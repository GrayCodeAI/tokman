package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// MetaTokenFilter implements Layer 15: Lossless Token Sequence Compression via Meta-Tokens.
//
// Research Source: "Lossless Token Sequence Compression via Meta-Tokens" (arXiv:2506.00307)
// Key Innovation: LZ77-style lossless compression operating on token sequences.
// Results: 27% token reduction = 47% compute reduction (due to quadratic attention)
// Critical Feature: ZERO semantic loss - trivially reversible.
//
// Methodology:
// 1. Scan for repeated token sequences (sliding window)
// 2. Replace with meta-tokens that reference the original sequence
// 3. Meta-tokens use special marker format: [META:hash:length]
// 4. Decompression expands meta-tokens back to original sequences
type MetaTokenFilter struct {
	config     MetaTokenConfig
	metaTokens map[string]MetaToken // hash -> meta-token info
	mu         sync.RWMutex
}

// MetaTokenConfig holds configuration for the meta-token filter
type MetaTokenConfig struct {
	// WindowSize is the maximum sequence length to consider for compression
	WindowSize int

	// MinPattern is the minimum sequence length to compress (shorter = more compression but more meta-tokens)
	MinPattern int

	// MaxMetaTokens limits the number of meta-tokens created (0 = unlimited)
	MaxMetaTokens int

	// EnableDecompression allows this filter to also decompress
	EnableDecompression bool
}

// MetaToken represents a compressed token sequence
type MetaToken struct {
	Hash     string // SHA256 hash of the original sequence
	Original string // Original text that was compressed
	Length   int    // Number of tokens in original sequence
	Count    int    // Number of times this pattern was found
}

// DefaultMetaTokenConfig returns the default configuration
func DefaultMetaTokenConfig() MetaTokenConfig {
	return MetaTokenConfig{
		WindowSize:          512, // Match research paper's window size
		MinPattern:          3,   // Minimum 3 tokens to compress
		MaxMetaTokens:       1000,
		EnableDecompression: true,
	}
}

// NewMetaTokenFilter creates a new meta-token lossless compression filter
func NewMetaTokenFilter() *MetaTokenFilter {
	return NewMetaTokenFilterWithConfig(DefaultMetaTokenConfig())
}

// NewMetaTokenFilterWithConfig creates a meta-token filter with custom config
func NewMetaTokenFilterWithConfig(cfg MetaTokenConfig) *MetaTokenFilter {
	return &MetaTokenFilter{
		config:     cfg,
		metaTokens: make(map[string]MetaToken),
	}
}

// Name returns the filter name
func (f *MetaTokenFilter) Name() string {
	return "meta_token"
}

// Apply applies lossless compression via meta-tokens
func (f *MetaTokenFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Tokenize input into space-separated tokens. Using strings.Split on " "
	// (not strings.Fields) so that non-space whitespace such as newlines is
	// preserved inside tokens and strings.Join(compressed, " ") faithfully
	// reconstructs the original content without inserting extra spaces.
	tokens := strings.Split(input, " ")
	if len(tokens) < f.config.MinPattern {
		return input, 0
	}

	// Compress by finding repeated patterns
	compressed, saved := f.compress(tokens)

	// Reconstruct string from compressed tokens
	output := strings.Join(compressed, " ")

	return output, saved
}

// compress finds repeated patterns and replaces them with meta-tokens
func (f *MetaTokenFilter) compress(tokens []string) ([]string, int) {
	result := make([]string, 0, len(tokens))
	saved := 0
	metaCount := 0

	i := 0
	for i < len(tokens) {
		if metaCount >= f.config.MaxMetaTokens && f.config.MaxMetaTokens > 0 {
			// Max meta-tokens reached, just copy remaining
			result = append(result, tokens[i:]...)
			break
		}

		// Try to find the longest repeated pattern starting at position i
		found := false
		for patternLen := min(f.config.WindowSize, len(tokens)-i); patternLen >= f.config.MinPattern; patternLen-- {
			pattern := tokens[i : i+patternLen]
			patternStr := strings.Join(pattern, " ")

			// Check if this pattern appears later in the sequence
			if matchStart := f.findPattern(tokens, pattern, i+patternLen); matchStart != -1 {
				// Create meta-token
				hash := f.hashPattern(patternStr)
				metaToken := fmt.Sprintf("[META:%s:%d]", hash[:8], patternLen)

				// Store meta-token info
				f.mu.Lock()
				if mt, exists := f.metaTokens[hash]; exists {
					mt.Count++
					f.metaTokens[hash] = mt
				} else {
					f.metaTokens[hash] = MetaToken{
						Hash:     hash,
						Original: patternStr,
						Length:   patternLen,
						Count:    1,
					}
				}
				f.mu.Unlock()

				result = append(result, metaToken)
				saved += patternLen - 1 // Save patternLen tokens, add 1 meta-token
				i += patternLen
				metaCount++
				found = true
				break
			}
		}

		if !found {
			result = append(result, tokens[i])
			i++
		}
	}

	return result, saved
}

// findPattern searches for pattern in tokens starting from startIdx
func (f *MetaTokenFilter) findPattern(tokens, pattern []string, startIdx int) int {
	for i := startIdx; i <= len(tokens)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if tokens[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// hashPattern creates a short hash of a pattern for the meta-token
func (f *MetaTokenFilter) hashPattern(pattern string) string {
	h := sha256.Sum256([]byte(pattern))
	return hex.EncodeToString(h[:])
}

// Decompress expands meta-tokens back to original sequences
func (f *MetaTokenFilter) Decompress(input string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := input

	// Replace each meta-token with its original content
	for hash, mt := range f.metaTokens {
		metaMarker := fmt.Sprintf("[META:%s:%d]", hash[:8], mt.Length)
		result = strings.ReplaceAll(result, metaMarker, mt.Original)
	}

	return result
}

// GetMetaTokens returns all stored meta-tokens (for serialization)
func (f *MetaTokenFilter) GetMetaTokens() map[string]MetaToken {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]MetaToken, len(f.metaTokens))
	for k, v := range f.metaTokens {
		result[k] = v
	}
	return result
}

// LoadMetaTokens loads meta-tokens (for deserialization)
func (f *MetaTokenFilter) LoadMetaTokens(tokens map[string]MetaToken) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for k, v := range tokens {
		f.metaTokens[k] = v
	}
}

// Stats returns compression statistics
func (f *MetaTokenFilter) Stats() MetaTokenStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	totalPatterns := 0
	totalSaved := 0
	for _, mt := range f.metaTokens {
		totalPatterns += mt.Count
		totalSaved += mt.Count * (mt.Length - 1)
	}

	return MetaTokenStats{
		UniquePatterns: len(f.metaTokens),
		TotalPatterns:  totalPatterns,
		EstTokensSaved: totalSaved,
	}
}

// MetaTokenStats holds statistics for meta-token compression
type MetaTokenStats struct {
	UniquePatterns int
	TotalPatterns  int
	EstTokensSaved int
}
