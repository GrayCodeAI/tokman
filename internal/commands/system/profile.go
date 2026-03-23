package system

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show per-layer performance breakdown",
	Long: `Display which compression layers are most effective,
sorted by total tokens saved.`,
	RunE: runProfile,
}

func init() {
	registry.Add(func() { registry.Register(profileCmd) })
}

func runProfile(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	layers, err := tracker.GetTopLayers(20)
	if err != nil {
		// Table might not exist yet
		fmt.Println("No layer statistics available yet.")
		fmt.Println("Run some commands through tokman to collect data.")
		return nil
	}

	if len(layers) == 0 {
		fmt.Println("No layer statistics available yet.")
		fmt.Println("Run some commands through tokman to collect data.")
		return nil
	}

	fmt.Println("Pipeline Layer Performance")
	fmt.Printf("%-25s %10s %10s %8s\n", "Layer", "Total Saved", "Avg Saved", "Calls")
	fmt.Printf("%-25s %10s %10s %8s\n", "─────────────────────────", "──────────", "──────────", "────────")

	for _, l := range layers {
		fmt.Printf("%-25s %10d %10.1f %8d\n",
			l.LayerName, l.TotalSaved, l.AvgSaved, l.CallCount)
	}

	return nil
}
