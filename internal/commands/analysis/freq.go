package analysis

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var freqLimit int

var freqCmd = &cobra.Command{
	Use:   "freq",
	Short: "Show command frequency analysis",
	Long: `Display which commands are compressed most often,
sorted by execution count and total tokens saved.`,
	RunE: runFreq,
}

func init() {
	freqCmd.Flags().IntVarP(&freqLimit, "limit", "n", 20, "number of commands to show")
	registry.Add(func() { registry.Register(freqCmd) })
}

func runFreq(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	cwd, _ := os.Getwd()
	stats, err := tracker.GetCommandStats(cwd)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	if len(stats) == 0 {
		fmt.Println("No command data found. Run some commands through tokman first.")
		return nil
	}

	fmt.Printf("Top %d Commands by Token Savings\n", freqLimit)
	fmt.Printf("%-30s %8s %10s %8s\n", "Command", "Count", "Saved", "Pct")
	fmt.Printf("%-30s %8s %10s %8s\n", "──────────────────────────────", "────────", "──────────", "────────")

	limit := freqLimit
	if limit > len(stats) {
		limit = len(stats)
	}

	for i := 0; i < limit; i++ {
		s := stats[i]
		fmt.Printf("%-30s %8d %10d %7.1f%%\n",
			truncateStr(s.Command, 30),
			s.ExecutionCount,
			s.TotalSaved,
			s.ReductionPct,
		)
	}

	return nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
