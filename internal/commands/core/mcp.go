package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/cache"
	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/contextread"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/httpmw"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

const maxRequestBodySize = 10 * 1024 * 1024 // 10 MB

var mcpPort int
var mcpAPIKey string

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for Claude/ChatGPT integration",
	Long: `Start an HTTP server that accepts text and returns compressed output.
Compatible with MCP (Model Context Protocol) for tool use.

POST /compress — compress text
POST /read     — read file context with smart modes
POST /bundle   — graph-aware file bundle for agents
POST /explain  — explain compression
GET  /health   — health check

Example:
  tokman mcp --port 8080
  tokman mcp --port 8080 --api-key mysecret`,
	RunE: runMCP,
}

func init() {
	mcpCmd.Flags().IntVar(&mcpPort, "port", 8080, "MCP server port")
	mcpCmd.Flags().StringVar(&mcpAPIKey, "api-key", "", "API key for authentication (disabled if empty)")
	registry.Add(func() { registry.Register(mcpCmd) })
}

// MCPRequest is the request body for /compress.
type MCPRequest struct {
	Text    string `json:"text"`
	Command string `json:"command,omitempty"`
	Budget  int    `json:"budget,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Query   string `json:"query,omitempty"`
}

// MCPReadRequest is the request body for /read.
type MCPReadRequest struct {
	Path         string `json:"path"`
	Mode         string `json:"mode,omitempty"`
	Level        string `json:"level,omitempty"`
	StartLine    int    `json:"start_line,omitempty"`
	EndLine      int    `json:"end_line,omitempty"`
	MaxLines     int    `json:"max_lines,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	LineNumbers  bool   `json:"line_numbers,omitempty"`
	SaveSnapshot bool   `json:"save_snapshot,omitempty"`
	RelatedFiles int    `json:"related_files,omitempty"`
}

// MCPResponse is the response body.
type MCPResponse struct {
	Compressed       string  `json:"compressed"`
	OriginalTokens   int     `json:"original_tokens"`
	CompressedTokens int     `json:"compressed_tokens"`
	SavedTokens      int     `json:"saved_tokens"`
	ReductionPct     float64 `json:"reduction_percent"`
	Hash             string  `json:"hash,omitempty"`
}

// MCPReadResponse is the response body for /read.
type MCPReadResponse struct {
	Path           string  `json:"path"`
	Mode           string  `json:"mode"`
	Content        string  `json:"content"`
	OriginalTokens int     `json:"original_tokens"`
	FinalTokens    int     `json:"final_tokens"`
	SavedTokens    int     `json:"saved_tokens"`
	ReductionPct   float64 `json:"reduction_percent"`
}

// MCPBundleResponse returns a graph-selected context bundle.
type MCPBundleResponse struct {
	Path           string   `json:"path"`
	RelatedFiles   []string `json:"related_files"`
	Content        string   `json:"content"`
	OriginalTokens int      `json:"original_tokens"`
	FinalTokens    int      `json:"final_tokens"`
	SavedTokens    int      `json:"saved_tokens"`
	ReductionPct   float64  `json:"reduction_percent"`
}

