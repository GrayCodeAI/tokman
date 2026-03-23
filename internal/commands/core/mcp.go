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
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/httpmw"
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

// MCPResponse is the response body.
type MCPResponse struct {
	Compressed       string  `json:"compressed"`
	OriginalTokens   int     `json:"original_tokens"`
	CompressedTokens int     `json:"compressed_tokens"`
	SavedTokens      int     `json:"saved_tokens"`
	ReductionPct     float64 `json:"reduction_percent"`
	Hash             string  `json:"hash,omitempty"`
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

func runMCP(cmd *cobra.Command, args []string) error {
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
			Hash:             filter.Fingerprint(req.Text),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("json encode error: %v", err)
		}
	})
	mux.Handle("/compress", contentTypeJSON(authMiddleware(mcpAPIKey, compressHandler)))

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

	addr := fmt.Sprintf(":%d", mcpPort)
	fmt.Fprintf(os.Stderr, "tokman MCP server listening on %s\n", addr)
	fmt.Fprintf(os.Stderr, "Endpoints: /compress, /explain, /restore, /health\n")

	srv := &http.Server{
		Addr:              addr,
		Handler:           rl.Middleware(mux),
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
