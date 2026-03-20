package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var sessionsDays int

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Show session history",
	Long:  `Display recent command sessions grouped by time.`,
	RunE:  runSessions,
}

func init() {
	sessionsCmd.Flags().IntVarP(&sessionsDays, "days", "d", 7, "days of history")
	rootCmd.AddCommand(sessionsCmd)
}

func runSessions(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	cwd, _ := os.Getwd()
	records, err := tracker.GetRecentCommands(cwd, 100)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("Recent Sessions")
	fmt.Printf("%-20s %-30s %8s %8s\n", "Time", "Command", "Original", "Saved")
	fmt.Printf("%-20s %-30s %8s %8s\n", "────────────────────", "──────────────────────────────", "────────", "────────")

	cutoff := time.Now().AddDate(0, 0, -sessionsDays)
	for _, r := range records {
		if r.Timestamp.Before(cutoff) {
			continue
		}
		cmdName := r.Command
		if len(cmdName) > 30 {
			cmdName = cmdName[:27] + "..."
		}
		fmt.Printf("%-20s %-30s %8d %8d\n",
			r.Timestamp.Format("01-02 15:04:05"),
			cmdName,
			r.OriginalTokens,
			r.SavedTokens,
		)
	}

	return nil
}
