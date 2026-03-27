package filter

import (
	"regexp"
	"strings"
)

// StackTracePreserveFilter preserves stack traces as atomic units during compression.
// R61: Stack traces should never be split — they're atomic error information.
type StackTracePreserveFilter struct {
	stackPatterns []*regexp.Regexp
}

// newStackTracePreserveFilter creates a stack trace preservation filter.
func newStackTracePreserveFilter() *StackTracePreserveFilter {
	return &StackTracePreserveFilter{
		stackPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^\s+at .+\(.+:\d+:\d+\)`),             // JavaScript
			regexp.MustCompile(`^\s+File ".+", line \d+`),             // Python
			regexp.MustCompile(`^\s+.+\.go:\d+`),                      // Go
			regexp.MustCompile(`^\s+at .+\.rs:\d+`),                   // Rust
			regexp.MustCompile(`^\s+#\d+\s+0x[0-9a-f]+`),              // Native
			regexp.MustCompile(`^\s+Exception in thread`),             // Java
			regexp.MustCompile(`Traceback \(most recent call last\)`), // Python header
		},
	}
}

// Name returns the filter name.
func (f *StackTracePreserveFilter) Name() string {
	return "stack_trace_preserve"
}

// Apply preserves stack traces as atomic blocks.
func (f *StackTracePreserveFilter) Apply(input string, mode Mode) (string, int) {
	lines := strings.Split(input, "\n")
	originalTokens := EstimateTokens(input)

	var result []string
	inStack := false
	stackBlock := []string{}

	for _, line := range lines {
		isStackLine := f.isStackLine(line)

		if isStackLine {
			inStack = true
			stackBlock = append(stackBlock, line)
		} else {
			if inStack {
				// End of stack trace — preserve entire block
				result = append(result, stackBlock...)
				result = append(result, "") // Separator
				stackBlock = []string{}
				inStack = false
			}
			result = append(result, line)
		}
	}

	// Handle trailing stack trace
	if len(stackBlock) > 0 {
		result = append(result, stackBlock...)
	}

	output := strings.Join(result, "\n")
	saved := originalTokens - EstimateTokens(output)
	if saved < 0 {
		saved = 0
	}
	return output, saved
}

// isStackLine checks if a line is part of a stack trace.
func (f *StackTracePreserveFilter) isStackLine(line string) bool {
	for _, pattern := range f.stackPatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

// isStackTraceBlock checks if an entire text block is a stack trace.
func isStackTraceBlock(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) < 3 {
		return false
	}

	f := newStackTracePreserveFilter()
	stackLines := 0
	for _, line := range lines {
		if f.isStackLine(line) {
			stackLines++
		}
	}

	return float64(stackLines) >= float64(len(lines))*0.5
}
