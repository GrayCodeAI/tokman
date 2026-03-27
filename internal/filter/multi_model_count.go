package filter

import (
	"github.com/GrayCodeAI/tokman/internal/core"
)

// ModelFamily identifies a family of LLM models that share a tokenizer.
type ModelFamily string

const (
	ModelFamilyOpenAI    ModelFamily = "openai"    // cl100k_base (GPT-4, GPT-3.5, Claude compat)
	ModelFamilyAnthropic ModelFamily = "anthropic" // Claude family (uses same cl100k approximation)
	ModelFamilyLlama     ModelFamily = "llama"     // Llama 2/3 (sentencepiece, ~same as openai)
	ModelFamilyGemini    ModelFamily = "gemini"    // Google Gemini (≈ openai but slightly different)
	ModelFamilyMistral   ModelFamily = "mistral"   // Mistral family
)

// ModelTokenLimits maps model family to approximate context window size in tokens.
var ModelTokenLimits = map[ModelFamily]int{
	ModelFamilyOpenAI:    128_000,
	ModelFamilyAnthropic: 200_000,
	ModelFamilyLlama:     128_000,
	ModelFamilyGemini:    1_000_000,
	ModelFamilyMistral:   128_000,
}

// MultiModelTokenCount holds token counts for multiple model families.
type MultiModelTokenCount struct {
	Counts map[ModelFamily]int
	// InputLength is the original string length in bytes.
	InputLength int
}

// MultiModelCounter counts tokens for multiple model families simultaneously.
// Since all supported models use approximately the same BPE tokenizer
// (or a sentencepiece variant that is within 5–10% of cl100k), we use
// the core estimator as the base and apply per-family correction factors.
type MultiModelCounter struct{}

// Correction factors relative to cl100k_base.
// Positive values mean this model uses more tokens than cl100k.
var modelCorrectionFactor = map[ModelFamily]float64{
	ModelFamilyOpenAI:    1.00, // baseline
	ModelFamilyAnthropic: 1.02, // Claude uses slightly more tokens for code
	ModelFamilyLlama:     0.97, // Llama SentencePiece is slightly more compact
	ModelFamilyGemini:    0.95, // Gemini tokenizer is more efficient
	ModelFamilyMistral:   0.98, // Mistral is close to Llama
}

// NewMultiModelCounter creates a multi-model token counter.
func NewMultiModelCounter() *MultiModelCounter {
	return &MultiModelCounter{}
}

// Count returns token estimates for all supported model families.
func (c *MultiModelCounter) Count(text string) MultiModelTokenCount {
	base := core.EstimateTokens(text)
	counts := make(map[ModelFamily]int, len(modelCorrectionFactor))
	for family, factor := range modelCorrectionFactor {
		counts[family] = int(float64(base)*factor + 0.5)
	}
	return MultiModelTokenCount{
		Counts:      counts,
		InputLength: len(text),
	}
}

// FitsInContext returns true if the text fits within the given model family's
// default context window.
func (c *MultiModelCounter) FitsInContext(text string, family ModelFamily) bool {
	result := c.Count(text)
	limit, ok := ModelTokenLimits[family]
	if !ok {
		return true // unknown family: assume it fits
	}
	return result.Counts[family] <= limit
}

// SelectBestModel returns the cheapest model family whose context window
// can fit the given text.
func (c *MultiModelCounter) SelectBestModel(text string, candidates []ModelFamily) ModelFamily {
	result := c.Count(text)
	for _, family := range candidates {
		limit, ok := ModelTokenLimits[family]
		if !ok {
			continue
		}
		if result.Counts[family] <= limit {
			return family
		}
	}
	// Return the one with the largest window as fallback
	best := ModelFamilyGemini
	bestLimit := 0
	for _, f := range candidates {
		if lim, ok := ModelTokenLimits[f]; ok && lim > bestLimit {
			bestLimit = lim
			best = f
		}
	}
	return best
}

// UtilizationPct returns the percentage of a model family's context window used.
func (c *MultiModelCounter) UtilizationPct(text string, family ModelFamily) float64 {
	result := c.Count(text)
	limit, ok := ModelTokenLimits[family]
	if !ok || limit == 0 {
		return 0
	}
	return float64(result.Counts[family]) / float64(limit) * 100
}
