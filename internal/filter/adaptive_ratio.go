package filter

import (
	"github.com/GrayCodeAI/tokman/internal/core"
)

// AdaptiveRatioCompressor compresses content to hit a target compression ratio.
// Instead of a fixed token budget, the user specifies a target ratio such as
// 0.5 (compress to 50% of original tokens).
//
// Algorithm:
//  1. Try ModeMinimal first.
//  2. If ratio is not met, try ModeAggressive.
//  3. If still not met, apply smart truncation as a final step.
//
// The compressor uses the pipeline's layered filters to approach the target.
type AdaptiveRatioCompressor struct {
	// TargetRatio is the desired output/input token ratio (0.0–1.0).
	// E.g., 0.5 means "compress to at most 50% of original tokens".
	TargetRatio float64

	// Filters to apply. If nil, uses a default set of lightweight filters.
	Filters []Filter
}

// NewAdaptiveRatioCompressor creates a compressor with the given target ratio.
func NewAdaptiveRatioCompressor(targetRatio float64) *AdaptiveRatioCompressor {
	return &AdaptiveRatioCompressor{TargetRatio: targetRatio}
}

// NewAdaptiveRatioCompressorWithFilters creates a compressor with custom filters.
func NewAdaptiveRatioCompressorWithFilters(targetRatio float64, filters ...Filter) *AdaptiveRatioCompressor {
	return &AdaptiveRatioCompressor{TargetRatio: targetRatio, Filters: filters}
}

// Name returns the filter name.
func (a *AdaptiveRatioCompressor) Name() string {
	return "adaptive_ratio"
}

// Apply compresses input to approach the target ratio.
// The mode parameter is used as a starting point; the compressor may
// escalate to ModeAggressive if needed.
func (a *AdaptiveRatioCompressor) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone || a.TargetRatio <= 0 || a.TargetRatio >= 1.0 {
		return input, 0
	}

	origToks := core.EstimateTokens(input)
	targetToks := int(float64(origToks) * a.TargetRatio)
	if targetToks <= 0 {
		targetToks = 1
	}

	filters := a.Filters
	if len(filters) == 0 {
		filters = defaultAdaptiveFilters()
	}

	// Phase 1: Try minimal mode
	output, _ := applyFilterChain(input, ModeMinimal, filters)
	currentToks := core.EstimateTokens(output)

	if currentToks <= targetToks {
		saved := origToks - currentToks
		if saved <= 0 {
			return input, 0
		}
		return output, saved
	}

	// Phase 2: Try aggressive mode
	output2, _ := applyFilterChain(input, ModeAggressive, filters)
	currentToks2 := core.EstimateTokens(output2)

	if currentToks2 <= currentToks {
		output = output2
		currentToks = currentToks2
	}

	if currentToks <= targetToks {
		saved := origToks - currentToks
		if saved <= 0 {
			return input, 0
		}
		return output, saved
	}

	// Phase 3: Final truncation to hit the exact target
	truncFilter := NewSmartTruncateFilterWithBudget(targetToks)
	finalOutput, _ := truncFilter.Apply(output, ModeAggressive)
	finalToks := core.EstimateTokens(finalOutput)

	saved := origToks - finalToks
	if saved <= 0 {
		return input, 0
	}
	return finalOutput, saved
}

// applyFilterChain applies a slice of filters in sequence, returning the
// final output and total tokens saved.
func applyFilterChain(input string, mode Mode, filters []Filter) (string, int) {
	current := input
	totalSaved := 0
	for _, f := range filters {
		out, saved := f.Apply(current, mode)
		if saved > 0 {
			current = out
			totalSaved += saved
		}
	}
	return current, totalSaved
}

// defaultAdaptiveFilters returns a fast default filter chain for adaptive compression.
func defaultAdaptiveFilters() []Filter {
	return []Filter{
		NewRLECompressFilter(),
		NewBoilerplateFilter(),
		NewHTMLCompressFilter(),
		NewShellOutputFilter(),
		NewErrorDedupFilter(),
		NewImportanceScoringFilter(),
		NewNumericCompressFilter(),
	}
}
