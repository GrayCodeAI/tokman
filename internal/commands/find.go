package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var findCmd = &cobra.Command{
	Use:   "find [args...]",
	Short: "Find files with compact tree output",
	Long: `Find files with token-optimized output.

Accepts native find flags like -name, -type.
Filters and formats output for minimal token usage.

Examples:
  tokman find . -name "*.go"
  tokman find . -type f -mtime -1`,
	RunE: runFind,
}

func init() {
	rootCmd.AddCommand(findCmd)
}

func runFind(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	findArgs := append([]string{}, args...)
	if len(findArgs) == 0 {
		findArgs = []string{"."}
	}

	c := exec.Command("find", findArgs...)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String()
	if output == "" && stderr.Len() > 0 {
		output = stderr.String()
	}

	// Apply filtering - compact output
	engine := filter.NewEngine(filter.ModeMinimal)
	filtered, tokensSaved := engine.Process(output)

	// Further compact: one file per line, strip common prefix
	filtered = compactFindOutput(filtered)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("find %s", strings.Join(args, " ")), "tokman find", originalTokens, filteredTokens)

	if verbose > 0 && tokensSaved > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", tokensSaved)
	}

	return err
}

func compactFindOutput(output string) string {
	lines := strings.Split(output, "\n")
	// Just strip empty lines and return
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n") + "\n"
}
