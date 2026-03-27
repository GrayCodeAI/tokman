package filter

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// DryRunPipeline wraps a filter pipeline and reports what would be compressed
// without actually modifying the content.
type DryRunPipeline struct {
	filters []Filter
}

// NewDryRunPipeline creates a dry-run wrapper.
func NewDryRunPipeline(filters []Filter) *DryRunPipeline {
	return &DryRunPipeline{filters: filters}
}

// DryRunResult holds the dry-run analysis.
type DryRunResult struct {
	OriginalTokens  int
	EstimatedTokens int
	EstimatedSaved  int
	ReductionPct    float64
	FilterResults   []FilterDryRunResult
}

// FilterDryRunResult holds the per-filter dry-run result.
type FilterDryRunResult struct {
	FilterName  string
	WouldSave   int
	WouldApply  bool
}

// Analyze runs each filter and reports what it WOULD do.
// The original input is returned unchanged — only numbers are recorded.
func (p *DryRunPipeline) Analyze(input string, mode Mode) DryRunResult {
	originalTokens := core.EstimateTokens(input)

	filterResults := make([]FilterDryRunResult, 0, len(p.filters))

	// Run the pipeline to get accurate numbers, but we will discard the output.
	current := input
	totalSaved := 0
	for _, f := range p.filters {
		out, saved := f.Apply(current, mode)
		wouldApply := saved > 0
		filterResults = append(filterResults, FilterDryRunResult{
			FilterName: f.Name(),
			WouldSave:  saved,
			WouldApply: wouldApply,
		})
		if wouldApply {
			current = out
			totalSaved += saved
		}
	}

	estimatedTokens := core.EstimateTokens(current)
	if estimatedTokens < 0 {
		estimatedTokens = 0
	}

	var reductionPct float64
	if originalTokens > 0 {
		reductionPct = float64(originalTokens-estimatedTokens) / float64(originalTokens) * 100.0
	}

	return DryRunResult{
		OriginalTokens:  originalTokens,
		EstimatedTokens: estimatedTokens,
		EstimatedSaved:  totalSaved,
		ReductionPct:    reductionPct,
		FilterResults:   filterResults,
	}
}

// FormatReport returns a human-readable dry-run report.
func (r DryRunResult) FormatReport() string {
	var sb strings.Builder

	sb.WriteString("=== Dry-Run Analysis ===\n")
	sb.WriteString(fmt.Sprintf("Original tokens  : %d\n", r.OriginalTokens))
	sb.WriteString(fmt.Sprintf("Estimated tokens : %d\n", r.EstimatedTokens))
	sb.WriteString(fmt.Sprintf("Estimated saved  : %d\n", r.EstimatedSaved))
	sb.WriteString(fmt.Sprintf("Reduction        : %.1f%%\n", r.ReductionPct))

	if len(r.FilterResults) > 0 {
		sb.WriteString("\nPer-filter breakdown:\n")
		for _, fr := range r.FilterResults {
			status := "skip"
			if fr.WouldApply {
				status = "apply"
			}
			sb.WriteString(fmt.Sprintf("  %-30s [%s] would save %d tokens\n", fr.FilterName, status, fr.WouldSave))
		}
	}

	return sb.String()
}
