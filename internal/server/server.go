package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/httpmw"
)

const (
	maxRequestBodySize = 10 * 1024 * 1024 // 10MB
	defaultLayerCount  = 31
	defaultRateLimit   = 100 // requests per minute
)

// Server provides REST API for token compression
type Server struct {
	port     int
	apiKey   string
	version  string
	selector *filter.AdaptiveLayerSelector
	metrics  *Metrics
	logger   *Logger

	// Rate limiting
	rateLimiter *rateLimiter

	// Readiness state
	ready   bool
	readyMu sync.RWMutex
}

// rateLimiter implements a simple IP-based rate limiter
type rateLimiter struct {
	requests  map[string]*clientInfo
	mu        sync.RWMutex
	limit     int           // requests per minute
	window    time.Duration // time window
	lastClean time.Time
}

type clientInfo struct {
	count     int
	resetTime time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string]*clientInfo),
		limit:    limit,
		window:   time.Minute,
	}
}

func (rl *rateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if now.Sub(rl.lastClean) > time.Minute {
		for key, info := range rl.requests {
			if now.After(info.resetTime) {
				delete(rl.requests, key)
			}
		}
		rl.lastClean = now
	}

	info, exists := rl.requests[ip]

	if !exists || now.After(info.resetTime) {
		rl.requests[ip] = &clientInfo{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if info.count >= rl.limit {
		return false
	}

	info.count++
	return true
}

// Config holds server configuration
type Config struct {
	Port      int
	APIKey    string // Optional API key for authentication (empty = no auth)
	LogLevel  string // "debug", "info", "error"
	Version   string
	RateLimit int // Requests per minute (0 = unlimited)
}

// New creates a new server
func New(config Config) *Server {
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.RateLimit == 0 {
		config.RateLimit = defaultRateLimit
	}
	return &Server{
		port:        config.Port,
		apiKey:      config.APIKey,
		version:     config.Version,
		selector:    filter.NewAdaptiveLayerSelector(),
		metrics:     NewMetrics(),
		logger:      NewLogger(config.LogLevel),
		rateLimiter: newRateLimiter(config.RateLimit),
		ready:       true, // Start as ready
	}
}

// Start begins listening for requests
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/ready", s.handleHealthReady)

	// Compression endpoints (require JSON content type)
	contentTypeJSON := httpmw.RequireContentType("application/json")
	mux.Handle("/compress", contentTypeJSON(http.HandlerFunc(s.handleCompress)))
	mux.Handle("/compress/adaptive", contentTypeJSON(http.HandlerFunc(s.handleCompressAdaptive)))
	mux.Handle("/analyze", contentTypeJSON(http.HandlerFunc(s.handleAnalyze)))

	// Stats and metrics endpoints
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/metrics", s.handleMetrics)

	var handler http.Handler = mux
	// Apply rate limiting
	handler = s.rateLimitMiddleware(handler)
	// Apply auth middleware if API key is configured
	if s.apiKey != "" {
		handler = s.authMiddleware(handler)
	}
	handler = s.loggingMiddleware(handler)

	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Info("TokMan server starting", map[string]any{"port": s.port})

	if s.apiKey == "" {
		log.Println("WARNING: Server running without authentication - API is open to all clients")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	return nil
}

// rateLimitMiddleware enforces rate limiting per IP
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health checks are exempt from rate limiting
		if r.URL.Path == "/health" || r.URL.Path == "/health/ready" {
			next.ServeHTTP(w, r)
			return
		}

		// Use RemoteAddr only to prevent X-Forwarded-For spoofing
		ip := r.RemoteAddr
		// Strip port if present
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}

		if !s.rateLimiter.Allow(ip) {
			s.metrics.RecordError()
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		s.logger.Debug("request", map[string]any{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrapped.statusCode,
			"duration_ms": duration.Milliseconds(),
		})
	})
}

// authMiddleware validates API key on protected endpoints
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check is always accessible
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("Authorization")
		// Strip "Bearer " prefix
		if len(key) > 7 && key[:7] == "Bearer " {
			key = key[7:]
		}

		if subtle.ConstantTimeCompare([]byte(key), []byte(s.apiKey)) != 1 {
			s.metrics.RecordError()
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Response types

// HealthResponse for health checks
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// CompressRequest for compression API
type CompressRequest struct {
	Input  string `json:"input"`
	Mode   string `json:"mode,omitempty"`   // "minimal" or "aggressive"
	Budget int    `json:"budget,omitempty"` // Target token budget
}

// CompressResponse for compression results
type CompressResponse struct {
	Output           string  `json:"output"`
	OriginalTokens   int     `json:"original_tokens"`
	FinalTokens      int     `json:"final_tokens"`
	TokensSaved      int     `json:"tokens_saved"`
	ReductionPercent float64 `json:"reduction_percent"`
	ProcessingTimeMs int64   `json:"processing_time_ms"`
}

// AnalyzeResponse for content analysis
type AnalyzeResponse struct {
	ContentType string `json:"content_type"`
}

// StatsResponse for server stats
type StatsResponse struct {
	Version    string `json:"version"`
	LayerCount int    `json:"layer_count"`
}

// Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: s.version,
	})
}

