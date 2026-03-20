package core

// HeuristicEstimator uses len/4 approximation for token counting.
// Fast but ~20-30% inaccurate vs real tiktoken counts.
type HeuristicEstimator struct{}

// NewHeuristicEstimator creates a heuristic token estimator.
func NewHeuristicEstimator() *HeuristicEstimator {
	return &HeuristicEstimator{}
}

// Estimate returns ceil(len(text) / 4.0).
func (e *HeuristicEstimator) Estimate(text string) int {
	return (len(text) + 3) / 4
}

// Compare returns heuristic vs heuristic (no actual tokenizer).
func (e *HeuristicEstimator) Compare(text string) (int, int, float64) {
	h := e.Estimate(text)
	return h, h, 0
}

// Encoding returns the estimator type.
func (e *HeuristicEstimator) Encoding() string {
	return "heuristic"
}

// UnifiedEstimator is the single source of truth for token estimation.
// Replaces duplicate EstimateTokens in filter.go, tracker.go, tokenizer.go.
func EstimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// CalculateTokensSaved computes token savings between original and filtered.
func CalculateTokensSaved(original, filtered string) int {
	origTokens := EstimateTokens(original)
	filterTokens := EstimateTokens(filtered)
	if origTokens > filterTokens {
		return origTokens - filterTokens
	}
	return 0
}
