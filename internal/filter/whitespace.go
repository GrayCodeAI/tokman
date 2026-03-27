package filter

import (
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// WhitespaceNormalizer collapses blank lines, strips trailing whitespace,
// and expands tabs. Task #68.
type WhitespaceNormalizer struct{}

// NewWhitespaceNormalizer creates a new WhitespaceNormalizer filter.
func NewWhitespaceNormalizer() *WhitespaceNormalizer {
	return &WhitespaceNormalizer{}
}

// Name returns the filter name.
func (f *WhitespaceNormalizer) Name() string {
	return "whitespace"
}

// Apply normalises whitespace in input according to mode:
//   - ModeMinimal:    collapses 3+ consecutive blank lines → 2, strips trailing
//     whitespace, expands tabs.
//   - ModeAggressive: collapses 2+ consecutive blank lines → 1, strips trailing
//     whitespace, expands tabs.
//   - ModeNone:       returns input unchanged.
func (f *WhitespaceNormalizer) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))

	// maxConsecutiveBlanks is the maximum number of blank lines to allow.
	// ModeMinimal keeps up to 2; ModeAggressive keeps up to 1.
	maxConsecutiveBlanks := 2
	if mode == ModeAggressive {
		maxConsecutiveBlanks = 1
	}

	consecutiveBlanks := 0

	for _, line := range lines {
		// Expand tabs to 4 spaces.
		line = strings.ReplaceAll(line, "\t", "    ")

		// Strip trailing whitespace.
		line = strings.TrimRight(line, " \t\r")

		if line == "" {
			consecutiveBlanks++
			if consecutiveBlanks <= maxConsecutiveBlanks {
				out = append(out, line)
			}
			// Lines beyond the limit are simply dropped.
		} else {
			consecutiveBlanks = 0
			out = append(out, line)
		}
	}

	output := strings.Join(out, "\n")
	saved := originalTokens - core.EstimateTokens(output)
	if saved < 0 {
		saved = 0
	}
	return output, saved
}
