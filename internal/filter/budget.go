package filter

import (
	"strconv"
	"strings"
)

// BudgetEnforcer enforces strict token limits on output.
// Research-based: Budget-Constrained Compression (2024) - provides predictable
// output size by scoring segments and keeping only the most important ones.
//
// Key insight: LLMs have finite context windows. Enforcing a strict budget
// ensures output fits within constraints while maximizing information content.
type BudgetEnforcer struct {
	budget int // Maximum tokens allowed (0 = no limit)
}

// BudgetConfig holds configuration for the budget enforcer
type BudgetConfig struct {
	Budget int // Maximum tokens (0 = unlimited)
}

// NewBudgetEnforcer creates a new budget enforcer.
func NewBudgetEnforcer(budget int) *BudgetEnforcer {
	return &BudgetEnforcer{
		budget: budget,
	}
}

// NewBudgetEnforcerWithConfig creates a budget enforcer with config.
func NewBudgetEnforcerWithConfig(cfg BudgetConfig) *BudgetEnforcer {
	return &BudgetEnforcer{
		budget: cfg.Budget,
	}
}

// Name returns the filter name.
func (f *BudgetEnforcer) Name() string {
	return "budget"
}

// Apply enforces the token budget on the output.
func (f *BudgetEnforcer) Apply(input string, mode Mode) (string, int) {
	// No budget set - pass through
	if f.budget <= 0 {
		return input, 0
	}

	tokens := EstimateTokens(input)

	// Under budget - no action needed
	if tokens <= f.budget {
		return input, 0
	}

	// Over budget - need to compress
	output := f.enforceBudget(input, mode)

	tokensSaved := tokens - EstimateTokens(output)
	return output, tokensSaved
}

// SetBudget updates the token budget
func (f *BudgetEnforcer) SetBudget(budget int) {
	f.budget = budget
}

// enforceBudget compresses output to fit within budget
func (f *BudgetEnforcer) enforceBudget(input string, mode Mode) string {
	lines := strings.Split(input, "\n")

	// Score each line by importance
	scored := f.scoreLines(lines)

	// Calculate how many tokens we can keep
	targetTokens := f.budget - 50 // Reserve 50 tokens for truncation marker

	// Select lines to keep
	kept := f.selectLines(scored, targetTokens, len(lines))

	// Build output with truncation marker
	output := strings.Join(kept, "\n")

	remainingTokens := EstimateTokens(input) - EstimateTokens(output)
	if remainingTokens > 0 {
		output += "\n\n[... truncated: " + strconv.Itoa(remainingTokens) + " tokens omitted]"
	}

	return output
}

// scoredLine represents a line with its importance score
type scoredLine struct {
	line  string
	score float64
	index int
}

// scoreLines assigns importance scores to each line
func (f *BudgetEnforcer) scoreLines(lines []string) []scoredLine {
	scored := make([]scoredLine, len(lines))

	for i, line := range lines {
		scored[i] = scoredLine{
			line:  line,
			score: f.scoreLine(line),
			index: i,
		}
	}

	return scored
}

// scoreLine calculates importance score for a line
func (f *BudgetEnforcer) scoreLine(line string) float64 {
	trimmed := strings.TrimSpace(line)

	// Empty lines are low priority
	if trimmed == "" {
		return 0.1
	}

	score := 0.0
	lower := strings.ToLower(trimmed)

	// Critical: errors and failures
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") ||
		strings.Contains(lower, "failure") || strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "panic") || strings.Contains(lower, "exception") {
		score += 0.9
	}

	// Important: warnings and diffs
	if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
		score += 0.7
	}
	if strings.Contains(trimmed, "@@") || strings.Contains(trimmed, "diff --git") {
		score += 0.8
	}

	// Important: file references
	filePatterns := []string{".go:", ".py:", ".js:", ".ts:", ".rs:", ".java:", ".c:", ".cpp:"}
	for _, pattern := range filePatterns {
		if strings.Contains(trimmed, pattern) {
			score += 0.6
		}
	}

	// Important: test results
	if strings.Contains(lower, "pass") || strings.Contains(lower, "fail") ||
		strings.Contains(lower, "test") {
		score += 0.5
	}

	// Less important: success messages
	if strings.Contains(lower, "success") || strings.Contains(lower, "ok") ||
		strings.Contains(lower, "complete") {
		score -= 0.2
	}

	// Less important: debug/trace
	if strings.Contains(lower, "debug") || strings.Contains(lower, "trace") ||
		strings.Contains(lower, "verbose") {
		score -= 0.3
	}

	// Length bonus: longer lines may have more information
	if len(trimmed) > 50 {
		score += 0.1
	}

	// Clamp to [0, 1]
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// selectLines selects lines to keep within budget
func (f *BudgetEnforcer) selectLines(scored []scoredLine, targetTokens int, totalLines int) []string {
	// Strategy: Keep high-score lines while preserving some structure
	// We use a hybrid approach:
	// 1. Always keep critical lines (score >= 0.8)
	// 2. Keep important lines (score >= 0.5) up to budget
	// 3. Keep structural lines (first/last) for context

	type keepDecision struct {
		keep   bool
		reason string
	}

	decisions := make([]keepDecision, len(scored))

	// First pass: Mark critical lines
	for i, sl := range scored {
		if sl.score >= 0.8 {
			decisions[i] = keepDecision{keep: true, reason: "critical"}
		}
	}

	// Second pass: Mark structural lines (first/last 5 lines)
	for i := 0; i < 5 && i < len(scored); i++ {
		if !decisions[i].keep {
			decisions[i] = keepDecision{keep: true, reason: "header"}
		}
	}
	footerStart := len(scored) - 5
	if footerStart < 0 {
		footerStart = 0
	}
	for i := footerStart; i < len(scored); i++ {
		if !decisions[i].keep {
			decisions[i] = keepDecision{keep: true, reason: "footer"}
		}
	}

	// Calculate tokens used by already-kept lines
	currentTokens := 0
	for i, d := range decisions {
		if d.keep {
			currentTokens += EstimateTokens(scored[i].line)
		}
	}

	// Third pass: Add important lines within budget
	for i, sl := range scored {
		if currentTokens >= targetTokens {
			break
		}

		if !decisions[i].keep && sl.score >= 0.5 {
			lineTokens := EstimateTokens(sl.line)
			if currentTokens+lineTokens <= targetTokens {
				decisions[i] = keepDecision{keep: true, reason: "important"}
				currentTokens += lineTokens
			}
		}
	}

	// Build result, with ellipsis for gaps
	var result []string
	lastKept := -1

	for i, d := range decisions {
		if d.keep {
			// Add ellipsis for skipped sections
			if lastKept >= 0 && i-lastKept > 1 {
				gap := i - lastKept - 1
				if gap > 3 {
					result = append(result, "... ["+strconv.Itoa(gap)+" lines omitted]")
				} else {
					// Small gaps - just add an empty line
					result = append(result, "")
				}
			}
			result = append(result, scored[i].line)
			lastKept = i
		}
	}

	return result
}
