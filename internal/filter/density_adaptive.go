package filter

import (
	"strings"
)

// DensityAdaptiveFilter implements DAST-style density-adaptive allocation.
// Research basis: "DAST: Context-Aware Compression via Dynamic Allocation of Soft Tokens"
// (Chen et al., 2025) - allocate compression capacity based on information density.
//
// T17: Key insight - dense content sections (code, data) need more tokens,
// while sparse sections (whitespace, repetition) can be heavily compressed.
//
// This filter:
// 1. Analyzes content density per section
// 2. Allocates budget proportionally to density
// 3. Applies non-uniform compression ratios
// 4. Preserves information-rich regions
type DensityAdaptiveFilter struct {
	config DensityAdaptiveConfig
}

// DensityAdaptiveConfig holds configuration for density-adaptive filtering
type DensityAdaptiveConfig struct {
	// Enable density-adaptive filtering
	Enabled bool

	// Target compression ratio (0.0-1.0, e.g., 0.3 = 30% of original)
	TargetRatio float64

	// Minimum density threshold for preservation (0.0-1.0)
	DensityThreshold float64

	// Window size for density calculation (in lines)
	WindowSize int

	// Boost factor for high-density regions
	DensityBoost float64

	// Penalty for low-density regions
	SparsePenalty float64
}

// DefaultDensityAdaptiveConfig returns default configuration
func DefaultDensityAdaptiveConfig() DensityAdaptiveConfig {
	return DensityAdaptiveConfig{
		Enabled:          true,
		TargetRatio:      0.4,  // Keep 40% of original
		DensityThreshold: 0.5,
		WindowSize:       10,   // lines
		DensityBoost:     1.3,
		SparsePenalty:    0.7,
	}
}

// NewDensityAdaptiveFilter creates a new density-adaptive filter
func NewDensityAdaptiveFilter() *DensityAdaptiveFilter {
	return &DensityAdaptiveFilter{
		config: DefaultDensityAdaptiveConfig(),
	}
}

// Name returns the filter name
func (d *DensityAdaptiveFilter) Name() string {
	return "density_adaptive"
}

// Apply applies density-adaptive compression to the input
func (d *DensityAdaptiveFilter) Apply(input string, mode Mode) (string, int) {
	if !d.config.Enabled {
		return input, 0
	}

	originalTokens := EstimateTokens(input)

	// Split into lines for density analysis
	lines := strings.Split(input, "\n")
	if len(lines) < d.config.WindowSize {
		return input, 0 // Too short for adaptive compression
	}

	// Calculate density for each line
	densities := make([]float64, len(lines))
	for i, line := range lines {
		densities[i] = d.calculateLineDensity(line)
	}

	// Calculate rolling average density
	windowDensities := d.calculateWindowDensities(densities)

	// Allocate budget per section based on density
	keep := d.allocateBudget(lines, windowDensities, originalTokens)

	// Build output
	var result strings.Builder
	for i, line := range lines {
		if keep[i] {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(line)
		}
	}

	output := result.String()
	finalTokens := EstimateTokens(output)
	saved := originalTokens - finalTokens

	return output, saved
}

