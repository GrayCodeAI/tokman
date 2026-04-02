package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/utils"
)

// SemanticAnchorFilter implements Layer 19: Semantic-Anchor Compression (SAC style).
//
// Research Source: "SAC: Semantic-Anchor Compression" (2024)
// Key Innovation: Autoencoding-free compression via anchor selection and aggregation.
// Results: Higher compression ratios without contextual amnesia.
//
// Methodology:
// 1. Anchor Detection - Identify high-connectivity tokens (semantic hubs)
// 2. Information Aggregation - Merge surrounding tokens into anchors
// 3. Prompt Reorganization - Restructure into anchor-based layout
type SemanticAnchorFilter struct {
	config        SemanticAnchorConfig
	anchorTokens  []AnchorToken
	anchorDensity map[string]float64 // Token -> density score
}

// SemanticAnchorConfig holds configuration for semantic anchor compression
type SemanticAnchorConfig struct {
	// AnchorRatio is the percentage of tokens to select as anchors (0.1 = 10%)
	AnchorRatio float64

	// MinAnchorSpacing is minimum tokens between anchors
	MinAnchorSpacing int

	// EnableAggregation allows non-anchor token aggregation
	EnableAggregation bool

	// PreserveStructure keeps structural tokens as anchors
	PreserveStructure bool
}

// AnchorToken represents a semantic anchor point
type AnchorToken struct {
	Text         string
	Position     int
	Score        float64
	Aggregated   []string // Tokens aggregated into this anchor
	IsStructural bool
}

// SemanticAnchorStats tracks compression statistics
type SemanticAnchorStats struct {
	TotalAnchors    int
	TotalAggregated int
	NonAnchorPruned int
	TokensSaved     int
}

