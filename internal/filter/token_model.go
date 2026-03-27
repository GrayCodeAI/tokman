package filter

import (
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// tokenModelContextSizes maps well-known model IDs to context window sizes.
// This extends ModelTokenLimits (which maps families, not specific model IDs).
var tokenModelContextSizes = map[string]int{
	"gpt-4o":             128_000,
	"gpt-4o-mini":        128_000,
	"gpt-4-turbo":        128_000,
	"gpt-3.5-turbo":      16_384,
	"claude-opus-4.6":    200_000,
	"claude-sonnet-4.6":  200_000,
	"claude-haiku-4.5":   200_000,
	"claude-3.5-sonnet":  200_000,
	"claude-3-haiku":     200_000,
	"gemini-1.5-pro":     1_000_000,
	"gemini-1.5-flash":   1_000_000,
	"llama-3-70b":        128_000,
	"llama-3-8b":         8_000,
	"mistral-large":      128_000,
	"mistral-7b":         32_000,
}

// TokenModelAbstraction provides a unified token counting interface
// that can be switched between model families without changing call sites.
// This wraps the existing MultiModelCounter with per-model-ID convenience.
// Task #130: Token model abstraction layer.
type TokenModelAbstraction struct {
	Family           ModelFamily
	MaxContextTokens int
	counter          *MultiModelCounter
}

// NewTokenModelAbstraction creates an abstraction for the given model ID.
func NewTokenModelAbstraction(modelID string) *TokenModelAbstraction {
	family := detectTokenModelFamily(modelID)
	ctx, ok := tokenModelContextSizes[modelID]
	if !ok {
		if lim, ok2 := ModelTokenLimits[family]; ok2 {
			ctx = lim
		} else {
			ctx = 128_000
		}
	}
	return &TokenModelAbstraction{
		Family:           family,
		MaxContextTokens: ctx,
		counter:          NewMultiModelCounter(),
	}
}

// NewTokenModelAbstractionForFamily creates an abstraction for a model family.
func NewTokenModelAbstractionForFamily(family ModelFamily, contextTokens int) *TokenModelAbstraction {
	if contextTokens <= 0 {
		if lim, ok := ModelTokenLimits[family]; ok {
			contextTokens = lim
		} else {
			contextTokens = 128_000
		}
	}
	return &TokenModelAbstraction{
		Family:           family,
		MaxContextTokens: contextTokens,
		counter:          NewMultiModelCounter(),
	}
}

// Count returns the estimated token count adjusted for this model's family.
func (m *TokenModelAbstraction) Count(text string) int {
	result := m.counter.Count(text)
	if count, ok := result.Counts[m.Family]; ok {
		return count
	}
	return core.EstimateTokens(text)
}

// FitsInContext returns true if the text fits within the model's context window.
func (m *TokenModelAbstraction) FitsInContext(text string) bool {
	return m.Count(text) <= m.MaxContextTokens
}

// UtilizationPct returns what percentage of the context window the text uses.
func (m *TokenModelAbstraction) UtilizationPct(text string) float64 {
	if m.MaxContextTokens <= 0 {
		return 0
	}
	return float64(m.Count(text)) / float64(m.MaxContextTokens) * 100
}

// Budget returns remaining token budget after the given text.
func (m *TokenModelAbstraction) Budget(text string) int {
	remaining := m.MaxContextTokens - m.Count(text)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// detectTokenModelFamily identifies the model family from a model ID string.
func detectTokenModelFamily(modelID string) ModelFamily {
	lower := strings.ToLower(modelID)
	switch {
	case strings.Contains(lower, "gpt") || strings.Contains(lower, "davinci") || strings.Contains(lower, "text-"):
		return ModelFamilyOpenAI
	case strings.Contains(lower, "claude") || strings.Contains(lower, "anthropic"):
		return ModelFamilyAnthropic
	case strings.Contains(lower, "llama"):
		return ModelFamilyLlama
	case strings.Contains(lower, "gemini") || strings.Contains(lower, "bard") || strings.Contains(lower, "palm"):
		return ModelFamilyGemini
	case strings.Contains(lower, "mistral") || strings.Contains(lower, "mixtral"):
		return ModelFamilyMistral
	default:
		return ModelFamilyOpenAI
	}
}
