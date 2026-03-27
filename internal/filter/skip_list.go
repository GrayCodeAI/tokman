package filter

import (
	"math"
	"regexp"
	"strings"
)

// SkipListGuard wraps a filter and skips it when the input is
// already highly compressed (high entropy) or falls into a known
// uncompressible category. Saves CPU on content that won't benefit.
//
// Known uncompressible categories:
//   - Pure base64 blobs (already encoded, high entropy)
//   - Minified JS/CSS (dense, no structural repetition)
//   - Already-compressed JSON (no whitespace, single-line dense)
//   - Binary-like content (high byte diversity)
//   - Very short content (< 20 tokens: overhead not worth it)
//
// Entropy threshold: Shannon entropy > 4.5 bits/byte → likely compressed.
type SkipListGuard struct {
	inner            Filter
	entropyThreshold float64 // default 4.5
	minTokens        int     // default 20
}

var (
	// Dense base64 lines (no whitespace, all base64 chars, long)
	base64DenseRe = regexp.MustCompile(`^[A-Za-z0-9+/]{200,}={0,2}$`)
	// Minified JS/CSS: semicolons, braces, no spaces
	minifiedCodeRe = regexp.MustCompile(`(?:[;{}]{5,}|[^\s]{100,})`)
	// Already-minified JSON (one line, dense)
	minifiedJSONRe = regexp.MustCompile(`^\{"[^"]+":`)
)

// NewSkipListGuard wraps a filter with the default skip-list thresholds.
func NewSkipListGuard(inner Filter) *SkipListGuard {
	return &SkipListGuard{
		inner:            inner,
		entropyThreshold: 4.5,
		minTokens:        20,
	}
}

// NewSkipListGuardWithThreshold wraps a filter with custom thresholds.
func NewSkipListGuardWithThreshold(inner Filter, entropyThreshold float64, minTokens int) *SkipListGuard {
	return &SkipListGuard{
		inner:            inner,
		entropyThreshold: entropyThreshold,
		minTokens:        minTokens,
	}
}

// Name returns the wrapped filter's name with a skip-guard prefix.
func (g *SkipListGuard) Name() string {
	return g.inner.Name()
}

// Apply runs the inner filter only if the content is likely compressible.
func (g *SkipListGuard) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Skip very short content
	if estimateWordCount(input) < g.minTokens {
		return input, 0
	}

	// Skip high-entropy (already compressed / binary) content
	if shannonEntropy(input) > g.entropyThreshold {
		return input, 0
	}

	// Skip known uncompressible patterns
	if isUncompressible(input) {
		return input, 0
	}

	return g.inner.Apply(input, mode)
}

// isUncompressible returns true for content known not to compress well.
func isUncompressible(input string) bool {
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return false
	}

	// Check first non-empty line for base64 or minified code
	for _, line := range lines[:min10(len(lines), 3)] {
		trimmed := strings.TrimSpace(line)
		if base64DenseRe.MatchString(trimmed) {
			return true
		}
		if minifiedCodeRe.MatchString(trimmed) {
			return true
		}
	}

	// Single-line JSON: likely already compact
	if len(lines) == 1 && minifiedJSONRe.MatchString(strings.TrimSpace(input)) {
		return len(input) > 500
	}

	return false
}

// shannonEntropy computes the Shannon entropy of a string in bits/byte.
// Values > 4.5 indicate high-entropy content (already compressed or binary).
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	// Sample up to 2048 bytes for speed
	sample := s
	if len(sample) > 2048 {
		sample = sample[:2048]
	}

	freq := make(map[byte]int, 256)
	for i := 0; i < len(sample); i++ {
		freq[sample[i]]++
	}

	entropy := 0.0
	n := float64(len(sample))
	for _, count := range freq {
		if count == 0 {
			continue
		}
		p := float64(count) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// estimateWordCount is a fast word count approximation (spaces + 1).
func estimateWordCount(s string) int {
	count := 1
	for _, ch := range s {
		if ch == ' ' || ch == '\n' || ch == '\t' {
			count++
		}
	}
	return count
}

func min10(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WrapWithSkipList wraps each filter in a SkipListGuard.
func WrapWithSkipList(filters []Filter) []Filter {
	result := make([]Filter, len(filters))
	for i, f := range filters {
		result[i] = NewSkipListGuard(f)
	}
	return result
}
