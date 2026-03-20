package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

var mcpPort int

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for Claude/ChatGPT integration",
	Long: `Start an HTTP server that accepts text and returns compressed output.
Compatible with MCP (Model Context Protocol) for tool use.

POST /compress — compress text
POST /explain  — explain compression
GET  /health   — health check

Example:
  tokman mcp --port 8080`,
	RunE: runMCP,
}

func init() {
	mcpCmd.Flags().IntVar(&mcpPort, "port", 8080, "MCP server port")
	rootCmd.AddCommand(mcpCmd)
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

func runMCP(cmd *cobra.Command, args []string) error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": Version})
	})

	// Compress endpoint
	mux.HandleFunc("/compress", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		var req MCPRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
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
		json.NewEncoder(w).Encode(resp)
	})

	// Explain endpoint
	mux.HandleFunc("/explain", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req MCPRequest
		json.Unmarshal(body, &req)

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

		resp := map[string]interface{}{
			"compressed":  compressed,
			"explanation": strings.Join(explanation, "\n"),
			"stats": map[string]interface{}{
				"original_tokens":   stats.OriginalTokens,
				"compressed_tokens": stats.FinalTokens,
				"saved_tokens":      stats.TotalSaved,
				"reduction_percent": stats.ReductionPercent,
			},
			"quality": map[string]interface{}{
				"score":             report.Score,
				"errors_preserved":  report.ErrorPreserved,
				"numbers_preserved": report.NumbersPreserved,
				"urls_preserved":    report.URLsPreserved,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Reverse/restore endpoint
	mux.HandleFunc("/restore", func(w http.ResponseWriter, r *http.Request) {
		store := filter.NewReversibleStore()
		hash := r.URL.Query().Get("hash")
		if hash == "" {
			http.Error(w, "hash required", http.StatusBadRequest)
			return
		}

		entry, err := store.Restore(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"original": entry.Original,
			"command":  entry.Command,
		})
	})

	addr := fmt.Sprintf(":%d", mcpPort)
	fmt.Fprintf(os.Stderr, "tokman MCP server listening on %s\n", addr)
	fmt.Fprintf(os.Stderr, "Endpoints: /compress, /explain, /restore, /health\n")

	return http.ListenAndServe(addr, mux)
}
