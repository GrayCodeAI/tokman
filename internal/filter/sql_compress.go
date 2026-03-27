package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// SQLCompressFilter compresses SQL query output and query plans.
// Targets: verbose EXPLAIN output, repeated SELECT boilerplate,
// long IN(...) lists, and redundant whitespace.
type SQLCompressFilter struct{}

// NewSQLCompressFilter creates a new SQL compress filter.
func NewSQLCompressFilter() *SQLCompressFilter { return &SQLCompressFilter{} }

// Name returns the filter name.
func (f *SQLCompressFilter) Name() string { return "sql_compress" }

var (
	// sqlExplainRowRe matches EXPLAIN ANALYZE rows with timing noise
	sqlExplainRowRe = regexp.MustCompile(`(?i)(actual time=[\d.]+\.\.[\d.]+ rows=\d+ loops=\d+)`)
	// sqlInListRe matches long IN(...) lists
	sqlInListRe = regexp.MustCompile(`(?i)\bIN\s*\(([^)]{60,})\)`)
	// sqlSelectStarRe collapses verbose column lists in SELECT
	sqlWhitespacRe = regexp.MustCompile(`[ \t]{2,}`)
	// sqlBlankLineRe collapses multiple blank lines
	sqlBlankLineRe = regexp.MustCompile(`\n{3,}`)
	// sqlPlanningRe strips verbose planner cost annotations
	sqlPlanningRe = regexp.MustCompile(`(?i)\(cost=[\d.]+\.\.[\d.]+ rows=\d+ width=\d+\)`)
)

// Apply compresses SQL output and query plans.
func (f *SQLCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input

	// Normalize excessive whitespace within lines
	output = sqlWhitespacRe.ReplaceAllString(output, " ")

	// Collapse 3+ blank lines to 2
	output = sqlBlankLineRe.ReplaceAllString(output, "\n\n")

	// Compress long IN(...) lists
	output = sqlInListRe.ReplaceAllStringFunc(output, func(match string) string {
		sub := sqlInListRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		items := strings.Split(sub[1], ",")
		if len(items) <= 5 {
			return match
		}
		return "IN(" + strings.TrimSpace(items[0]) + ", " +
			strings.TrimSpace(items[1]) + ", ... [" +
			itoa(len(items)-2) + " more])"
	})

	// Strip planner cost annotations in aggressive mode
	if mode == ModeAggressive {
		output = sqlPlanningRe.ReplaceAllString(output, "")
		output = sqlExplainRowRe.ReplaceAllStringFunc(output, func(match string) string {
			// Keep only the row count
			rowRe := regexp.MustCompile(`rows=(\d+)`)
			if m := rowRe.FindStringSubmatch(match); len(m) == 2 {
				return "rows=" + m[1]
			}
			return match
		})
	}

	// Clean up trailing spaces left after substitutions
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	output = strings.Join(lines, "\n")

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}
