package dashboard

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/ccusage"
	"github.com/GrayCodeAI/tokman/internal/economics"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// AlertConfig holds alert threshold configuration
type AlertConfig struct {
	DailyTokenLimit     int64   `json:"daily_token_limit"`
	WeeklyTokenLimit    int64   `json:"weekly_token_limit"`
	UsageSpikeThreshold float64 `json:"usage_spike_threshold"` // multiplier for spike detection
	Enabled             bool    `json:"enabled"`
}

// Config holds dashboard configuration
type Config struct {
	Port             int         `json:"port"`
	Bind             string      `json:"bind"`
	UpdateInterval   int         `json:"update_interval"`
	Theme            string      `json:"theme"`
	Alerts           AlertConfig `json:"alerts"`
	EnableExport     bool        `json:"enable_export"`
	HistoryRetention int         `json:"history_retention"`
}

// DefaultConfig returns default dashboard configuration
var defaultConfig = Config{
	Port:             8080,
	Bind:             "localhost",
	UpdateInterval:   30000,
	Theme:            "dark",
	EnableExport:     true,
	HistoryRetention: 90,
	Alerts: AlertConfig{
		DailyTokenLimit:     1000000,
		WeeklyTokenLimit:    5000000,
		UsageSpikeThreshold: 2.0,
		Enabled:             true,
	},
}

var (
	Port int
	Open bool
	Bind string
)

// Cmd returns the dashboard cobra command for registration.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch web dashboard for token savings visualization",
		Long: `Launch an interactive web dashboard to visualize token savings,
economics metrics, and usage trends over time.

The dashboard provides:
- Real-time token savings charts
- Daily/weekly/monthly breakdowns
- Command-level analytics
- Cost tracking with Claude API rates
- LLM usage integration via ccusage`,
		RunE: runDashboard,
	}

	cmd.Flags().IntVarP(&Port, "port", "p", 8080, "Port to run dashboard on")
	cmd.Flags().BoolVarP(&Open, "open", "o", false, "Open browser automatically")
	cmd.Flags().StringVar(&Bind, "bind", "localhost", "Address to bind server to (e.g., 0.0.0.0 for all interfaces)")

	return cmd
}

func runDashboard(cmd *cobra.Command, args []string) error {
	dbPath := tracking.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer tracker.Close()

	// API handlers
	http.HandleFunc("/api/stats", corsMiddleware(statsHandler(tracker)))
	http.HandleFunc("/api/daily", corsMiddleware(dailyHandler(tracker)))
	http.HandleFunc("/api/weekly", corsMiddleware(weeklyHandler(tracker)))
	http.HandleFunc("/api/monthly", corsMiddleware(monthlyHandler(tracker)))
	http.HandleFunc("/api/commands", corsMiddleware(commandsHandler(tracker)))
	http.HandleFunc("/api/recent", corsMiddleware(recentHandler(tracker)))
	http.HandleFunc("/api/economics", corsMiddleware(economicsHandler(tracker)))
	http.HandleFunc("/api/performance", corsMiddleware(performanceHandler(tracker)))
	http.HandleFunc("/api/failures", corsMiddleware(failuresHandler(tracker)))
	http.HandleFunc("/api/top-commands", corsMiddleware(topCommandsHandler(tracker)))
	http.HandleFunc("/api/hourly", corsMiddleware(hourlyHandler(tracker)))
	http.HandleFunc("/api/export/csv", corsMiddleware(exportCSVHandler(tracker)))
	// New endpoints for enhanced dashboard
	http.HandleFunc("/api/llm-status", corsMiddleware(llmStatusHandler(tracker)))
	http.HandleFunc("/api/daily-breakdown", corsMiddleware(dailyBreakdownHandler(tracker)))
	http.HandleFunc("/api/project-stats", corsMiddleware(projectStatsHandler(tracker)))
	http.HandleFunc("/api/session-stats", corsMiddleware(sessionStatsHandler(tracker)))
	http.HandleFunc("/api/savings-trend", corsMiddleware(savingsTrendHandler(tracker)))
	// New enhanced endpoints
	http.HandleFunc("/api/alerts", corsMiddleware(alertsHandler(tracker)))
	http.HandleFunc("/api/export/json", corsMiddleware(exportJSONHandler(tracker)))
	http.HandleFunc("/api/model-breakdown", corsMiddleware(modelBreakdownHandler(tracker)))
	http.HandleFunc("/api/config", corsMiddleware(configHandler(tracker)))
	http.HandleFunc("/api/report", corsMiddleware(reportHandler(tracker)))
	http.HandleFunc("/api/cache-metrics", corsMiddleware(cacheMetricsHandler(tracker)))
	http.HandleFunc("/logo", logoHandler)
	http.HandleFunc("/", dashboardIndexHandler)

	addr := fmt.Sprintf("%s:%d", Bind, Port)

	fmt.Printf("🌐 TokMan Dashboard running at http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")

	if Open {
		fmt.Println("Opening browser...")
		// Could use browser.OpenURL here
	}

	return http.ListenAndServe(addr, nil)
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func statsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetSavings("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get 24h and total savings
		saved24h, _ := tracker.TokensSaved24h()
		savedTotal, _ := tracker.TokensSavedTotal()

		response := map[string]interface{}{
			"tokens_saved":       stats.TotalSaved,
			"commands_count":     stats.TotalCommands,
			"original":           stats.TotalOriginal,
			"filtered":           stats.TotalFiltered,
			"tokens_saved_24h":   saved24h,
			"tokens_saved_total": savedTotal,
		}
		json.NewEncoder(w).Encode(response)
	}
}

func dailyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		days := 7
		if d := r.URL.Query().Get("days"); d != "" {
			fmt.Sscanf(d, "%d", &days)
		}

		records, err := tracker.GetDailySavings("", days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Convert to chart format
		result := make([]map[string]interface{}, len(records))
		for i, r := range records {
			result[i] = map[string]interface{}{
				"date":         r.Date,
				"tokens_saved": r.Saved,
				"original":     r.Original,
				"commands":     r.Commands,
			}
		}
		json.NewEncoder(w).Encode(result)
	}
}

func commandsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetCommandStats("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Convert to chart format
		limit := 5
		if len(stats) < limit {
			limit = len(stats)
		}
		result := make([]map[string]interface{}, limit)
		for i := 0; i < limit; i++ {
			result[i] = map[string]interface{}{
				"command":      stats[i].Command,
				"tokens_saved": stats[i].TotalSaved,
				"executions":   stats[i].ExecutionCount,
			}
		}
		json.NewEncoder(w).Encode(result)
	}
}

func economicsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 30)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Calculate estimated cost savings
		// Using $3 per million input tokens (Claude pricing)
		var totalSaved int
		for _, r := range records {
			totalSaved += r.Saved
		}
		estimatedCost := float64(totalSaved) * 3.0 / 1_000_000

		response := map[string]interface{}{
			"total_tokens_saved": totalSaved,
			"estimated_cost":     estimatedCost,
			"records_count":      len(records),
		}

		json.NewEncoder(w).Encode(response)
	}
}

func logoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	http.ServeFile(w, r, "docs/logo.svg")
}

// weeklyHandler returns weekly aggregated data
func weeklyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 28) // 4 weeks
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Aggregate by week
		weeks := make(map[string]struct{ saved, original, commands int })
		for _, r := range records {
			// Simple week grouping by date prefix
			week := r.Date[:7] + "-W" // Year-Month format
			wk := weeks[week]
			wk.saved += r.Saved
			wk.original += r.Original
			wk.commands += r.Commands
			weeks[week] = wk
		}
		result := make([]map[string]interface{}, 0)
		for week, data := range weeks {
			result = append(result, map[string]interface{}{
				"week":         week,
				"tokens_saved": data.saved,
				"original":     data.original,
				"commands":     data.commands,
			})
		}
		json.NewEncoder(w).Encode(result)
	}
}

// monthlyHandler returns monthly aggregated data
func monthlyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 90) // 3 months
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Aggregate by month
		months := make(map[string]struct{ saved, original, commands int })
		for _, r := range records {
			month := r.Date[:7] // YYYY-MM format
			m := months[month]
			m.saved += r.Saved
			m.original += r.Original
			m.commands += r.Commands
			months[month] = m
		}
		result := make([]map[string]interface{}, 0)
		for month, data := range months {
			result = append(result, map[string]interface{}{
				"month":        month,
				"tokens_saved": data.saved,
				"original":     data.original,
				"commands":     data.commands,
			})
		}
		json.NewEncoder(w).Encode(result)
	}
}

