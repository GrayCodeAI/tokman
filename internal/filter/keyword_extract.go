package filter

import (
	"sort"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// KeywordExtractFilter performs keyword extraction and query-guided compression.
// When a query/intent is provided, lines are scored by relevance to keywords
// from the query and low-relevance lines are pruned.
//
// Algorithm:
//  1. Extract keywords from the query (if set in mode context via config)
//  2. Score each line by keyword overlap (TF-style scoring)
//  3. Keep top-K% lines plus lines adjacent to high-score lines
//  4. Ensures minimum content is preserved
type KeywordExtractFilter struct {
	// Keywords to prioritize. Set via SetKeywords before Apply.
	Keywords []string
	// KeepFraction is fraction of lines to keep (0.0–1.0). Default: 0.7.
	KeepFraction float64
}

// NewKeywordExtractFilter creates a keyword extraction filter.
func NewKeywordExtractFilter() *KeywordExtractFilter {
	return &KeywordExtractFilter{
		KeepFraction: 0.7,
	}
}

// NewKeywordExtractFilterWithKeywords creates a filter pre-loaded with keywords.
func NewKeywordExtractFilterWithKeywords(keywords []string) *KeywordExtractFilter {
	return &KeywordExtractFilter{
		Keywords:     keywords,
		KeepFraction: 0.7,
	}
}

// Name returns the filter name.
func (f *KeywordExtractFilter) Name() string {
	return "keyword_extract"
}

// Apply prunes lines with low keyword relevance.
func (f *KeywordExtractFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone || len(f.Keywords) == 0 {
		return input, 0
	}

	keepFrac := f.KeepFraction
	if mode == ModeAggressive {
		keepFrac = 0.5
	}

	lines := strings.Split(input, "\n")
	if len(lines) < 10 {
		return input, 0
	}

	// Build lowercase keyword set
	kwSet := make(map[string]bool, len(f.Keywords))
	for _, kw := range f.Keywords {
		kwSet[strings.ToLower(kw)] = true
	}

	// Score each line
	type scoredLine struct {
		idx   int
		score float64
	}
	scores := make([]scoredLine, len(lines))
	for i, line := range lines {
		scores[i] = scoredLine{idx: i, score: f.scoreLine(line, kwSet)}
	}

	// Determine threshold: keep top keepFrac% of lines by score
	sortedScores := make([]float64, len(scores))
	for i, s := range scores {
		sortedScores[i] = s.score
	}
	sort.Float64s(sortedScores)
	keepCount := int(float64(len(lines)) * keepFrac)
	if keepCount < 5 {
		keepCount = 5
	}
	threshold := 0.0
	if len(sortedScores)-keepCount >= 0 {
		threshold = sortedScores[len(sortedScores)-keepCount]
	}

	// Keep lines at or above threshold; also keep neighbors of high-score lines
	keep := make([]bool, len(lines))
	for i, s := range scores {
		if s.score >= threshold || s.score > 0 {
			keep[i] = true
			// Keep 1 neighbor on each side for context
			if i > 0 {
				keep[i-1] = true
			}
			if i+1 < len(lines) {
				keep[i+1] = true
			}
		}
	}
	// Always keep first and last lines
	if len(lines) > 0 {
		keep[0] = true
		keep[len(lines)-1] = true
	}

	original := core.EstimateTokens(input)
	var result []string
	skipped := 0
	for i, line := range lines {
		if keep[i] {
			if skipped > 0 {
				result = append(result, "... ["+strconv.Itoa(skipped)+" lines omitted] ...")
				skipped = 0
			}
			result = append(result, line)
		} else {
			skipped++
		}
	}
	if skipped > 0 {
		result = append(result, "... ["+strconv.Itoa(skipped)+" lines omitted] ...")
	}

	output := strings.Join(result, "\n")
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// scoreLine scores a line by keyword relevance.
func (f *KeywordExtractFilter) scoreLine(line string, kwSet map[string]bool) float64 {
	lower := strings.ToLower(line)
	words := strings.Fields(lower)
	if len(words) == 0 {
		return 0
	}

	matches := 0
	for _, word := range words {
		// Strip punctuation
		word = strings.Trim(word, ".,;:!?\"'()[]{}")
		if kwSet[word] {
			matches++
		}
	}
	return float64(matches) / float64(len(words))
}
