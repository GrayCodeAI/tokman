package filter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ComparisonEntry holds the result of a single filter applied to the same input.
type ComparisonEntry struct {
	FilterName string
	Mode       Mode
	Output     string
	Tokens     int
	Saved      int
	ReductionPct float64
}

// ComparisonReport holds the side-by-side comparison of multiple filters.
type ComparisonReport struct {
	OriginalTokens int
	Entries        []ComparisonEntry
}

// FormatReport returns a human-readable comparison table.
func (r *ComparisonReport) FormatReport() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Compression Comparison (input: %d tokens)\n", r.OriginalTokens))
	sb.WriteString(strings.Repeat("─", 70) + "\n")
	sb.WriteString(fmt.Sprintf("%-30s %-10s %8s %8s %8s\n",
		"Filter", "Mode", "Tokens", "Saved", "Reduction"))
	sb.WriteString(strings.Repeat("─", 70) + "\n")

	// Sort by reduction descending
	sorted := make([]ComparisonEntry, len(r.Entries))
	copy(sorted, r.Entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ReductionPct > sorted[j].ReductionPct
	})

	for _, e := range sorted {
		sb.WriteString(fmt.Sprintf("%-30s %-10s %8d %8d %7.1f%%\n",
			e.FilterName, string(e.Mode), e.Tokens, e.Saved, e.ReductionPct))
	}
	sb.WriteString(strings.Repeat("─", 70) + "\n")
	return sb.String()
}

// Best returns the entry with the highest reduction.
func (r *ComparisonReport) Best() *ComparisonEntry {
	if len(r.Entries) == 0 {
		return nil
	}
	best := &r.Entries[0]
	for i := range r.Entries {
		if r.Entries[i].ReductionPct > best.ReductionPct {
			best = &r.Entries[i]
		}
	}
	return best
}

// CompressionComparator runs multiple filters on the same input and compares results.
// Task #163: Compression comparison tool.
type CompressionComparator struct {
	filters []Filter
	mode    Mode
}

// NewCompressionComparator creates a comparator for the given filters and mode.
func NewCompressionComparator(filters []Filter, mode Mode) *CompressionComparator {
	return &CompressionComparator{filters: filters, mode: mode}
}

// NewDefaultComparator creates a comparator with commonly used filters.
func NewDefaultComparator(mode Mode) *CompressionComparator {
	filters := []Filter{
		NewImportanceScoringFilter(),
		NewCommentStripFilter(),
		NewANSIStripFilter(),
		NewErrorDedupFilter(),
		NewLineTruncateFilter(),
		NewBoilerplateFilter(),
	}
	return NewCompressionComparator(filters, mode)
}

// Compare runs all filters on input and returns a comparison report.
func (c *CompressionComparator) Compare(input string) *ComparisonReport {
	originalTokens := core.EstimateTokens(input)
	report := &ComparisonReport{OriginalTokens: originalTokens}

	for _, f := range c.filters {
		output, saved := f.Apply(input, c.mode)
		toks := core.EstimateTokens(output)

		var reduction float64
		if originalTokens > 0 {
			reduction = float64(saved) / float64(originalTokens) * 100
		}

		report.Entries = append(report.Entries, ComparisonEntry{
			FilterName:   f.Name(),
			Mode:         c.mode,
			Output:       output,
			Tokens:       toks,
			Saved:        saved,
			ReductionPct: reduction,
		})
	}

	return report
}

// CompareAllModes runs each filter in both minimal and aggressive mode.
func (c *CompressionComparator) CompareAllModes(input string) *ComparisonReport {
	originalTokens := core.EstimateTokens(input)
	report := &ComparisonReport{OriginalTokens: originalTokens}

	for _, f := range c.filters {
		for _, mode := range []Mode{ModeMinimal, ModeAggressive} {
			output, saved := f.Apply(input, mode)
			toks := core.EstimateTokens(output)

			var reduction float64
			if originalTokens > 0 {
				reduction = float64(saved) / float64(originalTokens) * 100
			}

			report.Entries = append(report.Entries, ComparisonEntry{
				FilterName:   f.Name(),
				Mode:         mode,
				Output:       output,
				Tokens:       toks,
				Saved:        saved,
				ReductionPct: reduction,
			})
		}
	}

	return report
}
