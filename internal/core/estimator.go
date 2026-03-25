package core

import (
	"sync"

	tiktoken "github.com/tiktoken-go/tokenizer"
)

// HeuristicEstimator uses len/4 approximation for token counting.
// Fast but ~20-30% inaccurate vs real tiktoken counts.
type HeuristicEstimator struct{}

// newHeuristicEstimator creates a heuristic token estimator.
func newHeuristicEstimator() *HeuristicEstimator {
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

// BPETokenizer wraps tiktoken for accurate BPE token counting.
// P1.1: Replaces heuristic len/4 with real BPE tokenization.
// ~20-30% more accurate than heuristic estimation.
type BPETokenizer struct {
	codec tiktoken.Codec
	cache *tokenCache
}

// tokenCache caches BPE token counts for frequently seen strings.
// Phase 2.8: Avoids repeated BPE encoding for identical content.
type tokenCache struct {
	mu    sync.RWMutex
	items map[string]int
	size  int
	max   int
}

func newTokenCache(maxSize int) *tokenCache {
	return &tokenCache{
		items: make(map[string]int),
		max:   maxSize,
	}
}

func (c *tokenCache) get(text string) (int, bool) {
	c.mu.RLock()
	val, ok := c.items[text]
	c.mu.RUnlock()
	return val, ok
}

func (c *tokenCache) set(text string, count int) {
	c.mu.Lock()
	if c.size >= c.max {
		// Simple eviction: clear half the cache
		for k := range c.items {
			delete(c.items, k)
			c.size--
			if c.size <= c.max/2 {
				break
			}
		}
	}
	c.items[text] = count
	c.size++
	c.mu.Unlock()
}

var (
	bpeInstance *BPETokenizer
	bpeOnce     sync.Once
	bpeErr      error
)

// getBPETokenizer returns a singleton BPE tokenizer.
func getBPETokenizer() (*BPETokenizer, error) {
	bpeOnce.Do(func() {
		codec, err := tiktoken.Get(tiktoken.Cl100kBase)
		if err != nil {
			bpeErr = err
			return
		}
		bpeInstance = &BPETokenizer{
			codec: codec,
			cache: newTokenCache(1024),
		}
	})
	return bpeInstance, bpeErr
}

// Count returns the accurate BPE token count with caching.
func (b *BPETokenizer) Count(text string) int {
	if text == "" {
		return 0
	}

	// Check cache first (Phase 2.8 optimization)
	if val, ok := b.cache.get(text); ok {
		return val
	}

	count, err := b.codec.Count(text)
	if err != nil {
		return (len(text) + 3) / 4 // Fallback to heuristic
	}

	// Cache result for future lookups
	b.cache.set(text, count)
	return count
}

// useBPE controls whether to use BPE or heuristic estimation.
// Set to true by default for accuracy; can be toggled for performance.
var useBPE = true

// setBPEEnabled enables or disables BPE token counting.
func setBPEEnabled(enabled bool) {
	useBPE = enabled
}

// EstimateTokens is the single source of truth for token estimation.
// P1.1: Uses BPE tokenization when available, falls back to heuristic.
// Phase 2.8: Results are cached to avoid repeated encoding.
func EstimateTokens(text string) int {
	if useBPE {
		if tok, err := getBPETokenizer(); err == nil {
			return tok.Count(text)
		}
	}
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
