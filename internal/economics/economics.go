// Package economics provides Claude Code spending vs savings analysis.
//
// Combines ccusage (tokens spent) with tokman tracking (tokens saved) to provide
// dual-metric economic impact reporting with weighted cost-per-token calculations.
package economics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/GrayCodeAI/tokman/internal/ccusage"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// API pricing ratios (verified Feb 2026, consistent across Claude models <=200K context)
// Source: https://docs.anthropic.com/en/docs/about-claude/models
const (
	WeightOutput      = 5.0  // Output = 5x input
	WeightCacheCreate = 1.25 // Cache write = 1.25x input
	WeightCacheRead   = 0.1  // Cache read = 0.1x input
)

// PeriodEconomics represents economics data for a single time period
type PeriodEconomics struct {
	Label string

	// ccusage metrics (Option for graceful degradation)
	CCCost         *float64
	CCTotalTokens  *uint64
	CCActiveTokens *uint64 // input + output only (excluding cache)

	// Per-type token breakdown
	CCInputTokens       *uint64
	CCOutputTokens      *uint64
	CCCacheCreateTokens *uint64
	CCCacheReadTokens   *uint64

	// tokman metrics
	TMCommands    *int
	TMSavedTokens *int
	TMSavingsPct  *float64

	// Primary metric (weighted input CPT)
	WeightedInputCPT *float64 // Derived input CPT using API ratios
	SavingsWeighted  *float64 // saved * weighted_input_cpt (PRIMARY)

	// Legacy metrics (verbose mode only)
	BlendedCPT     *float64 // cost / total_tokens (diluted by cache)
	ActiveCPT      *float64 // cost / active_tokens (OVERESTIMATES)
	SavingsBlended *float64 // saved * blended_cpt (UNDERESTIMATES)
	SavingsActive  *float64 // saved * active_cpt (OVERESTIMATES)
}

// Totals represents aggregated economics across all periods
type Totals struct {
	CCCost           float64
	CCTotalTokens    uint64
	CCActiveTokens   uint64
	CCInputTokens    uint64
	CCOutputTokens   uint64
	CCCacheCreate    uint64
	CCCacheRead      uint64
	TMCommands       int
	TMSavedTokens    int
	TMAvgSavingsPct  float64
	WeightedInputCPT *float64
	SavingsWeighted  *float64
	BlendedCPT       *float64
	ActiveCPT        *float64
	SavingsBlended   *float64
	SavingsActive    *float64
}

// DayStats represents daily tracking stats (simplified from tracking package)
type DayStats struct {
	Date        string
	Commands    int
	SavedTokens int
	SavingsPct  float64
}

// MonthStats represents monthly tracking stats
type MonthStats struct {
	Month       string
	Commands    int
	SavedTokens int
	SavingsPct  float64
}

// RunOptions configures the economics report
type RunOptions struct {
	Daily   bool
	Weekly  bool
	Monthly bool
	All     bool
	Format  string // "text", "json", "csv"
	Verbose bool
}

// Run executes the economics analysis and displays results
func Run(opts RunOptions) error {
	tracker, err := tracking.NewTracker(tracking.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to initialize tracking database: %w", err)
	}
	defer tracker.Close()

	switch opts.Format {
	case "json":
		return exportJSON(tracker, opts)
	case "csv":
		return exportCSV(tracker, opts)
	default:
		return displayText(tracker, opts)
	}
}

// mergeMonthly combines ccusage and tokman monthly data
func mergeMonthly(cc []ccusage.Period, tm []MonthStats) []PeriodEconomics {
	periodMap := make(map[string]*PeriodEconomics)

	// Insert ccusage data
	for _, entry := range cc {
		if _, exists := periodMap[entry.Key]; !exists {
			periodMap[entry.Key] = &PeriodEconomics{Label: entry.Key}
		}
		p := periodMap[entry.Key]
		p.CCCost = &entry.Metrics.TotalCost
		p.CCTotalTokens = &entry.Metrics.TotalTokens
		p.CCInputTokens = &entry.Metrics.InputTokens
		p.CCOutputTokens = &entry.Metrics.OutputTokens
		p.CCCacheCreateTokens = &entry.Metrics.CacheCreationTokens
		p.CCCacheReadTokens = &entry.Metrics.CacheReadTokens
		active := entry.Metrics.InputTokens + entry.Metrics.OutputTokens
		p.CCActiveTokens = &active
	}

	// Merge tokman data
	for _, entry := range tm {
		if _, exists := periodMap[entry.Month]; !exists {
			periodMap[entry.Month] = &PeriodEconomics{Label: entry.Month}
		}
		p := periodMap[entry.Month]
		p.TMCommands = &entry.Commands
		p.TMSavedTokens = &entry.SavedTokens
		p.TMSavingsPct = &entry.SavingsPct
	}

	// Compute metrics and sort
	result := make([]PeriodEconomics, 0, len(periodMap))
	for _, p := range periodMap {
		p.computeWeightedMetrics()
		p.computeDualMetrics()
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Label < result[j].Label
	})

	return result
}

