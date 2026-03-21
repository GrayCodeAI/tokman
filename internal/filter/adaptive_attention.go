package filter

import (
	"math"
	"strings"
)

// AdaptiveAttentionFilter implements ADSC-style compression (Berkeley/Clemson, 2025).
// Attention-driven self-compression that mimics LLM attention patterns.
//
// Algorithm:
// 1. Simulate attention scoring for each token/line
// 2. Apply progressive downsampling based on attention scores
// 3. Use uniform downsampling for low-attention regions
// 4. Preserve high-attention regions in full detail
//
// Research Results: 66.7% - 88.9% compression with 98.2% quality preservation.
// Key insight: Attention patterns reveal token importance.
type AdaptiveAttentionFilter struct {
	// Attention simulation parameters
	attentionWindow int
	attentionDecay  float64

	// Downsampling parameters
	downsampleRatio float64
	minKeepRatio    float64

	// Threshold settings
	highAttnThreshold float64
	lowAttnThreshold  float64
}

// NewAdaptiveAttentionFilter creates a new adaptive attention filter
func NewAdaptiveAttentionFilter() *AdaptiveAttentionFilter {
	return &AdaptiveAttentionFilter{
		attentionWindow:   20,
		attentionDecay:    0.95,
		downsampleRatio:   0.5,
		minKeepRatio:      0.2,
		highAttnThreshold: 0.7,
		lowAttnThreshold:  0.3,
	}
}

// Name returns the filter name
func (f *AdaptiveAttentionFilter) Name() string {
	return "adaptive_attention"
}

// Apply applies adaptive attention compression
func (f *AdaptiveAttentionFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := len(input)

	// Calculate attention scores for each line
	lines := strings.Split(input, "\n")
	scores := f.calculateAttentionScores(lines)

	// Apply adaptive downsampling
	output := f.adaptiveDownsample(lines, scores, mode)

	saved := (original - len(output)) / 4
	return output, saved
}

// calculateAttentionScores simulates attention scoring for lines
func (f *AdaptiveAttentionFilter) calculateAttentionScores(lines []string) []float64 {
	scores := make([]float64, len(lines))

	for i := range lines {
		// Position-based attention (U-curve - higher at edges)
		posScore := f.positionAttention(i, len(lines))

		// Content-based attention
		contentScore := f.contentAttention(lines[i])

		// Context-based attention (influence of nearby high-attention lines)
		contextScore := f.contextAttention(lines, i)

		// Combine scores
		scores[i] = posScore*0.3 + contentScore*0.5 + contextScore*0.2
	}

	return scores
}

// positionAttention calculates position-based attention score
// Simulates the "lost in the middle" phenomenon
func (f *AdaptiveAttentionFilter) positionAttention(index, total int) float64 {
	if total <= 1 {
		return 1.0
	}

	// Normalize position
	pos := float64(index) / float64(total-1)

	// U-curve: higher at edges, lower in middle
	center := 0.5
	distance := math.Abs(pos - center)

	// Base score from U-curve
	score := distance * 2.0

	// Boost very beginning and end
	if pos < 0.1 {
		score = 0.9 + pos
	} else if pos > 0.9 {
		score = 1.9 - pos
	}

	return score
}

