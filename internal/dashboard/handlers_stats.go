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
		contextStats, _ := tracker.GetSavingsForContextReads("", "", "")

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
		kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
		mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
		if kind != "" && len(contextread.TrackedCommandPatternsForKind(kind)) == 0 {
			http.Error(w, "invalid context read kind", http.StatusBadRequest)
			return
		}

		records, err := tracker.GetRecentContextReads("", kind, mode, 20)
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
				"kind":          contextReadKind(rec),
				"mode":          contextReadMode(rec),
				"target":        contextReadTarget(rec),
				"bundle":        rec.ContextBundle,
				"related_files": rec.ContextRelatedFiles,
			})
		}

		httpmw.JSONResponse(w, http.StatusOK, result)
	}
}

func contextReadSummaryHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := make(map[string]any)
		for _, kind := range contextread.TrackedCommandKinds() {
			summary, err := tracker.GetSavingsForContextReads("", kind, "")
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
		`
		where, args := contextReadQueryFilters("", "")
		query += " WHERE " + where + " GROUP BY DATE(timestamp)"
		query += `
			ORDER BY day ASC
		`
		rows, err := tracker.Query(query, args...)
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
		query := `
			SELECT
				COALESCE(NULLIF(context_target, ''), command) as target,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original
			FROM commands
		`
		where, args := contextReadQueryFilters("", "")
		query += " WHERE " + where + `
			GROUP BY COALESCE(NULLIF(context_target, ''), command)
			ORDER BY saved DESC
			LIMIT 10
		`
		rows, err := tracker.Query(query, args...)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var result []map[string]any
		for rows.Next() {
			var target string
			var commands, saved, original int
			if err := rows.Scan(&target, &commands, &saved, &original); err != nil {
				continue
			}
			reduction := 0.0
			if original > 0 {
				reduction = float64(saved) / float64(original) * 100
			}
			result = append(result, map[string]any{
				"file":          normalizeContextTarget(target),
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
		query := `
			SELECT
				project_path,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original
			FROM commands
		`
		where, args := contextReadQueryFilters("", "")
		query += " WHERE " + where + `
			  AND project_path IS NOT NULL AND project_path != ''
			GROUP BY project_path
			ORDER BY saved DESC
			LIMIT 10
		`
		rows, err := tracker.Query(query, args...)
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

func contextReadComparisonHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		singleQuery := `
			SELECT COUNT(*), COALESCE(SUM(saved_tokens), 0), COALESCE(SUM(original_tokens), 0), COALESCE(AVG(filtered_tokens), 0)
			FROM commands
		`
		singleWhere, singleArgs := contextReadQueryFilters("", "")
		singleQuery += " WHERE " + singleWhere + " AND COALESCE(context_bundle, 0) = 0 AND command NOT GLOB 'tokman mcp bundle *'"
		var singleCommands, singleSaved, singleOriginal int
		var singleAvgDelivered float64
		err := tracker.QueryRow(singleQuery, singleArgs...).Scan(&singleCommands, &singleSaved, &singleOriginal, &singleAvgDelivered)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		bundleQuery := `
			SELECT COUNT(*), COALESCE(SUM(saved_tokens), 0), COALESCE(SUM(original_tokens), 0),
			       COALESCE(AVG(filtered_tokens), 0), COALESCE(AVG(context_related_files), 0)
			FROM commands
		`
		bundleWhere, bundleArgs := contextReadQueryFilters("", "")
		bundleQuery += " WHERE " + bundleWhere + " AND (COALESCE(context_bundle, 0) = 1 OR command GLOB 'tokman mcp bundle *')"
		var bundleCommands, bundleSaved, bundleOriginal int
		var bundleAvgDelivered, bundleAvgRelated float64
		err = tracker.QueryRow(bundleQuery, bundleArgs...).Scan(&bundleCommands, &bundleSaved, &bundleOriginal, &bundleAvgDelivered, &bundleAvgRelated)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		httpmw.JSONResponse(w, http.StatusOK, map[string]any{
			"single": map[string]any{
				"commands":             singleCommands,
				"tokens_saved":         singleSaved,
				"reduction_pct":        reductionPercent(singleOriginal, singleSaved),
				"avg_delivered_tokens": singleAvgDelivered,
			},
			"bundle": map[string]any{
				"commands":             bundleCommands,
				"tokens_saved":         bundleSaved,
				"reduction_pct":        reductionPercent(bundleOriginal, bundleSaved),
				"avg_delivered_tokens": bundleAvgDelivered,
				"avg_related_files":    bundleAvgRelated,
			},
		})
	}
}

func contextReadQualityHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := tracker.Query(`
			SELECT
				COALESCE(NULLIF(context_resolved_mode, ''), NULLIF(context_mode, ''), 'unknown') as mode,
				COUNT(*) as commands,
				COALESCE(SUM(saved_tokens), 0) as saved,
				COALESCE(SUM(original_tokens), 0) as original,
				COALESCE(AVG(filtered_tokens), 0) as avg_final,
				COALESCE(AVG(saved_tokens), 0) as avg_saved,
				COALESCE(AVG(CASE WHEN COALESCE(context_bundle, 0) = 1 THEN context_related_files ELSE NULL END), 0) as avg_related
			FROM commands
			WHERE `+mustContextReadWhere()+`
			GROUP BY COALESCE(NULLIF(context_resolved_mode, ''), NULLIF(context_mode, ''), 'unknown')
			ORDER BY saved DESC, commands DESC
		`, mustContextReadArgs()...)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var modes []map[string]any
		for rows.Next() {
			var mode string
			var commands, saved, original int
			var avgFinal, avgSaved, avgRelated float64
			if err := rows.Scan(&mode, &commands, &saved, &original, &avgFinal, &avgSaved, &avgRelated); err != nil {
				continue
			}
			modes = append(modes, map[string]any{
				"mode":                 mode,
				"commands":             commands,
				"tokens_saved":         saved,
				"reduction_pct":        reductionPercent(original, saved),
				"avg_delivered_tokens": avgFinal,
				"avg_saved_tokens":     avgSaved,
				"avg_related_files":    avgRelated,
			})
		}

		httpmw.JSONResponse(w, http.StatusOK, map[string]any{"modes": modes})
	}
}

func contextReadQueryFilters(kind, mode string) (string, []any) {
	filter, args := trackingBuildContextReadFilter(kind, mode)
	return filter, args
}

func mustContextReadWhere() string {
	where, _ := contextReadQueryFilters("", "")
	return where
}

func mustContextReadArgs() []any {
	_, args := contextReadQueryFilters("", "")
	return args
}

func trackingBuildContextReadFilter(kind, mode string) (string, []any) {
	var filters []string
	var args []any

	if strings.TrimSpace(kind) != "" {
		fallback := contextread.TrackedCommandPatternsForKind(kind)
		fallbackClause, fallbackArgs := contextReadFallbackClause(fallback)
		if fallbackClause != "" {
			filters = append(filters, "(context_kind = ? OR (COALESCE(context_kind, '') = '' AND ("+fallbackClause+")))")
			args = append(args, strings.ToLower(kind))
			args = append(args, fallbackArgs...)
		} else {
			filters = append(filters, "context_kind = ?")
			args = append(args, strings.ToLower(kind))
		}
	} else {
		fallbackClause, fallbackArgs := contextReadFallbackClause(contextread.TrackedCommandPatterns())
		filters = append(filters, "(COALESCE(context_kind, '') != '' OR ("+fallbackClause+"))")
		args = append(args, fallbackArgs...)
	}

	if strings.TrimSpace(mode) != "" {
		mode = strings.ToLower(mode)
		filters = append(filters, "(context_mode = ? OR context_resolved_mode = ?)")
		args = append(args, mode, mode)
	}

	return strings.Join(filters, " AND "), args
}

func contextReadFallbackClause(patterns []string) (string, []any) {
	if len(patterns) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(patterns))
	args := make([]any, 0, len(patterns))
	for _, pattern := range patterns {
		parts = append(parts, "command GLOB ?")
		args = append(args, pattern)
	}
	return strings.Join(parts, " OR "), args
}

func contextReadKind(rec tracking.CommandRecord) string {
	if rec.ContextKind != "" {
		return rec.ContextKind
	}
	command := rec.Command
	switch {
	case strings.Contains(command, " ctx delta "):
		return "delta"
	case strings.Contains(command, " mcp "):
		return "mcp"
	default:
		return "read"
	}
}

func contextReadMode(rec tracking.CommandRecord) string {
	if rec.ContextResolvedMode != "" {
		return rec.ContextResolvedMode
	}
	if rec.ContextMode != "" {
		return rec.ContextMode
	}
	command := rec.Command
	switch {
	case strings.Contains(command, " ctx delta "):
		return "delta"
	case strings.Contains(command, " mcp bundle "):
		return "graph"
	default:
		return ""
	}
}

func contextReadTarget(rec tracking.CommandRecord) string {
	if rec.ContextTarget != "" {
		return rec.ContextTarget
	}
	return normalizeContextTarget(rec.Command)
}

func normalizeContextTarget(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) >= 4 && parts[0] == "tokman" {
		return parts[len(parts)-1]
	}
	return value
}

func reductionPercent(original, saved int) float64 {
	if original <= 0 {
		return 0
	}
	return float64(saved) / float64(original) * 100
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