// mergeDaily combines ccusage and tokman daily data
func mergeDaily(cc []ccusage.Period, tm []DayStats) []PeriodEconomics {
	periodMap := make(map[string]*PeriodEconomics)

	// Insert ccusage data
	for _, entry := range cc {
		if _, exists := periodMap[entry.Key]; !exists {
			periodMap[entry.Key] = &PeriodEconomics{Label: entry.Key}
		}
		p := periodMap[entry.Key]
		p.CCCost = &entry.Metrics.TotalCost
		p.CCTotalTokens = &entry.Metrics.TotalTokens
		p.CCInputTokens = &entry.Metrics.InputTokens
		p.CCOutputTokens = &entry.Metrics.OutputTokens
		p.CCCacheCreateTokens = &entry.Metrics.CacheCreationTokens
		p.CCCacheReadTokens = &entry.Metrics.CacheReadTokens
		active := entry.Metrics.InputTokens + entry.Metrics.OutputTokens
		p.CCActiveTokens = &active
	}

	// Merge tokman data
	for _, entry := range tm {
		if _, exists := periodMap[entry.Date]; !exists {
			periodMap[entry.Date] = &PeriodEconomics{Label: entry.Date}
		}
		p := periodMap[entry.Date]
		p.TMCommands = &entry.Commands
		p.TMSavedTokens = &entry.SavedTokens
		p.TMSavingsPct = &entry.SavingsPct
	}

	// Compute metrics and sort
	result := make([]PeriodEconomics, 0, len(periodMap))
	for _, p := range periodMap {
		p.computeWeightedMetrics()
		p.computeDualMetrics()
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Label < result[j].Label
	})

	return result
}

func (p *PeriodEconomics) computeWeightedMetrics() {
	if p.CCCost == nil || p.TMSavedTokens == nil {
		return
	}
	if p.CCInputTokens == nil || p.CCOutputTokens == nil ||
		p.CCCacheCreateTokens == nil || p.CCCacheReadTokens == nil {
		return
	}

	// Weighted units = input + 5*output + 1.25*cache_create + 0.1*cache_read
	weightedUnits := float64(*p.CCInputTokens) +
		WeightOutput*float64(*p.CCOutputTokens) +
		WeightCacheCreate*float64(*p.CCCacheCreateTokens) +
		WeightCacheRead*float64(*p.CCCacheReadTokens)

	if weightedUnits <= 0 {
		return
	}

	inputCPT := *p.CCCost / weightedUnits
	savings := float64(*p.TMSavedTokens) * inputCPT

	p.WeightedInputCPT = &inputCPT
	p.SavingsWeighted = &savings
}

func (p *PeriodEconomics) computeDualMetrics() {
	if p.CCCost == nil || p.TMSavedTokens == nil {
		return
	}

	// Blended CPT (cost / total_tokens including cache)
	if p.CCTotalTokens != nil && *p.CCTotalTokens > 0 {
		blended := *p.CCCost / float64(*p.CCTotalTokens)
		p.BlendedCPT = &blended
		savings := float64(*p.TMSavedTokens) * blended
		p.SavingsBlended = &savings
	}

	// Active CPT (cost / active_tokens = input+output only)
	if p.CCActiveTokens != nil && *p.CCActiveTokens > 0 {
		active := *p.CCCost / float64(*p.CCActiveTokens)
		p.ActiveCPT = &active
		savings := float64(*p.TMSavedTokens) * active
		p.SavingsActive = &savings
	}
}