// contentAttention calculates content-based attention score
func (f *AdaptiveAttentionFilter) contentAttention(line string) float64 {
	score := 0.5 // Base score

	trimmed := strings.TrimSpace(line)

	// Empty lines get low attention
	if trimmed == "" {
		return 0.1
	}

	// Code content gets high attention
	if f.isCodeContent(trimmed) {
		score += 0.3
	}

	// Error/warning content gets very high attention
	if f.isErrorContent(trimmed) {
		score += 0.5
	}

	// Structural markers get high attention
	if f.isStructuralMarker(trimmed) {
		score += 0.2
	}

	// Length factor: longer lines may carry more information
	if len(trimmed) > 100 {
		score += 0.1
	}

	// Penalize repetitive content
	if f.isRepetitive(trimmed) {
		score -= 0.3
	}

	// Clamp to [0, 1]
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// contextAttention calculates context-based attention from nearby lines
func (f *AdaptiveAttentionFilter) contextAttention(lines []string, index int) float64 {
	if len(lines) == 0 {
		return 0.5
	}

	// Look at nearby lines
	windowStart := index - f.attentionWindow/2
	if windowStart < 0 {
		windowStart = 0
	}
	windowEnd := index + f.attentionWindow/2
	if windowEnd > len(lines) {
		windowEnd = len(lines)
	}

	// Calculate context score
	var nearbyScore float64
	var weight float64

	for i := windowStart; i < windowEnd; i++ {
		distance := math.Abs(float64(i - index))
		w := math.Pow(f.attentionDecay, distance)

		// Content attention of nearby line
		if i != index {
			nearbyScore += f.contentAttention(lines[i]) * w
			weight += w
		}
	}

	if weight > 0 {
		return nearbyScore / weight
	}
	return 0.5
}

// isCodeContent checks if line contains code
func (f *AdaptiveAttentionFilter) isCodeContent(line string) bool {
	// Code patterns
	if strings.HasPrefix(line, "func ") ||
		strings.HasPrefix(line, "def ") ||
		strings.HasPrefix(line, "class ") ||
		strings.HasPrefix(line, "if ") ||
		strings.HasPrefix(line, "for ") ||
		strings.HasPrefix(line, "return ") ||
		strings.Contains(line, " := ") ||
		strings.Contains(line, " = ") ||
		strings.Contains(line, "(") && strings.Contains(line, ")") {
		return true
	}
	return false
}

// isErrorContent checks if line contains errors
func (f *AdaptiveAttentionFilter) isErrorContent(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "error") ||
		strings.Contains(lower, "exception") ||
		strings.Contains(lower, "panic") ||
		strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "warning") ||
		strings.Contains(lower, "failed")
}

// isStructuralMarker checks if line is a structural marker
func (f *AdaptiveAttentionFilter) isStructuralMarker(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Markdown headings
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// File separators
	if strings.HasPrefix(trimmed, "---") ||
		strings.HasPrefix(trimmed, "===") ||
		strings.HasPrefix(trimmed, "```") {
		return true
	}

	return false
}

// isRepetitive checks if line is repetitive
func (f *AdaptiveAttentionFilter) isRepetitive(line string) bool {
	// Check for repeated characters
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 4 {
		return false
	}

	// Check if all characters are the same
	first := trimmed[0]
	for _, c := range trimmed {
		if byte(c) != first {
			return false
		}
	}
	return true
}

// adaptiveDownsample applies adaptive downsampling based on attention scores
func (f *AdaptiveAttentionFilter) adaptiveDownsample(lines []string, scores []float64, mode Mode) string {
	if len(lines) == 0 {
		return ""
	}

	var result []string
	lastKept := -10

	// Determine thresholds based on mode
	highThresh := f.highAttnThreshold
	lowThresh := f.lowAttnThreshold
	if mode == ModeAggressive {
		highThresh = 0.8
		lowThresh = 0.4
	}

	for i, score := range scores {
		keep := false

		if score >= highThresh {
			// High attention: always keep
			keep = true
		} else if score <= lowThresh {
			// Low attention: downsample
			if i-lastKept >= 3 {
				// Keep every 3rd low-attention line
				keep = true
			}
		} else {
			// Medium attention: probabilistic keep
			if float64(i-lastKept) >= 1.0/score {
				keep = true
			}
		}

		// Always keep structural lines
		if f.isStructuralMarker(lines[i]) {
			keep = true
		}

		// Enforce minimum keep ratio
		keptRatio := float64(len(result)) / float64(i+1)
		if keptRatio < f.minKeepRatio && i > 0 {
			keep = true
		}

		if keep {
			result = append(result, lines[i])
			lastKept = i
		}
	}

	return strings.Join(result, "\n")
}

// SetDownsampleRatio sets the downsampling ratio
func (f *AdaptiveAttentionFilter) SetDownsampleRatio(ratio float64) {
	f.downsampleRatio = ratio
}

// SetThresholds sets the attention thresholds
func (f *AdaptiveAttentionFilter) SetThresholds(high, low float64) {
	f.highAttnThreshold = high
	f.lowAttnThreshold = low
}
