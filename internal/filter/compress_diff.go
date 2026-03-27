package filter

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// CompressDiff represents the difference between original and compressed text.
// Task #168: Compression-aware diff generation.
type CompressDiff struct {
	OriginalTokens   int
	CompressedTokens int
	SavedTokens      int
	ReductionPct     float64
	// Hunks are the changed regions.
	Hunks []DiffHunk
}

// DiffHunk is a region that changed during compression.
type DiffHunk struct {
	// OrigLine is the 1-based line number in the original text.
	OrigLine int
	// CompLine is the 1-based line number in the compressed text.
	CompLine int
	// Removed are lines removed from the original.
	Removed []string
	// Added are lines added in the compressed output.
	Added []string
}

// Format returns a unified-diff-style representation of the compression changes.
func (d *CompressDiff) Format() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- original  (%d tokens)\n", d.OriginalTokens))
	sb.WriteString(fmt.Sprintf("+++ compressed (%d tokens, -%.1f%%)\n", d.CompressedTokens, d.ReductionPct))

	for _, h := range d.Hunks {
		sb.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", h.OrigLine, h.CompLine))
		for _, l := range h.Removed {
			sb.WriteString("- " + l + "\n")
		}
		for _, l := range h.Added {
			sb.WriteString("+ " + l + "\n")
		}
	}
	return sb.String()
}

// Summary returns a one-line diff summary.
func (d *CompressDiff) Summary() string {
	return fmt.Sprintf("%d→%d tokens (%.1f%% reduction), %d hunks changed",
		d.OriginalTokens, d.CompressedTokens, d.ReductionPct, len(d.Hunks))
}

// ComputeDiff computes the compression diff between original and compressed text.
// Uses a simple LCS-based line-level diff (no external library required).
func ComputeDiff(original, compressed string) *CompressDiff {
	origLines := strings.Split(original, "\n")
	compLines := strings.Split(compressed, "\n")

	origTokens := core.EstimateTokens(original)
	compTokens := core.EstimateTokens(compressed)
	saved := origTokens - compTokens
	var pct float64
	if origTokens > 0 {
		pct = float64(saved) / float64(origTokens) * 100
	}

	hunks := lineDiff(origLines, compLines)

	return &CompressDiff{
		OriginalTokens:   origTokens,
		CompressedTokens: compTokens,
		SavedTokens:      saved,
		ReductionPct:     pct,
		Hunks:            hunks,
	}
}

// lineDiff computes line-level diff hunks using a simple patience-like approach.
// This is O(n·m) LCS but fast enough for typical LLM output sizes.
func lineDiff(orig, comp []string) []DiffHunk {
	// Build LCS table
	n, m := len(orig), len(comp)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if orig[i] == comp[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] > lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	// Trace back to produce edit script
	var hunks []DiffHunk
	var curHunk *DiffHunk
	i, j := 0, 0
	for i < n || j < m {
		if i < n && j < m && orig[i] == comp[j] {
			// Match — close any open hunk
			if curHunk != nil {
				hunks = append(hunks, *curHunk)
				curHunk = nil
			}
			i++
			j++
		} else {
			// Change — open or extend a hunk
			if curHunk == nil {
				curHunk = &DiffHunk{OrigLine: i + 1, CompLine: j + 1}
			}
			if j < m && (i >= n || lcs[i][j+1] >= lcs[i+1][j]) {
				curHunk.Added = append(curHunk.Added, comp[j])
				j++
			} else {
				curHunk.Removed = append(curHunk.Removed, orig[i])
				i++
			}
		}
	}
	if curHunk != nil {
		hunks = append(hunks, *curHunk)
	}
	return hunks
}

// DiffFilter is a Filter that annotates its output with a compression diff header.
// Useful for debugging which lines were removed or modified by the pipeline.
type DiffFilter struct {
	Inner Filter
}

// Name returns the filter name.
func (f *DiffFilter) Name() string { return "diff:" + f.Inner.Name() }

// Apply runs the inner filter and prepends a diff summary comment.
func (f *DiffFilter) Apply(input string, mode Mode) (string, int) {
	compressed, saved := f.Inner.Apply(input, mode)
	if saved <= 0 {
		return input, 0
	}
	diff := ComputeDiff(input, compressed)
	header := fmt.Sprintf("// [tokman-diff] %s\n", diff.Summary())
	return header + compressed, saved
}
