package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	reportToday bool
	reportWeek  bool
	reportJSON  bool
	reportLimit int
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate detailed usage reports",
	Long: `Generate detailed reports of token savings with various filters
and output formats.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := GetConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dbPath := cfg.GetDatabasePath()
		tracker, err := tracking.NewTracker(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
			os.Exit(1)
		}
		defer tracker.Close()

		projectPath := config.ProjectPath()

		// Determine time range
		days := 30 // default to 30 days
		if reportToday {
			days = 1
		} else if reportWeek {
			days = 7
		}

		// Get daily savings
		daily, err := tracker.GetDailySavings(projectPath, days)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting daily savings: %v\n", err)
			os.Exit(1)
		}

		// Get command stats
		stats, err := tracker.GetCommandStats(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting command stats: %v\n", err)
			os.Exit(1)
		}

		// Get recent commands
		recent, err := tracker.GetRecentCommands(projectPath, reportLimit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting recent commands: %v\n", err)
			os.Exit(1)
		}

		if reportJSON {
			outputJSON(daily, stats, recent)
			return
		}

		outputTable(daily, stats, recent)
	},
}

func outputJSON(daily []struct {
	Date     string
	Saved    int
	Original int
	Commands int
}, stats []tracking.CommandStats, recent []tracking.CommandRecord) {
	report := map[string]interface{}{
		"daily_savings":   daily,
		"command_stats":   stats,
		"recent_commands": recent,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(report)
}

func outputTable(daily []struct {
	Date     string
	Saved    int
	Original int
	Commands int
}, stats []tracking.CommandStats, recent []tracking.CommandRecord) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("\n%s\n", green("📈 TokMan Report"))
	fmt.Println(strings.Repeat("─", 60))

	// Daily savings
	if len(daily) > 0 {
		fmt.Printf("\n%s\n", green("Daily Savings"))
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("%-12s  %-10s  %-10s  %-8s\n", "Date", "Saved", "Original", "Commands")
		fmt.Println(strings.Repeat("─", 60))
		for _, d := range daily {
			fmt.Printf("%-12s  %-10s  %-10s  %-8d\n",
				d.Date,
				green(fmt.Sprintf("%d", d.Saved)),
				cyan(fmt.Sprintf("%d", d.Original)),
				d.Commands,
			)
		}
	}

	// Command statistics
	if len(stats) > 0 {
		fmt.Printf("\n%s\n", green("Command Statistics"))
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("%-20s  %-8s  %-10s  %-8s\n", "Command", "Runs", "Saved", "Reduction")
		fmt.Println(strings.Repeat("─", 60))
		for _, s := range stats {
			fmt.Printf("%-20s  %-8d  %-10s  %s\n",
				truncate(s.Command, 20),
				s.ExecutionCount,
				green(fmt.Sprintf("%d", s.TotalSaved)),
				green(fmt.Sprintf("%.1f%%", s.ReductionPct)),
			)
		}
	}

	// Recent commands
	if len(recent) > 0 {
		fmt.Printf("\n%s\n", green("Recent Commands"))
		fmt.Println(strings.Repeat("─", 60))
		for _, r := range recent {
			status := green("✓")
			if !r.ParseSuccess {
				status = yellow("⚠")
			}
			fmt.Printf("  %s %-20s saved: %s\n",
				status,
				truncate(r.Command, 20),
				green(fmt.Sprintf("%d", r.SavedTokens)),
			)
		}
	}

	fmt.Println()
}

func truncateReport(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().BoolVar(&reportToday, "today", false, "Show only today's data")
	reportCmd.Flags().BoolVar(&reportWeek, "week", false, "Show last 7 days")
	reportCmd.Flags().BoolVar(&reportJSON, "json", false, "Output in JSON format")
	reportCmd.Flags().IntVarP(&reportLimit, "limit", "n", 10, "Number of recent commands to show")

	reportCmd.MarkFlagsMutuallyExclusive("today", "week")
}
