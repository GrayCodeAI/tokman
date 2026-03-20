package core

import "fmt"

// ModelPricing holds per-token pricing for a model.
type ModelPricing struct {
	Model            string
	InputPerMillion  float64 // Cost per 1M input tokens
	OutputPerMillion float64 // Cost per 1M output tokens
}

// CommonModelPricing provides pricing for popular models (as of 2025).
var CommonModelPricing = map[string]ModelPricing{
	"gpt-4o": {
		Model:            "gpt-4o",
		InputPerMillion:  2.50,
		OutputPerMillion: 10.00,
	},
	"gpt-4o-mini": {
		Model:            "gpt-4o-mini",
		InputPerMillion:  0.15,
		OutputPerMillion: 0.60,
	},
	"claude-3.5-sonnet": {
		Model:            "claude-3.5-sonnet",
		InputPerMillion:  3.00,
		OutputPerMillion: 15.00,
	},
	"claude-3-haiku": {
		Model:            "claude-3-haiku",
		InputPerMillion:  0.25,
		OutputPerMillion: 1.25,
	},
}

// CalculateSavings computes dollar savings from token reduction.
func CalculateSavings(tokensSaved int, model string) float64 {
	pricing, ok := CommonModelPricing[model]
	if !ok {
		// Default to gpt-4o-mini pricing
		pricing = CommonModelPricing["gpt-4o-mini"]
	}
	// Assume all saved tokens would have been input tokens
	return float64(tokensSaved) / 1_000_000 * pricing.InputPerMillion
}

// FormatSavings returns a human-readable savings string.
func FormatSavings(tokensSaved int, model string) string {
	savings := CalculateSavings(tokensSaved, model)
	if savings < 0.01 {
		return ">$0.01"
	}
	return fmt.Sprintf("$%.4f", savings)
}
