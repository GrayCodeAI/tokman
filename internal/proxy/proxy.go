package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

// APIFormat represents the LLM API format being proxied.
type APIFormat int

const (
	APIFormatOpenAI APIFormat = iota
	APIFormatAnthropic
	APIFormatGemini
	APIFormatUnknown
)

// Proxy handles HTTP proxying for LLM API calls.
type Proxy struct {
	mu           sync.RWMutex
	compressor   *filter.PipelineCoordinator
	stats        *ProxyStats
	targetURL    string
	listenAddr   string
	modelAliases map[string]string
	requestCache *requestCache
}

// ProxyStats tracks proxy usage statistics.
type ProxyStats struct {
	mu                sync.Mutex
	TotalRequests     int64
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalSavedTokens  int64
	ByFormat          map[APIFormat]int64
	ByModel           map[string]int64
	StartTime         time.Time
}

// requestCache caches identical requests.
type requestCache struct {
	mu    sync.RWMutex
	items map[string]*cachedResponse
	ttl   time.Duration
}

type cachedResponse struct {
	body      []byte
	headers   http.Header
	status    int
	expiresAt time.Time
}

// NewProxy creates a new HTTP proxy instance.
func NewProxy(listenAddr, targetURL string) *Proxy {
	cfg := filter.PipelineConfig{
		Mode:                   filter.ModeMinimal,
		SessionTracking:        true,
		NgramEnabled:           true,
		EnableCompaction:       true,
		EnableAttribution:      true,
		EnableH2O:              true,
		EnableAttentionSink:    true,
		EnableMetaToken:        true,
		EnableSemanticChunk:    true,
		EnableSketchStore:      true,
		EnableLazyPruner:       true,
		EnableSemanticAnchor:   true,
		EnableAgentMemory:      true,
		EnableTFIDF:            true,
		EnableSymbolicCompress: true,
		EnablePhraseGrouping:   true,
		EnableNumericalQuant:   true,
		EnableDynamicRatio:     true,
	}
	return &Proxy{
		compressor: filter.NewPipelineCoordinator(cfg),
		stats: &ProxyStats{
			ByFormat:  make(map[APIFormat]int64),
			ByModel:   make(map[string]int64),
			StartTime: time.Now(),
		},
		targetURL:    targetURL,
		listenAddr:   listenAddr,
		modelAliases: make(map[string]string),
		requestCache: &requestCache{
			items: make(map[string]*cachedResponse),
			ttl:   5 * time.Minute,
		},
	}
}

// SetModelAlias adds a model alias mapping.
func (p *Proxy) SetModelAlias(from, to string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.modelAliases[from] = to
}

// ServeHTTP implements http.Handler.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check
	if r.URL.Path == "/health" {
		p.handleHealth(w)
		return
	}

	// Metrics endpoint
	if r.URL.Path == "/metrics" {
		p.handleMetrics(w)
		return
	}

	// Detect API format
	apiFormat := detectAPIFormat(r)

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Check request cache
	if apiFormat == APIFormatOpenAI || apiFormat == APIFormatAnthropic {
		cacheKey := requestCacheKey(bodyBytes, r.URL.Path)
		if cached := p.requestCache.get(cacheKey); cached != nil {
			p.stats.record(apiFormat, 0, 0, 0, "")
			for k, v := range cached.headers {
				w.Header()[k] = v
			}
			w.WriteHeader(cached.status)
			_, _ = w.Write(cached.body)
			return
		}
	}

	// Compress request messages
	compressedBody, inputTokens, outputTokens := p.compressRequest(bodyBytes, apiFormat)

	// Create new request to target
	targetURL := p.targetURL + r.URL.Path
	targetReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(compressedBody))
	if err != nil {
		http.Error(w, "Failed to create target request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for k, v := range r.Header {
		if k != "Host" {
			targetReq.Header[k] = v
		}
	}
	targetReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(compressedBody)))

	// Apply model alias
	if apiFormat == APIFormatOpenAI {
		targetReq = p.applyModelAlias(targetReq)
	}

	// Forward request
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(targetReq)
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusBadGateway)
		return
	}

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)

	// Cache response
	if resp.StatusCode == http.StatusOK {
		cacheKey := requestCacheKey(bodyBytes, r.URL.Path)
		p.requestCache.set(cacheKey, respBody, resp.Header, resp.StatusCode)
	}

	// Record stats
	model := extractModelName(bodyBytes, apiFormat)
	p.stats.record(apiFormat, inputTokens, outputTokens, inputTokens-outputTokens, model)
}

func (p *Proxy) compressRequest(body []byte, format APIFormat) ([]byte, int, int) {
	switch format {
	case APIFormatOpenAI:
		return p.compressOpenAI(body)
	case APIFormatAnthropic:
		return p.compressAnthropic(body)
	case APIFormatGemini:
		return p.compressGemini(body)
	default:
		return body, 0, 0
	}
}

