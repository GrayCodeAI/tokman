package filter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// GitDiffFilter compresses git diff output by:
//  1. Collapsing identical context lines (lines starting with " ")
//  2. Removing repetitive diff headers for unchanged files
//  3. Summarizing large binary diff notices
//  4. Compacting hunk headers
type GitDiffFilter struct{}

var (
	diffFileHeaderRe = regexp.MustCompile(`^diff --git `)
	diffHunkHeaderRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@`)
	diffIndexRe      = regexp.MustCompile(`^(?:index |--- |[+]{3} )`)
	diffBinaryRe     = regexp.MustCompile(`^Binary files .* differ$`)
)

// NewGitDiffFilter creates a new git diff compression filter.
func NewGitDiffFilter() *GitDiffFilter {
	return &GitDiffFilter{}
}

// Name returns the filter name.
func (f *GitDiffFilter) Name() string {
	return "git_diff"
}

// Apply compresses git diff output.
func (f *GitDiffFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Only process if this looks like a git diff
	if !strings.Contains(input, "diff --git") && !strings.Contains(input, "@@") {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := f.processDiff(input, mode)
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

func (f *GitDiffFilter) processDiff(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string

	// Context collapse threshold
	maxContext := 3
	if mode == ModeAggressive {
		maxContext = 1
	}

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Binary diff notice: one line is enough
		if diffBinaryRe.MatchString(line) {
			result = append(result, line)
			i++
			continue
		}

		// Collapse runs of context lines (unchanged lines, prefixed with " ")
		if len(line) > 0 && line[0] == ' ' {
			runStart := i
			for i < len(lines) && len(lines[i]) > 0 && lines[i][0] == ' ' {
				i++
			}
			count := i - runStart
			if count > maxContext*2 {
				// Keep first maxContext and last maxContext, collapse middle
				result = append(result, lines[runStart:runStart+maxContext]...)
				result = append(result, " ... ["+strconv.Itoa(count-maxContext*2)+" unchanged lines] ...")
				result = append(result, lines[i-maxContext:i]...)
			} else {
				result = append(result, lines[runStart:i]...)
			}
			continue
		}

		// Collapse index/mode lines in aggressive mode
		if mode == ModeAggressive && diffIndexRe.MatchString(line) {
			// Skip redundant header lines (keep diff --git and @@ lines)
			i++
			continue
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}
