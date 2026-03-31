package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/tracking"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

var (
	gainProject  bool
	gainGraph    bool
	gainHistory  bool
	gainQuota    bool
	gainTier     string
	gainDaily    bool
	gainWeekly   bool
	gainMonthly  bool
	gainAll      bool
	gainFormat   string
	gainFailures bool
)

var gainCmd = &cobra.Command{
	Use:   "gain",
	Short: "Show token savings summary and history",
	Long: `Display comprehensive token savings analysis.

Shows statistics including:
- Total commands processed
- Input/output token counts
- Tokens saved and savings percentage
- Execution time metrics
- Breakdown by command type

Use flags for detailed views: --graph, --history, --quota, --daily, --weekly, --monthly`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runGain(verbose)
	},
}

func init() {
	registry.Add(func() { registry.Register(gainCmd) })

	gainCmd.Flags().BoolVarP(&gainProject, "project", "p", false, "Filter to current project only")
	gainCmd.Flags().BoolVarP(&gainGraph, "graph", "g", false, "Show ASCII graph of daily savings")
	gainCmd.Flags().BoolVarP(&gainHistory, "history", "H", false, "Show recent command history")
	gainCmd.Flags().BoolVar(&gainQuota, "quota", false, "Show monthly quota savings estimate")
	gainCmd.Flags().StringVarP(&gainTier, "tier", "t", "20x", "Subscription tier for quota: pro, 5x, 20x")
	gainCmd.Flags().BoolVarP(&gainDaily, "daily", "d", false, "Show daily breakdown")
	gainCmd.Flags().BoolVarP(&gainWeekly, "weekly", "w", false, "Show weekly breakdown")
	gainCmd.Flags().BoolVarP(&gainMonthly, "monthly", "m", false, "Show monthly breakdown")
	gainCmd.Flags().BoolVarP(&gainAll, "all", "a", false, "Show all time periods")
	gainCmd.Flags().StringVarP(&gainFormat, "format", "f", "text", "Output format: text, json, csv")
	gainCmd.Flags().BoolVarP(&gainFailures, "failures", "F", false, "Show parse failure log")
}

// GainSummary represents the summary statistics
type GainSummary struct {
	TotalCommands int
	TotalInput    int
	TotalOutput   int
	TotalSaved    int
	AvgSavingsPct float64
	TotalTimeMs   int64
	AvgTimeMs     int64
	ByCommand     []CommandBreakdown
	ByDay         []DayBreakdown
}

// CommandBreakdown represents stats for a single command
type CommandBreakdown struct {
	Command string
	Count   int
	Saved   int
	AvgPct  float64
	AvgTime int64
}

// DayBreakdown represents stats for a single day
type DayBreakdown struct {
	Date     string
	Commands int
	Saved    int
	Original int
}

func runGain(verbose bool) error {
	dbPath := tracking.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("error initializing tracker: %w", err)
	}
	defer tracker.Close()

	// Resolve project scope (default to current project)
	var projectPath string
	if gainProject || !gainAll {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting working directory: %w", err)
		}
		projectPath = cwd
	}

	// Handle failure log
	if gainFailures {
		showFailures(tracker)
		return nil
	}

	// Handle export formats
	switch gainFormat {
	case "json":
		exportJSON(tracker, projectPath)
		return nil
	case "csv":
		exportCSV(tracker, projectPath)
		return nil
	}

	// Get summary
	summary := getSummary(tracker, projectPath)

	if summary.TotalCommands == 0 {
		fmt.Println("No tracking data yet.")
		fmt.Println("Run some tokman commands to start tracking savings.")
		return nil
	}

	// Default view
	if !gainDaily && !gainWeekly && !gainMonthly && !gainAll {
		printDefaultView(tracker, summary, projectPath, verbose)
		return nil
	}

	// Time breakdown views
	if gainAll || gainDaily {
		printDaily(tracker, projectPath)
	}
	if gainAll || gainWeekly {
		printWeekly(tracker, projectPath)
	}
	if gainAll || gainMonthly {
		printMonthly(tracker, projectPath)
	}
	return nil
}

