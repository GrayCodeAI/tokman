package economics

import (
	"testing"

	"github.com/GrayCodeAI/tokman/internal/utils"
)

func TestWeightConstants(t *testing.T) {
	// Verify API pricing ratios are reasonable
	if WeightOutput != 5.0 {
		t.Errorf("WeightOutput = %f, want 5.0", WeightOutput)
	}
	if WeightCacheCreate != 1.25 {
		t.Errorf("WeightCacheCreate = %f, want 1.25", WeightCacheCreate)
	}
	if WeightCacheRead != 0.1 {
		t.Errorf("WeightCacheRead = %f, want 0.1", WeightCacheRead)
	}
}

func TestPeriodEconomics_ComputeWeightedMetrics(t *testing.T) {
	p := &PeriodEconomics{}

	// Test with nil values (should not panic)
	p.computeWeightedMetrics()

	// Test with valid values
	cost := 10.0
	saved := 1000
	input := uint64(1000)
	output := uint64(500)
	cacheCreate := uint64(100)
	cacheRead := uint64(200)

	p.CCCost = &cost
	p.TMSavedTokens = &saved
	p.CCInputTokens = &input
	p.CCOutputTokens = &output
	p.CCCacheCreateTokens = &cacheCreate
	p.CCCacheReadTokens = &cacheRead

	p.computeWeightedMetrics()

	if p.WeightedInputCPT == nil {
		t.Error("WeightedInputCPT should not be nil after computeWeightedMetrics")
	}
	if p.SavingsWeighted == nil {
		t.Error("SavingsWeighted should not be nil after computeWeightedMetrics")
	}
}

func TestPeriodEconomics_ComputeDualMetrics(t *testing.T) {
	p := &PeriodEconomics{}

	// Test with nil values (should not panic)
	p.computeDualMetrics()

	// Test with valid values
	cost := 10.0
	saved := 1000
	total := uint64(5000)
	active := uint64(3000)

	p.CCCost = &cost
	p.TMSavedTokens = &saved
	p.CCTotalTokens = &total
	p.CCActiveTokens = &active

	p.computeDualMetrics()

	if p.BlendedCPT == nil {
		t.Error("BlendedCPT should not be nil after computeDualMetrics")
	}
	if p.ActiveCPT == nil {
		t.Error("ActiveCPT should not be nil after computeDualMetrics")
	}
}

func TestComputeTotals(t *testing.T) {
	cost := 10.0
	saved := 1000
	cmds := 50
	tokens := uint64(5000)
	input := uint64(2000)
	output := uint64(1000)
	cacheCreate := uint64(500)
	cacheRead := uint64(1500)
	pct := 50.0

	periods := []PeriodEconomics{
		{
			CCCost:              &cost,
			CCTotalTokens:       &tokens,
			CCActiveTokens:      &tokens,
			CCInputTokens:       &input,
			CCOutputTokens:      &output,
			CCCacheCreateTokens: &cacheCreate,
			CCCacheReadTokens:   &cacheRead,
			TMCommands:          &cmds,
			TMSavedTokens:       &saved,
			TMSavingsPct:        &pct,
		},
	}

	totals := computeTotals(periods)

	if totals.CCCost != cost {
		t.Errorf("CCCost = %f, want %f", totals.CCCost, cost)
	}
	if totals.TMSavedTokens != saved {
		t.Errorf("TMSavedTokens = %d, want %d", totals.TMSavedTokens, saved)
	}
	if totals.TMCommands != cmds {
		t.Errorf("TMCommands = %d, want %d", totals.TMCommands, cmds)
	}
}

func TestFormatUSD(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		expected string
	}{
		{"zero", 0, "$0.00"},
		{"small", 0.01, "$0.01"},
		{"medium", 1.5, "$1.50"},
		{"large", 100.0, "$100.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUSD(tt.amount)
			if result != tt.expected {
				t.Errorf("formatUSD(%.2f) = %q, want %q", tt.amount, result, tt.expected)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name     string
		count    uint64
		expected string
	}{
		{"zero", 0, "0"},
		{"hundreds", 500, "500"},
		{"thousands", 5000, "5.0K"},
		{"millions", 5000000, "5.0M"},
		{"billions", 5000000000, "5.0B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatTokens64(tt.count)
			if result != tt.expected {
				t.Errorf("formatTokens(%d) = %q, want %q", tt.count, result, tt.expected)
			}
		})
	}
}
