package filter

import (
	"strings"
	"sync"
)

// InterLayerFeedback implements cross-layer feedback mechanism.
// This allows later layers to signal earlier layers to adjust aggressiveness,
// creating an adaptive pipeline that self-corrects based on compression results.
type InterLayerFeedback struct {
	config  FeedbackConfig
	signals map[string]FeedbackSignal
	mu      sync.RWMutex
}

// FeedbackConfig holds configuration for inter-layer feedback
type FeedbackConfig struct {
	// Enabled controls whether feedback is active
	Enabled bool

	// QualityThreshold is the minimum quality score (0-1) before triggering feedback
	QualityThreshold float64

	// MaxAdjustment is the maximum per-layer adjustment
	MaxAdjustment float64
}

// FeedbackSignal carries compression quality feedback between layers
type FeedbackSignal struct {
	// LayerName is the source layer
	LayerName string

	// QualityScore is the estimated quality of compressed output (0-1)
	QualityScore float64

	// CompressionRatio is the achieved compression ratio
	CompressionRatio float64

	// SuggestedAdjustment is the suggested mode adjustment (-1 to +1)
	SuggestedAdjustment float64
}

// DefaultFeedbackConfig returns default configuration
func DefaultFeedbackConfig() FeedbackConfig {
	return FeedbackConfig{
		Enabled:          true,
		QualityThreshold: 0.7,
		MaxAdjustment:    0.3,
	}
}

// NewInterLayerFeedback creates a new feedback mechanism
func NewInterLayerFeedback() *InterLayerFeedback {
	return &InterLayerFeedback{
		config:  DefaultFeedbackConfig(),
		signals: make(map[string]FeedbackSignal),
	}
}

// RecordSignal records a feedback signal from a layer
func (f *InterLayerFeedback) RecordSignal(signal FeedbackSignal) {
	if !f.config.Enabled {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.signals[signal.LayerName] = signal
}

// GetAdjustment returns the suggested adjustment for a given layer
func (f *InterLayerFeedback) GetAdjustment(layerName string) float64 {
	if !f.config.Enabled {
		return 0
	}
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Aggregate feedback from all layers that have run
	totalAdjust := 0.0
	count := 0
	for _, signal := range f.signals {
		if signal.QualityScore < f.config.QualityThreshold {
			totalAdjust += signal.SuggestedAdjustment
			count++
		}
	}

	if count == 0 {
		return 0
	}

	avgAdjust := totalAdjust / float64(count)
	// Clamp to max adjustment
	if avgAdjust > f.config.MaxAdjustment {
		avgAdjust = f.config.MaxAdjustment
	} else if avgAdjust < -f.config.MaxAdjustment {
		avgAdjust = -f.config.MaxAdjustment
	}

	return avgAdjust
}

// Reset clears all feedback signals
func (f *InterLayerFeedback) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.signals = make(map[string]FeedbackSignal)
}

// QualityEstimator estimates the quality of compressed output
type QualityEstimator struct{}

// NewQualityEstimator creates a new quality estimator
func NewQualityEstimator() *QualityEstimator {
	return &QualityEstimator{}
}

// EstimateQuality estimates the quality of compressed output vs original
func (q *QualityEstimator) EstimateQuality(original, compressed string) float64 {
	if len(compressed) == 0 {
		return 0
	}
	if len(original) == 0 {
		return 1
	}

	// 1. Key term retention
	keyScore := q.keyTermRetention(original, compressed)

	// 2. Structural integrity
	structScore := q.structuralIntegrity(original, compressed)

	// 3. Information density
	densityScore := q.informationDensity(compressed)

	// 4. Readability
	readScore := q.readability(compressed)

	// Combined score
	return keyScore*0.3 + structScore*0.3 + densityScore*0.2 + readScore*0.2
}

// keyTermRetention checks if important terms are preserved
func (q *QualityEstimator) keyTermRetention(original, compressed string) float64 {
	origTerms := extractKeyTerms(original)
	if len(origTerms) == 0 {
		return 1.0
	}

	compressedLower := strings.ToLower(compressed)
	retained := 0
	for term := range origTerms {
		if strings.Contains(compressedLower, term) {
			retained++
		}
	}

	return float64(retained) / float64(len(origTerms))
}

// structuralIntegrity checks if output structure is preserved
func (q *QualityEstimator) structuralIntegrity(original, compressed string) float64 {
	// Check for preserved structural elements
	score := 0.5

	// Line breaks preserved
	origLines := strings.Count(original, "\n")
	compLines := strings.Count(compressed, "\n")
	if origLines > 0 {
		ratio := float64(compLines) / float64(origLines)
		if ratio > 0.3 && ratio < 2.0 {
			score += 0.25
		}
	}

	// Code blocks preserved
	if strings.Contains(original, "{") && strings.Contains(compressed, "{") {
		score += 0.25
	}

	return score
}

// informationDensity measures information per token
func (q *QualityEstimator) informationDensity(text string) float64 {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}

	// Count unique words
	unique := make(map[string]bool)
	for _, w := range words {
		unique[strings.ToLower(w)] = true
	}

	// Higher unique ratio = higher density
	return float64(len(unique)) / float64(len(words))
}

// readability estimates how readable the output is
func (q *QualityEstimator) readability(text string) float64 {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return 0
	}

	score := 0.5

	// Average line length
	totalLen := 0
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			totalLen += len(line)
			nonEmptyLines++
		}
	}

	if nonEmptyLines > 0 {
		avgLen := float64(totalLen) / float64(nonEmptyLines)
		// Optimal line length ~40-80 chars
		if avgLen >= 20 && avgLen <= 120 {
			score += 0.3
		}
	}

	// Empty lines provide structure
	emptyRatio := 1.0 - float64(nonEmptyLines)/float64(len(lines))
	if emptyRatio > 0 && emptyRatio < 0.3 {
		score += 0.2
	}

	return score
}

// extractKeyTerms extracts important terms from text
func extractKeyTerms(text string) map[string]bool {
	terms := make(map[string]bool)
	words := strings.Fields(strings.ToLower(text))

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "shall": true, "can": true, "to": true,
		"of": true, "in": true, "for": true, "on": true, "with": true,
		"at": true, "by": true, "from": true, "it": true, "this": true,
		"that": true, "and": true, "or": true, "not": true, "but": true,
	}

	for _, w := range words {
		// Clean word
		cleaned := strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(cleaned) > 3 && !stopWords[cleaned] {
			terms[cleaned] = true
		}
	}

	return terms
}
