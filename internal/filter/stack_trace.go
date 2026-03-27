package filter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// StackTraceFilter deduplicates and compresses stack traces.
// Stack traces are highly redundant in test output and build logs.
// Repeated frames are collapsed and consecutive identical traces are merged.
type StackTraceFilter struct{}

var (
	// Go panic/goroutine header
	goPanicRe     = regexp.MustCompile(`(?m)^(?:panic:|goroutine \d+ \[)`)
	goFrameRe     = regexp.MustCompile(`(?m)^\s+[^\s(]+\.go:\d+`)
	javaFrameRe   = regexp.MustCompile(`(?m)^\s+at [a-zA-Z][\w.$]+\([^)]*\)`)
	pythonFrameRe = regexp.MustCompile(`(?m)^\s+File "[^"]+", line \d+`)
	// Lines that mark end of a trace
	traceEndRe = regexp.MustCompile(`(?m)^(?:FAIL|PASS|ok\s|---\s|=== RUN|=== PAUSE|=== CONT)`)
)

// NewStackTraceFilter creates a new stack trace filter.
func NewStackTraceFilter() *StackTraceFilter {
	return &StackTraceFilter{}
}

// Name returns the filter name.
func (f *StackTraceFilter) Name() string {
	return "stack_trace"
}

// Apply deduplicates stack traces in the input.
func (f *StackTraceFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := f.processTraces(input, mode)
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// processTraces finds and deduplicates stack trace blocks.
func (f *StackTraceFilter) processTraces(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string

	// Track seen traces for deduplication
	seenTraces := make(map[string]int)

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Detect start of a stack trace
		if f.isTraceStart(line) {
			traceLines, end := f.collectTrace(lines, i)
			trace := strings.Join(traceLines, "\n")
			normalized := f.normalizeTrace(trace)

			count := seenTraces[normalized]
			seenTraces[normalized] = count + 1

			if count > 0 {
				// Duplicate trace: emit a summary instead
				result = append(result, "[duplicate stack trace ×"+strconv.Itoa(count+1)+"]")
			} else {
				// First occurrence: compress the trace if aggressive
				if mode == ModeAggressive {
					result = append(result, f.compressTrace(traceLines)...)
				} else {
					result = append(result, f.truncateTrace(traceLines, 15)...)
				}
			}
			i = end
			continue
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// isTraceStart detects the beginning of a stack trace block.
func (f *StackTraceFilter) isTraceStart(line string) bool {
	return goPanicRe.MatchString(line) ||
		strings.HasPrefix(strings.TrimSpace(line), "Traceback (most recent call last)") ||
		strings.HasPrefix(strings.TrimSpace(line), "Exception in thread")
}

// collectTrace gathers all lines belonging to a single stack trace.
func (f *StackTraceFilter) collectTrace(lines []string, start int) ([]string, int) {
	var trace []string
	i := start
	for i < len(lines) {
		line := lines[i]
		// Stop at clear section boundaries
		if i > start && traceEndRe.MatchString(line) && !f.isFrameLine(line) {
			break
		}
		// Stop at two consecutive empty lines after we've seen some frames
		if len(trace) > 3 && line == "" && i+1 < len(lines) && lines[i+1] == "" {
			trace = append(trace, line)
			i++
			break
		}
		trace = append(trace, line)
		i++
	}
	return trace, i
}

// isFrameLine checks if a line is a stack frame.
func (f *StackTraceFilter) isFrameLine(line string) bool {
	return goFrameRe.MatchString(line) || javaFrameRe.MatchString(line) || pythonFrameRe.MatchString(line)
}

// normalizeTrace removes line numbers for dedup comparison.
func (f *StackTraceFilter) normalizeTrace(trace string) string {
	// Replace line numbers to normalize for dedup
	re := regexp.MustCompile(`:\d+`)
	return re.ReplaceAllString(trace, ":N")
}

// truncateTrace limits the trace to maxLines, appending a summary.
func (f *StackTraceFilter) truncateTrace(lines []string, maxLines int) []string {
	if len(lines) <= maxLines {
		return lines
	}
	result := make([]string, 0, maxLines+1)
	result = append(result, lines[:maxLines]...)
	result = append(result, "    ... ["+strconv.Itoa(len(lines)-maxLines)+" more frames]")
	return result
}

// compressTrace keeps header, first 5 frames, and last 3 frames.
func (f *StackTraceFilter) compressTrace(lines []string) []string {
	const keepTop = 5
	const keepBottom = 3
	total := len(lines)

	if total <= keepTop+keepBottom+2 {
		return lines
	}

	result := make([]string, 0, keepTop+keepBottom+2)
	result = append(result, lines[:keepTop]...)
	result = append(result, "    ... ["+strconv.Itoa(total-keepTop-keepBottom)+" frames omitted] ...")
	result = append(result, lines[total-keepBottom:]...)
	return result
}
