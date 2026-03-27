package filter

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// SmartTruncateFilter truncates content that exceeds a token budget while
// appending an informative summary suffix so the LLM knows content was cut.
//
// Strategy:
//  1. If content fits within budget, return as-is.
//  2. Otherwise: keep head (70% of budget), append suffix, keep tail (remaining 30%).
//     This preserves both the start and end of content — important for logs and traces.
type SmartTruncateFilter struct {
	// Budget is the max tokens to target. 0 = use mode-based default.
	Budget int
}

// NewSmartTruncateFilter creates a smart truncation filter.
func NewSmartTruncateFilter() *SmartTruncateFilter {
	return &SmartTruncateFilter{}
}

// NewSmartTruncateFilterWithBudget creates a smart truncation filter with a specific budget.
func NewSmartTruncateFilterWithBudget(budget int) *SmartTruncateFilter {
	return &SmartTruncateFilter{Budget: budget}
}

// Name returns the filter name.
func (f *SmartTruncateFilter) Name() string {
	return "smart_truncate"
}

// Apply applies smart truncation with a summary suffix.
func (f *SmartTruncateFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	budget := f.Budget
	if budget <= 0 {
		switch mode {
		case ModeAggressive:
			budget = 2000
		default:
			budget = 4000
		}
	}

	originalTokens := core.EstimateTokens(input)
	if originalTokens <= budget {
		return input, 0
	}

	// Split into lines for cleaner truncation
	lines := strings.Split(input, "\n")
	totalLines := len(lines)

	// Target: 70% head, 30% tail
	headBudget := budget * 7 / 10
	tailBudget := budget - headBudget

	// Estimate chars from budget (≈4 chars/token)
	headChars := headBudget * 4
	tailChars := tailBudget * 4

	// Collect head lines
	var headLines []string
	headLen := 0
	for _, line := range lines {
		if headLen+len(line)+1 > headChars {
			break
		}
		headLines = append(headLines, line)
		headLen += len(line) + 1
	}

	// Collect tail lines (from end)
	var tailLines []string
	tailLen := 0
	for i := totalLines - 1; i >= len(headLines); i-- {
		line := lines[i]
		if tailLen+len(line)+1 > tailChars {
			break
		}
		tailLines = append([]string{line}, tailLines...)
		tailLen += len(line) + 1
	}

	omittedLines := totalLines - len(headLines) - len(tailLines)
	omittedTokens := originalTokens - budget

	suffix := fmt.Sprintf(
		"\n... [%d lines / ~%d tokens omitted — content truncated to fit context window] ...\n",
		omittedLines, omittedTokens,
	)

	var parts []string
	parts = append(parts, strings.Join(headLines, "\n"))
	parts = append(parts, suffix)
	if len(tailLines) > 0 {
		parts = append(parts, strings.Join(tailLines, "\n"))
	}

	output := strings.Join(parts, "")
	saved := originalTokens - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}
