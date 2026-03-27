package filter

import (
	"math"
	"sort"
	"strings"
)

// PerplexityFilter implements LLMLingua-style compression (Microsoft/Tsinghua, 2023).
// Uses perplexity-based iterative pruning with a budget controller.
//
// Algorithm:
// 1. Calculate perplexity of each token given context
// 2. Rank tokens by perplexity (higher = more surprising = more important)
// 3. Iteratively remove lowest-perplexity tokens while staying within budget
//
// Research Results: Up to 20x compression with semantic preservation.
type PerplexityFilter struct {
	// Budget controller settings
	targetRatio float64 // Target compression ratio

	// Iterative pruning settings
	iterationSteps int
	pruneRatio     float64

	// Context window for perplexity calculation
	contextWindow int
	
	// Convergence threshold for early exit (Phase 1 optimization)
	convergenceThreshold float64
}

// NewPerplexityFilter creates a new perplexity-based filter
func NewPerplexityFilter() *PerplexityFilter {
	return &PerplexityFilter{
		targetRatio:          0.3,  // Keep 30% of tokens
		iterationSteps:       2,    // Reduced from 3 for performance (Phase 1)
		pruneRatio:           0.7,  // Prune 30% each iteration
		contextWindow:        10,   // Words to consider for context
		convergenceThreshold: 0.05, // 5% change threshold for early exit
	}
}

// Name returns the filter name
func (f *PerplexityFilter) Name() string {
	return "perplexity"
}

// Apply applies perplexity-based iterative pruning with early exit (Phase 1 optimization)
func (f *PerplexityFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := len(input)
	output := input
	prevLen := original

	// Iterative pruning with convergence check
	for i := 0; i < f.iterationSteps; i++ {
		output = f.pruneIteration(output, mode)
		
		// Early exit if convergence detected (Phase 1 optimization)
		currentLen := len(output)
		changeRatio := float64(prevLen-currentLen) / float64(prevLen)
		if changeRatio < f.convergenceThreshold && i > 0 {
			break // Converged - no significant improvement
		}
		prevLen = currentLen
	}

	saved := (original - len(output)) / 4
	return output, saved
}

// pruneIteration performs one iteration of perplexity-based pruning
func (f *PerplexityFilter) pruneIteration(input string, mode Mode) string {
	lines := strings.Split(input, "\n")
	var result []string

	for _, line := range lines {
		processed := f.pruneLine(line, mode)
		result = append(result, processed)
	}

	return strings.Join(result, "\n")
}

// pruneLine prunes a single line based on token importance
func (f *PerplexityFilter) pruneLine(line string, mode Mode) string {
	words := tokenize(line)
	if len(words) < 5 {
		return line // Keep short lines intact
	}

	// Score each token
	scores := f.scoreTokens(words)

	// Sort indices by score
	indices := make([]int, len(words))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return scores[indices[i]] > scores[indices[j]]
	})

	// Determine how many to keep
	keepCount := int(float64(len(words)) * f.targetRatio)
	if keepCount < 3 {
		keepCount = 3 // Keep at least 3 words
	}

	// Build set of indices to keep
	keepSet := make(map[int]bool)
	for i := 0; i < keepCount && i < len(indices); i++ {
		keepSet[indices[i]] = true
	}

	// Always keep first and last word for context
	keepSet[0] = true
	keepSet[len(words)-1] = true

	// Build result preserving order
	var result []string
	for i, word := range words {
		if keepSet[i] {
			result = append(result, word)
		}
	}

	return strings.Join(result, " ")
}

// scoreTokens calculates importance scores for each token
// Uses a simplified perplexity approximation based on:
// - Token length (longer = more specific)
// - Position (first/last = more important)
// - Local frequency (rare in context = more important)
// - Special characters (code/symbols = more important)
func (f *PerplexityFilter) scoreTokens(words []string) []float64 {
	scores := make([]float64, len(words))

	// Count local frequency
	freq := make(map[string]int)
	for _, w := range words {
		freq[strings.ToLower(w)]++
	}

	for i, word := range words {
		// Base score: inverse frequency (rare = important)
		localFreq := freq[strings.ToLower(word)]
		freqScore := 1.0 / float64(localFreq)

		// Length score: longer words often carry more meaning
		lenScore := math.Log(float64(len(word)) + 1)

		// Position score: beginning and end are more important
		posScore := 1.0
		if i == 0 || i == len(words)-1 {
			posScore = 2.0
		} else if i < 3 || i >= len(words)-3 {
			posScore = 1.5
		}

		// Special content score: code, numbers, symbols
		specialScore := 1.0
		if isCodeToken(word) || isNumeric(word) {
			specialScore = 3.0
		}

		// Combine scores
		scores[i] = freqScore * lenScore * posScore * specialScore
	}

	return scores
}

// isCodeToken checks if a token looks like code
func isCodeToken(word string) bool {
	// Contains programming-like patterns
	if strings.Contains(word, "(") || strings.Contains(word, ")") {
		return true
	}
	if strings.Contains(word, ".") && !strings.HasPrefix(word, ".") {
		return true
	}
	if strings.Contains(word, "_") || strings.Contains(word, "::") {
		return true
	}
	if strings.HasPrefix(word, "$") || strings.HasPrefix(word, "@") {
		return true
	}

	// CamelCase or PascalCase
	hasUpper := false
	hasLower := false
	for _, c := range word {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		}
		if c >= 'a' && c <= 'z' {
			hasLower = true
		}
	}

	return hasUpper && hasLower && len(word) > 3
}

// isNumeric checks if a token is numeric
func isNumeric(word string) bool {
	for _, c := range word {
		if (c < '0' || c > '9') && c != '.' && c != '-' && c != '+' {
			return false
		}
	}
	return len(word) > 0
}

// SetTargetRatio sets the target compression ratio
func (f *PerplexityFilter) SetTargetRatio(ratio float64) {
	f.targetRatio = ratio
}

// SetIterations sets the number of pruning iterations
func (f *PerplexityFilter) SetIterations(iterations int) {
	f.iterationSteps = iterations
}
