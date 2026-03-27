package filter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ShellOutputFilter compresses shell command output by:
//  1. Removing progress bars and spinner lines (overwritten with \r)
//  2. Collapsing repeated "downloading"/"fetching"/"resolving" lines
//  3. Removing verbose package manager output (npm/cargo/pip noise)
//  4. Deduplicating warning lines
//  5. Truncating very long file paths in output
type ShellOutputFilter struct{}

var (
	// Progress bar / spinner lines (contain \r or lots of dots/hashes)
	progressBarRe = regexp.MustCompile(`(?m)^[^\n]*(?:\r[^\n]*|\[={5,}\]|[#.]{10,}|downloading\s+\d+%)[^\n]*$`)
	// npm/yarn noise
	npmNoisyRe = regexp.MustCompile(`(?m)^(?:npm warn|npm notice|npm http|yarn info|yarn verbose)[^\n]*$`)
	// cargo fetch/download lines
	cargoNoisyRe = regexp.MustCompile(`(?m)^\s+(?:Downloading|Fetching|Blocking|Compiling|Checking)\s+[^\n]+ v\d[^\n]*$`)
	// pip download lines
	pipNoisyRe = regexp.MustCompile(`(?m)^(?:Collecting|Downloading|Using cached)\s+[^\n]*$`)
	// Repeated warning lines (same prefix)
	shellWarningRe = regexp.MustCompile(`(?m)^(?:WARNING|WARN|warning):\s+`)
	// Empty lines with only whitespace
	onlySpaceLineRe = regexp.MustCompile(`(?m)^\s+$`)
)

// NewShellOutputFilter creates a new shell output filter.
func NewShellOutputFilter() *ShellOutputFilter {
	return &ShellOutputFilter{}
}

// Name returns the filter name.
func (f *ShellOutputFilter) Name() string {
	return "shell_output"
}

// Apply compresses shell command output.
func (f *ShellOutputFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input

	// Remove progress bars/spinner overwrite lines
	output = progressBarRe.ReplaceAllString(output, "")

	// Remove npm/yarn noise
	output = npmNoisyRe.ReplaceAllString(output, "")

	if mode == ModeAggressive {
		// Remove cargo fetch lines (collapse to summary)
		output = f.collapseCargoLines(output)
		// Remove pip download lines
		output = pipNoisyRe.ReplaceAllString(output, "")
	}

	// Deduplicate warning lines
	output = f.deduplicateWarnings(output)

	// Clean up lines that are only whitespace
	output = onlySpaceLineRe.ReplaceAllString(output, "")

	// Collapse multiple blank lines
	output = collapseBlankLines(output, 2)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// collapseCargoLines collapses runs of cargo Compiling/Checking lines.
func (f *ShellOutputFilter) collapseCargoLines(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		if cargoNoisyRe.MatchString(lines[i]) {
			runStart := i
			for i < len(lines) && cargoNoisyRe.MatchString(lines[i]) {
				i++
			}
			count := i - runStart
			if count > 3 {
				// Show first 2 and last 1
				result = append(result, lines[runStart])
				result = append(result, lines[runStart+1])
				result = append(result, "    ... ["+strconv.Itoa(count-3)+" more packages] ...")
				result = append(result, lines[i-1])
			} else {
				result = append(result, lines[runStart:i]...)
			}
		} else {
			result = append(result, lines[i])
			i++
		}
	}
	return strings.Join(result, "\n")
}

// deduplicateWarnings removes duplicate warning lines (same prefix).
func (f *ShellOutputFilter) deduplicateWarnings(input string) string {
	lines := strings.Split(input, "\n")
	seenWarnings := make(map[string]bool)
	var result []string
	for _, line := range lines {
		if shellWarningRe.MatchString(line) {
			// Normalize: lowercase + trim
			key := strings.ToLower(strings.TrimSpace(line))
			if seenWarnings[key] {
				continue
			}
			seenWarnings[key] = true
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}
