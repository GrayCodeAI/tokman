package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/GrayCodeAI/tokman/internal/ccusage"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// LLMProvider represents detected LLM configuration
type LLMProvider struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Model      string  `json:"model"`
	BaseURL    string  `json:"base_url,omitempty"`
	CostInput  float64 `json:"cost_input_per_million"`
	CostOutput float64 `json:"cost_output_per_million"`
}

// ModelPricing contains pricing for common models
var modelPricing = map[string]struct {
	input  float64
	output float64
}{
	"claude-3-opus":     {15.0, 75.0},
	"claude-3-sonnet":   {3.0, 15.0},
	"claude-3-haiku":    {0.25, 1.25},
	"claude-3.5-sonnet": {3.0, 15.0},
	"claude-3.5-haiku":  {0.80, 4.0},
	"gpt-4-turbo":       {10.0, 30.0},
	"gpt-4":             {30.0, 60.0},
	"gpt-3.5-turbo":     {0.50, 1.50},
	"qwen2.5":           {0.35, 0.35},
	"qwen3":             {0.35, 0.35},
	"llama3":            {0.20, 0.20},
	"mistral":           {0.20, 0.60},
	"gemini":            {0.50, 1.50},
	"ollama":            {0.0, 0.0},
}

func economicsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 30)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		provider := detectLLMProvider()
		var totalSaved int
		for _, r := range records {
			totalSaved += r.Saved
		}

		costPerMillion := provider.CostInput
		if costPerMillion == 0 {
			costPerMillion = 3.0
		}
		estimatedCost := float64(totalSaved) * costPerMillion / 1_000_000

		response := map[string]any{
			"total_tokens_saved": totalSaved,
			"estimated_cost":     estimatedCost,
			"records_count":      len(records),
			"provider":           provider.Name,
			"provider_type":      provider.Type,
			"cost_per_million":   costPerMillion,
		}

		jsonResponse(w, http.StatusOK, response)
	}
}

func llmStatusHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		saved24h, _ := tracker.TokensSaved24h()
		savedTotal, _ := tracker.TokensSavedTotal()
		provider := detectLLMProvider()

		response := map[string]any{
			"provider":           provider.Name,
			"provider_type":      provider.Type,
			"model":              provider.Model,
			"tokens_saved_24h":   saved24h,
			"tokens_saved_total": savedTotal,
			"ccusage_available":  ccusage.IsAvailable(),
			"pricing": map[string]float64{
				"input_per_million":  provider.CostInput,
				"output_per_million": provider.CostOutput,
			},
		}

		if ccusage.IsAvailable() {
			response["usage_tracking"] = "ccusage"
		} else {
			response["usage_tracking"] = "internal"
		}

		jsonResponse(w, http.StatusOK, response)
	}
}

func modelBreakdownHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := detectLLMProvider()
		response := map[string]any{
			"provider":  provider.Name,
			"model":     provider.Model,
			"available": provider.Type != "unknown",
			"sessions":  []map[string]any{},
		}

		if !ccusage.IsAvailable() {
			jsonResponse(w, http.StatusOK, response)
			return
		}

		monthly, err := ccusage.Fetch(ccusage.Monthly)
		if err != nil || len(monthly) == 0 {
			jsonResponse(w, http.StatusOK, response)
			return
		}

		sessions := []map[string]any{}
		for _, m := range monthly {
			sessions = append(sessions, map[string]any{
				"period":        m.Key,
				"total_cost":    m.Metrics.TotalCost,
				"input_tokens":  m.Metrics.InputTokens,
				"output_tokens": m.Metrics.OutputTokens,
				"cache_read":    m.Metrics.CacheReadTokens,
			})
		}
		response["sessions"] = sessions
		jsonResponse(w, http.StatusOK, response)
	}
}

func detectLLMProvider() LLMProvider {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}

	// Check OpenCode first (most specific)
	opencodePath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if data, err := os.ReadFile(opencodePath); err == nil {
		var config map[string]any
		if json.Unmarshal(data, &config) == nil {
			providers, ok := config["provider"].(map[string]any)
			if ok && len(providers) > 0 {
				// Find the first provider with models
				for _, p := range providers {
					pmap, ok := p.(map[string]any)
					if !ok {
						continue
					}
					modelName := ""
					if models, ok := pmap["models"].(map[string]any); ok && len(models) > 0 {
						for m := range models {
							modelName = m
							break
						}
					}
					providerType := "custom"
					if npm, ok := pmap["npm"].(string); ok {
						if npm == "@ai-sdk/openai-compatible" {
							providerType = "ollama"
						}
					}
					baseURL := ""
					if opts, ok := pmap["options"].(map[string]any); ok {
						if url, ok := opts["baseURL"].(string); ok {
							baseURL = url
						}
					}
					// Determine pricing based on model type
					costInput := 0.0
					costOutput := 0.0
					if contains(modelName, "qwen") || contains(modelName, "llama") || contains(modelName, "mistral") {
						costInput = 0.35
						costOutput = 0.35
					} else if contains(modelName, "gpt-4") {
						costInput = 10.0
						costOutput = 30.0
					} else if contains(modelName, "gpt-3.5") {
						costInput = 0.50
						costOutput = 1.50
					}
					return LLMProvider{
						Name:       "OpenCode",
						Type:       providerType,
						Model:      modelName,
						BaseURL:    baseURL,
						CostInput:  costInput,
						CostOutput: costOutput,
					}
				}
			}
		}
	}

	// Check Claude Code
	claudePath := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(claudePath); err == nil {
		return LLMProvider{
			Name:       "Claude Code",
			Type:       "anthropic",
			Model:      "claude-3.5-sonnet",
			CostInput:  3.0,
			CostOutput: 15.0,
		}
	}

	// Check Cursor
	cursorPath := filepath.Join(home, ".cursor", "settings.json")
	if _, err := os.Stat(cursorPath); err == nil {
		return LLMProvider{
			Name:       "Cursor",
			Type:       "cursor",
			Model:      "claude-3.5-sonnet",
			CostInput:  3.0,
			CostOutput: 15.0,
		}
	}

	// Check Aider
	aiderPath := filepath.Join(home, ".aider", "aider.conf")
	if _, err := os.Stat(aiderPath); err == nil {
		return LLMProvider{
			Name:       "Aider",
			Type:       "aider",
			Model:      "unknown",
			CostInput:  0.0,
			CostOutput: 0.0,
		}
	}

	return LLMProvider{
		Name:       "Unknown",
		Type:       "unknown",
		Model:      "unknown",
		CostInput:  3.0,
		CostOutput: 15.0,
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && containsString(s, substr))
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getString(m map[string]any, path string) string {
	keys := splitPath(path)
	current := m
	for i, k := range keys {
		if i == len(keys)-1 {
			if v, ok := current[k].(string); ok {
				return v
			}
			return ""
		}
		if next, ok := current[k].(map[string]any); ok {
			current = next
		} else {
			return ""
		}
	}
	return ""
}

func getStringOr(m map[string]any, key string, fallback string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return fallback
}

func splitPath(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
