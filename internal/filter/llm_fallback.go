package filter

import (
	"strings"
)

// FallbackEntry pairs a named Filter with a minimum reduction threshold.
// If the filter achieves less than MinReductionPct reduction, the chain
// tries the next entry. Task #189.
type FallbackEntry struct {
	Name            string
	Filter          Filter
	MinReductionPct float64 // e.g. 0.05 = 5 %
}

// LLMFallbackChain tries multiple compression filters in order, advancing
// to the next when the current filter fails to meet the minimum reduction.
// It always returns the best (most-reduced) result seen across all attempts.
type LLMFallbackChain struct {
	entries []FallbackEntry
}

// NewLLMFallbackChain creates a fallback chain from the provided entries.
func NewLLMFallbackChain(entries []FallbackEntry) *LLMFallbackChain {
	return &LLMFallbackChain{entries: entries}
}

// NewDefaultFallbackChain builds the standard fallback chain:
// var_shorten → comment_strip → whitespace → body filter.
func NewDefaultFallbackChain(_ Mode) *LLMFallbackChain {
	entries := []FallbackEntry{
		{
			Name:            "var_shorten",
			Filter:          NewVarShortenFilter(),
			MinReductionPct: 0.05,
		},
		{
			Name:            "comment_strip",
			Filter:          NewCommentStripFilter(),
			MinReductionPct: 0.03,
		},
		{
			Name:            "whitespace",
			Filter:          NewWhitespaceNormalizer(),
			MinReductionPct: 0.01,
		},
		{
			Name:            "body",
			Filter:          NewBodyFilter(),
			MinReductionPct: 0.0,
		},
	}
	return NewLLMFallbackChain(entries)
}

// Name satisfies the Filter interface.
func (c *LLMFallbackChain) Name() string {
	return "llm_fallback"
}

// Apply tries each filter in order. If a filter achieves at least
// MinReductionPct reduction it is accepted and returned immediately.
// Otherwise the chain continues. The best result (lowest output length)
// seen across all attempts is returned together with tokens saved.
func (c *LLMFallbackChain) Apply(input string, mode Mode) (string, int) {
	if len(c.entries) == 0 || strings.TrimSpace(input) == "" {
		return input, 0
	}

	origLen := len(input)
	bestOutput := input
	bestSaved := 0

	for _, entry := range c.entries {
		out, saved := entry.Filter.Apply(input, mode)

		// Track the overall best result.
		if saved > bestSaved {
			bestOutput = out
			bestSaved = saved
		}

		if origLen == 0 {
			break
		}

		reductionPct := float64(origLen-len(out)) / float64(origLen)
		if reductionPct >= entry.MinReductionPct {
			// This filter met the threshold — accept it.
			return out, saved
		}
		// Below threshold; try next entry.
	}

	return bestOutput, bestSaved
}