// recentHandler returns recent command activity
func recentHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		records, err := tracker.GetRecentCommands("", limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result := make([]map[string]interface{}, len(records))
		for i, r := range records {
			result[i] = map[string]interface{}{
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
		json.NewEncoder(w).Encode(result)
	}
}

// performanceHandler returns performance metrics
func performanceHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetRecentCommands("", 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var totalExecTime int64
		var slowCommands []map[string]interface{}
		for _, rec := range records {
			totalExecTime += rec.ExecTimeMs
			if rec.ExecTimeMs > 1000 { // > 1 second
				slowCommands = append(slowCommands, map[string]interface{}{
					"command":      rec.Command,
					"exec_time_ms": rec.ExecTimeMs,
				})
			}
		}
		avgExecTime := float64(0)
		if len(records) > 0 {
			avgExecTime = float64(totalExecTime) / float64(len(records))
		}
		response := map[string]interface{}{
			"avg_exec_time_ms": avgExecTime,
			"total_commands":   len(records),
			"slow_commands":    slowCommands[:min(5, len(slowCommands))],
		}
		json.NewEncoder(w).Encode(response)
	}
}

// failuresHandler returns parse failure statistics
func failuresHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := tracker.GetParseFailureSummary()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := map[string]interface{}{
			"total_failures": summary.Total,
			"recovery_rate":  summary.RecoveryRate,
			"top_failures":   summary.TopCommands,
		}
		json.NewEncoder(w).Encode(response)
	}
}

// topCommandsHandler returns top commands by token savings
func topCommandsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetCommandStats("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		limit := 10
		if len(stats) < limit {
			limit = len(stats)
		}
		result := make([]map[string]interface{}, limit)
		for i := 0; i < limit; i++ {
			avgSaved := 0.0
			if stats[i].ExecutionCount > 0 {
				avgSaved = float64(stats[i].TotalSaved) / float64(stats[i].ExecutionCount)
			}
			result[i] = map[string]interface{}{
				"command":       stats[i].Command,
				"tokens_saved":  stats[i].TotalSaved,
				"executions":    stats[i].ExecutionCount,
				"avg_saved":     avgSaved,
				"reduction_pct": stats[i].ReductionPct,
			}
		}
		json.NewEncoder(w).Encode(result)
	}
}

// hourlyHandler returns hourly usage distribution
func hourlyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetRecentCommands("", 500)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hours := make(map[int]int)
		for _, rec := range records {
			hour := rec.Timestamp.Hour()
			hours[hour]++
		}
		result := make([]map[string]interface{}, 24)
		for h := 0; h < 24; h++ {
			result[h] = map[string]interface{}{
				"hour":     h,
				"commands": hours[h],
			}
		}
		json.NewEncoder(w).Encode(result)
	}
}

// exportCSVHandler exports data as CSV
func exportCSVHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetRecentCommands("", 1000)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

// === NEW HANDLERS FOR ENHANCED DASHBOARD ===

// llmStatusHandler returns LLM integration status and usage data
func llmStatusHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if ccusage is available
		ccAvailable := ccusage.IsAvailable()

		response := map[string]interface{}{
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

		json.NewEncoder(w).Encode(response)
	}
}

// dailyBreakdownHandler returns detailed daily breakdown with tokens saved per day
func dailyBreakdownHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		days := 30
		if d := r.URL.Query().Get("days"); d != "" {
			fmt.Sscanf(d, "%d", &days)
		}

		// Get tokman daily stats
		tmStats := economics.GetDailyStats(tracker)

		// Try to merge with ccusage data
		ccDaily, _ := ccusage.Fetch(ccusage.Daily)
		periods := economics.MergeDailyLite(ccDaily, tmStats)

		// Convert to response format
		result := make([]map[string]interface{}, len(periods))
		for i, p := range periods {
			entry := map[string]interface{}{
				"date":         p.Label,
				"tokens_saved": p.TMSavedTokens,
				"commands":     p.TMCommands,
				"savings_pct":  p.TMSavingsPct,
			}
			// Add ccusage data if available
			if p.CCCost != nil {
				entry["cc_cost"] = *p.CCCost
			}
			if p.CCInputTokens != nil {
				entry["cc_input"] = *p.CCInputTokens
			}
			if p.CCOutputTokens != nil {
				entry["cc_output"] = *p.CCOutputTokens
			}
			if p.SavingsWeighted != nil {
				entry["estimated_savings"] = *p.SavingsWeighted
			}
			result[i] = entry
		}

		json.NewEncoder(w).Encode(result)
	}
}

