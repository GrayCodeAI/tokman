package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show token savings summary",
	Long:  `Display a summary of token savings for the current project.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := GetConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dbPath := cfg.GetDatabasePath()
		tracker, err := tracking.NewTracker(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
			os.Exit(1)
		}
		defer tracker.Close()

		projectPath := config.ProjectPath()

		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()

		// Get overall summary
		summary, err := tracker.GetSavings(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting savings: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n%s\n", green("📊 TokMan Status"))
		fmt.Println(strings.Repeat("─", 50))
		fmt.Printf("Project: %s\n", cyan(projectPath))
		fmt.Println(strings.Repeat("─", 50))

		if summary.TotalCommands == 0 {
			fmt.Println(yellow("No commands recorded yet."))
			fmt.Println("\nRun some commands through TokMan to see savings.")
			return
		}

		// Display summary
		fmt.Printf("\n%s\n", green("Overall Summary"))
		fmt.Printf("  Commands executed:  %s\n", cyan(fmt.Sprintf("%d", summary.TotalCommands)))
		fmt.Printf("  Original tokens:    %s\n", cyan(fmt.Sprintf("%d", summary.TotalOriginal)))
		fmt.Printf("  Filtered tokens:    %s\n", cyan(fmt.Sprintf("%d", summary.TotalFiltered)))
		fmt.Printf("  Tokens saved:       %s\n", green(fmt.Sprintf("%d", summary.TotalSaved)))
		fmt.Printf("  Reduction:          %s\n", green(fmt.Sprintf("%.1f%%", summary.ReductionPct)))

		// Get per-command stats
		stats, err := tracker.GetCommandStats(projectPath)
		if err == nil && len(stats) > 0 {
			fmt.Printf("\n%s\n", green("Top Commands by Savings"))
			fmt.Println(strings.Repeat("─", 50))
			for i, s := range stats {
				if i >= 5 {
					break // Show top 5
				}
				fmt.Printf("  %-20s %s saved (%.1f%% reduction)\n",
					s.Command,
					green(fmt.Sprintf("%d", s.TotalSaved)),
					s.ReductionPct,
				)
			}
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
