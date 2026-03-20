package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var diffCmd = &cobra.Command{
	Use:   "diff <file1> [file2]",
	Short: "Ultra-condensed diff (only changed lines)",
	Long: `Show diff with only changed lines for token efficiency.

If file2 is omitted, reads from stdin.
Strips context lines, shows only additions/deletions.

Examples:
  tokman diff file1.go file2.go
  git diff | tokman diff -`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	var output string
	var err error

	if len(args) == 1 || args[0] == "-" {
		// Read from stdin
		var buf bytes.Buffer
		_, err = buf.ReadFrom(cmd.InOrStdin())
		output = buf.String()
	} else {
		// Run diff command
		c := exec.Command("diff", args...)
		var stdout, stderr bytes.Buffer
		c.Stdout = &stdout
		c.Stderr = &stderr
		err = c.Run()
		output = stdout.String()
		if output == "" && stderr.Len() > 0 {
			output = stderr.String()
		}
	}

	// Compact diff: only show changed lines
	filtered := compactDiff(output)

	fmt.Print(filtered)

	originalTokens := core.EstimateTokens(output)
	filteredTokens := core.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("diff %s", strings.Join(args, " ")), "tokman diff", originalTokens, filteredTokens)

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	return err
}

func compactDiff(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		// Keep only lines starting with + or - (but not +++
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "+") && !strings.HasPrefix(trimmed, "+++") {
			result.WriteString(line + "\n")
		} else if strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "---") {
			result.WriteString(line + "\n")
		} else if strings.HasPrefix(trimmed, "@@") {
			result.WriteString(line + "\n")
		}
	}
	return result.String()
}
