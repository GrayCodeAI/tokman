package core

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/core"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Find missed token savings opportunities",
	Long:  `Analyze command history to find commands that could benefit from TokMan compression.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDiscover()
	},
}

func runDiscover() error {
	analyzer := core.NewDiscoverAnalyzer()

	tracker, err := shared.OpenTracker()
	if err != nil {
		return err
	}
	defer tracker.Close()

	recent, err := tracker.GetRecentCommands("", 500)
	if err != nil {
		return err
	}

	var commands []string
	for _, r := range recent {
		commands = append(commands, r.Command)
	}

	results := analyzer.AnalyzeBatch(commands)
	if len(results) == 0 {
		fmt.Println("No missed savings found. All commands are already optimized!")
		return nil
	}

	totalSavings := 0
	fmt.Printf("Found %d opportunities for optimization:\n\n", len(results))
	fmt.Printf("%-30s  %-15s  %s\n", "Command", "Est. Savings", "Suggestion")
	fmt.Println("────────────────────────────────────────────────────────────────────────────")
	for _, r := range results {
		fmt.Printf("%-30s  %6d tokens  %s\n",
			truncCmd(r.Command, 30),
			r.EstSavings,
			r.Suggestion)
		totalSavings += r.EstSavings
	}
	fmt.Println("────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("\nTotal potential savings: %d tokens\n", totalSavings)
	return nil
}

func init() {
	registry.Add(func() { registry.Register(discoverCmd) })
}
