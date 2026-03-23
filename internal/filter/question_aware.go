package filter

import (
	"strings"
)

// QuestionAwareFilter implements LongLLMLingua-style question-aware recovery.
// Research basis: "LongLLMLingua: Accelerating and Enhancing LLMs in Long Context Scenarios"
// (Jiang et al., ACL 2024) - preserves query-relevant subsequences during compression.
//
// T12: Key insight - compression should be aware of the question/query and preserve
// subsequences that are likely relevant to answering it.
//
// This filter:
// 1. Extracts key terms from the query
// 2. Scores content segments by relevance to query
// 3. Preserves high-relevance segments even under aggressive compression
// 4. Enables "recovery" of important context post-compression
type QuestionAwareFilter struct {
	config QuestionAwareConfig
}

// QuestionAwareConfig holds configuration for question-aware filtering
type QuestionAwareConfig struct {
	// Enable question-aware filtering
	Enabled bool

	// The query/question to be aware of
	Query string

	// Minimum relevance score to preserve (0.0-1.0)
	RelevanceThreshold float64

	// Number of context tokens to preserve around matches
	ContextWindow int

	// Boost factor for exact matches
	ExactMatchBoost float64

	// Boost factor for partial matches
	PartialMatchBoost float64
}

// DefaultQuestionAwareConfig returns default configuration
func DefaultQuestionAwareConfig() QuestionAwareConfig {
	return QuestionAwareConfig{
		Enabled:            true,
		RelevanceThreshold: 0.3,
		ContextWindow:      20,  // tokens
		ExactMatchBoost:    1.0,
		PartialMatchBoost:  0.6,
	}
}

// NewQuestionAwareFilter creates a new question-aware filter
func NewQuestionAwareFilter(query string) *QuestionAwareFilter {
	cfg := DefaultQuestionAwareConfig()
	cfg.Query = query
	return &QuestionAwareFilter{config: cfg}
}

// Name returns the filter name
func (q *QuestionAwareFilter) Name() string {
	return "question_aware"
}

// Apply applies question-aware filtering to preserve query-relevant content
func (q *QuestionAwareFilter) Apply(input string, mode Mode) (string, int) {
	if !q.config.Enabled || q.config.Query == "" {
		return input, 0
	}

	originalTokens := EstimateTokens(input)

	// Extract query terms
	queryTerms := q.extractTerms(q.config.Query)
	if len(queryTerms) == 0 {
		return input, 0
	}

	// Split input into segments (lines or sentences)
	segments := q.segmentContent(input)

	// Score each segment for relevance
	scored := make([]segmentScore, len(segments))
	for i, seg := range segments {
		scored[i] = segmentScore{
			text:  seg,
			score: q.scoreRelevance(seg, queryTerms),
			index: i,
		}
	}

	// Preserve segments above threshold
	keep := make(map[int]bool)
	for _, s := range scored {
		if s.score >= q.config.RelevanceThreshold {
			keep[s.index] = true
			// Also preserve context window around match
			for j := s.index - 2; j <= s.index + 2 && j < len(segments); j++ {
				if j >= 0 {
					keep[j] = true
				}
			}
		}
	}

	// Always preserve first and last segments (structural importance)
	if len(segments) > 0 {
		keep[0] = true
		keep[len(segments)-1] = true
	}

	// Build output
	var result strings.Builder
	for i, seg := range segments {
		if keep[i] {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(seg)
		}
	}

	output := result.String()
	finalTokens := EstimateTokens(output)
	saved := originalTokens - finalTokens

	return output, saved
}

// segmentScore holds a segment with its relevance score
type segmentScore struct {
	text  string
	score float64
	index int
}

// extractTerms extracts important terms from a query
func (q *QuestionAwareFilter) extractTerms(query string) []string {
	// Lowercase and split
	words := strings.Fields(strings.ToLower(query))

	// Filter out stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true, "can": true,
		"need": true, "dare": true, "ought": true, "used": true, "to": true,
		"of": true, "in": true, "for": true, "on": true, "with": true,
		"at": true, "by": true, "from": true, "as": true, "into": true,
		"through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "under": true,
		"again": true, "further": true, "then": true, "once": true,
		"and": true, "but": true, "or": true, "nor": true, "so": true,
		"yet": true, "both": true, "either": true, "neither": true,
		"not": true, "only": true, "own": true, "same": true, "than": true,
		"too": true, "very": true, "just": true, "also": true, "now": true,
		"how": true, "what": true, "when": true, "where": true, "which": true,
		"who": true, "whom": true, "whose": true, "why": true, "that": true,
		"this": true, "these": true, "those": true, "it": true, "its": true,
	}

	var terms []string
	for _, w := range words {
		// Clean punctuation
		w = strings.Trim(w, ".,!?;:\"'()[]{}")
		if len(w) > 2 && !stopWords[w] {
			terms = append(terms, w)
		}
	}

	return terms
}

// segmentContent splits content into processable segments
func (q *QuestionAwareFilter) segmentContent(content string) []string {
	// Split by lines for code/logs, by sentences for prose
	lines := strings.Split(content, "\n")

	// For code-like content, use lines directly
	// For prose-like content, could split by sentences
	return lines
}

// scoreRelevance scores a segment's relevance to the query
func (q *QuestionAwareFilter) scoreRelevance(segment string, queryTerms []string) float64 {
	segLower := strings.ToLower(segment)

	var score float64
	matchCount := 0

	for _, term := range queryTerms {
		if strings.Contains(segLower, term) {
			// Exact match
			matchCount++
			score += q.config.ExactMatchBoost
		} else {
			// Check for partial match (substring)
			for _, word := range strings.Fields(segLower) {
				word = strings.Trim(word, ".,!?;:\"'()[]{}")
				if len(word) > 3 && len(term) > 3 {
					// Check for common prefix or substring
					if strings.HasPrefix(word, term) || strings.HasPrefix(term, word) {
						score += q.config.PartialMatchBoost
						break
					}
				}
			}
		}
	}

	// Normalize by number of terms
	if len(queryTerms) > 0 {
		score = score / float64(len(queryTerms))
	}

	// Bonus for multiple matches (higher density)
	if matchCount > 1 {
		score *= 1.2
	}

	return score
}

// SetQuery sets the query for question-aware filtering
func (q *QuestionAwareFilter) SetQuery(query string) {
	q.config.Query = query
}

// SetEnabled enables or disables the filter
func (q *QuestionAwareFilter) SetEnabled(enabled bool) {
	q.config.Enabled = enabled
}

// GetStats returns filter statistics
func (q *QuestionAwareFilter) GetStats() map[string]any {
	return map[string]any{
		"enabled":             q.config.Enabled,
		"query":               q.config.Query,
		"relevance_threshold": q.config.RelevanceThreshold,
		"context_window":      q.config.ContextWindow,
	}
}
