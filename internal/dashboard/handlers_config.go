package dashboard

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/GrayCodeAI/tokman/internal/ccusage"
	"github.com/GrayCodeAI/tokman/internal/contextread"
	"github.com/GrayCodeAI/tokman/internal/httpmw"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// alertsHandler returns alert status based on thresholds
func alertsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts := []map[string]any{}
		configMu.RLock()
		cfg := defaultConfig.Alerts
		configMu.RUnlock()

		if !cfg.Enabled {
			httpmw.JSONResponse(w, http.StatusOK, map[string]any{"enabled": false, "alerts": alerts})
			return
		}

		// Check daily token limit (best-effort)
		saved24h, _ := tracker.TokensSaved24h()
		if saved24h > cfg.DailyTokenLimit {
			alerts = append(alerts, map[string]any{
				"type":      "daily_limit",
				"severity":  "warning",
				"message":   fmt.Sprintf("Daily token savings (%d) exceeded limit (%d)", saved24h, cfg.DailyTokenLimit),
				"value":     saved24h,
				"threshold": cfg.DailyTokenLimit,
				"timestamp": time.Now().Format(time.RFC3339),
			})
		}

		// Check for usage spikes (compare last hour vs average)
		query := `
			SELECT
				COUNT(*) as count,
				AVG(hourly_count) as avg_count
			FROM (
				SELECT
					strftime('%Y-%m-%d %H', timestamp) as hour,
					COUNT(*) as hourly_count
				FROM commands
				WHERE timestamp >= datetime('now', '-24 hours')
				GROUP BY hour
			)
		`
		rows, err := tracker.Query(query)
		if err == nil {
			defer rows.Close()
			if rows.Next() {
				var count int
				var avgCount float64
				if scanErr := rows.Scan(&count, &avgCount); scanErr == nil && avgCount > 0 {
					// Get last hour count
					lastHourQuery := `
						SELECT COUNT(*) FROM commands
						WHERE timestamp >= datetime('now', '-1 hour')
					`
					lhRows, lhErr := tracker.Query(lastHourQuery)
					if lhErr == nil {
						defer lhRows.Close()
						if lhRows.Next() {
							var lastHourCount int
							if scanErr := lhRows.Scan(&lastHourCount); scanErr == nil {
								if float64(lastHourCount) > avgCount*cfg.UsageSpikeThreshold && avgCount > 0 {
									alerts = append(alerts, map[string]any{
										"type":      "usage_spike",
										"severity":  "info",
										"message":   fmt.Sprintf("Usage spike detected: %d commands in last hour vs %.1f avg", lastHourCount, avgCount),
										"value":     lastHourCount,
										"average":   avgCount,
										"timestamp": time.Now().Format(time.RFC3339),
									})
								}
							}
						}
					}
				}
			}
		}

		response := map[string]any{
			"enabled": true,
			"alerts":  alerts,
			"config": map[string]any{
				"daily_token_limit":     cfg.DailyTokenLimit,
				"weekly_token_limit":    cfg.WeeklyTokenLimit,
				"usage_spike_threshold": cfg.UsageSpikeThreshold,
			},
		}
		httpmw.JSONResponse(w, http.StatusOK, response)
	}
}

// exportCSVHandler exports data as CSV
func exportCSVHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetRecentCommands("", 1000)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=tokman-export.csv")
		writer := csv.NewWriter(w)
		defer writer.Flush()
		writer.Write([]string{"timestamp", "command", "tokens_saved", "original_tokens", "filtered_tokens", "exec_time_ms", "project_path"})
		for _, rec := range records {
			writer.Write([]string{
				rec.Timestamp.Format("2006-01-02 15:04:05"),
				rec.Command,
				fmt.Sprintf("%d", rec.SavedTokens),
				fmt.Sprintf("%d", rec.OriginalTokens),
				fmt.Sprintf("%d", rec.FilteredTokens),
				fmt.Sprintf("%d", rec.ExecTimeMs),
				rec.ProjectPath,
			})
		}
	}
}

// exportJSONHandler exports data as JSON
func exportJSONHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exportType := r.URL.Query().Get("type")
		if exportType == "" {
			exportType = "all"
		}

		response := map[string]any{
			"exported_at": time.Now().Format(time.RFC3339),
			"type":        exportType,
		}

		// Include stats (best-effort)
		stats, _ := tracker.GetSavings("")
		if stats != nil {
			response["stats"] = stats
		}

		// Include recent commands (best-effort)
		records, _ := tracker.GetRecentCommands("", 1000)
		if records != nil {
			response["commands"] = records
		}

		// Include daily breakdown (best-effort)
		daily, _ := tracker.GetDailySavings("", 30)
		if daily != nil {
			response["daily"] = daily
		}

		// Include command stats (best-effort)
		cmdStats, _ := tracker.GetCommandStats("")
		if cmdStats != nil {
			response["command_stats"] = cmdStats
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=tokman-export.json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("json encode error: %v", err)
		}
	}
}