func computeTotals(periods []PeriodEconomics) Totals {
	var totals Totals
	var pctSum float64
	var pctCount int

	for _, p := range periods {
		if p.CCCost != nil {
			totals.CCCost += *p.CCCost
		}
		if p.CCTotalTokens != nil {
			totals.CCTotalTokens += *p.CCTotalTokens
		}
		if p.CCActiveTokens != nil {
			totals.CCActiveTokens += *p.CCActiveTokens
		}
		if p.CCInputTokens != nil {
			totals.CCInputTokens += *p.CCInputTokens
		}
		if p.CCOutputTokens != nil {
			totals.CCOutputTokens += *p.CCOutputTokens
		}
		if p.CCCacheCreateTokens != nil {
			totals.CCCacheCreate += *p.CCCacheCreateTokens
		}
		if p.CCCacheReadTokens != nil {
			totals.CCCacheRead += *p.CCCacheReadTokens
		}
		if p.TMCommands != nil {
			totals.TMCommands += *p.TMCommands
		}
		if p.TMSavedTokens != nil {
			totals.TMSavedTokens += *p.TMSavedTokens
		}
		if p.TMSavingsPct != nil {
			pctSum += *p.TMSavingsPct
			pctCount++
		}
	}

	if pctCount > 0 {
		totals.TMAvgSavingsPct = pctSum / float64(pctCount)
	}

	// Compute global weighted metrics
	weightedUnits := float64(totals.CCInputTokens) +
		WeightOutput*float64(totals.CCOutputTokens) +
		WeightCacheCreate*float64(totals.CCCacheCreate) +
		WeightCacheRead*float64(totals.CCCacheRead)

	if weightedUnits > 0 {
		inputCPT := totals.CCCost / weightedUnits
		totals.WeightedInputCPT = &inputCPT
		savings := float64(totals.TMSavedTokens) * inputCPT
		totals.SavingsWeighted = &savings
	}

	// Compute global dual metrics
	if totals.CCTotalTokens > 0 {
		blended := totals.CCCost / float64(totals.CCTotalTokens)
		totals.BlendedCPT = &blended
		savings := float64(totals.TMSavedTokens) * blended
		totals.SavingsBlended = &savings
	}
	if totals.CCActiveTokens > 0 {
		active := totals.CCCost / float64(totals.CCActiveTokens)
		totals.ActiveCPT = &active
		savings := float64(totals.TMSavedTokens) * active
		totals.SavingsActive = &savings
	}

	return totals
}

func displayText(tracker *tracking.Tracker, opts RunOptions) error {
	// Default: summary view
	if !opts.Daily && !opts.Weekly && !opts.Monthly && !opts.All {
		return displaySummary(tracker, opts.Verbose)
	}

	if opts.All || opts.Daily {
		if err := displayDaily(tracker, opts.Verbose); err != nil {
			return err
		}
	}
	if opts.All || opts.Weekly {
		if err := displayWeekly(tracker, opts.Verbose); err != nil {
			return err
		}
	}
	if opts.All || opts.Monthly {
		if err := displayMonthly(tracker, opts.Verbose); err != nil {
			return err
		}
	}

	return nil
}

