package dashboard

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/contextread"
	"github.com/GrayCodeAI/tokman/internal/httpmw"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func statsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetSavings("")
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Get 24h and total savings (best-effort; partial data is acceptable)
		saved24h, _ := tracker.TokensSaved24h()
		savedTotal, _ := tracker.TokensSavedTotal()
		contextStats, _ := tracker.GetSavingsForCommands("", contextread.TrackedCommandPatterns())

		response := map[string]any{
			"tokens_saved":          stats.TotalSaved,
			"commands_count":        stats.TotalCommands,
			"original":              stats.TotalOriginal,
			"filtered":              stats.TotalFiltered,
			"tokens_saved_24h":      saved24h,
			"tokens_saved_total":    savedTotal,
			"context_read_commands": contextStats.TotalCommands,
			"context_read_saved":    contextStats.TotalSaved,
			"context_read_original": contextStats.TotalOriginal,
			"context_read_filtered": contextStats.TotalFiltered,
		}
		httpmw.JSONResponse(w, http.StatusOK, response)
	}
}

func contextReadsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind := r.URL.Query().Get("kind")
		patterns := contextread.TrackedCommandPatterns()
		if kind != "" {
			patterns = contextread.TrackedCommandPatternsForKind(kind)
			if len(patterns) == 0 {
				http.Error(w, "invalid context read kind", http.StatusBadRequest)
				return
			}
		}

		records, err := tracker.GetRecentCommandsForPatterns("", 20, patterns)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		result := make([]map[string]any, 0, len(records))
		for _, rec := range records {
			reductionPct := 0.0
			if rec.OriginalTokens > 0 {
				reductionPct = float64(rec.SavedTokens) / float64(rec.OriginalTokens) * 100
			}
			result = append(result, map[string]any{
				"command":       rec.Command,
				"tokens_saved":  rec.SavedTokens,
				"original":      rec.OriginalTokens,
				"filtered":      rec.FilteredTokens,
				"reduction_pct": reductionPct,
				"project_path":  rec.ProjectPath,
				"timestamp":     rec.Timestamp,
				"exec_time_ms":  rec.ExecTimeMs,
				"parse_success": rec.ParseSuccess,
			})
		}

		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

func contextReadSummaryHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := make(map[string]any)
		for _, kind := range contextread.TrackedCommandKinds() {
			summary, err := tracker.GetSavingsForCommands("", contextread.TrackedCommandPatternsForKind(kind))
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			result[kind] = map[string]any{
				"commands":  summary.TotalCommands,
				"saved":     summary.TotalSaved,
				"original":  summary.TotalOriginal,
				"filtered":  summary.TotalFiltered,
				"reduction": summary.ReductionPct,
			}
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

func contextReadTrendHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT
				DATE(timestamp) as day,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original
			FROM commands
			WHERE command GLOB 'tokman read *'
			   OR command GLOB 'tokman ctx read *'
			   OR command GLOB 'tokman ctx delta *'
			   OR command GLOB 'tokman mcp read *'
			GROUP BY DATE(timestamp)
			ORDER BY day ASC
		`
		rows, err := tracker.Query(query)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]any
		for rows.Next() {
			var day string
			var commands, saved, original int
			if err := rows.Scan(&day, &commands, &saved, &original); err != nil {
				continue
			}
			reduction := 0.0
			if original > 0 {
				reduction = float64(saved) / float64(original) * 100
			}
			result = append(result, map[string]any{
				"date":         day,
				"commands":     commands,
				"tokens_saved": saved,
				"original":     original,
				"reduction":    reduction,
			})
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

func contextReadTopFilesHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := tracker.Query(`
			SELECT
				command,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original
			FROM commands
			WHERE command GLOB 'tokman read *'
			   OR command GLOB 'tokman ctx read *'
			   OR command GLOB 'tokman ctx delta *'
			   OR command GLOB 'tokman mcp read *'
			GROUP BY command
			ORDER BY saved DESC
			LIMIT 10
		`)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]any
		for rows.Next() {
			var command string
			var commands, saved, original int
			if err := rows.Scan(&command, &commands, &saved, &original); err != nil {
				continue
			}
			reduction := 0.0
			if original > 0 {
				reduction = float64(saved) / float64(original) * 100
			}
			result = append(result, map[string]any{
				"command":       command,
				"file":          extractReadTarget(command),
				"commands":      commands,
				"tokens_saved":  saved,
				"reduction_pct": reduction,
			})
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

func contextReadProjectsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := tracker.Query(`
			SELECT
				project_path,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original
			FROM commands
			WHERE (command GLOB 'tokman read *'
			    OR command GLOB 'tokman ctx read *'
			    OR command GLOB 'tokman ctx delta *'
			    OR command GLOB 'tokman mcp read *')
			  AND project_path IS NOT NULL AND project_path != ''
			GROUP BY project_path
			ORDER BY saved DESC
			LIMIT 10
		`)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]any
		for rows.Next() {
			var project string
			var commands, saved, original int
			if err := rows.Scan(&project, &commands, &saved, &original); err != nil {
				continue
			}
			reduction := 0.0
			if original > 0 {
				reduction = float64(saved) / float64(original) * 100
			}
			result = append(result, map[string]any{
				"project":       project,
				"commands":      commands,
				"tokens_saved":  saved,
				"reduction_pct": reduction,
			})
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

func extractReadTarget(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func dailyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		days := 7
		if d := r.URL.Query().Get("days"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
				days = parsed
			}
		}

		records, err := tracker.GetDailySavings("", days)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		// Convert to chart format
		result := make([]map[string]any, len(records))
		for i, r := range records {
			result[i] = map[string]any{
				"date":         r.Date,
				"tokens_saved": r.Saved,
				"original":     r.Original,
				"commands":     r.Commands,
			}
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

// weeklyHandler returns weekly aggregated data
func weeklyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 28) // 4 weeks
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
		result := make([]map[string]any, 0)
		for week, data := range weeks {
			result = append(result, map[string]any{
				"week":         week,
				"tokens_saved": data.saved,
				"original":     data.original,
				"commands":     data.commands,
			})
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

// monthlyHandler returns monthly aggregated data
func monthlyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 90) // 3 months
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
		result := make([]map[string]any, 0)
		for month, data := range months {
			result = append(result, map[string]any{
				"month":        month,
				"tokens_saved": data.saved,
				"original":     data.original,
				"commands":     data.commands,
			})
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

// performanceHandler returns performance metrics
func performanceHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetRecentCommands("", 100)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		var totalExecTime int64
		var slowCommands []map[string]any
		for _, rec := range records {
			totalExecTime += rec.ExecTimeMs
			if rec.ExecTimeMs > 1000 { // > 1 second
				slowCommands = append(slowCommands, map[string]any{
					"command":      rec.Command,
					"exec_time_ms": rec.ExecTimeMs,
				})
			}
		}
		avgExecTime := float64(0)
		if len(records) > 0 {
			avgExecTime = float64(totalExecTime) / float64(len(records))
		}
		response := map[string]any{
			"avg_exec_time_ms": avgExecTime,
			"total_commands":   len(records),
			"slow_commands":    slowCommands[:min(5, len(slowCommands))],
		}
		httpmw.JSONResponse(w, http.StatusOK, response)
	}
}

// topCommandsHandler returns top commands by token savings
func topCommandsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetCommandStats("")
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		limit := 10
		if len(stats) < limit {
			limit = len(stats)
		}
		result := make([]map[string]any, limit)
		for i := 0; i < limit; i++ {
			avgSaved := 0.0
			if stats[i].ExecutionCount > 0 {
				avgSaved = float64(stats[i].TotalSaved) / float64(stats[i].ExecutionCount)
			}
			result[i] = map[string]any{
				"command":       stats[i].Command,
				"tokens_saved":  stats[i].TotalSaved,
				"executions":    stats[i].ExecutionCount,
				"avg_saved":     avgSaved,
				"reduction_pct": stats[i].ReductionPct,
			}
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

// hourlyHandler returns hourly usage distribution
func hourlyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetRecentCommands("", 500)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		hours := make(map[int]int)
		for _, rec := range records {
			hour := rec.Timestamp.Hour()
			hours[hour]++
		}
		result := make([]map[string]any, 24)
		for h := 0; h < 24; h++ {
			result[h] = map[string]any{
				"hour":     h,
				"commands": hours[h],
			}
		}
		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

// dailyBreakdownHandler returns detailed daily breakdown with tokens saved per day
// Shows only tokman tracking data (not ccusage historical data)
func dailyBreakdownHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		days := 30
		if d := r.URL.Query().Get("days"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
				days = parsed
			}
		}

		// Get tokman daily stats only
		tmStats, err := tracker.GetDailySavings("", days)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Convert to response format - only include days with activity
		var result []map[string]any
		for _, d := range tmStats {
			// Only include days with actual commands
			if d.Commands == 0 && d.Saved == 0 && d.Original == 0 {
				continue
			}
			savingsPct := 0.0
			if d.Original > 0 {
				savingsPct = float64(d.Saved) / float64(d.Original) * 100
			}
			result = append(result, map[string]any{
				"date":         d.Date,
				"tokens_saved": d.Saved,
				"commands":     d.Commands,
				"savings_pct":  savingsPct,
				"original":     d.Original,
			})
		}

		httpmw.JSONResponse(w, http.StatusOK, result)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]any
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
			result = append(result, map[string]any{
				"project":       projectPath,
				"commands":      commands,
				"tokens_saved":  saved,
				"original":      original,
				"savings_pct":   savingsPct,
				"avg_exec_time": avgExecTime,
			})
		}

		httpmw.JSONResponse(w, http.StatusOK, result)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]any
		for rows.Next() {
			var sessionID, projectPath, startTime, endTime string
			var commands, saved int
			if err := rows.Scan(&sessionID, &startTime, &endTime, &commands, &saved, &projectPath); err != nil {
				continue
			}
			result = append(result, map[string]any{
				"session_id":   sessionID,
				"start_time":   startTime,
				"end_time":     endTime,
				"commands":     commands,
				"tokens_saved": saved,
				"project":      projectPath,
			})
		}

		httpmw.JSONResponse(w, http.StatusOK, result)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]any
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
			result = append(result, map[string]any{
				"date":         date,
				"commands":     commands,
				"tokens_saved": saved,
				"original":     original,
				"savings_pct":  savingsPct,
				"avg_exec_ms":  avgExec,
			})
		}

		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}
