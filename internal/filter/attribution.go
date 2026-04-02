package filter

import (
	"math"
	"strings"
	"unicode"

	"github.com/GrayCodeAI/tokman/internal/simd"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

// Pre-compiled keywords for SIMD matching
var semanticKeywords = []string{
	"error", "fail", "success", "done", "complete", "warning",
	"file", "path", "line", "column", "function", "class",
	"import", "export", "return", "def", "var", "const",
	"true", "false", "null", "nil", "undefined",
	"http", "api", "url", "id", "key", "token",
}

// File extensions for fast matching
var fileExtensions = []string{
	".go:", ".rs:", ".py:", ".js:", ".ts:", ".java:", ".cpp:", ".c:",
	".json:", ".yaml:", ".yml:", ".toml:", ".md:", ".txt:",
}

// AttributionFilter implements attribution-based token pruning.
// Research basis: "ProCut: Progressive Pruning via Attribution" (LinkedIn, 2025)
// Achieves 78% token reduction by using importance scoring.
//
// Key technique: Attribution scores (simplified SHAP) identify which tokens
// contribute most to the output. Low-importance tokens are pruned.
//
// This is Layer 12 in the pipeline, adding ML-style importance without
// requiring actual model training.
type AttributionFilter struct {
	config AttributionConfig
}

// AttributionConfig holds configuration for attribution-based pruning
type AttributionConfig struct {
	// Enable attribution filtering
	Enabled bool

	// Threshold for token importance (0.0-1.0)
	// Tokens below this score are candidates for removal
	ImportanceThreshold float64

	// Minimum content length to apply attribution
	MinContentLength int

	// Use positional bias (later tokens often less important)
	PositionalBias bool

	// Use frequency-based importance (repeated tokens may be less important)
	FrequencyBias bool

	// Use semantic markers (preserve keywords, numbers, code)
	SemanticPreservation bool

	// Maximum tokens to analyze (for performance)
	MaxAnalyzeTokens int
}

// DefaultAttributionConfig returns default configuration
func DefaultAttributionConfig() AttributionConfig {
	return AttributionConfig{
		Enabled:              true,
		ImportanceThreshold:  0.25,
		MinContentLength:     50,
		PositionalBias:       true,
		FrequencyBias:        true,
		SemanticPreservation: true,
		MaxAnalyzeTokens:     10000,
	}
}

// NewAttributionFilter creates a new attribution filter
func NewAttributionFilter() *AttributionFilter {
	return &AttributionFilter{
		config: DefaultAttributionConfig(),
	}
}

// Name returns the filter name
func (a *AttributionFilter) Name() string {
	return "attribution"
}

// Apply applies attribution-based pruning to the input
func (a *AttributionFilter) Apply(input string, mode Mode) (string, int) {
	if !a.config.Enabled {
		return input, 0
	}

	// Skip short content
	if len(input) < a.config.MinContentLength {
		return input, 0
	}

	originalTokens := EstimateTokens(input)

	// Tokenize for analysis
	tokens := a.tokenize(input)
	if len(tokens) < 10 {
		return input, 0
	}

	// Calculate importance scores
	scores := a.calculateImportance(tokens, input)

	// Apply pruning based on mode
	threshold := a.config.ImportanceThreshold
	if mode == ModeAggressive {
		threshold += 0.1
	}

	// Build output with high-importance tokens
	// Always preserve whitespace to maintain structure
	var output strings.Builder
	var keptTokens int

	for i, token := range tokens {
		// Always preserve whitespace
		if strings.TrimSpace(token.text) == "" {
			output.WriteString(token.text)
			continue
		}

		score := scores[i]

		if score >= threshold || a.shouldPreserve(token.text) {
			output.WriteString(token.text)
			keptTokens++
		}
	}

	result := output.String()
	finalTokens := EstimateTokens(result)
	saved := originalTokens - finalTokens

	// Return original if result is empty or we didn't save much
	if len(result) == 0 || saved < 5 {
		return input, 0
	}

	return result, saved
}

// token represents a token with its text content
type token struct {
	text string
}

// tokenize splits content into tokens using SIMD-optimized scanning
// Avoids regex overhead for better P99 latency
func (a *AttributionFilter) tokenize(content string) []token {
	n := len(content)
	if n == 0 {
		return nil
	}

	var tokens []token
	start := 0
	inWhitespace := false

	// Fast byte scanning (no regex)
	for i := 0; i < n; i++ {
		c := content[i]
		isWhitespace := c == ' ' || c == '\t' || c == '\n' || c == '\r'

		if i == 0 {
			inWhitespace = isWhitespace
			continue
		}

		// State transition: whitespace <-> non-whitespace
		if isWhitespace != inWhitespace {
			// Extract token from start to i
			tokens = append(tokens, token{text: content[start:i]})
			start = i
			inWhitespace = isWhitespace
		}
	}

	// Add final token
	if start < n {
		tokens = append(tokens, token{text: content[start:]})
	}

	return tokens
}

// calculateImportance calculates importance scores for each token
// P2.3: Enhanced with GlobEnc-style attention-based salience scoring.
// Research Source: "FrugalPrompt: Reducing Contextual Overhead via Token Attribution" (Oct 2025)
// Uses attention contribution scoring inspired by GlobEnc and DecompX methods.
func (a *AttributionFilter) calculateImportance(tokens []token, content string) []float64 {
	n := len(tokens)
	scores := make([]float64, n)

	// Track token frequencies for frequency bias
	freq := make(map[string]int)
	for _, t := range tokens {
		freq[strings.ToLower(strings.TrimSpace(t.text))]++
	}

	// Build token connectivity matrix for GlobEnc-style attention simulation
	connectivity := a.computeTokenConnectivity(tokens)

	// Track positions for positional bias
	for i, t := range tokens {
		var score float64

		// 1. Positional importance (introduction and conclusion are important)
		if a.config.PositionalBias {
			pos := float64(i) / float64(n)
			// Higher importance at start and end (U-shaped)
			if pos < 0.2 {
				score += 0.3 * (1 - pos/0.2)
			} else if pos > 0.8 {
				score += 0.3 * (pos - 0.8) / 0.2
			}
			// Lost-in-the-middle: penalize middle positions
			if pos > 0.3 && pos < 0.7 {
				score -= 0.05
			}
		}

		// 2. Frequency-based importance (unique tokens are more important)
		if a.config.FrequencyBias {
			text := strings.ToLower(strings.TrimSpace(t.text))
			if freq[text] == 1 {
				score += 0.2 // Unique token
			} else if freq[text] > 3 {
				score -= 0.1 // Very common token
			}
		}

		// 3. Semantic importance
		if a.config.SemanticPreservation {
			score += a.semanticScore(t.text)
		}

		// 4. GlobEnc-style attention contribution (NEW - P2.3)
		// Tokens that are highly connected (attended to by many others) are important
		if connectivity[i] > 0 {
			score += connectivity[i] * 0.3
		}

		// 5. Length-based importance (very short tokens often less important)
		if len(strings.TrimSpace(t.text)) <= 2 && !isPunctuation(t.text) {
			score -= 0.1
		}

		// 6. DecompX-style decomposition: tokens in important regions get bonus
		if a.isInImportantRegion(tokens, i) {
			score += 0.15
		}

		// Ensure score is in [0, 1] range
		// Lower baseline so filler words can be pruned
		scores[i] = math.Max(0, math.Min(1, 0.2+score))
	}

	// Normalize scores
	if n > 0 {
		maxScore := 0.0
		for _, s := range scores {
			if s > maxScore {
				maxScore = s
			}
		}
		if maxScore > 0 {
			for i := range scores {
				scores[i] /= maxScore
			}
		}
	}

	return scores
}

// computeTokenConnectivity computes a simplified attention connectivity score.
// GlobEnc-inspired: tokens that are "hubs" (many tokens reference them) are important.
// Optimized: Reduced window size for P99 latency improvement
func (a *AttributionFilter) computeTokenConnectivity(tokens []token) []float64 {
	n := len(tokens)
	connectivity := make([]float64, n)

	if n < 3 {
		return connectivity
	}

	// Reduced window size from 5 to 3 for better P99 performance
	windowSize := 3
	for i := 0; i < n; i++ {
		tokenI := strings.ToLower(strings.TrimSpace(tokens[i].text))
		if len(tokenI) < 2 {
			continue
		}

		// Early exit for common filler words
		if tokenI == "the" || tokenI == "a" || tokenI == "an" || tokenI == "is" || tokenI == "are" {
			continue
		}

		for j := max(0, i-windowSize); j < min(n, i+windowSize+1); j++ {
			if i == j {
				continue
			}
			tokenJ := strings.ToLower(strings.TrimSpace(tokens[j].text))
			// Simple semantic relatedness: shared prefix or substring
			if len(tokenI) >= 3 && len(tokenJ) >= 3 {
				if strings.HasPrefix(tokenI, tokenJ[:3]) || strings.HasPrefix(tokenJ, tokenI[:3]) {
					connectivity[i] += 0.1
				}
			}
			// Structural co-occurrence: adjacent tokens in code
			if utils.Abs(i-j) == 1 && (isCodeSymbol(tokens[i].text) || isCodeSymbol(tokens[j].text)) {
				connectivity[i] += 0.15
			}
		}
	}

	// Normalize
	maxConn := 0.0
	for _, c := range connectivity {
		if c > maxConn {
			maxConn = c
		}
	}
	if maxConn > 0 {
		for i := range connectivity {
			connectivity[i] /= maxConn
		}
	}

	return connectivity
}

// isInImportantRegion checks if a token is in a semantically important region.
// DecompX-inspired: regions with high semantic density are more important.
func (a *AttributionFilter) isInImportantRegion(tokens []token, idx int) bool {
	// Check if surrounded by important tokens
	windowSize := 3
	importantCount := 0
	for j := max(0, idx-windowSize); j < min(len(tokens), idx+windowSize+1); j++ {
		if j == idx {
			continue
		}
		text := strings.TrimSpace(tokens[j].text)
		if isNumber(text) || isCodeSymbol(text) || isFilePath(text) || isURL(text) {
			importantCount++
		}
	}
	return importantCount >= 2
}

// semanticScore returns importance score for semantic content
// Optimized: Uses SIMD ContainsAny for keyword matching
func (a *AttributionFilter) semanticScore(text string) float64 {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return 0
	}

	var score float64

	// Preserve numbers
	if isNumber(text) {
		score += 0.4
	}

	// Preserve code symbols
	if isCodeSymbol(text) {
		score += 0.3
	}

	// SIMD-optimized keyword matching
	lower := strings.ToLower(text)
	if simd.ContainsAny(lower, semanticKeywords) {
		score += 0.3
	}

	// Preserve file paths - SIMD check for extensions
	if simd.ContainsAny(text, fileExtensions) || isFilePath(text) {
		score += 0.4
	}

	// Preserve URLs
	if isURL(text) {
		score += 0.4
	}

	// Preserve important punctuation
	if text == ":" || text == "=" || text == "->" || text == "=>" {
		score += 0.2
	}

	return score
}

