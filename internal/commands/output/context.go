package output

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show context window usage analysis",
	Long:  `Analyze how your context window is being used and suggest optimizations.`,
	RunE:  runContext,
}

func init() {
	registry.Add(func() { registry.Register(contextCmd) })
}

func runContext(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	cwd, _ := os.Getwd()
	savings, err := tracker.GetSavings(cwd)
	if err != nil {
		return fmt.Errorf("failed to get context data: %w", err)
	}

	fmt.Println("Context Window Analysis")
	fmt.Println("======================")
	fmt.Println()

	if savings.TotalCommands == 0 {
		fmt.Println("No data yet. Run some commands through tokman first.")
		return nil
	}

	fmt.Printf("Commands analyzed: %d\n", savings.TotalCommands)
	fmt.Printf("Original context:  %d tokens\n", savings.TotalOriginal)
	fmt.Printf("Filtered context:  %d tokens\n", savings.TotalFiltered)
	fmt.Printf("Tokens saved:      %d tokens\n", savings.TotalSaved)
	fmt.Printf("Reduction:         %.1f%%\n\n", savings.ReductionPct)

	contextSizes := []struct {
		name  string
		limit int
	}{
		{"GPT-4o-mini (128K)", 128000},
		{"GPT-4o (128K)", 128000},
		{"Claude 3.5 (200K)", 200000},
		{"Claude 3 Opus (200K)", 200000},
		{"Gemini 1.5 (1M)", 1000000},
	}

	fmt.Println("Context window capacity with tokman:")
	fmt.Printf("%-25s %12s %12s %10s\n", "Model", "Without", "With", "Extra")
	fmt.Printf("%-25s %12s %12s %10s\n", "─────────────────────────", "────────────", "────────────", "──────────")

	for _, cs := range contextSizes {
		without := cs.limit / savings.TotalOriginal
		if without == 0 {
			without = 1
		}
		with := cs.limit / savings.TotalFiltered
		if with == 0 {
			with = 1
		}
		extra := with - without
		fmt.Printf("%-25s %10dx %10dx +%dx\n", cs.name, without, with, extra)
	}

	return nil
}