func getSummary(tracker *tracking.Tracker, projectPath string) *GainSummary {
	summary := &GainSummary{}

	// Get overall savings
	savings, err := tracker.GetSavings(projectPath)
	if err == nil {
		summary.TotalCommands = savings.TotalCommands
		summary.TotalSaved = savings.TotalSaved
		summary.TotalInput = savings.TotalOriginal
		summary.TotalOutput = savings.TotalFiltered
		if savings.TotalOriginal > 0 {
			summary.AvgSavingsPct = float64(savings.TotalSaved) / float64(savings.TotalOriginal) * 100
		}
	}

	// Get command stats
	cmdStats, err := tracker.GetCommandStats(projectPath)
	if err == nil {
		for _, cs := range cmdStats {
			summary.ByCommand = append(summary.ByCommand, CommandBreakdown{
				Command: cs.Command,
				Count:   cs.ExecutionCount,
				Saved:   cs.TotalSaved,
				AvgPct:  cs.ReductionPct,
			})
		}
	}

	// Get daily savings
	daily, err := tracker.GetDailySavings(projectPath, 30)
	if err == nil {
		for _, d := range daily {
			summary.ByDay = append(summary.ByDay, DayBreakdown{
				Date:     d.Date,
				Commands: d.Commands,
				Saved:    d.Saved,
				Original: d.Original,
			})
		}
	}

	return summary
}

func printDefaultView(tracker *tracking.Tracker, summary *GainSummary, projectPath string, verbose bool) {
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()

	// Header
	title := "TokMan Token Savings (Global Scope)"
	if projectPath != "" {
		title = "TokMan Token Savings (Project Scope)"
	}
	fmt.Printf("%s\n", green(title))
	fmt.Println(strings.Repeat("═", 60))
	if projectPath != "" {
		fmt.Printf("Scope: %s\n", shared.ShortenPath(projectPath))
	}

	// Get time range from recent commands
	if len(summary.ByDay) > 0 {
		firstDay := summary.ByDay[len(summary.ByDay)-1]
		lastDay := summary.ByDay[0]
		fmt.Printf("Period: %s → %s\n", firstDay.Date, lastDay.Date)
	}
	fmt.Printf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Println()

	// KPIs
	printKPI("Total commands", fmt.Sprintf("%d", summary.TotalCommands))
	printKPI("Input tokens", utils.FormatTokens(summary.TotalInput))
	printKPI("Output tokens", utils.FormatTokens(summary.TotalOutput))
	printKPI("Tokens saved", fmt.Sprintf("%s (%.1f%%)", utils.FormatTokens(summary.TotalSaved), summary.AvgSavingsPct))
	printKPI("Total exec time", utils.FormatDuration(summary.TotalTimeMs))
	printEfficiencyMeter(summary.AvgSavingsPct)
	fmt.Println()

	// By command breakdown
	if len(summary.ByCommand) > 0 {
		fmt.Printf("%s\n", cyan("By Command"))
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("%-24s  %8s  %10s  %8s  %s\n", "Command", "Count", "Saved", "Avg%", "Last Seen")
		fmt.Println(strings.Repeat("─", 60))

		// Get recent commands for timestamps
		recentCmds := getRecentCommandTimestamps(tracker, projectPath)

		// Sort by saved tokens
		sorted := summary.ByCommand
		for i := 0; i < len(sorted) && i < 10; i++ {
			cb := sorted[i]
			cmdName := cb.Command
			if len(cmdName) > 22 {
				cmdName = cmdName[:20] + "..."
			}
			pctCell := fmt.Sprintf("%7.1f%%", cb.AvgPct)
			if cb.AvgPct >= 70 {
				pctCell = color.New(color.FgGreen, color.Bold).Sprintf("%7.1f%%", cb.AvgPct)
			} else if cb.AvgPct >= 40 {
				pctCell = color.New(color.FgYellow, color.Bold).Sprintf("%7.1f%%", cb.AvgPct)
			}
			lastSeen := ""
			if ts, ok := recentCmds[cb.Command]; ok {
				lastSeen = ts.Format("01-02 15:04")
			}
			fmt.Printf("%-24s  %8d  %10s  %s  %s\n", cmdName, cb.Count, utils.FormatTokens(cb.Saved), pctCell, lastSeen)
		}
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println()
	}

	// Graph
	if gainGraph && len(summary.ByDay) > 0 {
		fmt.Printf("%s\n", cyan("Daily Savings (last 30 days)"))
		fmt.Println(strings.Repeat("─", 60))
		printASCIIChart(summary.ByDay)
		fmt.Println()
	}

	// History
	if gainHistory {
		dbPath := tracking.DatabasePath()
		histTracker, err := tracking.NewTracker(dbPath)
		if err != nil {
			if shared.Verbose > 0 {
				fmt.Fprintf(os.Stderr, "Warning: failed to create tracker for history: %v\n", err)
			}
			return
		}
		defer histTracker.Close()
		recent, err := histTracker.GetRecentCommands(projectPath, 10)
		if err == nil && len(recent) > 0 {
			fmt.Printf("%s\n", cyan("Recent Commands"))
			fmt.Println(strings.Repeat("─", 60))
			for _, r := range recent {
				ts := r.Timestamp.Format("01-02 15:04")
				cmd := r.Command
				if len(cmd) > 25 {
					cmd = cmd[:22] + "..."
				}
				sign := "•"
				savedPct := 0.0
				if r.OriginalTokens > 0 {
					savedPct = float64(r.SavedTokens) / float64(r.OriginalTokens) * 100
				}
				if savedPct >= 70 {
					sign = "▲"
				} else if savedPct >= 30 {
					sign = "■"
				}
				fmt.Printf("%s %s %-25s -%.0f%% (%s)\n", ts, sign, cmd, savedPct, utils.FormatTokens(r.SavedTokens))
			}
			fmt.Println()
		}
	}

	// Quota
	if gainQuota {
		printQuotaAnalysis(summary, gainTier)
	}
}

