package filter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// OutputFormat controls how pipeline results are serialized.
type OutputFormat string

const (
	// OutputFormatPlain emits only the compressed text.
	OutputFormatPlain OutputFormat = "plain"
	// OutputFormatAnnotated emits compressed text with inline markers showing
	// what was removed and by which filter.
	OutputFormatAnnotated OutputFormat = "annotated"
	// OutputFormatJSON emits a structured JSON object with full metadata.
	OutputFormatJSON OutputFormat = "json"
)

// PipelineOutput is the result of a pipeline run, ready for rendering.
type PipelineOutput struct {
	// Compressed is the final compressed text.
	Compressed string `json:"compressed"`
	// OriginalTokens is the token count before compression.
	OriginalTokens int `json:"tokens_in"`
	// FinalTokens is the token count after compression.
	FinalTokens int `json:"tokens_out"`
	// TokensSaved is OriginalTokens − FinalTokens.
	TokensSaved int `json:"tokens_saved"`
	// Ratio is FinalTokens / OriginalTokens.
	Ratio float64 `json:"ratio"`
	// FiltersUsed lists the filters that produced token savings.
	FiltersUsed []string `json:"filters_used"`
	// Mode is the pipeline mode used.
	Mode string `json:"mode"`
	// Annotations maps ranges/markers to removal explanations (annotated mode only).
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Annotation marks a removed or replaced section of content.
type Annotation struct {
	FilterName string `json:"filter"`
	Marker     string `json:"marker"`    // placeholder text in annotated output
	TokensRemoved int `json:"tokens_removed"`
}

// NewPipelineOutput constructs a PipelineOutput from raw pipeline results.
func NewPipelineOutput(original, compressed string, mode Mode, stageResults []ExplainedStage) PipelineOutput {
	origTok := core.EstimateTokens(original)
	finalTok := core.EstimateTokens(compressed)
	saved := origTok - finalTok

	ratio := 1.0
	if origTok > 0 {
		ratio = float64(finalTok) / float64(origTok)
	}

	var used []string
	for _, s := range stageResults {
		if s.Applied {
			used = append(used, s.FilterName)
		}
	}

	return PipelineOutput{
		Compressed:     compressed,
		OriginalTokens: origTok,
		FinalTokens:    finalTok,
		TokensSaved:    saved,
		Ratio:          ratio,
		FiltersUsed:    used,
		Mode:           string(mode),
	}
}

// Render formats the output according to the requested format.
func (o PipelineOutput) Render(format OutputFormat) string {
	switch format {
	case OutputFormatJSON:
		return o.renderJSON()
	case OutputFormatAnnotated:
		return o.renderAnnotated()
	default:
		return o.Compressed
	}
}

func (o PipelineOutput) renderJSON() string {
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(b)
}

func (o PipelineOutput) renderAnnotated() string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf(
		"[tokman: %d→%d tokens (%.1f%% reduction) via %s | filters: %s]\n\n",
		o.OriginalTokens, o.FinalTokens,
		(1.0-o.Ratio)*100,
		o.Mode,
		strings.Join(o.FiltersUsed, ", "),
	))

	// Content
	sb.WriteString(o.Compressed)

	return sb.String()
}

// ParseOutputFormat parses a string into an OutputFormat.
// Unknown values default to OutputFormatPlain.
func ParseOutputFormat(s string) OutputFormat {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return OutputFormatJSON
	case "annotated":
		return OutputFormatAnnotated
	default:
		return OutputFormatPlain
	}
}
