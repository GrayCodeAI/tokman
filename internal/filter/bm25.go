package filter

import (
	"math"
	"strings"
)

// BM25Scorer implements Okapi BM25 scoring for relevance ranking.
// R8: Better relevance ranking than TF-IDF (used in IR systems for decades).
type BM25Scorer struct {
	avgDocLength float64
	k1           float64
	b            float64
	docFreqs     map[string]int // word -> number of documents containing it
	docCount     int
}

// newBM25Scorer creates a scorer with default parameters.
func newBM25Scorer() *BM25Scorer {
	return &BM25Scorer{
		k1:       1.2,  // Term frequency saturation
		b:        0.75, // Length normalization
		docFreqs: make(map[string]int),
	}
}

// Fit builds document frequency statistics from corpus.
func (s *BM25Scorer) Fit(docs []string) {
	s.docCount = len(docs)
	totalLen := 0

	for _, doc := range docs {
		words := uniqueWords(doc)
		totalLen += len(strings.Fields(doc))
		for _, w := range words {
			s.docFreqs[w]++
		}
	}

	if s.docCount > 0 {
		s.avgDocLength = float64(totalLen) / float64(s.docCount)
	}
}

// Score computes BM25 relevance of a document to a query.
func (s *BM25Scorer) Score(doc string, query string) float64 {
	docWords := strings.Fields(strings.ToLower(doc))
	queryWords := strings.Fields(strings.ToLower(query))
	docLen := float64(len(docWords))

	// Build term frequency map for document
	tf := make(map[string]int)
	for _, w := range docWords {
		tf[w]++
	}

	score := 0.0
	for _, qw := range queryWords {
		qw = cleanWord(qw)
		if len(qw) == 0 {
			continue
		}

		// IDF component
		df := float64(s.docFreqs[qw])
		if df == 0 {
			df = 0.5 // Smoothing for unseen terms
		}
		idf := math.Log((float64(s.docCount)-df+0.5)/(df+0.5) + 1.0)

		// TF component with saturation
		tfVal := float64(tf[qw])
		numerator := tfVal * (s.k1 + 1)
		denominator := tfVal + s.k1*(1-s.b+s.b*docLen/s.avgDocLength)

		score += idf * numerator / denominator
	}

	return score
}

// ScoreLines scores each line against a query and returns sorted indices.
func (s *BM25Scorer) ScoreLines(lines []string, query string) []LineScore {
	scores := make([]LineScore, len(lines))
	for i, line := range lines {
		scores[i] = LineScore{
			Index: i,
			Line:  line,
			Score: s.Score(line, query),
		}
	}
	return scores
}

// LineScore holds a line with its relevance score.
type LineScore struct {
	Index int
	Line  string
	Score float64
}

func uniqueWords(text string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, w := range strings.Fields(strings.ToLower(text)) {
		w = cleanWord(w)
		if !seen[w] && len(w) > 0 {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}

// QuestionAwareRecovery restores query-relevant subsequences after compression.
// R7: LongLLMLingua insight — question-aware post-compression recovery.
type QuestionAwareRecovery struct {
	scorer *BM25Scorer
}

// newQuestionAwareRecovery creates a recovery strategy.
func newQuestionAwareRecovery() *QuestionAwareRecovery {
	return &QuestionAwareRecovery{
		scorer: newBM25Scorer(),
	}
}

// Recover adds back important lines that were removed during compression.
func (r *QuestionAwareRecovery) Recover(original, compressed, query string) string {
	if query == "" {
		return compressed
	}

	origLines := strings.Split(original, "\n")
	compLines := strings.Split(compressed, "\n")

	// Build set of compressed lines
	compSet := make(map[string]bool)
	for _, line := range compLines {
		compSet[strings.TrimSpace(line)] = true
	}

	// Score original lines against query
	scored := r.scorer.ScoreLines(origLines, query)

	// Find high-scoring lines missing from compressed output
	var recovered []string
	for _, ss := range scored {
		trimmed := strings.TrimSpace(ss.Line)
		if !compSet[trimmed] && ss.Score > 0.5 {
			recovered = append(recovered, ss.Line)
			compSet[trimmed] = true
		}
	}

	// Merge compressed + recovered, preserving order
	result := append(compLines, recovered...)
	return strings.Join(result, "\n")
}