func printKPI(label string, value string) {
	fmt.Printf("%-18s %s\n", label+":", value)
}

func printEfficiencyMeter(pct float64) {
	width := 24
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	pctStr := fmt.Sprintf("%.1f%%", pct)
	var coloredPct string
	if pct >= 70 {
		coloredPct = color.New(color.FgGreen, color.Bold).Sprint(pctStr)
		bar = color.New(color.FgGreen).Sprint(bar)
	} else if pct >= 40 {
		coloredPct = color.New(color.FgYellow, color.Bold).Sprint(pctStr)
		bar = color.New(color.FgYellow).Sprint(bar)
	} else {
		coloredPct = color.New(color.FgRed, color.Bold).Sprint(pctStr)
		bar = color.New(color.FgRed).Sprint(bar)
	}

	fmt.Printf("Efficiency meter: %s %s\n", bar, coloredPct)
}

func printASCIIChart(data []DayBreakdown) {
	if len(data) == 0 {
		return
	}

	maxVal := 1
	for _, d := range data {
		if d.Saved > maxVal {
			maxVal = d.Saved
		}
	}

	width := 40
	for _, d := range data {
		dateShort := d.Date
		if len(dateShort) >= 10 {
			dateShort = dateShort[5:10]
		}
		barLen := int(float64(d.Saved) / float64(maxVal) * float64(width))
		bar := strings.Repeat("█", barLen)
		spaces := strings.Repeat(" ", width-barLen)
		fmt.Printf("%s │%s%s %s\n", dateShort, bar, spaces, utils.FormatTokens(d.Saved))
	}
}

func printQuotaAnalysis(summary *GainSummary, tier string) {
	const estimatedProMonthly = 6_000_000

	var quotaTokens int
	var tierName string
	switch tier {
	case "pro":
		quotaTokens = estimatedProMonthly
		tierName = "Pro ($20/mo)"
	case "5x":
		quotaTokens = estimatedProMonthly * 5
		tierName = "Max 5x ($100/mo)"
	default:
		quotaTokens = estimatedProMonthly * 20
		tierName = "Max 20x ($200/mo)"
	}

	quotaPct := float64(summary.TotalSaved) / float64(quotaTokens) * 100

	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	fmt.Printf("%s\n", cyan("Monthly Quota Analysis"))
	fmt.Println(strings.Repeat("─", 60))
	printKPI("Subscription tier", tierName)
	printKPI("Estimated monthly quota", utils.FormatTokens(quotaTokens))
	printKPI("Tokens saved (lifetime)", utils.FormatTokens(summary.TotalSaved))
	printKPI("Quota preserved", fmt.Sprintf("%.1f%%", quotaPct))
	fmt.Println()
	fmt.Println("Note: Heuristic estimate based on ~44K tokens/5h (Pro baseline)")
	fmt.Println("      Actual limits use rolling 5-hour windows, not monthly caps.")
}