// configHandler returns dashboard configuration
func configHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			configMu.RLock()
			cfg := defaultConfig
			configMu.RUnlock()
			httpmw.JSONResponse(w, http.StatusOK, cfg)
			return
		}

		// POST - update config (for future use)
		r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024) // 1MB limit for config
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		configMu.Lock()
		if newConfig.Port > 0 {
			defaultConfig.Port = newConfig.Port
		}
		if newConfig.Bind != "" {
			defaultConfig.Bind = newConfig.Bind
		}
		if newConfig.UpdateInterval > 0 {
			defaultConfig.UpdateInterval = newConfig.UpdateInterval
		}
		if newConfig.Alerts.Enabled || newConfig.Alerts.DailyTokenLimit > 0 {
			defaultConfig.Alerts = newConfig.Alerts
		}
		cfg := defaultConfig
		configMu.Unlock()

		httpmw.JSONResponse(w, http.StatusOK, cfg)
	}
}

// reportHandler generates summary reports
func reportHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reportType := r.URL.Query().Get("type")
		if reportType == "" {
			reportType = "weekly"
		}

		now := time.Now()
		response := map[string]any{
			"report_type":  reportType,
			"generated_at": now.Format(time.RFC3339),
		}

		var days int
		switch reportType {
		case "daily":
			days = 1
		case "weekly":
			days = 7
		case "monthly":
			days = 30
		default:
			days = 7
		}

		// Get stats for period
		daily, err := tracker.GetDailySavings("", days)
		if err == nil {
			var totalSaved, totalCommands, totalOriginal int
			for _, d := range daily {
				totalSaved += d.Saved
				totalCommands += d.Commands
				totalOriginal += d.Original
			}

			efficiency := 0.0
			if totalOriginal > 0 {
				efficiency = float64(totalSaved) / float64(totalOriginal) * 100
			}

			estimatedSavings := float64(totalSaved) * 3.0 / 1_000_000 // $3/1M tokens

			response["summary"] = map[string]any{
				"period_days":          days,
				"tokens_saved":         totalSaved,
				"commands_processed":   totalCommands,
				"efficiency_pct":       efficiency,
				"estimated_cost_saved": estimatedSavings,
			}

			response["daily_breakdown"] = daily
		}

		// Get top commands (best-effort)
		cmdStats, _ := tracker.GetCommandStats("")
		if len(cmdStats) > 0 {
			response["top_commands"] = cmdStats[:min(5, len(cmdStats))]
		}

		// Get project stats
		rows, err := tracker.Query(`
			SELECT project_path, COUNT(*), COALESCE(SUM(saved_tokens), 0)
			FROM commands
			WHERE timestamp >= datetime('now', '-' || ? || ' days')
			AND project_path IS NOT NULL AND project_path != ''
			GROUP BY project_path
			ORDER BY SUM(saved_tokens) DESC
			LIMIT 5
		`, days)
		if err == nil {
			defer rows.Close()
			projects := []map[string]any{}
			for rows.Next() {
				var path string
				var count, saved int
				if scanErr := rows.Scan(&path, &count, &saved); scanErr == nil {
					projects = append(projects, map[string]any{
						"project":      filepath.Base(path),
						"commands":     count,
						"tokens_saved": saved,
					})
				}
			}
			response["top_projects"] = projects
		}

		httpmw.JSONResponse(w, http.StatusOK, response)
	}
}

// cacheMetricsHandler returns cache efficiency metrics
func cacheMetricsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get total stats (best-effort)
		stats, _ := tracker.GetSavings("")
		if stats == nil {
			httpmw.JSONResponse(w, http.StatusOK, map[string]any{"error": "no data"})
			return
		}

		// Calculate cache efficiency
		efficiency := 0.0
		if stats.TotalOriginal > 0 {
			efficiency = float64(stats.TotalSaved) / float64(stats.TotalOriginal) * 100
		}

		// Get hourly distribution for cache patterns
		rows, err := tracker.Query(`
			SELECT
				strftime('%H', timestamp) as hour,
				COUNT(*) as count,
				AVG(saved_tokens) as avg_saved
			FROM commands
			WHERE timestamp >= datetime('now', '-7 days')
			GROUP BY hour
			ORDER BY hour
		`)
		hourlyPattern := []map[string]any{}
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var hour string
				var count int
				var avgSaved float64
				if scanErr := rows.Scan(&hour, &count, &avgSaved); scanErr == nil {
					hourlyPattern = append(hourlyPattern, map[string]any{
						"hour":      hour,
						"count":     count,
						"avg_saved": avgSaved,
					})
				}
			}
		}

		response := map[string]any{
			"total_original": stats.TotalOriginal,
			"total_filtered": stats.TotalFiltered,
			"total_saved":    stats.TotalSaved,
			"efficiency_pct": efficiency,
			"commands_count": stats.TotalCommands,
			"hourly_pattern": hourlyPattern,
		}
		contextCache := contextread.CacheStats()
		response["context_cache_entries"] = contextCache.Entries
		response["context_cache_hits"] = contextCache.Hits
		response["context_cache_misses"] = contextCache.Misses
		response["context_cache_hit_rate"] = contextCache.HitRate * 100

		// Add ccusage cache stats if available
		if ccusage.IsAvailable() {
			monthly, _ := ccusage.Fetch(ccusage.Monthly) // best-effort
			if len(monthly) > 0 {
				var totalCacheRead, totalCacheCreate uint64
				for _, m := range monthly {
					totalCacheRead += m.Metrics.CacheReadTokens
					totalCacheCreate += m.Metrics.CacheCreationTokens
				}
				response["cc_cache_read"] = totalCacheRead
				response["cc_cache_create"] = totalCacheCreate
			}
		}

		httpmw.JSONResponse(w, http.StatusOK, response)
	}
}
