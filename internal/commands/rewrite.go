package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Patel230/tokman/internal/discover"
)

var rewriteCmd = &cobra.Command{
	Use:   "rewrite <command>",
	Short: "Rewrite a command to use TokMan wrappers",
	Long: `Check if a command should be rewritten and output the TokMan version.
Used by shell hooks to automatically intercept commands.

Example:
  tokman rewrite "git status"     # Output: tokman git status
  tokman rewrite "ls -la"         # Output: tokman ls
  tokman rewrite "cat file.txt"   # Output: cat file.txt (no rewrite)`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Join all args as the command
		originalCmd := args[0]
		if len(args) > 1 {
			originalCmd = args[0] + " " + args[1]
			// Check if the first two words form a known command
			if !discover.ShouldRewrite(originalCmd) {
				// Just use the first word
				originalCmd = args[0]
			}
		}

		// Get full command string
		fullCmd := originalCmd
		if len(args) > 1 && discover.ShouldRewrite(originalCmd) {
			// Add remaining args
			for i := 2; i < len(args); i++ {
				fullCmd += " " + args[i]
			}
		} else if len(args) > 1 {
			// No rewrite for first word, use all args
			fullCmd = args[0]
			for i := 1; i < len(args); i++ {
				fullCmd += " " + args[i]
			}
		}

		// Rewrite the command
		rewritten := discover.Rewrite(fullCmd)

		// Output the rewritten command (for shell hooks)
		fmt.Println(rewritten)

		// If verbose, show what happened
		if verbose && rewritten != fullCmd {
			cyan := color.New(color.FgCyan).SprintFunc()
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Fprintf(cmd.ErrOrStderr(), "%s → %s\n", cyan(fullCmd), green(rewritten))
		}
	},
}

var rewriteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered command rewrites",
	Run: func(cmd *cobra.Command, args []string) {
		cyan := color.New(color.FgCyan).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		dim := color.New(color.FgHiBlack).SprintFunc()

		fmt.Println(cyan("📋 Registered Command Rewrites"))
		fmt.Println(dim("─────────────────────────────────────"))

		rewrites := discover.ListRewrites()
		for _, mapping := range rewrites {
			fmt.Printf("  %s → %s\n", green(mapping.Original), cyan(mapping.TokManCmd))
		}

		fmt.Println(dim("─────────────────────────────────────"))
		fmt.Printf("  %d commands registered\n", len(rewrites))
	},
}

func init() {
	rootCmd.AddCommand(rewriteCmd)
	rewriteCmd.AddCommand(rewriteListCmd)
}
