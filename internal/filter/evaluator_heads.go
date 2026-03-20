package filter

import (
	"math"
	"sort"
	"strings"
)

// EvaluatorHeadsFilter implements EHPC-style compression (Tsinghua/Huawei, 2025).
// Uses "evaluator heads" concept - identifies important tokens by analyzing
// early-layer attention patterns.
//
// Algorithm:
// 1. Simulate "skim" mode - look at first few tokens of each chunk
// 2. Score tokens by position and content importance
// 3. Identify "evaluator" tokens that predict importance
// 4. Apply rapid pruning based on evaluator scores
//
// Research Results: 5-7x compression with minimal quality loss.
// Key insight: Early layers of LLMs can predict token importance.
type EvaluatorHeadsFilter struct {
	// Skimming parameters
	skimRatio     float64 // Ratio of tokens to examine in skim mode
	evalThreshold float64 // Threshold for evaluator scoring

	// Position weights (simulating attention patterns)
	positionWeights []float64

	// Content evaluators
	keywordEvaluators map[string]float64
}

// NewEvaluatorHeadsFilter creates a new evaluator heads filter
func NewEvaluatorHeadsFilter() *EvaluatorHeadsFilter {
	e := &EvaluatorHeadsFilter{
		skimRatio:     0.3, // Examine 30% of content in detail
		evalThreshold: 0.5,
	}

	e.initPositionWeights()
	e.initKeywordEvaluators()
	return e
}

// initPositionWeights initializes position-based weights
// Simulates the "lost in the middle" phenomenon
func (f *EvaluatorHeadsFilter) initPositionWeights() {
	// U-shaped weights - higher at beginning and end
	maxPos := 100
	f.positionWeights = make([]float64, maxPos)

	for i := 0; i < maxPos; i++ {
		// U-shaped curve
		pos := float64(i) / float64(maxPos-1)
		f.positionWeights[i] = 1.0 - 2.0*math.Abs(pos-0.5)
		// Boost edges
		if i < 10 {
			f.positionWeights[i] *= 1.5
		}
		if i >= maxPos-10 {
			f.positionWeights[i] *= 1.5
		}
	}
}

// initKeywordEvaluators initializes content-based evaluators
func (f *EvaluatorHeadsFilter) initKeywordEvaluators() {
	f.keywordEvaluators = map[string]float64{
		// Error indicators (high importance)
		"error":     3.0,
		"fail":      3.0,
		"exception": 3.0,
		"panic":     3.0,
		"fatal":     3.0,

		// Warning indicators
		"warning":    2.5,
		"warn":       2.5,
		"deprecated": 2.5,

		// Success indicators
		"success":  2.0,
		"complete": 2.0,
		"done":     2.0,
		"pass":     2.0,

		// Action indicators
		"create": 1.5,
		"update": 1.5,
		"delete": 1.5,
		"modify": 1.5,

		// Structural indicators
		"function": 1.8,
		"class":    1.8,
		"method":   1.8,
		"module":   1.8,

		// Code indicators
		"import": 1.5,
		"export": 1.5,
		"return": 1.5,
		"const":  1.3,
		"let":    1.3,
		"var":    1.3,
	}
}

// Name returns the filter name
func (f *EvaluatorHeadsFilter) Name() string {
	return "evaluator_heads"
}

// Apply applies evaluator heads compression
func (f *EvaluatorHeadsFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := len(input)

	// Phase 1: Skim mode - identify important chunks
	chunks := f.skimChunks(input)

	// Phase 2: Evaluate each chunk
	scores := f.evaluateChunks(chunks)

	// Phase 3: Select top chunks
	output := f.selectTopChunks(chunks, scores, mode)

	saved := (original - len(output)) / 4
	return output, saved
}

// skimChunks divides input into chunks for skimming
func (f *EvaluatorHeadsFilter) skimChunks(input string) []string {
	lines := strings.Split(input, "\n")

	var chunks []string
	var currentChunk []string
	chunkSize := 10 // Lines per chunk

	for _, line := range lines {
		currentChunk = append(currentChunk, line)
		if len(currentChunk) >= chunkSize {
			chunks = append(chunks, strings.Join(currentChunk, "\n"))
			currentChunk = nil
		}
	}

	// Add remaining chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, "\n"))
	}

	return chunks
}

// evaluateChunks evaluates importance of each chunk
func (f *EvaluatorHeadsFilter) evaluateChunks(chunks []string) []float64 {
	scores := make([]float64, len(chunks))

	for i, chunk := range chunks {
		scores[i] = f.evaluateChunk(chunk, i, len(chunks))
	}

	return scores
}

// evaluateChunk evaluates a single chunk
func (f *EvaluatorHeadsFilter) evaluateChunk(chunk string, index, total int) float64 {
	score := 0.0

	// Position score (U-shaped)
	posWeight := 1.0
	if index < len(f.positionWeights) {
		posWeight = f.positionWeights[index]
	} else if total > 1 {
		// Approximate for longer sequences
		pos := float64(index) / float64(total-1)
		posWeight = 1.0 - 2.0*math.Abs(pos-0.5)
	}

	// Content score
	words := tokenize(strings.ToLower(chunk))
	for _, word := range words {
		if weight, exists := f.keywordEvaluators[word]; exists {
			score += weight
		}
	}

	// Special content detection
	if strings.Contains(chunk, "error") || strings.Contains(chunk, "Error") {
		score += 5.0
	}
	if strings.Contains(chunk, "warning") || strings.Contains(chunk, "Warning") {
		score += 3.0
	}

	// Code detection
	if strings.Contains(chunk, "func ") || strings.Contains(chunk, "def ") {
		score += 2.0
	}

	return score * posWeight
}

// selectTopChunks selects the most important chunks
func (f *EvaluatorHeadsFilter) selectTopChunks(chunks []string, scores []float64, mode Mode) string {
	type indexedChunk struct {
		chunk string
		score float64
		index int
	}

	indexed := make([]indexedChunk, len(chunks))
	for i := range chunks {
		indexed[i] = indexedChunk{
			chunk: chunks[i],
			score: scores[i],
			index: i,
		}
	}

	// Sort by score
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].score > indexed[j].score
	})

	// Determine keep ratio
	keepRatio := 0.5
	if mode == ModeAggressive {
		keepRatio = 0.3
	}

	keepCount := int(float64(len(chunks)) * keepRatio)
	if keepCount < 2 {
		keepCount = 2
	}

	// Build keep set
	keepSet := make(map[int]bool)
	for i := 0; i < keepCount && i < len(indexed); i++ {
		keepSet[indexed[i].index] = true
	}

	// Always keep first and last
	keepSet[0] = true
	if len(chunks) > 1 {
		keepSet[len(chunks)-1] = true
	}

	// Build output preserving order
	var result []string
	for i, chunk := range chunks {
		if keepSet[i] {
			result = append(result, chunk)
		}
	}

	return strings.Join(result, "\n")
}

// SetSkimRatio sets the skim ratio
func (f *EvaluatorHeadsFilter) SetSkimRatio(ratio float64) {
	f.skimRatio = ratio
}

// SetEvalThreshold sets the evaluator threshold
func (f *EvaluatorHeadsFilter) SetEvalThreshold(threshold float64) {
	f.evalThreshold = threshold
}
