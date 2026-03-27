package filter

import (
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// SlidingWindowBudget enforces a token budget on streaming content using
// a sliding window approach. When the window fills up, oldest content is
// dropped to make room for new content, ensuring the total never exceeds budget.
// Task #134: Sliding window token budget for streaming.
type SlidingWindowBudget struct {
	// Budget is the maximum tokens in the window at any time.
	Budget int
	// MinKeepFrac is the minimum fraction of the window to always preserve
	// at the tail (most recent content). Default: 0.3 (keep last 30%).
	MinKeepFrac float64

	// internal state
	window []string // lines currently in window
	tokens int      // current token count
}

// NewSlidingWindowBudget creates a sliding window with the given token budget.
func NewSlidingWindowBudget(budget int) *SlidingWindowBudget {
	if budget <= 0 {
		budget = 4096
	}
	return &SlidingWindowBudget{
		Budget:      budget,
		MinKeepFrac: 0.3,
	}
}

// Name returns the filter name.
func (s *SlidingWindowBudget) Name() string { return "sliding_window" }

// Feed adds new content to the window, evicting old content if needed.
// Returns the current window content.
func (s *SlidingWindowBudget) Feed(input string) string {
	// Add new lines to window
	newLines := strings.Split(input, "\n")
	for _, line := range newLines {
		toks := core.EstimateTokens(line)
		s.window = append(s.window, line)
		s.tokens += toks
	}

	// Evict oldest content until within budget
	minKeep := int(float64(len(s.window)) * s.MinKeepFrac)
	if minKeep < 1 {
		minKeep = 1
	}

	for s.tokens > s.Budget && len(s.window) > minKeep {
		evicted := s.window[0]
		s.window = s.window[1:]
		s.tokens -= core.EstimateTokens(evicted)
		if s.tokens < 0 {
			s.tokens = 0
		}
	}

	return strings.Join(s.window, "\n")
}

// Apply implements the Filter interface. Applies the sliding window to the full input,
// returning the most recent content that fits within the budget.
func (s *SlidingWindowBudget) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	if original <= s.Budget {
		return input, 0
	}

	budget := s.Budget
	if mode == ModeAggressive {
		budget = budget * 2 / 3
	}

	// Split into lines and fill from the end (keep most recent)
	lines := strings.Split(input, "\n")
	kept := make([]string, 0, len(lines))
	keptTokens := 0

	// Walk backwards to keep most recent content
	for i := len(lines) - 1; i >= 0; i-- {
		toks := core.EstimateTokens(lines[i])
		if keptTokens+toks > budget {
			if len(kept) == 0 {
				// Must keep at least one line
				kept = append(kept, lines[i])
			}
			break
		}
		kept = append(kept, lines[i])
		keptTokens += toks
	}

	// Reverse kept (we built it backwards)
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}

	omitted := len(lines) - len(kept)
	var result []string
	if omitted > 0 {
		result = append(result, "[... "+itoa(omitted)+" lines omitted (sliding window) ...]")
	}
	result = append(result, kept...)

	output := strings.Join(result, "\n")
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// TokensInWindow returns the current token count in the window.
func (s *SlidingWindowBudget) TokensInWindow() int { return s.tokens }

// Reset clears the window.
func (s *SlidingWindowBudget) Reset() {
	s.window = nil
	s.tokens = 0
}
