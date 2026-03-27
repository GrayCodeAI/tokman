package filter

import (
	"fmt"
	"strings"
)

// ModelPrice holds per-token pricing for an LLM model.
type ModelPrice struct {
	Name            string
	InputPerMillion float64 // $ per 1M input tokens
}

// DefaultModelPrices is the built-in pricing table (approximate, as of 2026-Q1).
// Prices are per 1M input tokens.
var DefaultModelPrices = []ModelPrice{
	{"claude-opus-4.6", 15.00},
	{"claude-sonnet-4.6", 3.00},
	{"claude-haiku-4.5", 0.80},
	{"gpt-4o", 2.50},
	{"gpt-4o-mini", 0.15},
	{"gemini-1.5-pro", 3.50},
	{"gemini-1.5-flash", 0.075},
	{"llama-3-70b", 0.59},
	{"mistral-large", 2.00},
}

// CostReport summarizes the API cost savings from a compression operation.
type CostReport struct {
	OriginalTokens   int
	CompressedTokens int
	TokensSaved      int
	ReductionPct     float64
	ByModel          []ModelCostLine
}

// ModelCostLine holds the savings for a single model.
type ModelCostLine struct {
	Model         string
	OriginalCost  float64 // USD
	CompressedCost float64 // USD
	Saved         float64 // USD
}

// GenerateCostReport builds a cost report for the given original/compressed token counts.
// If prices is nil, DefaultModelPrices is used.
func GenerateCostReport(originalTokens, compressedTokens int, prices []ModelPrice) CostReport {
	if prices == nil {
		prices = DefaultModelPrices
	}

	tokensSaved := originalTokens - compressedTokens
	reductionPct := 0.0
	if originalTokens > 0 {
		reductionPct = float64(tokensSaved) / float64(originalTokens) * 100
	}

	byModel := make([]ModelCostLine, 0, len(prices))
	for _, p := range prices {
		origCost := float64(originalTokens) / 1_000_000 * p.InputPerMillion
		compCost := float64(compressedTokens) / 1_000_000 * p.InputPerMillion
		byModel = append(byModel, ModelCostLine{
			Model:          p.Name,
			OriginalCost:   origCost,
			CompressedCost: compCost,
			Saved:          origCost - compCost,
		})
	}

	return CostReport{
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		TokensSaved:      tokensSaved,
		ReductionPct:     reductionPct,
		ByModel:          byModel,
	}
}

// FormatReport returns a human-readable cost savings table.
func (r CostReport) FormatReport() string {
	var sb strings.Builder
	sb.WriteString("=== Token Budget & Cost Report ===\n\n")
	sb.WriteString(fmt.Sprintf("Tokens: %d → %d (saved %d, %.1f%%)\n\n",
		r.OriginalTokens, r.CompressedTokens, r.TokensSaved, r.ReductionPct))

	sb.WriteString(fmt.Sprintf("%-22s  %10s  %10s  %10s\n",
		"Model", "Original($)", "Compressed($)", "Saved($)"))
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	for _, line := range r.ByModel {
		sb.WriteString(fmt.Sprintf("%-22s  %10.6f  %13.6f  %10.6f\n",
			line.Model, line.OriginalCost, line.CompressedCost, line.Saved))
	}

	return sb.String()
}

// TotalSaved returns the sum of savings across all models.
func (r CostReport) TotalSaved() float64 {
	total := 0.0
	for _, l := range r.ByModel {
		total += l.Saved
	}
	return total
}
