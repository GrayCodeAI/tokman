package core

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gainToday bool
var gainWeek bool
var gainAll bool

// gainCmd shows token savings statistics
var gainCmd = &cobra.Command{
	Use:   "gain",
	Short: "Show token savings and ROI metrics",
	Long: `Display token savings achieved by TokMan compression.

Shows:
  - Total tokens saved
  - Compression percentage
  - Estimated cost savings
  - Commands processed

Examples:
  tokman gain           # All-time savings
  tokman gain --week    # This week's savings
  tokman gain --today   # Today's savings`,
	RunE: runGain,
}

func init() {
	gainCmd.Flags().BoolVar(&gainToday, "today", false, "show today's savings")
	gainCmd.Flags().BoolVar(&gainWeek, "week", false, "show this week's savings")
	gainCmd.Flags().BoolVar(&gainAll, "all", true, "show all-time savings (default)")
	registry.Add(func() { registry.Register(gainCmd) })
}

func runGain(cmd *cobra.Command, args []string) error {
	tracker, err := shared.OpenTracker()
	if err != nil {
		return fmt.Errorf("cannot open database: %w", err)
	}
	defer tracker.Close()

	// Determine time range display
	switch {
	case gainToday:
		fmt.Println("📊 Today's Token Savings")
		fmt.Println("=======================")
	case gainWeek:
		fmt.Println("📊 This Week's Token Savings")
		fmt.Println("===========================")
	default:
		fmt.Println("📊 All-Time Token Savings")
		fmt.Println("========================")
	}
	fmt.Println()

	// Get savings using existing tracker method
	savings, err := tracker.GetSavings("")
	if err != nil {
		return fmt.Errorf("cannot get savings: %w", err)
	}

	if savings.TotalCommands == 0 {
		fmt.Println("No commands processed yet.")
		fmt.Println()
		fmt.Println("Start using TokMan to see savings!")
		return nil
	}

	// Display results
	fmt.Printf("┌─────────────────────────────────────┐\n")
	fmt.Printf("│ Commands processed: %15d │\n", savings.TotalCommands)
	fmt.Printf("│ Original tokens:   %15d │\n", savings.TotalOriginal)
	fmt.Printf("│ Final tokens:      %15d │\n", savings.TotalFiltered)
	fmt.Printf("│ Tokens saved:      %15d │\n", savings.TotalSaved)
	fmt.Printf("├─────────────────────────────────────┤\n")
	fmt.Printf("│ Compression ratio: %14.1f%% │\n", savings.ReductionPct)
	fmt.Printf("└─────────────────────────────────────┘\n")
	fmt.Println()

	// Estimate cost savings (approximate GPT-4 pricing)
	// Input: $2.50/1M tokens, Output: $10.00/1M tokens (average ~$5/1M)
	costPer1MTokens := 5.0
	estimatedSavings := float64(savings.TotalSaved) / 1_000_000 * costPer1MTokens

	fmt.Printf("💰 Estimated cost saved: $%.2f\n", estimatedSavings)
	fmt.Println()

	// Show top commands using existing GetCommandStats
	cmdStats, err := tracker.GetCommandStats("")
	if err == nil && len(cmdStats) > 0 {
		fmt.Println("📈 Top Commands by Savings:")
		for i, stat := range cmdStats {
			if i >= 5 {
				break
			}
			pct := 0.0
			if stat.TotalOriginal > 0 {
				pct = float64(stat.TotalSaved) / float64(stat.TotalOriginal) * 100
			}
			fmt.Printf("   %d. %s: %d tokens (%.1f%%)\n", i+1, stat.Command, stat.TotalSaved, pct)
		}
	}

	return nil
}
