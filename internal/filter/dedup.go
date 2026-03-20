package filter

import (
	"fmt"
	"strings"
)

// DeduplicationFilter removes duplicate lines from output.
// R60: Common in logs and test output — same line repeated many times.
type DeduplicationFilter struct {
	// MaxConsecutiveDedup: if N consecutive identical lines, keep 1 + count
	MaxConsecutiveDedup int
}

// NewDeduplicationFilter creates a dedup filter.
func NewDeduplicationFilter() *DeduplicationFilter {
	return &DeduplicationFilter{
		MaxConsecutiveDedup: 3,
	}
}

// Name returns the filter name.
func (f *DeduplicationFilter) Name() string {
	return "dedup"
}

// Apply removes duplicate consecutive lines.
func (f *DeduplicationFilter) Apply(input string, mode Mode) (string, int) {
	lines := strings.Split(input, "\n")
	originalTokens := EstimateTokens(input)

	var result []string
	consecutive := 0
	lastLine := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == lastLine && trimmed != "" {
			consecutive++
			if consecutive == f.MaxConsecutiveDedup {
				result = append(result, "[... repeated ...]")
			} else if consecutive > f.MaxConsecutiveDedup {
				// Skip duplicate
			} else {
				result = append(result, line)
			}
		} else {
			if consecutive > f.MaxConsecutiveDedup {
				result = append(result, fmt.Sprintf("[repeated %d times]", consecutive))
			}
			result = append(result, line)
			consecutive = 0
			lastLine = trimmed
		}
	}

	if consecutive > f.MaxConsecutiveDedup {
		result = append(result, fmt.Sprintf("[repeated %d times]", consecutive))
	}

	output := strings.Join(result, "\n")
	saved := originalTokens - EstimateTokens(output)
	if saved < 0 {
		saved = 0
	}
	return output, saved
}
