package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// TestOutputFilter compresses test runner output by:
//  1. Collapsing passing test lines (PASS/ok lines) into a summary count
//  2. Preserving all FAIL lines and their context
//  3. Removing verbose timing lines for passing tests in aggressive mode
//  4. Deduplicating identical error messages across test cases
type TestOutputFilter struct{}

var (
	testPassLineRe   = regexp.MustCompile(`^(?:--- PASS:|ok\s+\S+\s+\d)`)
	testFailLineRe   = regexp.MustCompile(`^(?:--- FAIL:|FAIL\s+\S+|FAIL$)`)
	testRunLineRe    = regexp.MustCompile(`^=== RUN\s+`)
	testPauseLineRe  = regexp.MustCompile(`^=== (?:PAUSE|CONT)\s+`)
	goTestTimingRe   = regexp.MustCompile(`^\s+--- (?:PASS|SKIP):\s+\S+\s+\(\d+\.\d+s\)`)
	testSummaryLineRe = regexp.MustCompile(`^(?:ok|FAIL)\s+\S+\s+\d`)
)

// NewTestOutputFilter creates a test output filter.
func NewTestOutputFilter() *TestOutputFilter {
	return &TestOutputFilter{}
}

// Name returns the filter name.
func (f *TestOutputFilter) Name() string {
	return "test_output"
}

// Apply compresses test output while preserving all failures.
func (f *TestOutputFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Only apply to test output
	if !strings.Contains(input, "--- PASS") && !strings.Contains(input, "--- FAIL") &&
		!strings.Contains(input, "=== RUN") && !strings.Contains(input, "ok  \t") {
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

func (f *TestOutputFilter) process(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string

	passCount := 0
	skipCount := 0
	seenErrors := make(map[string]bool)

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Always keep FAIL lines and summary
		if testFailLineRe.MatchString(line) || testSummaryLineRe.MatchString(line) {
			// Flush pass count before failure
			if passCount > 0 {
				result = append(result, formatPassSummary(passCount, skipCount))
				passCount = 0
				skipCount = 0
			}
			result = append(result, line)
			i++
			continue
		}

		// Collapse passing test lines
		if testPassLineRe.MatchString(line) {
			if strings.Contains(line, "--- SKIP:") {
				skipCount++
			} else {
				passCount++
			}
			i++
			continue
		}

		// In aggressive mode: skip RUN/PAUSE/CONT lines and timing lines
		if mode == ModeAggressive {
			if testRunLineRe.MatchString(line) || testPauseLineRe.MatchString(line) ||
				goTestTimingRe.MatchString(line) {
				i++
				continue
			}
		}

		// Deduplicate error lines
		if seenErrors[line] && strings.Contains(strings.ToLower(line), "error") {
			i++
			continue
		}
		if strings.Contains(strings.ToLower(line), "error") {
			seenErrors[line] = true
		}

		// Flush pass count before non-test line
		if passCount > 0 && line == "" {
			result = append(result, formatPassSummary(passCount, skipCount))
			passCount = 0
			skipCount = 0
		}

		result = append(result, line)
		i++
	}

	if passCount > 0 {
		result = append(result, formatPassSummary(passCount, skipCount))
	}

	return strings.Join(result, "\n")
}

func formatPassSummary(pass, skip int) string {
	if skip > 0 {
		return "--- [" + numStr(pass) + " tests passed, " + numStr(skip) + " skipped]"
	}
	return "--- [" + numStr(pass) + " tests passed]"
}

func numStr(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
