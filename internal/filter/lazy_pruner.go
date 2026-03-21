package filter

import (
	"math"
	"sort"
	"strings"
	"sync"
)

// LazyPrunerFilter implements Layer 18: Budget-aware Dynamic Pruning (LazyLLM style).
//
// Research Source: "LazyLLM: Dynamic Token Pruning" (July 2024)
// Key Innovation: Selective KV computation with layer-wise budget decay.
// Results: 2.34x speedup in prefill phase with maintained accuracy.
//
// Methodology:
// 1. Dynamic token selection based on attention scores
// 2. Layer-wise budget decay (deeper layers = smaller budgets)
// 3. Prune-and-Revive mechanism for recoverable pruning
// 4. Selective prefill to accelerate inference
type LazyPrunerFilter struct {
	config       LazyPrunerConfig
	layerBudgets []int           // Layer-specific token limits
	prunedTokens map[int][]Token // Storage for recoverable tokens
	mu           sync.RWMutex
}

// LazyPrunerConfig holds configuration for dynamic pruning
type LazyPrunerConfig struct {
	// BaseBudget is the initial token budget for layer 0
	BaseBudget int

	// DecayRate is the budget decay per layer (0.9 = 10% reduction)
	DecayRate float64

	// NumLayers is the number of layers to compute budgets for
	NumLayers int

	// RevivalBudget is max tokens to pull back from pruned pool
	RevivalBudget int

	// AttentionThreshold is the minimum score to keep a token
	AttentionThreshold float64

	// EnableRevival allows on-demand token recovery
	EnableRevival bool
}

// Token represents a scored token for pruning decisions
type Token struct {
	Text      string
	Score     float64
	Position  int
	Layer     int
	Pruned    bool
	Revived   bool
}

// LazyPrunerStats tracks pruning statistics
type LazyPrunerStats struct {
	TotalPruned   int
	TotalRevived  int
	TokensSaved   int
	LayersApplied int
}

// DefaultLazyPrunerConfig returns default configuration
func DefaultLazyPrunerConfig() LazyPrunerConfig {
	return LazyPrunerConfig{
		BaseBudget:         4000,
		DecayRate:          0.9,
		NumLayers:          10,
		RevivalBudget:      100,
		AttentionThreshold: 0.3,
		EnableRevival:      true,
	}
}

// NewLazyPrunerFilter creates a new lazy pruner filter
func NewLazyPrunerFilter() *LazyPrunerFilter {
	return NewLazyPrunerFilterWithConfig(DefaultLazyPrunerConfig())
}

// NewLazyPrunerFilterWithConfig creates a filter with custom config
func NewLazyPrunerFilterWithConfig(cfg LazyPrunerConfig) *LazyPrunerFilter {
	lp := &LazyPrunerFilter{
		config:       cfg,
		prunedTokens: make(map[int][]Token),
	}

	// Compute layer budgets with decay
	lp.layerBudgets = lp.computeLayerBudgets()

	return lp
}

// Name returns the filter name
func (f *LazyPrunerFilter) Name() string {
	return "lazy_pruner"
}

// computeLayerBudgets calculates budgets with exponential decay
func (f *LazyPrunerFilter) computeLayerBudgets() []int {
	budgets := make([]int, f.config.NumLayers)

	for i := 0; i < f.config.NumLayers; i++ {
		// Budget decay: budgets[i] = baseBudget * (decayRate)^i
		budgets[i] = int(float64(f.config.BaseBudget) * math.Pow(f.config.DecayRate, float64(i)))
	}

	return budgets
}

// Apply applies budget-aware dynamic pruning
func (f *LazyPrunerFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Tokenize input
	tokens := f.tokenize(input)
	if len(tokens) == 0 {
		return input, 0
	}

	// Score tokens
	scored := f.scoreTokens(tokens)

	// Apply layer-wise pruning
	output, saved := f.applyLayerWisePruning(scored, mode)

	return output, saved
}

// tokenize splits input into tokens
func (f *LazyPrunerFilter) tokenize(input string) []Token {
	words := strings.Fields(input)
	tokens := make([]Token, len(words))

	for i, word := range words {
		tokens[i] = Token{
			Text:     word,
			Position: i,
			Score:    0.5, // Base score
		}
	}

	return tokens
}

// scoreTokens assigns importance scores to tokens
func (f *LazyPrunerFilter) scoreTokens(tokens []Token) []Token {
	for i := range tokens {
		tokens[i].Score = f.calculateTokenScore(tokens[i])
	}
	return tokens
}

