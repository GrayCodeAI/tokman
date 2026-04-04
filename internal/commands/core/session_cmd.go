package core

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Show TokMan adoption across sessions",
	Long:  `Display session history, adoption rate, and token savings per session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSession()
	},
}

func runSession() error {
	tracker, err := shared.OpenTracker()
	if err != nil {
		return err
	}
	defer tracker.Close()

	recent, err := tracker.GetRecentCommands("", 500)
	if err != nil {
		return err
	}

	if len(recent) == 0 {
		fmt.Println("No session data available.")
		return nil
	}

	totalCmds := len(recent)
	totalSaved := 0
	totalOriginal := 0
	for _, r := range recent {
		totalSaved += r.SavedTokens
		totalOriginal += r.OriginalTokens
	}

	avgSavings := percentOf(totalSaved, totalOriginal)

	fmt.Printf("Session Summary (last %d commands)\n", totalCmds)
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("  Commands:     %d\n", totalCmds)
	fmt.Printf("  Tokens saved: %s\n", formatTokens(totalSaved))
	fmt.Printf("  Avg savings:  %.1f%%\n", avgSavings)
	fmt.Printf("  Adoption:     %.0f%%\n", percentOf(totalSaved, totalOriginal))
	return nil
}

func percentOf(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func formatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func init() {
	registry.Add(func() { registry.Register(sessionCmd) })
}
