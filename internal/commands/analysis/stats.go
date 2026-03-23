package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var statsJSON bool

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show token savings statistics",
	Long:  `Display aggregate token savings for the current project.`,
	RunE:  runStats,
}

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "JSON output for machine consumption")
	registry.Add(func() { registry.Register(statsCmd) })
}

func runStats(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	cwd, _ := os.Getwd()

	// Get overall savings
	savings, err := tracker.GetSavings(cwd)
	if err != nil {
		return fmt.Errorf("failed to get savings: %w", err)
	}

	// Get 24h savings
	saved24h, _ := tracker.TokensSaved24h()
	totalSaved, _ := tracker.TokensSavedTotal()
	overallPct, _ := tracker.OverallSavingsPct()
	topCommands, _ := tracker.TopCommands(5)

	// Get today's command count
	today := time.Now().Add(-24 * time.Hour)
	todayCount, _ := tracker.CountCommandsSince(today)

	if statsJSON {
		output := map[string]any{
			"project":           cwd,
			"total_commands":    savings.TotalCommands,
			"total_original":    savings.TotalOriginal,
			"total_filtered":    savings.TotalFiltered,
			"total_saved":       savings.TotalSaved,
			"reduction_percent": savings.ReductionPct,
			"saved_24h":         saved24h,
			"saved_total":       totalSaved,
			"overall_percent":   overallPct,
			"commands_today":    todayCount,
			"top_commands":      topCommands,
			"cost_savings":      core.CalculateSavings(int(totalSaved), "gpt-4o-mini"),
		}
		return json.NewEncoder(os.Stdout).Encode(output)
	}

	fmt.Println("Token Savings Statistics")
	fmt.Println("========================")
	fmt.Printf("Project: %s\n\n", cwd)

	fmt.Printf("Commands tracked: %d\n", savings.TotalCommands)
	fmt.Printf("Commands today:   %d\n\n", todayCount)

	fmt.Printf("Original tokens:  %d\n", savings.TotalOriginal)
	fmt.Printf("Filtered tokens:  %d\n", savings.TotalFiltered)
	fmt.Printf("Tokens saved:     %d\n", savings.TotalSaved)
	fmt.Printf("Reduction:        %.1f%%\n\n", savings.ReductionPct)

	fmt.Printf("Saved (24h):      %d tokens\n", saved24h)
	fmt.Printf("Saved (total):    %d tokens\n", totalSaved)
	fmt.Printf("Overall:          %.1f%%\n\n", overallPct)

	// Cost savings
	dailyCost := core.CalculateSavings(int(saved24h), "gpt-4o-mini")
	totalCost := core.CalculateSavings(int(totalSaved), "gpt-4o-mini")
	fmt.Printf("Estimated savings:\n")
	fmt.Printf("  Today: $%.4f\n", dailyCost)
	fmt.Printf("  Total: $%.4f\n", totalCost)

	if len(topCommands) > 0 {
		fmt.Printf("\nTop commands:\n")
		for i, cmd := range topCommands {
			fmt.Printf("  %d. %s\n", i+1, cmd)
		}
	}

	return nil
}
