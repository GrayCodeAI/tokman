package filter

import (
	"math"
	"strings"
)

// PhaseTransitionDetector implements critical compression threshold detection.
// Research: "Phase transitions in LLM compression" (Nature 2026)
// Key Insight: LLMs have critical compression thresholds. Below a threshold,
// performance collapses catastrophically. Above it, you can compress 4-5x
// with 90% performance preserved.
//
// This detects when compression is about to cross a quality cliff and backs off.
// It monitors: entropy loss, structure preservation, key-term retention.
// If any metric drops below threshold, compression stops.
type PhaseTransitionDetector struct {
	config PhaseConfig
}

// PhaseConfig holds configuration for phase transition detection
type PhaseConfig struct {
	Enabled            bool
	EntropyThreshold   float64 // Min entropy ratio (compressed/original)
	StructureThreshold float64 // Min structure preservation ratio
	KeyTermThreshold   float64 // Min key-term retention ratio
	MinContentLength   int
	BackoffRatio       float64 // How much to reduce compression when threshold hit
}

// DefaultPhaseConfig returns default configuration
func DefaultPhaseConfig() PhaseConfig {
	return PhaseConfig{
		Enabled:            true,
		EntropyThreshold:   0.4, // Below 40% entropy = quality collapse
		StructureThreshold: 0.3, // Below 30% structure = broken output
		KeyTermThreshold:   0.5, // Below 50% key terms = information loss
		MinContentLength:   50,
		BackoffRatio:       0.7, // Reduce compression by 30%
	}
}

// NewPhaseTransitionDetector creates a new phase transition detector
func NewPhaseTransitionDetector() *PhaseTransitionDetector {
	return &PhaseTransitionDetector{config: DefaultPhaseConfig()}
}

// Name returns the filter name
func (p *PhaseTransitionDetector) Name() string { return "phase_transition" }

// PhaseQualityMetrics holds quality measurements
type PhaseQualityMetrics struct {
	EntropyRatio   float64
	StructureRatio float64
	KeyTermRatio   float64
	OverallQuality float64
	IsCollapsed    bool
}

// CheckQuality measures quality of compressed output vs original
func (p *PhaseTransitionDetector) CheckQuality(original, compressed string) PhaseQualityMetrics {
	if len(original) == 0 || len(compressed) == 0 {
		return PhaseQualityMetrics{IsCollapsed: true}
	}

	metrics := PhaseQualityMetrics{}

	// 1. Entropy ratio
	metrics.EntropyRatio = p.entropyRatio(original, compressed)

	// 2. Structure preservation
	metrics.StructureRatio = p.structureRatio(original, compressed)

	// 3. Key-term retention
	metrics.KeyTermRatio = p.keyTermRatio(original, compressed)

	// Overall quality (weighted average)
	metrics.OverallQuality = metrics.EntropyRatio*0.3 +
		metrics.StructureRatio*0.3 +
		metrics.KeyTermRatio*0.4

	// Check for phase transition (quality collapse)
	metrics.IsCollapsed = metrics.EntropyRatio < p.config.EntropyThreshold ||
		metrics.StructureRatio < p.config.StructureThreshold ||
		metrics.KeyTermRatio < p.config.KeyTermThreshold

	return metrics
}

// ShouldBackoff checks if compression should be reduced
func (p *PhaseTransitionDetector) ShouldBackoff(original, compressed string) bool {
	if !p.config.Enabled {
		return false
	}
	metrics := p.CheckQuality(original, compressed)
	return metrics.IsCollapsed
}

// Apply applies phase transition detection
func (p *PhaseTransitionDetector) Apply(input string, mode Mode) (string, int) {
	// Phase detector doesn't compress - it monitors
	return input, 0
}

// entropyRatio computes entropy(compressed) / entropy(original)
func (p *PhaseTransitionDetector) entropyRatio(original, compressed string) float64 {
	origEntropy := p.computeEntropy(original)
	compEntropy := p.computeEntropy(compressed)

	if origEntropy == 0 {
		return 1.0
	}
	return compEntropy / origEntropy
}

// computeEntropy computes Shannon entropy of text
func (p *PhaseTransitionDetector) computeEntropy(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	freq := make(map[byte]int)
	for i := 0; i < len(text); i++ {
		freq[text[i]]++
	}

	entropy := 0.0
	total := float64(len(text))
	for _, count := range freq {
		p := float64(count) / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// structureRatio computes structure preservation ratio
func (p *PhaseTransitionDetector) structureRatio(original, compressed string) float64 {
	origLines := strings.Count(original, "\n")
	compLines := strings.Count(compressed, "\n")

	if origLines == 0 {
		return 1.0
	}

	lineRatio := float64(compLines) / float64(origLines)

	// Check for preserved structural elements
	structScore := 0.0
	origHasBraces := strings.Contains(original, "{") && strings.Contains(original, "}")
	compHasBraces := strings.Contains(compressed, "{") && strings.Contains(compressed, "}")
	if origHasBraces && compHasBraces {
		structScore += 0.3
	}

	origHasFunc := strings.Contains(original, "func ") || strings.Contains(original, "function ")
	compHasFunc := strings.Contains(compressed, "func ") || strings.Contains(compressed, "function ")
	if origHasFunc && compHasFunc {
		structScore += 0.3
	}

	origHasImport := strings.Contains(original, "import ") || strings.Contains(original, "package ")
	compHasImport := strings.Contains(compressed, "import ") || strings.Contains(compressed, "package ")
	if origHasImport && compHasImport {
		structScore += 0.2
	}

	return lineRatio*0.2 + structScore*0.8
}

// keyTermRatio computes key-term retention ratio
func (p *PhaseTransitionDetector) keyTermRatio(original, compressed string) float64 {
	origTerms := p.extractKeyTerms(original)
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

// extractKeyTerms extracts important terms from text
func (p *PhaseTransitionDetector) extractKeyTerms(text string) map[string]bool {
	terms := make(map[string]bool)
	words := strings.Fields(strings.ToLower(text))

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "it": true,
		"and": true, "or": true, "not": true, "but": true,
	}

	for _, w := range words {
		cleaned := strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(cleaned) > 3 && !stopWords[cleaned] {
			terms[cleaned] = true
		}
	}
	return terms
}