// shouldPreserve returns true if token must be kept regardless of score
func (a *AttributionFilter) shouldPreserve(text string) bool {
	text = strings.TrimSpace(text)

	// Always preserve whitespace structure
	if text == "\n\n" {
		return true
	}

	// Preserve code blocks
	if strings.HasPrefix(text, "```") {
		return true
	}

	return false
}

// isNumber checks if text is a number
func isNumber(text string) bool {
	for _, c := range text {
		if !unicode.IsDigit(c) && c != '.' && c != '-' && c != '+' && c != 'e' && c != 'E' {
			return false
		}
	}
	return len(text) > 0
}

// codeSymbols are symbols used in programming languages
var codeSymbols = []string{
	"{", "}", "[", "]", "(", ")", ";", ",", ".", "->", "=>", "::",
	"==", "!=", "<", ">", "<=", ">=", "&&", "||", "++", "--",
}

// isCodeSymbol checks if text is a code symbol
func isCodeSymbol(text string) bool {
	for _, s := range codeSymbols {
		if text == s {
			return true
		}
	}
	return false
}

// isPunctuation checks if text is purely punctuation
func isPunctuation(text string) bool {
	for _, c := range text {
		if !unicode.IsPunct(c) && !unicode.IsSpace(c) {
			return false
		}
	}
	return len(text) > 0
}

