package dashboard

import (
	"net/http"

	"github.com/GrayCodeAI/tokman/internal/ccusage"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func economicsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 30)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Calculate estimated cost savings
		// Using $3 per million input tokens (Claude pricing)
		var totalSaved int
		for _, r := range records {
			totalSaved += r.Saved
		}
		estimatedCost := float64(totalSaved) * 3.0 / 1_000_000

		response := map[string]any{
			"total_tokens_saved": totalSaved,
			"estimated_cost":     estimatedCost,
			"records_count":      len(records),
		}

		jsonResponse(w, http.StatusOK, response)
	}
}

// llmStatusHandler returns LLM integration status and usage data
func llmStatusHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if ccusage is available
		ccAvailable := ccusage.IsAvailable()

		response := map[string]any{
			"ccusage_available": ccAvailable,
		}

		// If ccusage is available, fetch real usage data
		if ccAvailable {
			// Get monthly usage
			monthly, err := ccusage.Fetch(ccusage.Monthly)
			if err == nil && len(monthly) > 0 {
				// Calculate totals and detect models used
				var totalCost float64
				var totalInput, totalOutput, totalCacheCreate, totalCacheRead uint64
				for _, m := range monthly {
					totalCost += m.Metrics.TotalCost
					totalInput += m.Metrics.InputTokens
					totalOutput += m.Metrics.OutputTokens
					totalCacheCreate += m.Metrics.CacheCreationTokens
					totalCacheRead += m.Metrics.CacheReadTokens
				}
				response["total_cost"] = totalCost
				response["total_input_tokens"] = totalInput
				response["total_output_tokens"] = totalOutput
				response["total_cache_create"] = totalCacheCreate
				response["total_cache_read"] = totalCacheRead
				response["monthly_data"] = monthly

				// Set provider based on ccusage availability (Claude Code uses ccusage)
				response["provider"] = "Claude Code"
				response["provider_model"] = "Detected from usage"
			}

			// Get daily usage for trend
			daily, err := ccusage.Fetch(ccusage.Daily)
			if err == nil && len(daily) > 0 {
				response["daily_data"] = daily
			}
		}

		// If no ccusage data, set generic values
		if _, ok := response["provider"]; !ok {
			response["provider"] = "Unknown"
			response["provider_model"] = "Install ccusage for LLM tracking"
		}

		jsonResponse(w, http.StatusOK, response)
	}
}

// modelBreakdownHandler returns LLM usage breakdown by model (from ccusage)
func modelBreakdownHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"available": false,
			"models":    []map[string]any{},
		}

		if !ccusage.IsAvailable() {
			jsonResponse(w, http.StatusOK, response)
			return
		}

		// Get monthly data which includes usage info
		monthly, err := ccusage.Fetch(ccusage.Monthly)
		if err != nil || len(monthly) == 0 {
			jsonResponse(w, http.StatusOK, response)
			return
		}

		// Aggregate by period (ccusage doesn't provide model breakdown, so group by month)
		models := []map[string]any{}
		for _, m := range monthly {
			models = append(models, map[string]any{
				"period":        m.Key,
				"total_cost":    m.Metrics.TotalCost,
				"input_tokens":  m.Metrics.InputTokens,
				"output_tokens": m.Metrics.OutputTokens,
				"cache_read":    m.Metrics.CacheReadTokens,
				"cache_create":  m.Metrics.CacheCreationTokens,
				"total_tokens":  m.Metrics.TotalTokens,
			})
		}

		response["available"] = true
		response["models"] = models
		jsonResponse(w, http.StatusOK, response)
	}
}