func (s *Server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	s.readyMu.RLock()
	ready := s.ready
	s.readyMu.RUnlock()

	if !ready {
		http.Error(w, `{"status":"not ready"}`, http.StatusServiceUnavailable)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"status":  "ready",
		"version": s.version,
	})
}

// SetReady sets the readiness state for health checks
func (s *Server) SetReady(ready bool) {
	s.readyMu.Lock()
	s.ready = ready
	s.readyMu.Unlock()
}

func (s *Server) handleCompress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.metrics.RecordError()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req CompressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.metrics.RecordError()
		s.logger.Error("invalid JSON", map[string]any{"error": err.Error()})
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Input == "" {
		s.metrics.RecordError()
		http.Error(w, "input required", http.StatusBadRequest)
		return
	}

	// Set defaults
	mode := filter.ModeMinimal
	if req.Mode == "aggressive" {
		mode = filter.ModeAggressive
	}
	budget := req.Budget
	if budget == 0 {
		budget = 4000
	}

	// Process
	start := time.Now()
	config := filter.PipelineConfig{
		Mode:                mode,
		Budget:              budget,
		SessionTracking:     true,
		NgramEnabled:        true,
		EnableCompaction:    true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	}
	coordinator := filter.NewPipelineCoordinator(config)
	output, stats := coordinator.Process(req.Input)
	elapsed := time.Since(start)

	// Record metrics
	s.metrics.RecordCompression(stats.OriginalTokens, stats.FinalTokens, elapsed, "unknown")

	s.logger.Info("compression", map[string]any{
		"original_tokens": stats.OriginalTokens,
		"final_tokens":    stats.FinalTokens,
		"reduction_pct":   stats.ReductionPercent,
		"processing_ms":   elapsed.Milliseconds(),
	})

	jsonResponse(w, http.StatusOK, CompressResponse{
		Output:           output,
		OriginalTokens:   stats.OriginalTokens,
		FinalTokens:      stats.FinalTokens,
		TokensSaved:      stats.TotalSaved,
		ReductionPercent: stats.ReductionPercent,
		ProcessingTimeMs: elapsed.Milliseconds(),
	})
}

func (s *Server) handleCompressAdaptive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.metrics.RecordError()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req CompressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.metrics.RecordError()
		s.logger.Error("invalid JSON", map[string]any{"error": err.Error()})
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Input == "" {
		s.metrics.RecordError()
		http.Error(w, "input required", http.StatusBadRequest)
		return
	}

	mode := filter.ModeMinimal
	if req.Mode == "aggressive" {
		mode = filter.ModeAggressive
	}

	start := time.Now()
	coordinator := s.selector.OptimizePipeline(req.Input, mode)
	output, stats := coordinator.Process(req.Input)
	elapsed := time.Since(start)

	// Detect content type for metrics
	contentType := s.selector.AnalyzeContent(req.Input).String()
	s.metrics.RecordCompression(stats.OriginalTokens, stats.FinalTokens, elapsed, contentType)

	s.logger.Info("adaptive compression", map[string]any{
		"content_type":    contentType,
		"original_tokens": stats.OriginalTokens,
		"final_tokens":    stats.FinalTokens,
		"reduction_pct":   stats.ReductionPercent,
		"processing_ms":   elapsed.Milliseconds(),
	})

	jsonResponse(w, http.StatusOK, CompressResponse{
		Output:           output,
		OriginalTokens:   stats.OriginalTokens,
		FinalTokens:      stats.FinalTokens,
		TokensSaved:      stats.TotalSaved,
		ReductionPercent: stats.ReductionPercent,
		ProcessingTimeMs: elapsed.Milliseconds(),
	})
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req CompressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Input == "" {
		http.Error(w, "input required", http.StatusBadRequest)
		return
	}

	ct := s.selector.AnalyzeContent(req.Input)
	jsonResponse(w, http.StatusOK, AnalyzeResponse{
		ContentType: ct.String(),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	snapshot := s.metrics.Snapshot()
	jsonResponse(w, http.StatusOK, map[string]any{
		"version":            s.version,
		"layer_count":        defaultLayerCount,
		"total_requests":     snapshot.TotalRequests,
		"total_compressions": snapshot.TotalCompressions,
		"total_tokens_saved": snapshot.TotalTokensSaved,
		"avg_reduction_pct":  snapshot.AvgReductionPct,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(s.metrics.PrometheusFormat()))
}

// Helper

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}
