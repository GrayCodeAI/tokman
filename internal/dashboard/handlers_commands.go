package dashboard

import (
	"net/http"

	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func commandsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetCommandStats("")
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		// Convert to chart format
		limit := 5
		if len(stats) < limit {
			limit = len(stats)
		}
		result := make([]map[string]any, limit)
		for i := 0; i < limit; i++ {
			result[i] = map[string]any{
				"command":      stats[i].Command,
				"tokens_saved": stats[i].TotalSaved,
				"executions":   stats[i].ExecutionCount,
			}
		}
		jsonResponse(w, http.StatusOK, result)
	}
}

// recentHandler returns recent command activity
func recentHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		records, err := tracker.GetRecentCommands("", limit)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		result := make([]map[string]any, len(records))
		for i, r := range records {
			result[i] = map[string]any{
				"command":       r.Command,
				"tokens_saved":  r.SavedTokens,
				"original":      r.OriginalTokens,
				"filtered":      r.FilteredTokens,
				"exec_time_ms":  r.ExecTimeMs,
				"timestamp":     r.Timestamp,
				"project_path":  r.ProjectPath,
				"parse_success": r.ParseSuccess,
			}
		}
		jsonResponse(w, http.StatusOK, result)
	}
}

// failuresHandler returns parse failure statistics
func failuresHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := tracker.GetParseFailureSummary()
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		response := map[string]any{
			"total_failures": summary.Total,
			"recovery_rate":  summary.RecoveryRate,
			"top_failures":   summary.TopCommands,
		}
		jsonResponse(w, http.StatusOK, response)
	}
}
