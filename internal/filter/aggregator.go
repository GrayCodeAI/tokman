package filter

import (
	"regexp"
	"strconv"
	"strings"
)

// LogAggregator deduplicates and compresses log output.
type LogAggregator struct {
	timestampPatterns []*regexp.Regexp
}

// newLogAggregator creates a new log aggregator.
func newLogAggregator() *LogAggregator {
	return &LogAggregator{
		timestampPatterns: LogTimestampPatterns,
	}
}

// Name returns the filter name.
func (f *LogAggregator) Name() string {
	return "aggregator"
}

// Apply aggregates log output and returns token savings.
func (f *LogAggregator) Apply(input string, mode Mode) (string, int) {
	original := len(input)

	// Check if this looks like log output
	if !f.isLogOutput(input) {
		return input, 0
	}

	// Apply aggregations
	output := f.aggregate(input)

	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// isLogOutput checks if the output looks like log/terminal output.
func (f *LogAggregator) isLogOutput(input string) bool {
	lines := strings.Split(input, "\n")
	if len(lines) < 3 {
		return false
	}

	// Check for log patterns
	logIndicators := 0
	for _, line := range lines {
		for _, pattern := range f.timestampPatterns {
			if pattern.MatchString(line) {
				logIndicators++
				break
			}
		}
	}

	// If more than 20% of lines have timestamps, it's likely log output
	return float64(logIndicators)/float64(len(lines)) > 0.2
}

// aggregate applies all aggregation strategies.
func (f *LogAggregator) aggregate(input string) string {
	output := f.deduplicateLines(input)
	output = f.aggregateTestResults(output)
	output = f.aggregateBuildOutput(output)
	return output
}

// deduplicateLines collapses consecutive duplicate lines.
// Example: 10 lines of "error: connection refused"
//
//	→ "error: connection refused (x10)"
func (f *LogAggregator) deduplicateLines(input string) string {
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return input
	}

	var result []string
	var prev string
	count := 1

	for i, line := range lines {
		// Strip timestamps for comparison
		normalized := f.stripTimestamp(line)

		if normalized == prev && normalized != "" {
			count++
		} else {
			// Output previous line with count
			if prev != "" {
				if count > 1 {
					result = append(result, prev+" (x"+strconv.Itoa(count)+")")
				} else {
					result = append(result, prev)
				}
			}
			prev = line
			count = 1
		}

		// Handle last line
		if i == len(lines)-1 {
			if count > 1 {
				result = append(result, line+" (x"+strconv.Itoa(count)+")")
			} else {
				result = append(result, line)
			}
		}
	}

	return strings.Join(result, "\n")
}

// stripTimestamp removes timestamp prefix from a line.
func (f *LogAggregator) stripTimestamp(line string) string {
	for _, pattern := range f.timestampPatterns {
		line = pattern.ReplaceAllString(line, "")
	}
	return strings.TrimSpace(line)
}

// aggregateTestResults compresses test output into summaries.
// Aggregates multiple test suites into single line.
func (f *LogAggregator) aggregateTestResults(input string) string {
	// Check for test output patterns
	hasTestOutput := false
	for _, pattern := range TestResultPatterns {
		if pattern.MatchString(input) {
			hasTestOutput = true
			break
		}
	}

	if !hasTestOutput {
		return input
	}

	lines := strings.Split(input, "\n")
	var result []string
	var testSuites []testSuite
	currentSuite := testSuite{}
	inTestOutput := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect test suite boundaries
		if strings.Contains(trimmed, "test result:") ||
			strings.Contains(trimmed, "PASS") ||
			strings.Contains(trimmed, "FAIL") {
			inTestOutput = true

			// Parse test results
			if strings.Contains(trimmed, "passed") {
				currentSuite.Passed = f.extractNumber(trimmed, "passed")
			}
			if strings.Contains(trimmed, "failed") {
				currentSuite.Failed = f.extractNumber(trimmed, "failed")
			}
			if strings.Contains(trimmed, "ignored") || strings.Contains(trimmed, "skipped") {
				currentSuite.Ignored = f.extractNumber(trimmed, "ignored")
				currentSuite.Ignored += f.extractNumber(trimmed, "skipped")
			}

			// Check for failure
			if strings.Contains(trimmed, "FAILED") || strings.Contains(trimmed, "FAIL") {
				currentSuite.Failed++
			}

			// Finalize suite
			if strings.Contains(trimmed, "test result:") ||
				(strings.Contains(trimmed, "PASS") && !strings.Contains(trimmed, "passed")) {
				testSuites = append(testSuites, currentSuite)
				currentSuite = testSuite{}
			}
		} else if !inTestOutput {
			result = append(result, line)
		}
	}

	// Generate summary
	if len(testSuites) > 0 {
		totalPassed := 0
		totalFailed := 0
		totalIgnored := 0

		for _, suite := range testSuites {
			totalPassed += suite.Passed
			totalFailed += suite.Failed
			totalIgnored += suite.Ignored
		}

		// If no failures, show compact summary
		if totalFailed == 0 && totalPassed > 0 {
			summary := "✓ " + strconv.Itoa(totalPassed) + " passed"
			if totalIgnored > 0 {
				summary += ", " + strconv.Itoa(totalIgnored) + " ignored"
			}
			summary += " (" + strconv.Itoa(len(testSuites)) + " suites)"
			result = append(result, "", summary)
		} else {
			// Show failures in detail
			summary := "✗ " + strconv.Itoa(totalFailed) + " failed, " +
				strconv.Itoa(totalPassed) + " passed"
			result = append(result, "", summary)
		}
	}

	return strings.Join(result, "\n")
}

type testSuite struct {
	Passed  int
	Failed  int
	Ignored int
}

// aggregateBuildOutput compresses build/compilation output.
func (f *LogAggregator) aggregateBuildOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	warningCount := 0
	errorCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Count warnings and errors
		if strings.Contains(lower, "warning:") {
			warningCount++
			// Keep first few warnings
			if warningCount <= 3 {
				result = append(result, line)
			}
			continue
		}

		if strings.Contains(lower, "error:") {
			errorCount++
			result = append(result, line)
			continue
		}

		result = append(result, line)
	}

	// Add summary if we condensed warnings
	if warningCount > 3 {
		result = append(result, "",
			"... and "+strconv.Itoa(warningCount-3)+" more warnings")
	}

	return strings.Join(result, "\n")
}

// extractNumber extracts a number preceding a keyword.
func (f *LogAggregator) extractNumber(text string, keyword string) int {
	// Pattern: "N keyword"
	re := regexp.MustCompile(`(\d+)\s+` + keyword)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}
	return 0
}

// Aggregate is a utility function to aggregate log output.
func Aggregate(input string) string {
	filter := newLogAggregator()
	output, _ := filter.Apply(input, ModeMinimal)
	return output
}

// groupLines groups lines by common prefix or pattern.
func groupLines(input string) string {
	lines := strings.Split(input, "\n")

	// Group by first word/pattern
	groups := make(map[string][]string)
	var order []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Extract key (first word or pattern)
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		key := fields[0]
		if len(key) > 20 {
			key = key[:20]
		}

		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], line)
	}

	// Reconstruct with groups
	var result []string
	for _, key := range order {
		lines := groups[key]
		if len(lines) > 1 {
			result = append(result, "["+key+"] ("+strconv.Itoa(len(lines))+" lines)")
		} else {
			result = append(result, lines[0])
		}
	}

	return strings.Join(result, "\n")
}
