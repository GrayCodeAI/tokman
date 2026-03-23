package analysis

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var topLimit int

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Show top commands by token savings",
	Long:  `Display commands ranked by total tokens saved.`,
	RunE:  runTop,
}

func init() {
	topCmd.Flags().IntVarP(&topLimit, "limit", "n", 10, "number of commands to show")
	registry.Add(func() { registry.Register(topCmd) })
}

func runTop(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	topCommands, err := tracker.TopCommands(topLimit)
	if err != nil {
		return fmt.Errorf("failed to get top commands: %w", err)
	}

	if len(topCommands) == 0 {
		fmt.Println("No command data found.")
		fmt.Println("Run some commands through tokman to collect data.")
		return nil
	}

	fmt.Printf("Top %d Commands\n", topLimit)
	fmt.Println("───────────────")

	for i, cmd := range topCommands {
		fmt.Printf("  %2d. %s\n", i+1, cmd)
	}

	return nil
}
