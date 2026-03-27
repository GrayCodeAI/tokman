package filter

import (
	"sort"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ContextWindowOptimizer selects and orders content sections to maximize
// information density within a target token budget.
//
// Given a set of content sections, it:
//  1. Scores each section by relevance (keyword match + recency + structural importance)
//  2. Sorts sections by score (descending)
//  3. Greedily fills the token budget from highest to lowest score
//  4. Appends a summary of omitted sections at the end
//
// This is ideal for multi-file or multi-context situations where not all
// content can fit within the LLM's context window.
type ContextWindowOptimizer struct {
	// Budget is the target token limit. 0 = use default (100,000).
	Budget int
	// Keywords are used to boost relevance of sections containing them.
	Keywords []string
}

// ContentSection represents one piece of content to be ranked and selected.
type ContentSection struct {
	Name      string    // Display name (e.g., file path, section title)
	Content   string    // The actual content
	Priority  int       // Explicit priority (0 = auto-computed)
	ModTime   time.Time // Last modification time (zero = unknown)
	IsError   bool      // True for error/failure sections (always high priority)
}

// OptimizationResult is the result of context window optimization.
type OptimizationResult struct {
	Selected []ContentSection // Sections included, in ranked order
	Omitted  []ContentSection // Sections excluded due to budget
	UsedTokens int
	BudgetTokens int
}

// NewContextWindowOptimizer creates an optimizer with a default budget.
func NewContextWindowOptimizer() *ContextWindowOptimizer {
	return &ContextWindowOptimizer{Budget: 100_000}
}

// NewContextWindowOptimizerWithBudget creates an optimizer with a specific token budget.
func NewContextWindowOptimizerWithBudget(budget int) *ContextWindowOptimizer {
	return &ContextWindowOptimizer{Budget: budget}
}

// Optimize selects sections that fit within the token budget, ranked by relevance.
func (o *ContextWindowOptimizer) Optimize(sections []ContentSection) OptimizationResult {
	budget := o.Budget
	if budget <= 0 {
		budget = 100_000
	}

	type scoredSection struct {
		section ContentSection
		score   float64
		tokens  int
	}

	// Score all sections
	scored := make([]scoredSection, 0, len(sections))
	for _, s := range sections {
		toks := core.EstimateTokens(s.Content)
		score := o.scoreSection(s)
		scored = append(scored, scoredSection{section: s, score: score, tokens: toks})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Greedy fill
	result := OptimizationResult{BudgetTokens: budget}
	usedTokens := 0
	for _, ss := range scored {
		if usedTokens+ss.tokens <= budget {
			result.Selected = append(result.Selected, ss.section)
			usedTokens += ss.tokens
		} else {
			result.Omitted = append(result.Omitted, ss.section)
		}
	}
	result.UsedTokens = usedTokens
	return result
}

// Apply implements the Filter interface. It treats the input as a single
// section and applies smart truncation if it exceeds the budget.
func (o *ContextWindowOptimizer) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}
	budget := o.Budget
	if budget <= 0 {
		budget = 100_000
	}
	if mode == ModeAggressive {
		budget = budget / 2
	}

	original := core.EstimateTokens(input)
	if original <= budget {
		return input, 0
	}

	// Split into sections by blank lines / headers
	sections := splitIntoSections(input)
	if len(sections) <= 1 {
		// Fall back to smart truncation
		f := NewSmartTruncateFilterWithBudget(budget)
		return f.Apply(input, mode)
	}

	// Build ContentSection list
	cs := make([]ContentSection, len(sections))
	for i, s := range sections {
		cs[i] = ContentSection{
			Name:    "section-" + itoa(i+1),
			Content: s,
			IsError: containsErrorHint(s),
		}
	}

	result := o.Optimize(cs)

	// Reconstruct output in original order (preserve narrative flow)
	origOrder := make([]string, 0, len(result.Selected))
	for _, s := range result.Selected {
		origOrder = append(origOrder, s.Content)
	}
	if len(result.Omitted) > 0 {
		omittedTokens := 0
		for _, s := range result.Omitted {
			omittedTokens += core.EstimateTokens(s.Content)
		}
		origOrder = append(origOrder,
			"\n... ["+itoa(len(result.Omitted))+" sections / ~"+itoa(omittedTokens)+" tokens omitted — exceeded context window] ...\n")
	}

	output := strings.Join(origOrder, "\n\n")
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// Name returns the filter name.
func (o *ContextWindowOptimizer) Name() string {
	return "context_window_optimizer"
}

// scoreSection computes a relevance score for a content section.
func (o *ContextWindowOptimizer) scoreSection(s ContentSection) float64 {
	score := float64(s.Priority)

	// Error sections are always highest priority
	if s.IsError {
		score += 100
	}

	// Keyword matching
	if len(o.Keywords) > 0 {
		lower := strings.ToLower(s.Content)
		matches := 0
		for _, kw := range o.Keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				matches++
			}
		}
		score += float64(matches) * 10
	}

	// Recency boost (recently modified files score higher)
	if !s.ModTime.IsZero() {
		age := time.Since(s.ModTime)
		if age < time.Hour {
			score += 20
		} else if age < 24*time.Hour {
			score += 10
		} else if age < 7*24*time.Hour {
			score += 5
		}
	}

	// Structural importance: sections with function signatures, class decls
	if importantLineRe.MatchString(s.Content) {
		score += 5
	}

	// Prefer denser sections (more tokens = more information)
	tokens := core.EstimateTokens(s.Content)
	if tokens > 0 && tokens < 500 {
		score += 2 // small, dense sections are often high value
	}

	return score
}

// splitIntoSections splits a document into logical sections.
func splitIntoSections(input string) []string {
	var sections []string
	var current strings.Builder
	lines := strings.Split(input, "\n")
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount >= 2 && current.Len() > 0 {
				sections = append(sections, strings.TrimSpace(current.String()))
				current.Reset()
				blankCount = 0
				continue
			}
		} else {
			blankCount = 0
		}
		current.WriteString(line + "\n")
	}
	if current.Len() > 0 {
		s := strings.TrimSpace(current.String())
		if s != "" {
			sections = append(sections, s)
		}
	}
	return sections
}

// containsErrorHint returns true if the content looks like an error/failure.
func containsErrorHint(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "error:") ||
		strings.Contains(lower, "fatal:") ||
		strings.Contains(lower, "panic:") ||
		strings.Contains(lower, "failed:") ||
		strings.Contains(lower, "exception:")
}
