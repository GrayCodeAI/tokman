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

var npmCmd = &cobra.Command{
	Use:   "npm [args...]",
	Short: "npm run with filtered output",
	Long: `npm run with token-optimized output.

Strips boilerplate and progress bars from npm output.

Examples:
  tokman npm run build
  tokman npm install
  tokman npm test`,
	RunE: runNpm,
}

func init() {
	rootCmd.AddCommand(npmCmd)
}

func runNpm(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	npmArgs := append([]string{}, args...)

	c := exec.Command("npm", npmArgs...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	// Filter npm output
	filtered := filterNpmOutput(output)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npm %s", strings.Join(args, " ")), "tokman npm", originalTokens, filteredTokens)

	if verbose {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	return err
}

func filterNpmOutput(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip progress bars and spinners
		if strings.Contains(trimmed, "\\") || strings.Contains(trimmed, "|") || strings.Contains(trimmed, "/") {
			continue
		}
		// Skip npm WARN without actionable info
		if strings.HasPrefix(trimmed, "npm WARN") && !strings.Contains(trimmed, "deprecated") {
			continue
		}
		// Skip empty lines
		if trimmed == "" {
			continue
		}

		result.WriteString(line + "\n")
	}
	return result.String()
}
