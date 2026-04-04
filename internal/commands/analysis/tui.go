package analysis

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var tuiLegacyCmd = &cobra.Command{
	Use:   "tui-legacy",
	Short: "Legacy TUI for token analytics",
	Long:  `Legacy terminal UI for monitoring token usage. Use "tokman tui" for the new interactive TUI.`,
	RunE:  runTUILegacy,
}

func init() {
	registry.Add(func() { registry.Register(tuiLegacyCmd) })
}

func runTUILegacy(cmd *cobra.Command, args []string) error {
	tracker, err := shared.OpenTracker()
	if err != nil {
		return fmt.Errorf("tracker not initialized: %w", err)
	}
	defer tracker.Close()
	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Println("║          TokMan Token Analytics TUI                ║")
	fmt.Println("╠════════════════════════════════════════════════════╣")
	savings, _ := tracker.GetSavings("")
	if savings != nil {
		fmt.Printf("║ Total Commands: %-6d                         ║\n", savings.TotalCommands)
		fmt.Printf("║ Total Saved:    %-6d tokens                      ║\n", savings.TotalSaved)
		fmt.Printf("║ Reduction:      %-5.1f%%                          ║\n", savings.ReductionPct)
	}
	tokens24h, _ := tracker.TokensSaved24h()
	fmt.Printf("║ Saved (24h):    %-6d tokens                      ║\n", tokens24h)
	totalTokens, _ := tracker.TokensSavedTotal()
	fmt.Printf("║ Saved (total):  %-6d tokens                      ║\n", totalTokens)
	fmt.Println("╠════════════════════════════════════════════════════╣")
	topCmds, _ := tracker.TopCommands(5)
	if len(topCmds) > 0 {
		fmt.Println("║ Top Commands:                                    ║")
		for i, c := range topCmds {
			fmt.Printf("║   %d. %-40s ║\n", i+1, trunc(c, 40))
		}
	}
	daily, _ := tracker.GetDailySavings("", 7)
	if len(daily) > 0 {
		fmt.Println("╠════════════════════════════════════════════════════╣")
		fmt.Println("║ Last 7 Days:                                     ║")
		for _, d := range daily {
			bar := strings.Repeat("█", min(d.Saved/100, 40))
			fmt.Printf("║   %s: %-4d %s ║\n", d.Date, d.Saved, bar)
		}
	}
	fmt.Println("╚════════════════════════════════════════════════════╝")
	return nil
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}
