package filter

import (
	"strings"
	"unicode/utf8"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// LineTruncateFilter truncates lines that exceed a token-aware length limit.
// Long lines (e.g., minified JS, base64 blobs, single-line JSON) waste tokens
// without adding semantic value when a summary suffix is provided.
type LineTruncateFilter struct {
	// MaxLineTokens is the maximum tokens per line before truncation.
	// Defaults to 60 for ModeMinimal, 30 for ModeAggressive.
	MaxLineTokens int
}

// NewLineTruncateFilter creates a line truncate filter with defaults.
func NewLineTruncateFilter() *LineTruncateFilter {
	return &LineTruncateFilter{}
}

// Name returns the filter name.
func (f *LineTruncateFilter) Name() string {
	return "line_truncate"
}

// Apply truncates overly long lines.
func (f *LineTruncateFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	maxTokens := 60
	if f.MaxLineTokens > 0 {
		maxTokens = f.MaxLineTokens
	} else if mode == ModeAggressive {
		maxTokens = 30
	}

	// Estimate max chars: ~4 chars per token on average
	maxChars := maxTokens * 4

	original := core.EstimateTokens(input)
	lines := strings.Split(input, "\n")
	changed := false

	for i, line := range lines {
		if utf8.RuneCountInString(line) > maxChars {
			// Truncate at the char boundary
			runes := []rune(line)
			truncated := string(runes[:maxChars])
			// Find a clean word boundary near the end
			if idx := strings.LastIndexAny(truncated, " \t,;"); idx > maxChars*3/4 {
				truncated = truncated[:idx]
			}
			lines[i] = truncated + " …[truncated]"
			changed = true
		}
	}

	if !changed {
		return input, 0
	}

	output := strings.Join(lines, "\n")
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}
