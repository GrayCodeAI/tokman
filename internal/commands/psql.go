package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var psqlCmd = &cobra.Command{
	Use:   "psql [args...]",
	Short: "PostgreSQL client with filtered output",
	Long: `PostgreSQL client with token-optimized output.

Automatically detects and compresses:
  - Table format: Strips borders, outputs tab-separated
  - Expanded format (\x): Converts to one-liner key=val

Examples:
  tokman psql -c "SELECT * FROM users"
  tokman psql -d mydb -c "SELECT * FROM orders LIMIT 100"`,
	RunE: runPsql,
}

func init() {
	rootCmd.AddCommand(psqlCmd)
}

var (
	expandedRecord = regexp.MustCompile(`-\[ RECORD \d+ \]-`)
	separator      = regexp.MustCompile(`^[-+]+$`)
	rowCount       = regexp.MustCompile(`^\(\d+ rows?\)$`)
	recordHeader   = regexp.MustCompile(`^-\[ RECORD (\d+) \]-`)
)

func runPsql(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	c := exec.Command("psql", args...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String()

	if stderr.Len() > 0 {
		fmt.Fprint(os.Stderr, stderr.String())
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}

	filtered := filterPsqlOutput(output)
	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("psql %s", strings.Join(args, " ")), "tokman psql", originalTokens, filteredTokens)

	return nil
}

func filterPsqlOutput(output string) string {
	if strings.TrimSpace(output) == "" {
		return ""
	}

	if isExpandedFormat(output) {
		return filterExpanded(output)
	} else if isTableFormat(output) {
		return filterTable(output)
	}

	// Passthrough: COPY results, notices, etc.
	return output
}

func isTableFormat(output string) bool {
	return strings.Contains(output, "-+-") || strings.Contains(output, "---+---")
}

func isExpandedFormat(output string) bool {
	return expandedRecord.MatchString(output)
}

func filterTable(output string) string {
	var result []string
	dataRows := 0
	totalRows := 0

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip separator lines
		if separator.MatchString(trimmed) {
			continue
		}

		// Skip row count footer
		if rowCount.MatchString(trimmed) {
			continue
		}

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// This is a data or header row with | delimiters
		if strings.Contains(trimmed, "|") {
			totalRows++
			// First row is header, don't count it as data
			if totalRows > 1 {
				dataRows++
			}

			if dataRows <= 30 || totalRows == 1 {
				cols := []string{}
				for _, col := range strings.Split(trimmed, "|") {
					cols = append(cols, strings.TrimSpace(col))
				}
				result = append(result, strings.Join(cols, "\t"))
			}
		} else {
			// Non-table line (e.g., command output like SET, NOTICE)
			result = append(result, trimmed)
		}
	}

	if dataRows > 30 {
		result = append(result, fmt.Sprintf("... +%d more rows", dataRows-30))
	}

	return strings.Join(result, "\n")
}

func filterExpanded(output string) string {
	var result []string
	var currentPairs []string
	var currentRecord string
	recordCount := 0

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		if rowCount.MatchString(trimmed) {
			continue
		}

		if matches := recordHeader.FindStringSubmatch(trimmed); len(matches) > 1 {
			// Flush previous record
			if currentRecord != "" {
				if recordCount <= 20 {
					result = append(result, fmt.Sprintf("[%s] %s", currentRecord, strings.Join(currentPairs, " ")))
				}
				currentPairs = nil
			}
			recordCount++
			currentRecord = matches[1]
		} else if strings.Contains(trimmed, "|") && currentRecord != "" {
			// key | value line
			parts := strings.SplitN(trimmed, "|", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				currentPairs = append(currentPairs, fmt.Sprintf("%s=%s", key, val))
			}
		} else if trimmed == "" {
			continue
		} else if currentRecord == "" {
			// Non-record line before any record (notices, etc.)
			result = append(result, trimmed)
		}
	}

	// Flush last record
	if currentRecord != "" {
		if recordCount <= 20 {
			result = append(result, fmt.Sprintf("[%s] %s", currentRecord, strings.Join(currentPairs, " ")))
		}
	}

	if recordCount > 20 {
		result = append(result, fmt.Sprintf("... +%d more records", recordCount-20))
	}

	return strings.Join(result, "\n")
}
