package filter

import (
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// AttentionPredictor implements learned pattern prediction for compression.
// Research: "AttentionPredictor: Temporal Patterns Matter for KV Cache Compression" (NeurIPS 2025)
// Key Innovation: Uses learned patterns to predict which tokens/lines will be important
// before processing them, enabling pre-computation of importance scores.
//
// In TokMan: Instead of scoring each line independently, we learn patterns from
// previous compressions and predict importance based on content type, position,
// and structural patterns. This is faster than computing importance from scratch.
type AttentionPredictor struct {
	config   PredictorConfig
	patterns map[string]PredictorPattern
}

// PredictorConfig holds configuration for attention prediction
type PredictorConfig struct {
	Enabled          bool
	LearningRate     float64
	MinSamples       int
	MinContentLength int
}

// PredictorPattern learned from previous compressions
type PredictorPattern struct {
	ContentType     string
	AvgImportance   map[string]float64 // word -> importance score
	PositionWeights []float64          // position -> weight
	TotalSamples    int
}

// DefaultPredictorConfig returns default configuration
func DefaultPredictorConfig() PredictorConfig {
	return PredictorConfig{
		Enabled:          true,
		LearningRate:     0.1,
		MinSamples:       3,
		MinContentLength: 100,
	}
}

// NewAttentionPredictor creates a new attention predictor
func NewAttentionPredictor() *AttentionPredictor {
	return &AttentionPredictor{
		config:   DefaultPredictorConfig(),
		patterns: make(map[string]PredictorPattern),
	}
}

// Name returns the filter name
func (a *AttentionPredictor) Name() string { return "attention_predictor" }

// Apply applies attention prediction-based compression
func (a *AttentionPredictor) Apply(input string, mode Mode) (string, int) {
	if !a.config.Enabled || mode == ModeNone {
		return input, 0
	}

	if len(input) < a.config.MinContentLength {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	// Detect content type
	contentType := a.detectContentType(input)

	// Get learned pattern for this content type
	pattern := a.patterns[contentType]

	lines := strings.Split(input, "\n")
	var result strings.Builder

	// Predict importance for each line using learned patterns
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result.WriteString("\n")
			continue
		}

		importance := a.predictImportance(trimmed, i, len(lines), pattern, mode)

		// Keep lines above threshold
		threshold := 0.3
		if mode == ModeAggressive {
			threshold = 0.5
		}

		if importance >= threshold {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	output := strings.TrimSpace(result.String())
	finalTokens := core.EstimateTokens(output)
	saved := originalTokens - finalTokens

	// Learn from this compression
	if saved > 0 {
		a.learnFromCompression(input, output, contentType)
	}

	if saved < 3 {
		return input, 0
	}

	return output, saved
}

// predictImportance predicts line importance using learned patterns
func (a *AttentionPredictor) predictImportance(line string, position, totalLines int, pattern PredictorPattern, mode Mode) float64 {
	score := 0.5

	// 1. Position-based prediction (learned from previous compressions)
	if len(pattern.PositionWeights) > 0 {
		posIdx := int(float64(position) / float64(totalLines) * float64(len(pattern.PositionWeights)))
		if posIdx >= len(pattern.PositionWeights) {
			posIdx = len(pattern.PositionWeights) - 1
		}
		score = pattern.PositionWeights[posIdx]
	}

	// 2. Word-based prediction (learned importance per word)
	if len(pattern.AvgImportance) > 0 {
		words := strings.Fields(strings.ToLower(line))
		wordScore := 0.0
		wordCount := 0
		for _, w := range words {
			if importance, ok := pattern.AvgImportance[w]; ok {
				wordScore += importance
				wordCount++
			}
		}
		if wordCount > 0 {
			score = score*0.5 + (wordScore/float64(wordCount))*0.5
		}
	}

	// 3. Structural heuristics (always applied)
	lower := strings.ToLower(line)
	if strings.Contains(lower, "error") || strings.Contains(lower, "fail") {
		score += 0.3
	}
	if strings.Contains(line, "func ") || strings.Contains(line, "class ") {
		score += 0.2
	}

	return score
}

// detectContentType detects content type for pattern matching
func (a *AttentionPredictor) detectContentType(input string) string {
	lines := strings.Split(input, "\n")

	codeScore := 0
	logScore := 0
	diffScore := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 4 {
			continue
		}

		prefix := trimmed[:min(4, len(trimmed))]

		if prefix == "func" || prefix == "def " || prefix == "clas" || prefix == "impo" {
			codeScore++
		}
		if (len(prefix) == 4 && prefix[0] == '2' && prefix[1] == '0' && prefix[2] >= '0' && prefix[2] <= '9' && prefix[3] >= '0' && prefix[3] <= '9') || prefix == "ERRO" || prefix == "WARN" {
			logScore++
		}
		if prefix == "diff" || prefix == "comm" || prefix == "--- " || prefix == "+++ " {
			diffScore++
		}
	}

	if codeScore > logScore && codeScore > diffScore {
		return "code"
	}
	if logScore > diffScore {
		return "logs"
	}
	if diffScore > 0 {
		return "diff"
	}
	return "text"
}

// learnFromCompression updates patterns based on compression results
func (a *AttentionPredictor) learnFromCompression(original, compressed, contentType string) {
	pattern := a.patterns[contentType]
	if pattern.AvgImportance == nil {
		pattern.AvgImportance = make(map[string]float64)
	}
	if len(pattern.PositionWeights) == 0 {
		pattern.PositionWeights = make([]float64, 10)
		for i := range pattern.PositionWeights {
			pattern.PositionWeights[i] = 0.5
		}
	}

	// Build set of surviving words
	survivedWords := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(compressed)) {
		cleaned := strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(cleaned) > 2 {
			survivedWords[cleaned] = true
		}
	}

	// Update word importance based on survival
	origLines := strings.Split(original, "\n")
	totalLines := len(origLines)

	for i, line := range origLines {
		words := strings.Fields(strings.ToLower(line))
		lineImportance := 0.0

		for _, w := range words {
			cleaned := strings.Trim(w, ".,;:!?\"'()[]{}")
			if len(cleaned) > 2 {
				if survivedWords[cleaned] {
					lineImportance += 1.0
				}
			}
		}

		if len(words) > 0 {
			lineImportance /= float64(len(words))
		}

		// Update word importance with exponential moving average
		lr := a.config.LearningRate
		for _, w := range words {
			cleaned := strings.Trim(w, ".,;:!?\"'()[]{}")
			if len(cleaned) > 2 {
				current := pattern.AvgImportance[cleaned]
				pattern.AvgImportance[cleaned] = current*(1-lr) + lineImportance*lr
			}
		}

		// Update position weights
		posIdx := int(float64(i) / float64(totalLines) * float64(len(pattern.PositionWeights)))
		if posIdx >= len(pattern.PositionWeights) {
			posIdx = len(pattern.PositionWeights) - 1
		}
		pattern.PositionWeights[posIdx] = pattern.PositionWeights[posIdx]*(1-lr) + lineImportance*lr
	}

	pattern.TotalSamples++
	pattern.ContentType = contentType
	a.patterns[contentType] = pattern
}
