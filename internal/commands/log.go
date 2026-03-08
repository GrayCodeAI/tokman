package commands

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var logCmd = &cobra.Command{
	Use:   "log [file]",
	Short: "Filter and deduplicate log output",
	Long: `Filter and deduplicate log output.

Reads from stdin or file. Strips timestamps, deduplicates lines.

Examples:
  tokman log app.log
  cat debug.log | tokman log`,
	RunE: runLog,
}

func init() {
	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	var input string
	if len(args) > 0 {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		input = string(data)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		input = strings.Join(lines, "\n")
	}

	// Filter logs
	filtered := filterLogs(input)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(input)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("log", "tokman log", originalTokens, filteredTokens)

	if verbose > 0 {
		origLines := len(strings.Split(input, "\n"))
		filtLines := len(strings.Split(filtered, "\n"))
		fmt.Fprintf(os.Stderr, "Lines: %d -> %d\n", origLines, filtLines)
	}

	return nil
}

var timestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?\s*`)

func filterLogs(input string) string {
	lines := strings.Split(input, "\n")
	seen := make(map[string]bool)
	var result []string

	for _, line := range lines {
		// Strip timestamp
		cleaned := timestampPattern.ReplaceAllString(line, "")
		cleaned = strings.TrimSpace(cleaned)

		if cleaned == "" {
			continue
		}

		// Deduplicate
		if !seen[cleaned] {
			seen[cleaned] = true
			result = append(result, cleaned)
		}
	}

	return strings.Join(result, "\n") + "\n"
}