func displaySummary(tracker *tracking.Tracker, verbose bool) error {
	green := color.New(color.FgGreen).SprintFunc()

	ccMonthly, err := ccusage.Fetch(ccusage.Monthly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// Get monthly savings from tracker
	tmMonthly := getMonthlyStats(tracker)
	periods := mergeMonthly(ccMonthly, tmMonthly)

	if len(periods) == 0 {
		fmt.Println("No data available. Run some tokman commands to start tracking.")
		return nil
	}

	totals := computeTotals(periods)

	fmt.Println()
	fmt.Printf("%s\n", green("💰 Claude Code Economics"))
	fmt.Println("════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("  Spent (ccusage):              %s\n", formatUSD(totals.CCCost))
	fmt.Println("  Token breakdown:")
	fmt.Printf("    Input:                      %s\n", formatTokens(totals.CCInputTokens))
	fmt.Printf("    Output:                     %s\n", formatTokens(totals.CCOutputTokens))
	fmt.Printf("    Cache writes:               %s\n", formatTokens(totals.CCCacheCreate))
	fmt.Printf("    Cache reads:                %s\n", formatTokens(totals.CCCacheRead))
	fmt.Println()

	fmt.Printf("  TokMan commands:              %d\n", totals.TMCommands)
	fmt.Printf("  Tokens saved:                 %s\n", formatTokens(uint64(totals.TMSavedTokens)))
	fmt.Println()

	fmt.Println("  Estimated Savings:")
	fmt.Println("  ┌─────────────────────────────────────────────────┐")

	if totals.SavingsWeighted != nil {
		weightedPct := 0.0
		if totals.CCCost > 0 {
			weightedPct = (*totals.SavingsWeighted / totals.CCCost) * 100
		}
		fmt.Printf("  │ Input token pricing:   %-12s (%.1f%%)        │\n",
			strings.TrimSuffix(formatUSD(*totals.SavingsWeighted), " "), weightedPct)
		if totals.WeightedInputCPT != nil {
			fmt.Printf("  │ Derived input CPT:     %-24s        │\n",
				formatCPT(*totals.WeightedInputCPT))
		}
	} else {
		fmt.Println("  │ Input token pricing:   —                         │")
	}

	fmt.Println("  └─────────────────────────────────────────────────┘")
	fmt.Println()

	fmt.Println("  How it works:")
	fmt.Println("  TokMan compresses CLI outputs before they enter Claude's context.")
	fmt.Println("  Savings derived using API price ratios (out=5x, cache_w=1.25x, cache_r=0.1x).")
	fmt.Println()

	if verbose {
		fmt.Println("  Legacy metrics (reference only):")
		if totals.SavingsActive != nil {
			activePct := 0.0
			if totals.CCCost > 0 {
				activePct = (*totals.SavingsActive / totals.CCCost) * 100
			}
			fmt.Printf("    Active (OVERESTIMATES):  %s (%.1f%%)\n",
				formatUSD(*totals.SavingsActive), activePct)
		}
		if totals.SavingsBlended != nil {
			blendedPct := 0.0
			if totals.CCCost > 0 {
				blendedPct = (*totals.SavingsBlended / totals.CCCost) * 100
			}
			fmt.Printf("    Blended (UNDERESTIMATES): %s (%.2f%%)\n",
				formatUSD(*totals.SavingsBlended), blendedPct)
		}
		fmt.Println("  Note: Saved tokens estimated via chars/4 heuristic, not exact tokenizer.")
		fmt.Println()
	}

	return nil
}

func displayDaily(tracker *tracking.Tracker, verbose bool) error {
	ccDaily, err := ccusage.Fetch(ccusage.Daily)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	tmDaily := getDailyStats(tracker)
	periods := mergeDaily(ccDaily, tmDaily)

	fmt.Println()
	fmt.Println("📅 Daily Economics")
	fmt.Println("════════════════════════════════════════════════════")
	printPeriodTable(periods, verbose)
	return nil
}

func displayWeekly(tracker *tracking.Tracker, verbose bool) error {
	ccWeekly, err := ccusage.Fetch(ccusage.Weekly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	tmWeekly := getWeeklyStats(tracker)
	periods := mergeWeekly(ccWeekly, tmWeekly)

	fmt.Println()
	fmt.Println("📅 Weekly Economics")
	fmt.Println("════════════════════════════════════════════════════")
	printPeriodTable(periods, verbose)
	return nil
}

func displayMonthly(tracker *tracking.Tracker, verbose bool) error {
	ccMonthly, err := ccusage.Fetch(ccusage.Monthly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	tmMonthly := getMonthlyStats(tracker)
	periods := mergeMonthly(ccMonthly, tmMonthly)

	fmt.Println()
	fmt.Println("📅 Monthly Economics")
	fmt.Println("════════════════════════════════════════════════════")
	printPeriodTable(periods, verbose)
	return nil
}

// mergeWeekly combines ccusage and tokman weekly data
func mergeWeekly(cc []ccusage.Period, tm []WeekStats) []PeriodEconomics {
	periodMap := make(map[string]*PeriodEconomics)

	// Insert ccusage data (key = ISO Monday "2026-01-20")
	for _, entry := range cc {
		if _, exists := periodMap[entry.Key]; !exists {
			periodMap[entry.Key] = &PeriodEconomics{Label: entry.Key}
		}
		p := periodMap[entry.Key]
		p.CCCost = &entry.Metrics.TotalCost
		p.CCTotalTokens = &entry.Metrics.TotalTokens
		p.CCInputTokens = &entry.Metrics.InputTokens
		p.CCOutputTokens = &entry.Metrics.OutputTokens
		p.CCCacheCreateTokens = &entry.Metrics.CacheCreationTokens
		p.CCCacheReadTokens = &entry.Metrics.CacheReadTokens
		active := entry.Metrics.InputTokens + entry.Metrics.OutputTokens
		p.CCActiveTokens = &active
	}

	// Merge tokman data (week_start = legacy Saturday)
	// TODO: Convert Saturday to Monday for alignment if needed
	for _, entry := range tm {
		key := entry.WeekStart
		if _, exists := periodMap[key]; !exists {
			periodMap[key] = &PeriodEconomics{Label: key}
		}
		p := periodMap[key]
		p.TMCommands = &entry.Commands
		p.TMSavedTokens = &entry.SavedTokens
		p.TMSavingsPct = &entry.SavingsPct
	}

	// Compute metrics and sort
	result := make([]PeriodEconomics, 0, len(periodMap))
	for _, p := range periodMap {
		p.computeWeightedMetrics()
		p.computeDualMetrics()
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Label < result[j].Label
	})

	return result
}

// WeekStats represents weekly tracking stats
type WeekStats struct {
	WeekStart   string
	Commands    int
	SavedTokens int
	SavingsPct  float64
}

func printPeriodTable(periods []PeriodEconomics, verbose bool) {
	fmt.Println()

	green := color.New(color.FgGreen).SprintFunc()

	if verbose {
		fmt.Printf("%-12s %10s %10s %10s %12s %12s\n",
			"Period", "Spent", "Saved", "Savings", "Active$", "Blended$")
		fmt.Println(strings.Repeat("-", 78))

		for _, p := range periods {
			spent := "—"
			if p.CCCost != nil {
				spent = formatUSD(*p.CCCost)
			}
			saved := "—"
			if p.TMSavedTokens != nil {
				saved = formatTokens(uint64(*p.TMSavedTokens))
			}
			weighted := "—"
			if p.SavingsWeighted != nil {
				weighted = formatUSD(*p.SavingsWeighted)
			}
			active := "—"
			if p.SavingsActive != nil {
				active = formatUSD(*p.SavingsActive)
			}
			blended := "—"
			if p.SavingsBlended != nil {
				blended = formatUSD(*p.SavingsBlended)
			}

			fmt.Printf("%-12s %10s %10s %10s %12s %12s\n",
				p.Label, spent, saved, green(weighted), active, blended)
		}
	} else {
		fmt.Printf("%-12s %10s %10s %10s %12s\n",
			"Period", "Spent", "Saved", "Savings", "TM Cmds")
		fmt.Println(strings.Repeat("-", 58))

		for _, p := range periods {
			spent := "—"
			if p.CCCost != nil {
				spent = formatUSD(*p.CCCost)
			}
			saved := "—"
			if p.TMSavedTokens != nil {
				saved = formatTokens(uint64(*p.TMSavedTokens))
			}
			weighted := "—"
			if p.SavingsWeighted != nil {
				weighted = formatUSD(*p.SavingsWeighted)
			}
			cmds := "—"
			if p.TMCommands != nil {
				cmds = fmt.Sprintf("%d", *p.TMCommands)
			}

			fmt.Printf("%-12s %10s %10s %10s %12s\n",
				p.Label, spent, saved, green(weighted), cmds)
		}
	}
	fmt.Println()
}

// getMonthlyStats retrieves monthly stats from tracker
func getMonthlyStats(tracker *tracking.Tracker) []MonthStats {
	query := `
		SELECT 
			STRFTIME('%Y-%m', timestamp) as month,
			COUNT(*) as commands,
			COALESCE(SUM(saved_tokens), 0) as saved,
			COALESCE(SUM(original_tokens), 0) as original
		FROM commands
		GROUP BY STRFTIME('%Y-%m', timestamp)
		ORDER BY month DESC
	`

	rows, err := tracker.Query(query)
	if err != nil {
		return []MonthStats{}
	}
	defer rows.Close()

	var stats []MonthStats
	for rows.Next() {
		var s MonthStats
		var saved, original int
		if err := rows.Scan(&s.Month, &s.Commands, &saved, &original); err != nil {
			continue
		}
		s.SavedTokens = saved
		if original > 0 {
			s.SavingsPct = float64(saved) / float64(original) * 100
		}
		stats = append(stats, s)
	}

	return stats
}

// getDailyStats retrieves daily stats from tracker
func getDailyStats(tracker *tracking.Tracker) []DayStats {
	query := `
		SELECT 
			DATE(timestamp) as date,
			COUNT(*) as commands,
			COALESCE(SUM(saved_tokens), 0) as saved,
			COALESCE(SUM(original_tokens), 0) as original
		FROM commands
		GROUP BY DATE(timestamp)
		ORDER BY date DESC
	`

	rows, err := tracker.Query(query)
	if err != nil {
		return []DayStats{}
	}
	defer rows.Close()

	var stats []DayStats
	for rows.Next() {
		var s DayStats
		var saved, original int
		if err := rows.Scan(&s.Date, &s.Commands, &saved, &original); err != nil {
			continue
		}
		s.SavedTokens = saved
		if original > 0 {
			s.SavingsPct = float64(saved) / float64(original) * 100
		}
		stats = append(stats, s)
	}

	return stats
}

// getWeeklyStats retrieves weekly stats from tracker
func getWeeklyStats(tracker *tracking.Tracker) []WeekStats {
	query := `
		SELECT 
			DATE(timestamp, 'weekday 0', '-6 days') as week_start,
			COUNT(*) as commands,
			COALESCE(SUM(saved_tokens), 0) as saved,
			COALESCE(SUM(original_tokens), 0) as original
		FROM commands
		GROUP BY DATE(timestamp, 'weekday 0', '-6 days')
		ORDER BY week_start DESC
	`

	rows, err := tracker.Query(query)
	if err != nil {
		return []WeekStats{}
	}
	defer rows.Close()

	var stats []WeekStats
	for rows.Next() {
		var s WeekStats
		var saved, original int
		if err := rows.Scan(&s.WeekStart, &s.Commands, &saved, &original); err != nil {
			continue
		}
		s.SavedTokens = saved
		if original > 0 {
			s.SavingsPct = float64(saved) / float64(original) * 100
		}
		stats = append(stats, s)
	}

	return stats
}

// formatUSD formats a USD amount
func formatUSD(amount float64) string {
	return fmt.Sprintf("$%.2f", amount)
}

// formatTokens formats a token count with K/M/B suffixes
func formatTokens(count uint64) string {
	const billion = 1e9
	const million = 1e6
	const thousand = 1e3

	if count >= billion {
		return fmt.Sprintf("%.1fB", float64(count)/billion)
	}
	if count >= million {
		return fmt.Sprintf("%.1fM", float64(count)/million)
	}
	if count >= thousand {
		return fmt.Sprintf("%.1fK", float64(count)/thousand)
	}
	return fmt.Sprintf("%d", count)
}

// formatCPT formats cost-per-token
func formatCPT(cpt float64) string {
	return fmt.Sprintf("$%.6f/tok", cpt)
}

func exportJSON(tracker *tracking.Tracker, opts RunOptions) error {
	// Gather data based on options
	ccMonthly, _ := ccusage.Fetch(ccusage.Monthly)
	ccDaily, _ := ccusage.Fetch(ccusage.Daily)
	ccWeekly, _ := ccusage.Fetch(ccusage.Weekly)

	tmMonthly := getMonthlyStats(tracker)
	tmDaily := getDailyStats(tracker)
	tmWeekly := getWeeklyStats(tracker)

	export := struct {
		GeneratedAt   string             `json:"generated_at"`
		PricingRatios map[string]float64 `json:"pricing_ratios"`
		Daily         []PeriodEconomics  `json:"daily,omitempty"`
		Weekly        []PeriodEconomics  `json:"weekly,omitempty"`
		Monthly       []PeriodEconomics  `json:"monthly,omitempty"`
		Summary       *Totals            `json:"summary,omitempty"`
	}{
		GeneratedAt: time.Now().Format(time.RFC3339),
		PricingRatios: map[string]float64{
			"output":       WeightOutput,
			"cache_create": WeightCacheCreate,
			"cache_read":   WeightCacheRead,
		},
	}

	// Include requested periods
	if opts.All || opts.Daily {
		export.Daily = mergeDaily(ccDaily, tmDaily)
	}
	if opts.All || opts.Weekly {
		export.Weekly = mergeWeekly(ccWeekly, tmWeekly)
	}
	if opts.All || opts.Monthly || (!opts.Daily && !opts.Weekly && !opts.All) {
		export.Monthly = mergeMonthly(ccMonthly, tmMonthly)
	}

	// Compute summary
	allPeriods := append(export.Daily, export.Weekly...)
	allPeriods = append(allPeriods, export.Monthly...)
	summary := computeTotals(allPeriods)
	export.Summary = &summary

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(export)
}

func exportCSV(tracker *tracking.Tracker, opts RunOptions) error {
	// Gather data
	ccMonthly, _ := ccusage.Fetch(ccusage.Monthly)
	ccDaily, _ := ccusage.Fetch(ccusage.Daily)
	ccWeekly, _ := ccusage.Fetch(ccusage.Weekly)

	tmMonthly := getMonthlyStats(tracker)
	tmDaily := getDailyStats(tracker)
	tmWeekly := getWeeklyStats(tracker)

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Header
	header := []string{
		"period_type", "period", "cc_cost", "cc_total_tokens", "cc_input_tokens",
		"cc_output_tokens", "cc_cache_create", "cc_cache_read",
		"tm_commands", "tm_saved_tokens", "tm_savings_pct",
		"weighted_input_cpt", "savings_weighted",
	}
	if opts.Verbose {
		header = append(header, "blended_cpt", "active_cpt", "savings_blended", "savings_active")
	}
	writer.Write(header)

	writeRows := func(periodType string, periods []PeriodEconomics) error {
		for _, p := range periods {
			row := []string{periodType, p.Label}

			// ccusage fields
			row = append(row, formatOptionalFloat(p.CCCost))
			row = append(row, formatOptionalUint(p.CCTotalTokens))
			row = append(row, formatOptionalUint(p.CCInputTokens))
			row = append(row, formatOptionalUint(p.CCOutputTokens))
			row = append(row, formatOptionalUint(p.CCCacheCreateTokens))
			row = append(row, formatOptionalUint(p.CCCacheReadTokens))

			// tokman fields
			row = append(row, formatOptionalInt(p.TMCommands))
			row = append(row, formatOptionalInt(p.TMSavedTokens))
			row = append(row, formatOptionalFloat(p.TMSavingsPct))

			// weighted metrics
			row = append(row, formatOptionalFloat(p.WeightedInputCPT))
			row = append(row, formatOptionalFloat(p.SavingsWeighted))

			// verbose metrics
			if opts.Verbose {
				row = append(row, formatOptionalFloat(p.BlendedCPT))
				row = append(row, formatOptionalFloat(p.ActiveCPT))
				row = append(row, formatOptionalFloat(p.SavingsBlended))
				row = append(row, formatOptionalFloat(p.SavingsActive))
			}

			if err := writer.Write(row); err != nil {
				return err
			}
		}
		return nil
	}

	// Write data based on options
	if opts.All || opts.Daily {
		if err := writeRows("daily", mergeDaily(ccDaily, tmDaily)); err != nil {
			return err
		}
	}
	if opts.All || opts.Weekly {
		if err := writeRows("weekly", mergeWeekly(ccWeekly, tmWeekly)); err != nil {
			return err
		}
	}
	if opts.All || opts.Monthly || (!opts.Daily && !opts.Weekly && !opts.All) {
		if err := writeRows("monthly", mergeMonthly(ccMonthly, tmMonthly)); err != nil {
			return err
		}
	}

	return nil
}

// formatOptionalFloat formats an optional float for CSV output
func formatOptionalFloat(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.6f", *v)
}

// formatOptionalUint formats an optional uint64 for CSV output
func formatOptionalUint(v *uint64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

// formatOptionalInt formats an optional int for CSV output
func formatOptionalInt(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

// GetDailyStats is a public wrapper for daily stats (used by dashboard)
func GetDailyStats(tracker *tracking.Tracker) []DayStats {
	return getDailyStats(tracker)
}

// MergeDailyLite merges ccusage and tokman daily data (used by dashboard)
func MergeDailyLite(cc []ccusage.Period, tm []DayStats) []PeriodEconomics {
	return mergeDaily(cc, tm)
}