// calculateTokenScore calculates importance score for a token
func (f *LazyPrunerFilter) calculateTokenScore(token Token) float64 {
	score := token.Score // Start with base score

	lower := strings.ToLower(token.Text)

	// Structural importance
	structuralTokens := []string{"func", "function", "class", "struct", "return", "if", "else", "for", "while"}
	for _, st := range structuralTokens {
		if lower == st {
			score += 0.3
			break
		}
	}

	// Content importance
	contentIndicators := []string{"error", "main", "api", "handler", "config", "init"}
	for _, ci := range contentIndicators {
		if strings.Contains(lower, ci) {
			score += 0.2
			break
		}
	}

	// Position importance (beginning and end are often more important)
	position := token.Position
	totalTokens := 1000 // Approximation
	if position < 50 {
		score += 0.1 // Beginning boost
	} else if position > totalTokens-50 {
		score += 0.05 // End boost
	}

	// Normalize
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// applyLayerWisePruning applies pruning with layer-specific budgets
func (f *LazyPrunerFilter) applyLayerWisePruning(tokens []Token, mode Mode) (string, int) {
	// Sort tokens by score
	sortedTokens := make([]Token, len(tokens))
	copy(sortedTokens, tokens)
	sort.Slice(sortedTokens, func(i, j int) bool {
		return sortedTokens[i].Score > sortedTokens[j].Score
	})

	// Use first layer budget for initial pruning
	// In a real implementation, this would be called per-layer
	budget := f.layerBudgets[0]
	if mode == ModeAggressive {
		budget = int(float64(budget) * 0.7)
	}

	// Select top tokens within budget
	var keptTokens []Token
	var prunedTokens []Token

	for i, token := range sortedTokens {
		if i < budget {
			keptTokens = append(keptTokens, token)
		} else {
			token.Pruned = true
			prunedTokens = append(prunedTokens, token)
		}
	}

	// Store pruned tokens for potential revival
	f.mu.Lock()
	f.prunedTokens[0] = prunedTokens
	f.mu.Unlock()

	// Sort kept tokens by position to preserve order
	sort.Slice(keptTokens, func(i, j int) bool {
		return keptTokens[i].Position < keptTokens[j].Position
	})

	// Reconstruct output
	var result strings.Builder
	for _, token := range keptTokens {
		result.WriteString(token.Text)
		result.WriteString(" ")
	}

	saved := len(prunedTokens)

	return strings.TrimSpace(result.String()), saved
}

// SelectTokens selects tokens based on attention scores
func (f *LazyPrunerFilter) SelectTokens(tokens []Token, layer int, threshold float64) []Token {
	var selected []Token

	budget := f.layerBudgets[0]
	if layer < len(f.layerBudgets) {
		budget = f.layerBudgets[layer]
	}

	// Sort by score
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].Score > tokens[j].Score
	})

	for i, token := range tokens {
		if i < budget && token.Score >= threshold {
			selected = append(selected, token)
		}
	}

	return selected
}

// StorePruned stores pruned tokens for potential revival
func (f *LazyPrunerFilter) StorePruned(tokens []Token, layer int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.prunedTokens[layer] = tokens
}

// ReviveTokens recovers previously pruned tokens
func (f *LazyPrunerFilter) ReviveTokens(layer int, count int) []Token {
	f.mu.Lock()
	defer f.mu.Unlock()

	pruned, exists := f.prunedTokens[layer]
	if !exists || len(pruned) == 0 {
		return nil
	}

	// Sort by score to get most important pruned tokens
	sort.Slice(pruned, func(i, j int) bool {
		return pruned[i].Score > pruned[j].Score
	})

	// Revive up to count or revivalBudget
	maxRevive := count
	if maxRevive > f.config.RevivalBudget {
		maxRevive = f.config.RevivalBudget
	}
	if maxRevive > len(pruned) {
		maxRevive = len(pruned)
	}

	revived := make([]Token, maxRevive)
	for i := 0; i < maxRevive; i++ {
		pruned[i].Revived = true
		revived[i] = pruned[i]
	}

	// Update stored pruned tokens
	f.prunedTokens[layer] = pruned[maxRevive:]

	return revived
}

// GetLayerBudget returns the budget for a specific layer
func (f *LazyPrunerFilter) GetLayerBudget(layer int) int {
	if layer < len(f.layerBudgets) {
		return f.layerBudgets[layer]
	}
	// Return last budget for out-of-range layers
	return f.layerBudgets[len(f.layerBudgets)-1]
}

// GetStats returns pruning statistics
func (f *LazyPrunerFilter) GetStats() LazyPrunerStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := LazyPrunerStats{}
	for _, tokens := range f.prunedTokens {
		stats.TotalPruned += len(tokens)
		for _, t := range tokens {
			if t.Revived {
				stats.TotalRevived++
			}
		}
	}
	stats.TokensSaved = stats.TotalPruned - stats.TotalRevived
	stats.LayersApplied = len(f.layerBudgets)

	return stats
}

// GetLayerBudgets returns all layer budgets
func (f *LazyPrunerFilter) GetLayerBudgets() []int {
	return f.layerBudgets
}

// Clear clears the pruned token storage
func (f *LazyPrunerFilter) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.prunedTokens = make(map[int][]Token)
}