// projectStatsHandler returns statistics grouped by project
func projectStatsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT 
				project_path,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original,
				AVG(exec_time_ms) as avg_exec_time
			FROM commands
			WHERE project_path IS NOT NULL AND project_path != ''
			GROUP BY project_path
			ORDER BY saved DESC
			LIMIT 20
		`

		rows, err := tracker.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			var projectPath string
			var commands, saved, original int
			var avgExecTime float64
			if err := rows.Scan(&projectPath, &commands, &saved, &original, &avgExecTime); err != nil {
				continue
			}
			savingsPct := 0.0
			if original > 0 {
				savingsPct = float64(saved) / float64(original) * 100
			}
			result = append(result, map[string]interface{}{
				"project":       projectPath,
				"commands":      commands,
				"tokens_saved":  saved,
				"original":      original,
				"savings_pct":   savingsPct,
				"avg_exec_time": avgExecTime,
			})
		}

		json.NewEncoder(w).Encode(result)
	}
}

// sessionStatsHandler returns statistics grouped by session
func sessionStatsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT 
				session_id,
				MIN(timestamp) as start_time,
				MAX(timestamp) as end_time,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				project_path
			FROM commands
			WHERE session_id IS NOT NULL AND session_id != ''
			GROUP BY session_id
			ORDER BY start_time DESC
			LIMIT 20
		`

		rows, err := tracker.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			var sessionID, projectPath, startTime, endTime string
			var commands, saved int
			if err := rows.Scan(&sessionID, &startTime, &endTime, &commands, &saved, &projectPath); err != nil {
				continue
			}
			result = append(result, map[string]interface{}{
				"session_id":   sessionID,
				"start_time":   startTime,
				"end_time":     endTime,
				"commands":     commands,
				"tokens_saved": saved,
				"project":      projectPath,
			})
		}

		json.NewEncoder(w).Encode(result)
	}
}

// savingsTrendHandler returns savings trend data for charts
func savingsTrendHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get daily savings for the last 30 days
		query := `
			SELECT 
				DATE(timestamp) as date,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original,
				COALESCE(SUM(exec_time_ms), 0) as total_exec_time
			FROM commands
			WHERE timestamp >= DATE('now', '-30 days')
			GROUP BY DATE(timestamp)
			ORDER BY date ASC
		`

		rows, err := tracker.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			var date string
			var commands, saved, original int
			var totalExecTime int64
			if err := rows.Scan(&date, &commands, &saved, &original, &totalExecTime); err != nil {
				continue
			}
			savingsPct := 0.0
			if original > 0 {
				savingsPct = float64(saved) / float64(original) * 100
			}
			avgExec := 0.0
			if commands > 0 {
				avgExec = float64(totalExecTime) / float64(commands)
			}
			result = append(result, map[string]interface{}{
				"date":         date,
				"commands":     commands,
				"tokens_saved": saved,
				"original":     original,
				"savings_pct":  savingsPct,
				"avg_exec_ms":  avgExec,
			})
		}

		json.NewEncoder(w).Encode(result)
	}
}

