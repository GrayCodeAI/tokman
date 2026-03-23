package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	grepMaxLen   int
	grepMax      int
	grepFileType string
)

var grepCmd = &cobra.Command{
	Use:   "grep [args...]",
	Short: "Compact grep - strips whitespace, truncates, groups by file",
	Long: `Compact grep with token-optimized output.

Strips whitespace, truncates long lines, and groups results by file.
Passes native grep/ripgrep flags through.

Examples:
  tokman grep -r "TODO" .
  tokman grep "func " . -t go
  tokman grep -r "error" . --max-len 60 --max 20`,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE:               runGrep,
}

func init() {
	registry.Add(func() { registry.Register(grepCmd) })
	grepCmd.Flags().IntVarP(&grepMaxLen, "max-len", "l", 80, "Max line length")
	grepCmd.Flags().IntVarP(&grepMax, "max", "m", 50, "Max results to show")
	grepCmd.Flags().StringVarP(&grepFileType, "type", "t", "", "Filter by file type (go, py, js, rust)")
}

func runGrep(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Use standard grep with all args passed through
	grepArgs := append([]string{}, args...)

	// Add --color=never to avoid ANSI codes
	grepArgs = append([]string{"--color=never"}, grepArgs...)

	c := exec.Command("grep", grepArgs...)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String()

	// Grep returns exit code 1 when no matches - that's not an error for us
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// No matches found - not an error
			if output == "" {
				fmt.Println("(no matches)")
				return nil
			}
		} else {
			// Real error
			return fmt.Errorf("grep failed: %w\n%s", err, stderr.String())
		}
	}

	// Compact output for minimal tokens
	filtered := compactGrepOutputSimple(output, grepMaxLen, grepMax)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("grep %s", strings.Join(args, " ")), "tokman grep", originalTokens, filteredTokens)

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	return nil
}

func compactGrepOutputSimple(output string, maxLen, maxResults int) string {
	var result strings.Builder
	count := 0

	for _, line := range strings.Split(output, "\n") {
		if count >= maxResults {
			result.WriteString(fmt.Sprintf("... (%d more)\n", count-maxResults+1))
			break
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Truncate long lines
		if len(line) > maxLen {
			line = line[:maxLen] + "..."
		}
		result.WriteString(line + "\n")
		count++
	}

	return result.String()
}

func compactGrepOutput(output string, maxLen, maxResults int) string {
	var result strings.Builder
	count := 0

	for _, line := range strings.Split(output, "\n") {
		if count >= maxResults {
			result.WriteString(fmt.Sprintf("// ... truncated (showing %d of more results)\n", maxResults))
			break
		}
		if strings.HasPrefix(line, "{\"type\":\"match\"") {
			// Parse match JSON and format compactly
			result.WriteString(compactGrepMatch(line, maxLen))
			count++
		}
	}

	return result.String()
}

func compactGrepMatch(jsonLine string, maxLen int) string {
	// Simplified: just extract file and line number
	// In production, parse JSON properly
	if strings.Contains(jsonLine, "\"path\":") {
		return "// match found\n"
	}
	return ""
}