func authMiddleware(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	if apiKey == "" {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func newMCPHandler(apiKey string) http.Handler {
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": shared.Version}); err != nil {
			log.Printf("json encode error: %v", err)
		}
	})

	// Compress endpoint
	contentTypeJSON := httpmw.RequireContentType("application/json")
	compressHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var req MCPRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Text == "" {
			http.Error(w, "text required", http.StatusBadRequest)
			return
		}

		mode := filter.ModeMinimal
		if req.Mode == "aggressive" {
			mode = filter.ModeAggressive
		}

		pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
			Mode:                mode,
			Budget:              req.Budget,
			QueryIntent:         req.Query,
			SessionTracking:     false,
			NgramEnabled:        true,
			EnableCompaction:    true,
			EnableAttribution:   true,
			EnableH2O:           true,
			EnableAttentionSink: true,
		})

		compressed, stats := pipeline.Process(req.Text)

		resp := MCPResponse{
			Compressed:       compressed,
			OriginalTokens:   stats.OriginalTokens,
			CompressedTokens: stats.FinalTokens,
			SavedTokens:      stats.TotalSaved,
			ReductionPct:     stats.ReductionPercent,
			Hash:             cache.ComputeFingerprint(req.Text),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode error: %v", err)
		}
	})
	mux.Handle("/compress", contentTypeJSON(authMiddleware(mcpAPIKey, compressHandler)))

	mux.Handle("/read", contentTypeJSON(authMiddleware(mcpAPIKey, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var req MCPReadRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Path == "" {
			http.Error(w, "path required", http.StatusBadRequest)
			return
		}

		cleanPath := filepath.Clean(req.Path)
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			http.Error(w, "unable to read file", http.StatusBadRequest)
			return
		}

		mode := req.Mode
		if mode == "" {
			mode = "auto"
		}
		level := req.Level
		if level == "" {
			level = "minimal"
		}

		content, originalTokens, finalTokens, err := contextread.Build(cleanPath, string(data), contextread.Options{
			Level:        level,
			Mode:         mode,
			MaxLines:     req.MaxLines,
			MaxTokens:    req.MaxTokens,
			LineNumbers:  req.LineNumbers,
			StartLine:    req.StartLine,
			EndLine:      req.EndLine,
			SaveSnapshot: req.SaveSnapshot,
			RelatedFiles: req.RelatedFiles,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		saved := originalTokens - finalTokens
		if saved < 0 {
			saved = 0
		}
		resp := MCPReadResponse{
			Path:           cleanPath,
			Mode:           mode,
			Content:        content,
			OriginalTokens: originalTokens,
			FinalTokens:    finalTokens,
			SavedTokens:    saved,
			ReductionPct:   percentSaved(originalTokens, finalTokens),
		}

		if tracker := tracking.GetGlobalTracker(); tracker != nil {
			cwd, err := os.Getwd()
			if err == nil {
				meta := contextread.Describe("mcp", cleanPath, string(data), contextread.Options{
					Level:        level,
					Mode:         mode,
					MaxLines:     req.MaxLines,
					MaxTokens:    req.MaxTokens,
					LineNumbers:  req.LineNumbers,
					StartLine:    req.StartLine,
					EndLine:      req.EndLine,
					SaveSnapshot: req.SaveSnapshot,
					RelatedFiles: req.RelatedFiles,
				})
				_ = tracker.Record(&tracking.CommandRecord{
					Command:             fmt.Sprintf("tokman mcp read %s", cleanPath),
					OriginalTokens:      originalTokens,
					FilteredTokens:      finalTokens,
					SavedTokens:         saved,
					ProjectPath:         cwd,
					ParseSuccess:        true,
					ContextKind:         meta.Kind,
					ContextMode:         meta.RequestedMode,
					ContextResolvedMode: meta.ResolvedMode,
					ContextTarget:       meta.Target,
					ContextRelatedFiles: meta.RelatedFiles,
					ContextBundle:       meta.Bundle,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode error: %v", err)
		}
	}))))

	mux.Handle("/bundle", contentTypeJSON(authMiddleware(mcpAPIKey, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var req MCPReadRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Path == "" {
			http.Error(w, "path required", http.StatusBadRequest)
			return
		}

		cleanPath := filepath.Clean(req.Path)
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			http.Error(w, "unable to read file", http.StatusBadRequest)
			return
		}

		mode := req.Mode
		if mode == "" {
			mode = "graph"
		}
		if mode != "graph" {
			http.Error(w, "bundle mode requires graph", http.StatusBadRequest)
			return
		}

		bundle, err := contextread.BuildBundle(cleanPath, string(data), contextread.Options{
			Level:        req.Level,
			Mode:         mode,
			MaxLines:     req.MaxLines,
			MaxTokens:    req.MaxTokens,
			LineNumbers:  req.LineNumbers,
			StartLine:    req.StartLine,
			EndLine:      req.EndLine,
			SaveSnapshot: req.SaveSnapshot,
			RelatedFiles: req.RelatedFiles,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		saved := bundle.OriginalTokens - bundle.FinalTokens
		if saved < 0 {
			saved = 0
		}
		resp := MCPBundleResponse{
			Path:           bundle.TargetFile,
			RelatedFiles:   bundle.RelatedFiles,
			Content:        bundle.Content,
			OriginalTokens: bundle.OriginalTokens,
			FinalTokens:    bundle.FinalTokens,
			SavedTokens:    saved,
			ReductionPct:   percentSaved(bundle.OriginalTokens, bundle.FinalTokens),
		}

		if tracker := tracking.GetGlobalTracker(); tracker != nil {
			cwd, err := os.Getwd()
			if err == nil {
				_ = tracker.Record(&tracking.CommandRecord{
					Command:             fmt.Sprintf("tokman mcp bundle %s", cleanPath),
					OriginalTokens:      bundle.OriginalTokens,
					FilteredTokens:      bundle.FinalTokens,
					SavedTokens:         saved,
					ProjectPath:         cwd,
					ParseSuccess:        true,
					ContextKind:         "mcp",
					ContextMode:         "graph",
					ContextResolvedMode: "graph",
					ContextTarget:       cleanPath,
					ContextRelatedFiles: len(bundle.RelatedFiles),
					ContextBundle:       true,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode error: %v", err)
		}
	}))))

	// Explain endpoint
	mux.HandleFunc("/explain", authMiddleware(mcpAPIKey, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var req MCPRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Text == "" {
			http.Error(w, "text required", http.StatusBadRequest)
			return
		}

		pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
			Mode:                filter.ModeMinimal,
			QueryIntent:         req.Query,
			SessionTracking:     false,
			NgramEnabled:        true,
			EnableCompaction:    true,
			EnableAttribution:   true,
			EnableH2O:           true,
			EnableAttentionSink: true,
		})

		compressed, stats := pipeline.Process(req.Text)

		// Build explanation
		var explanation []string
		explanation = append(explanation, fmt.Sprintf("Input: %d tokens", stats.OriginalTokens))
		explanation = append(explanation, fmt.Sprintf("Output: %d tokens", stats.FinalTokens))
		explanation = append(explanation, fmt.Sprintf("Saved: %d tokens (%.1f%%)", stats.TotalSaved, stats.ReductionPercent))
		explanation = append(explanation, "")
		explanation = append(explanation, "Layer breakdown:")

		for name, stat := range stats.LayerStats {
			if stat.TokensSaved > 0 {
				explanation = append(explanation, fmt.Sprintf("  %s: -%d tokens", name, stat.TokensSaved))
			}
		}

		equiv := filter.NewSemanticEquivalence()
		report := equiv.Check(req.Text, compressed)

		explanation = append(explanation, "")
		explanation = append(explanation, fmt.Sprintf("Quality score: %.1f%%", report.Score*100))

		resp := map[string]any{
			"compressed":  compressed,
			"explanation": strings.Join(explanation, "\n"),
			"stats": map[string]any{
				"original_tokens":   stats.OriginalTokens,
				"compressed_tokens": stats.FinalTokens,
				"saved_tokens":      stats.TotalSaved,
				"reduction_percent": stats.ReductionPercent,
			},
			"quality": map[string]any{
				"score":             report.Score,
				"errors_preserved":  report.ErrorPreserved,
				"numbers_preserved": report.NumbersPreserved,
				"urls_preserved":    report.URLsPreserved,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode error: %v", err)
		}
	}))

	// Reverse/restore endpoint
	mux.HandleFunc("/restore", authMiddleware(mcpAPIKey, func(w http.ResponseWriter, r *http.Request) {
		store := filter.NewReversibleStore()
		hash := r.URL.Query().Get("hash")
		if hash == "" {
			http.Error(w, "hash required", http.StatusBadRequest)
			return
		}

		entry, err := store.Restore(hash)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"original": entry.Original,
			"command":  entry.Command,
		}); err != nil {
			log.Printf("json encode error: %v", err)
		}
	}))

	rl := httpmw.NewDefault()
	return rl.Middleware(mux)
}

func runMCP(cmd *cobra.Command, args []string) error {
	addr := fmt.Sprintf(":%d", mcpPort)
	fmt.Fprintf(os.Stderr, "tokman MCP server listening on %s\n", addr)
	fmt.Fprintf(os.Stderr, "Endpoints: /compress, /read, /bundle, /explain, /restore, /health\n")

	srv := &http.Server{
		Addr:              addr,
		Handler:           newMCPHandler(mcpAPIKey),
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

func percentSaved(original, final int) float64 {
	if original <= 0 {
		return 0
	}
	saved := original - final
	if saved < 0 {
		saved = 0
	}
	return float64(saved) / float64(original) * 100
}
