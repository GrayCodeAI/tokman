package filter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// CSVCompressFilter compresses CSV and table data by:
//  1. Detecting CSV/TSV blocks (3+ lines with consistent delimiters)
//  2. Summarizing large tables: show header + first N rows + "... M more rows"
//  3. Removing duplicate rows
//  4. Collapsing whitespace-padded table cells
type CSVCompressFilter struct{}

var (
	// Detects lines that look like CSV (3+ comma-separated values)
	csvLineRe = regexp.MustCompile(`^[^,\n]{0,100},[^,\n]{0,100},[^,\n]{0,100}`)
	// Detects pipe-formatted markdown tables
	mdTableRe = regexp.MustCompile(`^\s*\|`)
	// Detects TSV lines
	tsvLineRe = regexp.MustCompile(`^[^\t\n]+\t[^\t\n]+\t[^\t\n]+`)
)

// NewCSVCompressFilter creates a new CSV/table compression filter.
func NewCSVCompressFilter() *CSVCompressFilter {
	return &CSVCompressFilter{}
}

// Name returns the filter name.
func (f *CSVCompressFilter) Name() string {
	return "csv_compress"
}

// Apply compresses CSV and table data.
func (f *CSVCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := f.process(input, mode)
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

func (f *CSVCompressFilter) process(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string

	maxRows := 10
	if mode == ModeAggressive {
		maxRows = 5
	}

	i := 0
	for i < len(lines) {
		line := lines[i]
		delim, isData := f.detectTableStart(line)

		if isData {
			// Collect the table block
			blockStart := i
			for i < len(lines) && f.isTableLine(lines[i], delim) {
				i++
			}
			block := lines[blockStart:i]
			result = append(result, f.compressTable(block, delim, maxRows)...)
		} else {
			result = append(result, line)
			i++
		}
	}

	return strings.Join(result, "\n")
}

// detectTableStart checks if a line is the start of a table and returns the delimiter.
func (f *CSVCompressFilter) detectTableStart(line string) (string, bool) {
	if mdTableRe.MatchString(line) {
		return "|", true
	}
	if tsvLineRe.MatchString(line) {
		return "\t", true
	}
	if csvLineRe.MatchString(line) {
		return ",", true
	}
	return "", false
}

// isTableLine checks if a line belongs to the current table.
func (f *CSVCompressFilter) isTableLine(line, delim string) bool {
	if line == "" {
		return false
	}
	if delim == "|" {
		return mdTableRe.MatchString(line) || strings.TrimSpace(line) == ""
	}
	return strings.Contains(line, delim)
}

// compressTable reduces a table block to header + sample rows.
func (f *CSVCompressFilter) compressTable(lines []string, delim string, maxRows int) []string {
	if len(lines) <= maxRows+2 {
		return lines
	}

	// Deduplicate rows
	seen := make(map[string]bool)
	var unique []string
	for _, line := range lines {
		key := strings.TrimSpace(line)
		if key == "" || !seen[key] {
			seen[key] = true
			unique = append(unique, line)
		}
	}

	if len(unique) <= maxRows+2 {
		return unique
	}

	// Keep header (first 1-2 lines) + maxRows data rows + summary
	headerLines := 1
	if len(unique) > 1 && isTableSeparator(unique[1], delim) {
		headerLines = 2
	}

	result := make([]string, 0, maxRows+headerLines+1)
	result = append(result, unique[:headerLines]...)
	result = append(result, unique[headerLines:headerLines+maxRows]...)
	omitted := len(unique) - headerLines - maxRows
	if omitted > 0 {
		result = append(result, "... ["+strconv.Itoa(omitted)+" more rows] ...")
	}
	return result
}

// isTableSeparator detects markdown table separator lines like |---|---|.
func isTableSeparator(line, delim string) bool {
	if delim != "|" {
		return false
	}
	trimmed := strings.TrimSpace(line)
	return strings.ContainsAny(trimmed, "-") && strings.Contains(trimmed, "|")
}