// filePathExtensions are common file extensions for quick suffix matching
var filePathExtensions = []string{
	".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".rs",
	".rb", ".php", ".json", ".yaml", ".yml", ".toml", ".md", ".txt",
}

// isFilePath checks if text looks like a file path
func isFilePath(text string) bool {
	// Common file path patterns
	if strings.Contains(text, "/") && (strings.Contains(text, ".") || strings.Contains(text, "_")) {
		return true
	}
	if strings.Contains(text, "\\") && (strings.Contains(text, ".") || strings.Contains(text, "_")) {
		return true
	}
	for _, ext := range filePathExtensions {
		if strings.HasSuffix(text, ext) {
			return true
		}
	}
	return false
}

// isURL checks if text looks like a URL
func isURL(text string) bool {
	return strings.HasPrefix(text, "http://") ||
		strings.HasPrefix(text, "https://") ||
		strings.HasPrefix(text, "ftp://") ||
		strings.HasPrefix(text, "file://")
}

// SetEnabled enables or disables the filter
func (a *AttributionFilter) SetEnabled(enabled bool) {
	a.config.Enabled = enabled
}

// GetStats returns filter statistics
func (a *AttributionFilter) GetStats() map[string]any {
	return map[string]any{
		"enabled":    a.config.Enabled,
		"threshold":  a.config.ImportanceThreshold,
		"positional": a.config.PositionalBias,
		"frequency":  a.config.FrequencyBias,
		"semantic":   a.config.SemanticPreservation,
	}
}
