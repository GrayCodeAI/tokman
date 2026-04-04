package sessioncmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show command history",
	Long:  `Display recent commands with their token savings.`,
	RunE:  runHistory,
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "number of entries to show")
	registry.Add(func() { registry.Register(historyCmd) })
}

func runHistory(cmd *cobra.Command, args []string) error {
	tracker, err := shared.OpenTracker()
	if err != nil {
		return fmt.Errorf("tracking not available: %w", err)
	}
	defer tracker.Close()

	records, err := tracker.GetRecentCommands(shared.GetProjectPath(), historyLimit)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("No command history found.")
		return nil
	}

	fmt.Printf("Command History (last %d)\n", historyLimit)
	fmt.Printf("%-20s %-35s %8s %8s %6s\n", "Time", "Command", "Original", "Saved", "Parse")
	fmt.Printf("%-20s %-35s %8s %8s %6s\n", "────────────────────", "───────────────────────────────────", "────────", "────────", "──────")

	for _, r := range records {
		cmdName := r.Command
		if len(cmdName) > 35 {
			cmdName = cmdName[:32] + "..."
		}
		parseOk := "✓"
		if !r.ParseSuccess {
			parseOk = "✗"
		}
		fmt.Printf("%-20s %-35s %8d %8d %6s\n",
			r.Timestamp.Format("01-02 15:04:05"),
			cmdName,
			r.OriginalTokens,
			r.SavedTokens,
			parseOk,
		)
	}

	return nil
}
