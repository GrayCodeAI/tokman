package commands

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var wcCmd = &cobra.Command{
	Use:   "wc [args...]",
	Short: "Word/line/byte count with compact output",
	Long: `Word/line/byte count with token-optimized output.

Strips paths and padding for minimal token usage.

Examples:
  tokman wc -l file.go
  tokman wc -w *.go`,
	RunE: runWc,
}

func init() {
	rootCmd.AddCommand(wcCmd)
}

func runWc(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	c := exec.Command("wc", args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String()
	if output == "" && stderr.Len() > 0 {
		output = stderr.String()
	}

	// Compact output: strip padding
	filtered := compactWcOutput(output)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("wc %s", strings.Join(args, " ")), "tokman wc", originalTokens, filteredTokens)

	return err
}

func compactWcOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		// Collapse multiple spaces
		fields := strings.Fields(line)
		if len(fields) > 0 {
			result = append(result, strings.Join(fields, " "))
		}
	}
	return strings.Join(result, "\n") + "\n"
}
