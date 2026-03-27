package filter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ExplainResult holds the full explanation of a pipeline compression run.
// Emitted when explain mode is enabled. Safe to marshal as JSON.
type ExplainResult struct {
	OriginalTokens int              `json:"original_tokens"`
	FinalTokens    int              `json:"final_tokens"`
	TokensSaved    int              `json:"tokens_saved"`
	ReductionPct   float64          `json:"reduction_pct"`
	Mode           string           `json:"mode"`
	TotalLatencyMs float64          `json:"total_latency_ms"`
	Stages         []ExplainedStage `json:"stages"`
}

// ExplainedStage records what a single filter stage did.
type ExplainedStage struct {
	FilterName     string  `json:"filter"`
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	TokensSaved    int     `json:"tokens_saved"`
	ReductionPct   float64 `json:"reduction_pct"`
	LatencyMicros  float64 `json:"latency_us"`
	Applied        bool    `json:"applied"`
	Reason         string  `json:"reason,omitempty"`
}

// ExplainingPipeline wraps a list of filters and records what each one does.
type ExplainingPipeline struct {
	filters []Filter
	mode    Mode
}

// NewExplainingPipeline creates an explaining pipeline with the given filters.
func NewExplainingPipeline(mode Mode, filters ...Filter) *ExplainingPipeline {
	return &ExplainingPipeline{filters: filters, mode: mode}
}

// Run applies all filters in sequence and returns the compressed output plus
// a full explanation of every stage's contribution.
func (p *ExplainingPipeline) Run(input string) (output string, result ExplainResult) {
	result.OriginalTokens = core.EstimateTokens(input)
	result.Mode = string(p.mode)

	start := time.Now()
	current := input

	for _, f := range p.filters {
		stageIn := current
		stageInToks := core.EstimateTokens(stageIn)

		stageStart := time.Now()
		out, saved := f.Apply(stageIn, p.mode)
		stageElapsed := time.Since(stageStart)

		stageOutToks := core.EstimateTokens(out)
		applied := saved > 0
		if applied {
			current = out
		}

		reductionPct := 0.0
		if stageInToks > 0 {
			reductionPct = float64(saved) / float64(stageInToks) * 100
		}

		reason := ""
		if !applied {
			reason = "no savings"
		}

		result.Stages = append(result.Stages, ExplainedStage{
			FilterName:    f.Name(),
			InputTokens:   stageInToks,
			OutputTokens:  stageOutToks,
			TokensSaved:   saved,
			ReductionPct:  reductionPct,
			LatencyMicros: float64(stageElapsed.Microseconds()),
			Applied:       applied,
			Reason:        reason,
		})
	}

	result.TotalLatencyMs = float64(time.Since(start).Microseconds()) / 1000.0
	result.FinalTokens = core.EstimateTokens(current)
	result.TokensSaved = result.OriginalTokens - result.FinalTokens
	if result.OriginalTokens > 0 {
		result.ReductionPct = float64(result.TokensSaved) / float64(result.OriginalTokens) * 100
	}

	return current, result
}

// FormatText formats an ExplainResult as a human-readable text report.
func (r ExplainResult) FormatText() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Compression Explain Report ===\n"))
	sb.WriteString(fmt.Sprintf("Mode: %s | Original: %d tokens → Final: %d tokens (saved %d, %.1f%%)\n",
		r.Mode, r.OriginalTokens, r.FinalTokens, r.TokensSaved, r.ReductionPct))
	sb.WriteString(fmt.Sprintf("Total latency: %.2f ms\n\n", r.TotalLatencyMs))

	sb.WriteString(fmt.Sprintf("%-30s  %8s  %8s  %7s  %9s  %s\n",
		"Filter", "In(tok)", "Out(tok)", "Saved%", "Lat(µs)", "Applied"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, s := range r.Stages {
		mark := "✗"
		if s.Applied {
			mark = "✓"
		}
		sb.WriteString(fmt.Sprintf("%-30s  %8d  %8d  %6.1f%%  %9.1f  %s\n",
			truncStr(s.FilterName, 30),
			s.InputTokens, s.OutputTokens,
			s.ReductionPct, s.LatencyMicros,
			mark,
		))
		if s.Reason != "" {
			sb.WriteString(fmt.Sprintf("  → %s\n", s.Reason))
		}
	}
	return sb.String()
}

// FormatJSON marshals the result to indented JSON.
func (r ExplainResult) FormatJSON() string {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(b)
}
