package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// SmartLogFilter compresses structured log output intelligently.
// Strategies:
//  1. Deduplicate identical log lines (keeping count annotation)
//  2. Collapse verbose health-check / ping entries
//  3. Strip low-signal DEBUG/TRACE lines in aggressive mode
//  4. Normalize timestamp formats to save tokens
//  5. Collapse repeating stack frame sequences
type SmartLogFilter struct{}

// NewSmartLogFilter creates a new smart log filter.
func NewSmartLogFilter() *SmartLogFilter { return &SmartLogFilter{} }

// Name returns the filter name.
func (f *SmartLogFilter) Name() string { return "smart_log" }

var (
	// logLevelRe detects log level prefix
	logLevelRe = regexp.MustCompile(`(?i)^\s*(?:\[?(?:TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL|NOTICE)\]?\s*[-:])`)
	// logDebugRe matches DEBUG/TRACE lines
	logDebugRe = regexp.MustCompile(`(?i)^\s*\[?(?:TRACE|DEBUG)\]?`)
	// logHealthRe matches health/ping/readiness log lines
	logHealthRe = regexp.MustCompile(`(?i)(?:GET|POST)\s+/(?:health|ping|ready|alive|liveness|readiness)`)
	// logTimestampFullRe normalizes full ISO timestamps
	logTimestampFullRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`)
	// logStackFrameRe matches stack frame lines
	logStackFrameRe = regexp.MustCompile(`(?m)^\s+at\s+[\w.$<>[\]]+\([\w.:/]+:\d+\)$`)
)

// Apply compresses log output.
func (f *SmartLogFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	lines := strings.Split(input, "\n")

	// Pass 1: deduplicate identical lines
	type lineGroup struct {
		line  string
		count int
	}
	var groups []lineGroup
	for _, line := range lines {
		if len(groups) > 0 && groups[len(groups)-1].line == line {
			groups[len(groups)-1].count++
		} else {
			groups = append(groups, lineGroup{line: line, count: 1})
		}
	}

	// Pass 2: filter and annotate
	var result []string
	healthCount := 0

	for _, g := range groups {
		line := g.line
		trimmed := strings.TrimSpace(line)

		// Skip empty
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// In aggressive mode: drop DEBUG/TRACE lines
		if mode == ModeAggressive && logDebugRe.MatchString(trimmed) {
			continue
		}

		// Collapse health-check lines
		if logHealthRe.MatchString(line) {
			healthCount++
			continue // accumulated, emitted at end
		}

		// Normalize timestamps: keep only HH:MM:SS
		if mode == ModeAggressive {
			line = logTimestampFullRe.ReplaceAllStringFunc(line, func(ts string) string {
				// Extract time portion HH:MM:SS
				if len(ts) >= 19 {
					sep := 10
					if ts[10] == 'T' || ts[10] == ' ' {
						return ts[11:19]
					}
					_ = sep
				}
				return ts
			})
		}

		// Annotate deduplicated lines
		if g.count > 1 {
			line = line + " [×" + itoa(g.count) + "]"
		}
		result = append(result, line)
	}

	// Emit health-check summary
	if healthCount > 0 {
		result = append(result, "["+itoa(healthCount)+" health-check/ping requests omitted]")
	}

	// Collapse repeated stack frames in aggressive mode
	output := strings.Join(result, "\n")
	if mode == ModeAggressive {
		output = collapseStackFrames(output)
	}

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// collapseStackFrames replaces long identical stack frame runs with a summary.
func collapseStackFrames(input string) string {
	lines := strings.Split(input, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		if logStackFrameRe.MatchString(lines[i]) {
			// Count consecutive stack frames
			j := i
			for j < len(lines) && logStackFrameRe.MatchString(lines[j]) {
				j++
			}
			frameCount := j - i
			if frameCount > 4 {
				out = append(out, lines[i], lines[i+1])
				out = append(out, "\t... ["+itoa(frameCount-2)+" more frames] ...")
				out = append(out, lines[j-1])
			} else {
				out = append(out, lines[i:j]...)
			}
			i = j
		} else {
			out = append(out, lines[i])
			i++
		}
	}
	return strings.Join(out, "\n")
}
