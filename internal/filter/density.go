package filter

import (
	"fmt"
	"strings"
)

// DensityAdaptiveAllocator allocates compression budget based on content density.
// R9: DAST (Chen et al., 2025) — more budget for dense content sections.
type DensityAdaptiveAllocator struct{}

// newDensityAdaptiveAllocator creates a density-aware allocator.
func newDensityAdaptiveAllocator() *DensityAdaptiveAllocator {
	return &DensityAdaptiveAllocator{}
}

// AllocateBudget distributes token budget across sections based on density.
func (a *DensityAdaptiveAllocator) AllocateBudget(input string, totalBudget int) []SectionBudget {
	sections := splitSections(input)
	if len(sections) == 0 {
		return nil
	}

	// Compute density for each section
	densities := make([]float64, len(sections))
	totalDensity := 0.0
	for i, sec := range sections {
		densities[i] = computeDensity(sec)
		totalDensity += densities[i]
	}

	// Allocate budget proportionally to density
	budgets := make([]SectionBudget, len(sections))
	for i, sec := range sections {
		ratio := densities[i] / totalDensity
		if totalDensity == 0 {
			ratio = 1.0 / float64(len(sections))
		}
		budget := int(float64(totalBudget) * ratio)
		if budget < 10 {
			budget = 10 // Minimum budget per section
		}
		budgets[i] = SectionBudget{
			Section: sec,
			Budget:  budget,
			Density: densities[i],
		}
	}

	return budgets
}

// SectionBudget holds a section with its allocated budget.
type SectionBudget struct {
	Section string
	Budget  int
	Density float64
}

// computeDensity estimates information density (higher = more info per char).
func computeDensity(section string) float64 {
	if len(section) == 0 {
		return 0
	}

	lines := strings.Split(section, "\n")
	uniqueLines := make(map[string]bool)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 {
			uniqueLines[trimmed] = true
		}
	}

	// Density = unique lines / total lines (higher = less redundant)
	if len(lines) == 0 {
		return 0
	}
	return float64(len(uniqueLines)) / float64(len(lines))
}

// splitSections splits input into logical sections.
func splitSections(input string) []string {
	lines := strings.Split(input, "\n")
	var sections []string
	var current []string

	for _, line := range lines {
		// Section boundary: empty lines or markers
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(current) > 3 {
			sections = append(sections, strings.Join(current, "\n"))
			current = nil
		} else {
			current = append(current, line)
		}
	}

	if len(current) > 0 {
		sections = append(sections, strings.Join(current, "\n"))
	}

	// If no sections found, treat entire input as one section
	if len(sections) == 0 && len(input) > 0 {
		sections = append(sections, input)
	}

	return sections
}

// TieredCompaction implements hot/warm/cold memory tiers.
// R10: MemGPT insight — tiered memory for better compaction.
type TieredCompaction struct {
	HotMaxTokens  int // Recent, always accessible
	WarmMaxTokens int // Summarized, partially accessible
	ColdMaxTokens int // Heavily compressed, archived
}

// newTieredCompaction creates default tiered compaction.
func newTieredCompaction() *TieredCompaction {
	return &TieredCompaction{
		HotMaxTokens:  2000,  // Recent 2K tokens kept verbatim
		WarmMaxTokens: 5000,  // Next 5K tokens summarized
		ColdMaxTokens: 10000, // Beyond 10K heavily compressed
	}
}

// Compact applies tiered compaction to conversation history.
func (t *TieredCompaction) Compact(turns []string) string {
	if len(turns) == 0 {
		return ""
	}

	totalTokens := 0
	for _, turn := range turns {
		totalTokens += EstimateTokens(turn)
	}

	if totalTokens <= t.HotMaxTokens {
		// All fits in hot tier
		return strings.Join(turns, "\n\n")
	}

	// Split into tiers
	var hot, warm, cold []string
	hotTokens := 0

	// Hot tier: most recent turns
	for i := len(turns) - 1; i >= 0; i-- {
		turnTokens := EstimateTokens(turns[i])
		if hotTokens+turnTokens <= t.HotMaxTokens {
			hot = append([]string{turns[i]}, hot...)
			hotTokens += turnTokens
		} else {
			// Remaining goes to warm/cold
			remaining := turns[:i+1]
			warmTokens := 0
			for j := len(remaining) - 1; j >= 0; j-- {
				wt := EstimateTokens(remaining[j])
				if warmTokens+wt <= t.WarmMaxTokens {
					warm = append([]string{remaining[j]}, warm...)
					warmTokens += wt
				} else {
					cold = append([]string{remaining[j]}, cold...)
				}
			}
			break
		}
	}

	// Build output
	var parts []string
	if len(cold) > 0 {
		parts = append(parts, "[Archived: "+fmt.Sprintf("%d", len(cold))+" older turns summarized]")
	}
	if len(warm) > 0 {
		parts = append(parts, strings.Join(warm, "\n"))
	}
	if len(hot) > 0 {
		parts = append(parts, strings.Join(hot, "\n"))
	}

	return strings.Join(parts, "\n\n")
}
