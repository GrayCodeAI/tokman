package analysis

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/economics"
)

var (
	econDaily   bool
	econWeekly  bool
	econMonthly bool
	econAll     bool
	econFormat  string
)

var economicsCmd = &cobra.Command{
	Use:   "economics",
	Short: "Show spending vs savings analysis",
	Long: `Analyze Claude Code spending vs TokMan token savings.

Combines ccusage (Claude Code API usage) with TokMan tracking data
to show real-world cost savings from using TokMan.

Savings are calculated using Anthropic API price ratios:
- Output tokens: 5x input token cost
- Cache writes: 1.25x input token cost  
- Cache reads: 0.1x input token cost

Requires ccusage installed (npm i -g ccusage).`,
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")

		opts := economics.RunOptions{
			Daily:   econDaily,
			Weekly:  econWeekly,
			Monthly: econMonthly,
			All:     econAll,
			Format:  econFormat,
			Verbose: verbose,
		}

		if err := economics.Run(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	registry.Add(func() { registry.Register(economicsCmd) })

	economicsCmd.Flags().BoolVar(&econDaily, "daily", false, "Show daily breakdown")
	economicsCmd.Flags().BoolVar(&econWeekly, "weekly", false, "Show weekly breakdown")
	economicsCmd.Flags().BoolVar(&econMonthly, "monthly", false, "Show monthly breakdown")
	economicsCmd.Flags().BoolVar(&econAll, "all", false, "Show all time periods")
	economicsCmd.Flags().StringVar(&econFormat, "format", "text", "Output format: text, json, csv")
}
