package filter

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// TokenCountOverlay annotates text with inline token count markers.
// This is useful for editors and IDEs to display real-time token usage
// as a user edits prompts or code to be sent to an LLM.
// Task #191: Real-time token counting overlay for editors.
type TokenCountOverlay struct {
	// GranularityLines is the number of lines per annotated segment.
	// 0 or negative means annotate every line.
	GranularityLines int

	// Format is the annotation format string.
	// Available placeholders: {tokens}, {cumulative}, {pct}
	// Default: "// [~{tokens} tokens | cumulative: {cumulative}]"
	Format string

	// MaxTokens is the total context limit (for % calculation). 0 disables %.
	MaxTokens int

	// Suffix: if true, annotation appears at end of segment; false = new line after.
	Suffix bool
}

// NewTokenCountOverlay creates an overlay with sensible defaults.
func NewTokenCountOverlay() *TokenCountOverlay {
	return &TokenCountOverlay{
		GranularityLines: 10,
		Format:           "// [~{tokens} toks | total: {cumulative}]",
		MaxTokens:        128_000,
	}
}

// Name returns the filter name.
func (f *TokenCountOverlay) Name() string { return "token_count_overlay" }

// Apply annotates the input with token count markers and returns the annotated text.
// Note: this filter *adds* content so saved is always 0 (or negative).
// It is primarily for diagnostic/editor use rather than compression.
func (f *TokenCountOverlay) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	granularity := f.GranularityLines
	if granularity <= 0 {
		granularity = 1
	}

	lines := strings.Split(input, "\n")
	var sb strings.Builder
	cumulative := 0

	for i := 0; i < len(lines); i++ {
		sb.WriteString(lines[i])
		sb.WriteByte('\n')

		if (i+1)%granularity == 0 || i == len(lines)-1 {
			start := i + 1 - granularity
		if start < 0 {
			start = 0
		}
		segment := strings.Join(lines[start:i+1], "\n")
			segTokens := core.EstimateTokens(segment)
			cumulative += segTokens

			ann := f.formatAnnotation(segTokens, cumulative)
			sb.WriteString(ann)
			sb.WriteByte('\n')
		}
	}

	result := sb.String()
	// This filter adds content (annotations), so token savings are 0.
	return result, 0
}

func (f *TokenCountOverlay) formatAnnotation(tokens, cumulative int) string {
	tmpl := f.Format
	if tmpl == "" {
		tmpl = "// [~{tokens} toks | total: {cumulative}]"
	}
	ann := strings.ReplaceAll(tmpl, "{tokens}", itoa(tokens))
	ann = strings.ReplaceAll(ann, "{cumulative}", itoa(cumulative))
	if f.MaxTokens > 0 {
		pct := float64(cumulative) / float64(f.MaxTokens) * 100
		ann = strings.ReplaceAll(ann, "{pct}", fmt.Sprintf("%.1f%%", pct))
	} else {
		ann = strings.ReplaceAll(ann, "{pct}", "?%")
	}
	return ann
}

// OverlaySummary produces a single-line summary of token usage for a text.
func OverlaySummary(input string, contextLimit int) string {
	tokens := core.EstimateTokens(input)
	if contextLimit <= 0 {
		return fmt.Sprintf("~%d tokens", tokens)
	}
	pct := float64(tokens) / float64(contextLimit) * 100
	remaining := contextLimit - tokens
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("~%d/%d tokens (%.1f%% used, %d remaining)",
		tokens, contextLimit, pct, remaining)
}

