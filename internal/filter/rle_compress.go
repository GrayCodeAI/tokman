package filter

import (
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// RLECompressFilter implements repeating pattern detection and run-length encoding.
// Compresses consecutive identical or near-identical lines into a summary.
//
// Examples:
//
//	"foo\nfoo\nfoo\nfoo" → "foo [×4]"
//	"WARNING: disk\nWARNING: disk\nWARNING: disk" → "WARNING: disk [×3]"
type RLECompressFilter struct {
	// MinRun is the minimum consecutive repeats before compressing. Default: 3.
	MinRun int
}

// NewRLECompressFilter creates a new RLE compression filter.
func NewRLECompressFilter() *RLECompressFilter {
	return &RLECompressFilter{MinRun: 3}
}

// Name returns the filter name.
func (f *RLECompressFilter) Name() string {
	return "rle_compress"
}

// Apply applies run-length encoding to repeated lines.
func (f *RLECompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	minRun := f.MinRun
	if minRun <= 0 {
		minRun = 3
	}
	if mode == ModeAggressive {
		minRun = 2
	}

	original := core.EstimateTokens(input)
	lines := strings.Split(input, "\n")
	var result []string

	i := 0
	for i < len(lines) {
		line := lines[i]
		// Count consecutive identical lines
		j := i + 1
		for j < len(lines) && lines[j] == line {
			j++
		}
		count := j - i

		if count >= minRun {
			if line == "" {
				// Blank lines: just keep 1
				result = append(result, "")
			} else {
				result = append(result, line+" [×"+strconv.Itoa(count)+"]")
			}
		} else {
			result = append(result, lines[i:j]...)
		}
		i = j
	}

	output := strings.Join(result, "\n")
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}