// calculateLineDensity calculates information density of a line
func (d *DensityAdaptiveFilter) calculateLineDensity(line string) float64 {
	if len(line) == 0 {
		return 0.0
	}

	// Factors that indicate high density:
	// 1. Unique character ratio
	uniqueChars := make(map[rune]bool)
	for _, r := range line {
		uniqueChars[r] = true
	}
	uniqueRatio := float64(len(uniqueChars)) / float64(len(line))

	// 2. Non-whitespace ratio
	nonWhitespace := 0
	for _, r := range line {
		if r != ' ' && r != '\t' {
			nonWhitespace++
		}
	}
	contentRatio := float64(nonWhitespace) / float64(len(line))

	// 3. Symbol density (code-like)
	symbols := 0
	for _, r := range line {
		if isCodeSymbol(string(r)) || (r >= '!' && r <= '/') ||
			(r >= ':' && r <= '@') || (r >= '[' && r <= '`') {
			symbols++
		}
	}
	symbolRatio := float64(symbols) / float64(len(line))

	// 4. Number density
	numbers := 0
	for _, r := range line {
		if r >= '0' && r <= '9' {
			numbers++
		}
	}
	numberRatio := float64(numbers) / float64(len(line))

	// Combined density score
	density := uniqueRatio*0.3 + contentRatio*0.3 + symbolRatio*0.2 + numberRatio*0.2

	// Boost for code-like patterns
	if strings.Contains(line, "{") || strings.Contains(line, "}") ||
		strings.Contains(line, "(") || strings.Contains(line, ")") ||
		strings.Contains(line, ";") || strings.Contains(line, "=") {
		density *= 1.2
	}

	// Boost for important keywords
	keywords := []string{"error", "fail", "warning", "success", "import", "export",
		"function", "class", "def", "var", "const", "let", "return", "if", "else"}
	lineLower := strings.ToLower(line)
	for _, kw := range keywords {
		if strings.Contains(lineLower, kw) {
			density *= 1.15
			break
		}
	}

	if density > 1.0 {
		density = 1.0
	}

	return density
}

// calculateWindowDensities calculates rolling average density
func (d *DensityAdaptiveFilter) calculateWindowDensities(densities []float64) []float64 {
	n := len(densities)
	windowDensities := make([]float64, n)
	window := d.config.WindowSize

	for i := 0; i < n; i++ {
		start := i - window/2
		if start < 0 {
			start = 0
		}
		end := i + window/2 + 1
		if end > n {
			end = n
		}

		sum := 0.0
		for j := start; j < end; j++ {
			sum += densities[j]
		}
		windowDensities[i] = sum / float64(end-start)
	}

	return windowDensities
}

// allocateBudget decides which lines to keep based on density
func (d *DensityAdaptiveFilter) allocateBudget(lines []string, windowDensities []float64, totalTokens int) map[int]bool {
	n := len(lines)
	keep := make(map[int]bool)

	// Calculate target tokens to keep
	targetTokens := int(float64(totalTokens) * d.config.TargetRatio)

	// Always keep first and last few lines (structure)
	for i := 0; i < 3 && i < n; i++ {
		keep[i] = true
	}
	for i := n - 3; i < n; i++ {
		if i >= 0 {
			keep[i] = true
		}
	}

	// Score lines for priority (high density = high priority)
	type lineScore struct {
		index int
		score float64
	}
	scores := make([]lineScore, 0, n)

	for i := 3; i < n-3; i++ {
		// Base score is density
		score := windowDensities[i]

		// Apply boost/penalty
		if score >= d.config.DensityThreshold {
			score *= d.config.DensityBoost
		} else {
			score *= d.config.SparsePenalty
		}

		scores = append(scores, lineScore{index: i, score: score})
	}

	// Sort by score (descending) - simple selection sort for small arrays
	for i := 0; i < len(scores)-1; i++ {
		maxIdx := i
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[maxIdx].score {
				maxIdx = j
			}
		}
		scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
	}

	// Allocate budget - keep lines until we hit target
	currentTokens := 6 * 5 // Approximate tokens for first/last 3 lines each

	for _, s := range scores {
		if currentTokens >= targetTokens {
			break
		}

		lineTokens := EstimateTokens(lines[s.index])
		if currentTokens+lineTokens <= int(float64(targetTokens)*1.1) { // Allow 10% overflow
			keep[s.index] = true
			currentTokens += lineTokens
		}
	}

	return keep
}

// SetEnabled enables or disables the filter
func (d *DensityAdaptiveFilter) SetEnabled(enabled bool) {
	d.config.Enabled = enabled
}

// SetTargetRatio sets the target compression ratio
func (d *DensityAdaptiveFilter) SetTargetRatio(ratio float64) {
	d.config.TargetRatio = ratio
}

// GetStats returns filter statistics
func (d *DensityAdaptiveFilter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":           d.config.Enabled,
		"target_ratio":      d.config.TargetRatio,
		"density_threshold": d.config.DensityThreshold,
		"window_size":       d.config.WindowSize,
	}
}