func (p *Proxy) compressOpenAI(body []byte) ([]byte, int, int) {
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
		Tools     []any `json:"tools,omitempty"`
		Stream    bool  `json:"stream,omitempty"`
		MaxTokens int   `json:"max_tokens,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return body, 0, 0
	}

	totalInput := 0
	totalOutput := 0

	for i := range req.Messages {
		content, ok := req.Messages[i].Content.(string)
		if !ok {
			continue
		}
		origTokens := core.EstimateTokens(content)
		totalInput += origTokens

		compressed, stats := p.compressor.Process(content)
		req.Messages[i].Content = compressed
		totalOutput += origTokens - stats.TotalSaved
	}

	newBody, err := json.Marshal(req)
	if err != nil {
		return body, totalInput, totalOutput
	}
	return newBody, totalInput, totalOutput
}

func (p *Proxy) compressAnthropic(body []byte) ([]byte, int, int) {
	var req struct {
		Model    string `json:"model"`
		System   any    `json:"system"`
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return body, 0, 0
	}

	totalInput := 0
	totalOutput := 0

	// Compress system prompt
	if sysStr, ok := req.System.(string); ok {
		origTokens := core.EstimateTokens(sysStr)
		totalInput += origTokens
		compressed, stats := p.compressor.Process(sysStr)
		req.System = compressed
		totalOutput += origTokens - stats.TotalSaved
	}

	for i := range req.Messages {
		content, ok := req.Messages[i].Content.(string)
		if !ok {
			continue
		}
		origTokens := core.EstimateTokens(content)
		totalInput += origTokens
		compressed, stats := p.compressor.Process(content)
		req.Messages[i].Content = compressed
		totalOutput += origTokens - stats.TotalSaved
	}

	newBody, err := json.Marshal(req)
	if err != nil {
		return body, totalInput, totalOutput
	}
	return newBody, totalInput, totalOutput
}

func (p *Proxy) compressGemini(body []byte) ([]byte, int, int) {
	// Gemini uses a different format with contents array
	var req struct {
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
		SystemInstruction struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"systemInstruction"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return body, 0, 0
	}

	totalInput := 0
	totalOutput := 0

	for i := range req.Contents {
		for j := range req.Contents[i].Parts {
			content := req.Contents[i].Parts[j].Text
			origTokens := core.EstimateTokens(content)
			totalInput += origTokens
			compressed, stats := p.compressor.Process(content)
			req.Contents[i].Parts[j].Text = compressed
			totalOutput += origTokens - stats.TotalSaved
		}
	}

	newBody, err := json.Marshal(req)
	if err != nil {
		return body, totalInput, totalOutput
	}
	return newBody, totalInput, totalOutput
}

func (p *Proxy) applyModelAlias(req *http.Request) *http.Request {
	p.mu.RLock()
	defer p.mu.RUnlock()

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return req
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return req
	}

	if model, ok := body["model"].(string); ok {
		if alias, exists := p.modelAliases[model]; exists {
			body["model"] = alias
			newBody, err := json.Marshal(body)
			if err != nil {
				return req
			}
			req.Body = io.NopCloser(bytes.NewReader(newBody))
			req.ContentLength = int64(len(newBody))
		}
	}

	return req
}

func (p *Proxy) handleHealth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"uptime":   time.Since(p.stats.StartTime).String(),
		"requests": p.stats.TotalRequests,
	})
}

func (p *Proxy) handleMetrics(w http.ResponseWriter) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"total_requests":      p.stats.TotalRequests,
		"total_input_tokens":  p.stats.TotalInputTokens,
		"total_output_tokens": p.stats.TotalOutputTokens,
		"total_saved_tokens":  p.stats.TotalSavedTokens,
		"savings_percent":     float64(p.stats.TotalSavedTokens) / float64(p.stats.TotalInputTokens) * 100,
		"by_model":            p.stats.ByModel,
		"uptime":              time.Since(p.stats.StartTime).String(),
	})
}

func (p *Proxy) Stats() *ProxyStats {
	return p.stats
}

func (p *Proxy) ListenAddr() string {
	return p.listenAddr
}

// requestCache methods
func (rc *requestCache) get(key string) *cachedResponse {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	item, ok := rc.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		return nil
	}
	return item
}

func (rc *requestCache) set(key string, body []byte, headers http.Header, status int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.items[key] = &cachedResponse{
		body:      body,
		headers:   headers,
		status:    status,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

// Stats methods
func (s *ProxyStats) record(format APIFormat, input, output, saved int, model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRequests++
	s.TotalInputTokens += int64(input)
	s.TotalOutputTokens += int64(output)
	s.TotalSavedTokens += int64(saved)
	s.ByFormat[format]++
	if model != "" {
		s.ByModel[model]++
	}
}

// Helper functions
func detectAPIFormat(r *http.Request) APIFormat {
	path := r.URL.Path
	if strings.Contains(path, "/v1/chat/completions") || strings.Contains(path, "/chat/completions") {
		return APIFormatOpenAI
	}
	if strings.Contains(path, "/v1/messages") || strings.Contains(path, "/messages") {
		return APIFormatAnthropic
	}
	if strings.Contains(path, "/v1beta") || strings.Contains(path, "/generateContent") {
		return APIFormatGemini
	}
	return APIFormatUnknown
}

func requestCacheKey(body []byte, path string) string {
	return fmt.Sprintf("%x", []byte(fmt.Sprintf("%s:%s", path, string(body))))
}

func extractModelName(body []byte, format APIFormat) string {
	switch format {
	case APIFormatOpenAI:
		var req struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &req)
		return req.Model
	case APIFormatAnthropic:
		var req struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &req)
		return req.Model
	default:
		return ""
	}
}
