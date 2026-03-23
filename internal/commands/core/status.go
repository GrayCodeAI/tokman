package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Quick token savings summary",
	Long: `Display a quick one-line summary of token savings.
For a comprehensive report with graphs and history, use 'tokman gain'.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := shared.GetConfig()
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

		// Show enabled/disabled state
		if isEnabled() {
			fmt.Printf("%s\n", green("TokMan: enabled"))
		} else {
			fmt.Printf("%s\n", yellow("TokMan: disabled (run 'tokman enable')"))
		}

		// Get overall summary
		summary, err := tracker.GetSavings(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting savings: %v\n", err)
			os.Exit(1)
		}

		if summary.TotalCommands == 0 {
			fmt.Println(yellow("No commands recorded yet. Run some commands through TokMan."))
			return
		}

		// Quick summary line
		fmt.Printf("%s\n", strings.Repeat("─", 50))
		fmt.Printf("  Commands: %s  |  Saved: %s tokens (%s)  |  Project: %s\n",
			cyan(fmt.Sprintf("%d", summary.TotalCommands)),
			green(fmt.Sprintf("%d", summary.TotalSaved)),
			green(fmt.Sprintf("%.1f%%", summary.ReductionPct)),
			cyan(shared.ShortenPath(projectPath)),
		)

		// Top 3 commands
		stats, err := tracker.GetCommandStats(projectPath)
		if err == nil && len(stats) > 0 {
			fmt.Printf("  Top: ")
			for i, s := range stats {
				if i >= 3 {
					break
				}
				if i > 0 {
					fmt.Printf(", ")
				}
				cmdName := s.Command
				if len(cmdName) > 15 {
					cmdName = cmdName[:13] + ".."
				}
				fmt.Printf("%s (%d)", cmdName, s.TotalSaved)
			}
			fmt.Println()
		}

		fmt.Printf("%s\n", strings.Repeat("─", 50))
		fmt.Printf("Run %s for detailed report.\n", cyan("tokman gain"))
	},
}

func init() {
	registry.Add(func() { registry.Register(statusCmd) })
}