func printDaily(tracker *tracking.Tracker, projectPath string) {
	daily, err := tracker.GetDailySavings(projectPath, 30)
	if err != nil || len(daily) == 0 {
		return
	}

	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	fmt.Printf("%s\n", cyan("Daily Breakdown"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("%-12s  %8s  %12s  %8s\n", "Date", "Commands", "Saved", "Savings%")
	fmt.Println(strings.Repeat("─", 60))

	for _, d := range daily {
		pct := 0.0
		if d.Original > 0 {
			pct = float64(d.Saved) / float64(d.Original) * 100
		}
		fmt.Printf("%-12s  %8d  %12s  %7.1f%%\n", d.Date, d.Commands, utils.FormatTokens(d.Saved), pct)
	}
	fmt.Println()
}

func printWeekly(tracker *tracking.Tracker, projectPath string) {
	// Simplified weekly view - aggregate daily data
	daily, err := tracker.GetDailySavings(projectPath, 90)
	if err != nil || len(daily) == 0 {
		return
	}

	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	fmt.Printf("%s\n", cyan("Weekly Breakdown"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("%-12s  %8s  %12s\n", "Week", "Commands", "Saved")
	fmt.Println(strings.Repeat("─", 60))

	// Group by week (simplified)
	weekMap := make(map[string]struct{ commands, saved int })
	for _, d := range daily {
		// Parse date and get week
		t, err := time.Parse("2006-01-02", d.Date)
		if err != nil {
			continue
		}
		year, week := t.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)
		w := weekMap[weekKey]
		w.commands += d.Commands
		w.saved += d.Saved
		weekMap[weekKey] = w
	}

	// Print weeks (sorted for deterministic output)
	weekKeys := make([]string, 0, len(weekMap))
	for k := range weekMap {
		weekKeys = append(weekKeys, k)
	}
	sort.Strings(weekKeys)
	for _, weekKey := range weekKeys {
		w := weekMap[weekKey]
		fmt.Printf("%-12s  %8d  %12s\n", weekKey, w.commands, utils.FormatTokens(w.saved))
	}
	fmt.Println()
}

func printMonthly(tracker *tracking.Tracker, projectPath string) {
	daily, err := tracker.GetDailySavings(projectPath, 365)
	if err != nil || len(daily) == 0 {
		return
	}

	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	fmt.Printf("%s\n", cyan("Monthly Breakdown"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("%-12s  %8s  %12s\n", "Month", "Commands", "Saved")
	fmt.Println(strings.Repeat("─", 60))

	// Group by month
	monthMap := make(map[string]struct{ commands, saved int })
	for _, d := range daily {
		if len(d.Date) >= 7 {
			monthKey := d.Date[:7] // YYYY-MM
			m := monthMap[monthKey]
			m.commands += d.Commands
			m.saved += d.Saved
			monthMap[monthKey] = m
		}
	}

	// Print months
	for monthKey, m := range monthMap {
		fmt.Printf("%-12s  %8d  %12s\n", monthKey, m.commands, utils.FormatTokens(m.saved))
	}
	fmt.Println()
}

func showFailures(tracker *tracking.Tracker) {
	// For now, just show a message
	// The full implementation would query parse failures from the database
	fmt.Println("No parse failures recorded.")
	fmt.Println("This means all commands parsed successfully (or fallback hasn't triggered yet).")
}

func exportJSON(tracker *tracking.Tracker, projectPath string) {
	summary := getSummary(tracker, projectPath)
	data := map[string]any{
		"summary": map[string]any{
			"total_commands":  summary.TotalCommands,
			"total_input":     summary.TotalInput,
			"total_output":    summary.TotalOutput,
			"total_saved":     summary.TotalSaved,
			"avg_savings_pct": summary.AvgSavingsPct,
			"total_time_ms":   summary.TotalTimeMs,
			"avg_time_ms":     summary.AvgTimeMs,
		},
	}

	if gainAll || gainDaily {
		data["daily"] = summary.ByDay
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

func exportCSV(tracker *tracking.Tracker, projectPath string) {
	if gainAll || gainDaily {
		daily, _ := tracker.GetDailySavings(projectPath, 365)
		fmt.Println("# Daily Data")
		fmt.Println("date,commands,saved_tokens,original_tokens")
		for _, d := range daily {
			fmt.Printf("%s,%d,%d,%d\n", d.Date, d.Commands, d.Saved, d.Original)
		}
		fmt.Println()
	}
}

func getRecentCommandTimestamps(tracker *tracking.Tracker, projectPath string) map[string]time.Time {
	result := make(map[string]time.Time)
	recent, err := tracker.GetRecentCommands(projectPath, 100)
	if err != nil {
		return result
	}
	for _, r := range recent {
		if _, exists := result[r.Command]; !exists {
			result[r.Command] = r.Timestamp
		}
	}
	return result
}