// alertsHandler returns alert status based on thresholds
func alertsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts := []map[string]interface{}{}
		cfg := defaultConfig.Alerts

		if !cfg.Enabled {
			json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false, "alerts": alerts})
			return
		}

		// Check daily token limit
		saved24h, _ := tracker.TokensSaved24h()
		if saved24h > cfg.DailyTokenLimit {
			alerts = append(alerts, map[string]interface{}{
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
				if rows.Scan(&count, &avgCount); err == nil && avgCount > 0 {
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
							if lhRows.Scan(&lastHourCount); err == nil {
								if float64(lastHourCount) > avgCount*cfg.UsageSpikeThreshold && avgCount > 0 {
									alerts = append(alerts, map[string]interface{}{
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

		response := map[string]interface{}{
			"enabled": true,
			"alerts":  alerts,
			"config": map[string]interface{}{
				"daily_token_limit":     cfg.DailyTokenLimit,
				"weekly_token_limit":    cfg.WeeklyTokenLimit,
				"usage_spike_threshold": cfg.UsageSpikeThreshold,
			},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// exportJSONHandler exports data as JSON
func exportJSONHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exportType := r.URL.Query().Get("type")
		if exportType == "" {
			exportType = "all"
		}

		response := map[string]interface{}{
			"exported_at": time.Now().Format(time.RFC3339),
			"type":        exportType,
		}

		// Include stats
		stats, _ := tracker.GetSavings("")
		if stats != nil {
			response["stats"] = stats
		}

		// Include recent commands
		records, _ := tracker.GetRecentCommands("", 1000)
		if records != nil {
			response["commands"] = records
		}

		// Include daily breakdown
		daily, _ := tracker.GetDailySavings("", 30)
		if daily != nil {
			response["daily"] = daily
		}

		// Include command stats
		cmdStats, _ := tracker.GetCommandStats("")
		if cmdStats != nil {
			response["command_stats"] = cmdStats
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=tokman-export.json")
		json.NewEncoder(w).Encode(response)
	}
}

// modelBreakdownHandler returns LLM usage breakdown by model (from ccusage)
func modelBreakdownHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"available": false,
			"models":    []map[string]interface{}{},
		}

		if !ccusage.IsAvailable() {
			json.NewEncoder(w).Encode(response)
			return
		}

		// Get monthly data which includes usage info
		monthly, err := ccusage.Fetch(ccusage.Monthly)
		if err != nil || len(monthly) == 0 {
			json.NewEncoder(w).Encode(response)
			return
		}

		// Aggregate by period (ccusage doesn't provide model breakdown, so group by month)
		models := []map[string]interface{}{}
		for _, m := range monthly {
			models = append(models, map[string]interface{}{
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
		json.NewEncoder(w).Encode(response)
	}
}

// configHandler returns dashboard configuration
func configHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(defaultConfig)
			return
		}

		// POST - update config (for future use)
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Update config (merge with defaults for any missing fields)
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

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultConfig)
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
		response := map[string]interface{}{
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

			response["summary"] = map[string]interface{}{
				"period_days":          days,
				"tokens_saved":         totalSaved,
				"commands_processed":   totalCommands,
				"efficiency_pct":       efficiency,
				"estimated_cost_saved": estimatedSavings,
			}

			response["daily_breakdown"] = daily
		}

		// Get top commands
		cmdStats, _ := tracker.GetCommandStats("")
		if cmdStats != nil && len(cmdStats) > 0 {
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
			projects := []map[string]interface{}{}
			for rows.Next() {
				var path string
				var count, saved int
				if rows.Scan(&path, &count, &saved); err == nil {
					projects = append(projects, map[string]interface{}{
						"project":      filepath.Base(path),
						"commands":     count,
						"tokens_saved": saved,
					})
				}
			}
			response["top_projects"] = projects
		}

		json.NewEncoder(w).Encode(response)
	}
}

// cacheMetricsHandler returns cache efficiency metrics
func cacheMetricsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get total stats
		stats, _ := tracker.GetSavings("")
		if stats == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "no data"})
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
		hourlyPattern := []map[string]interface{}{}
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var hour string
				var count int
				var avgSaved float64
				if rows.Scan(&hour, &count, &avgSaved); err == nil {
					hourlyPattern = append(hourlyPattern, map[string]interface{}{
						"hour":      hour,
						"count":     count,
						"avg_saved": avgSaved,
					})
				}
			}
		}

		response := map[string]interface{}{
			"total_original": stats.TotalOriginal,
			"total_filtered": stats.TotalFiltered,
			"total_saved":    stats.TotalSaved,
			"efficiency_pct": efficiency,
			"commands_count": stats.TotalCommands,
			"hourly_pattern": hourlyPattern,
		}

		// Add ccusage cache stats if available
		if ccusage.IsAvailable() {
			monthly, _ := ccusage.Fetch(ccusage.Monthly)
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

		json.NewEncoder(w).Encode(response)
	}
}

func dashboardIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}