// Anchor patterns - high-connectivity tokens that serve as semantic hubs
var (
	// Named entities and identifiers
	identifierPattern = regexp.MustCompile(`[A-Z][a-zA-Z0-9_]*`)

	// Code symbols (functions, classes, etc.)
	codeSymbolPattern = regexp.MustCompile(`(func|class|def|fn|struct|interface|type|const|var|let)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

	// Important keywords
	importantKeywords = map[string]bool{
		"main": true, "error": true, "return": true, "if": true,
		"else": true, "for": true, "while": true, "func": true,
		"class": true, "struct": true, "interface": true, "type": true,
		"import": true, "export": true, "public": true, "private": true,
		"api": true, "handler": true, "config": true, "init": true,
	}

	// Structural markers that should always be anchors
	structuralMarkers = map[string]bool{
		"{": true, "}": true, "(": true, ")": true,
		"[": true, "]": true, ";": true, ":": true,
	}
)

// DefaultSemanticAnchorConfig returns default configuration
func DefaultSemanticAnchorConfig() SemanticAnchorConfig {
	return SemanticAnchorConfig{
		AnchorRatio:       0.15, // 15% of tokens as anchors
		MinAnchorSpacing:  3,
		EnableAggregation: true,
		PreserveStructure: true,
	}
}

// NewSemanticAnchorFilter creates a new semantic anchor filter
func NewSemanticAnchorFilter() *SemanticAnchorFilter {
	return NewSemanticAnchorFilterWithConfig(DefaultSemanticAnchorConfig())
}

// NewSemanticAnchorFilterWithConfig creates a filter with custom config
func NewSemanticAnchorFilterWithConfig(cfg SemanticAnchorConfig) *SemanticAnchorFilter {
	return &SemanticAnchorFilter{
		config:        cfg,
		anchorTokens:  make([]AnchorToken, 0),
		anchorDensity: make(map[string]float64),
	}
}

// Name returns the filter name
func (f *SemanticAnchorFilter) Name() string {
	return "semantic_anchor"
}

// Apply applies semantic-anchor compression
func (f *SemanticAnchorFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Tokenize input
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return input, 0
	}

	// Step 1: Detect anchors
	f.detectAnchors(tokens)

	// Step 2: Aggregate non-anchor tokens
	aggregated := f.aggregateTokens(tokens)

	// Step 3: Reconstruct with anchors only
	output, saved := f.reconstructWithAnchors(tokens, aggregated, mode)

	return output, saved
}

// detectAnchors identifies high-connectivity tokens as semantic anchors
func (f *SemanticAnchorFilter) detectAnchors(tokens []string) {
	f.anchorTokens = make([]AnchorToken, 0)
	f.anchorDensity = make(map[string]float64)

	// Calculate anchor density for each token
	for i, token := range tokens {
		density := f.calculateAnchorDensity(token, tokens, i)
		f.anchorDensity[token] = density
	}

	// Sort tokens by density (descending) and select top anchors
	type tokenScore struct {
		token string
		score float64
		pos   int
	}

	scores := make([]tokenScore, len(tokens))
	for i, token := range tokens {
		scores[i] = tokenScore{
			token: token,
			score: f.anchorDensity[token],
			pos:   i,
		}
	}

	// Simple sort (descending by score)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Select anchors with minimum spacing
	anchorCount := int(float64(len(tokens)) * f.config.AnchorRatio)
	if anchorCount < 3 {
		anchorCount = 3
	}

	selectedPositions := make(map[int]bool)

	for _, s := range scores {
		if len(f.anchorTokens) >= anchorCount {
			break
		}

		// Check spacing constraint
		tooClose := false
		for pos := range selectedPositions {
			if utils.Abs(s.pos-pos) < f.config.MinAnchorSpacing {
				tooClose = true
				break
			}
		}

		// Always include structural markers
		if structuralMarkers[s.token] && f.config.PreserveStructure {
			f.anchorTokens = append(f.anchorTokens, AnchorToken{
				Text:         s.token,
				Position:     s.pos,
				Score:        s.score,
				IsStructural: true,
			})
			selectedPositions[s.pos] = true
			continue
		}

		if !tooClose {
			f.anchorTokens = append(f.anchorTokens, AnchorToken{
				Text:     s.token,
				Position: s.pos,
				Score:    s.score,
			})
			selectedPositions[s.pos] = true
		}
	}
}

// calculateAnchorDensity calculates the anchor density score for a token
func (f *SemanticAnchorFilter) calculateAnchorDensity(token string, tokens []string, pos int) float64 {
	score := 0.5 // Base score

	lower := strings.ToLower(token)

	// Important keywords boost score significantly
	if importantKeywords[lower] {
		score += 0.4
	}

	// Identifier patterns (CamelCase, etc.)
	if identifierPattern.MatchString(token) {
		score += 0.2
	}

	// Code symbols
	if codeSymbolPattern.MatchString(token) {
		score += 0.3
	}

	// Structural markers
	if structuralMarkers[token] {
		score += 0.1
	}

	// Position importance (beginning and end of context)
	total := len(tokens)
	if pos < 20 {
		score += 0.15 // Beginning boost
	} else if pos > total-20 {
		score += 0.1 // End boost
	}

	// Connectivity - count how often this token appears
	occurrences := 0
	for _, t := range tokens {
		if strings.EqualFold(t, token) {
			occurrences++
		}
	}
	if occurrences > 1 {
		score += float64(occurrences) * 0.05
	}

	// Normalize
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// aggregateTokens aggregates non-anchor tokens into nearest anchors
func (f *SemanticAnchorFilter) aggregateTokens(tokens []string) map[int][]string {
	aggregated := make(map[int][]string)

	// Create position map for anchors
	anchorPositions := make(map[int]bool)
	for _, a := range f.anchorTokens {
		anchorPositions[a.Position] = true
	}

	// Aggregate non-anchor tokens to nearest anchor
	for i, token := range tokens {
		if anchorPositions[i] {
			continue // Skip anchors
		}

		// Find nearest anchor
		nearestAnchor := f.findNearestAnchor(i)
		if nearestAnchor != -1 {
			aggregated[nearestAnchor] = append(aggregated[nearestAnchor], token)
		}
	}

	return aggregated
}

// findNearestAnchor finds the nearest anchor position
func (f *SemanticAnchorFilter) findNearestAnchor(pos int) int {
	minDist := int(1<<30 - 1)
	nearest := -1

	for _, a := range f.anchorTokens {
		dist := utils.Abs(a.Position - pos)
		if dist < minDist {
			minDist = dist
			nearest = a.Position
		}
	}

	return nearest
}

// reconstructWithAnchors reconstructs output using only anchors
func (f *SemanticAnchorFilter) reconstructWithAnchors(tokens []string, aggregated map[int][]string, mode Mode) (string, int) {
	// Safety: if no anchors detected, return original input
	if len(f.anchorTokens) == 0 {
		return strings.Join(tokens, " "), 0
	}

	// Sort anchor positions
	positions := make([]int, len(f.anchorTokens))
	for i, a := range f.anchorTokens {
		positions[i] = a.Position
	}

	// Simple sort ascending
	for i := 0; i < len(positions); i++ {
		for j := i + 1; j < len(positions); j++ {
			if positions[j] < positions[i] {
				positions[i], positions[j] = positions[j], positions[i]
			}
		}
	}

	// Build output with anchors
	var result strings.Builder
	saved := len(tokens) - len(f.anchorTokens)

	for i, pos := range positions {
		// Add anchor token
		result.WriteString(tokens[pos])
		result.WriteString(" ")

		// Optionally add aggregated hint (for context preservation)
		if f.config.EnableAggregation && mode != ModeAggressive {
			if agg, exists := aggregated[pos]; exists && len(agg) > 0 {
				// Add a brief hint of aggregated content
				// This helps preserve semantic context
				hintCount := min(2, len(agg))
				for j := 0; j < hintCount; j++ {
					result.WriteString(agg[j])
					result.WriteString(" ")
				}
				saved -= hintCount // Count these as not saved
			}
		}

		// Add spacing between anchors
		if i < len(positions)-1 {
			result.WriteString(" ")
		}
	}

	return strings.TrimSpace(result.String()), saved
}


// GetAnchors returns all detected anchor tokens
func (f *SemanticAnchorFilter) GetAnchors() []AnchorToken {
	return f.anchorTokens
}

// GetAnchorDensity returns the density score for a token
func (f *SemanticAnchorFilter) GetAnchorDensity(token string) float64 {
	return f.anchorDensity[token]
}

// GetStats returns compression statistics
func (f *SemanticAnchorFilter) GetStats() SemanticAnchorStats {
	stats := SemanticAnchorStats{
		TotalAnchors: len(f.anchorTokens),
	}

	for _, a := range f.anchorTokens {
		stats.TotalAggregated += len(a.Aggregated)
	}

	return stats
}
