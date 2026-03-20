package filter

// PipelineState is an immutable snapshot of pipeline processing state.
// R20: Claw-compactor style immutable data flow — thread-safe, cacheable, debuggable.
type PipelineState struct {
	Input          string
	ContentType    ContentType
	LayerResults   []StateLayerResult
	CurrentOutput  string
	OriginalTokens int
	Budget         int
	Query          string
	Mode           Mode
}

// StateLayerResult holds the result of a single layer processing.
type StateLayerResult struct {
	LayerName   string
	Input       string
	Output      string
	TokensSaved int
	Skipped     bool
	SkipReason  string
}

// NewPipelineState creates an immutable initial state.
func NewPipelineState(input string, mode Mode, budget int, query string) *PipelineState {
	selector := NewAdaptiveLayerSelector()
	ct := selector.AnalyzeContent(input)
	return &PipelineState{
		Input:          input,
		ContentType:    ct,
		LayerResults:   nil,
		CurrentOutput:  input,
		OriginalTokens: EstimateTokens(input),
		Budget:         budget,
		Query:          query,
		Mode:           mode,
	}
}

// WithLayerResult returns a new state with an additional layer result.
// Original state is not modified (immutable pattern).
func (s *PipelineState) WithLayerResult(result StateLayerResult) *PipelineState {
	newResults := make([]StateLayerResult, len(s.LayerResults)+1)
	copy(newResults, s.LayerResults)
	newResults[len(s.LayerResults)] = result

	return &PipelineState{
		Input:          s.Input,
		ContentType:    s.ContentType,
		LayerResults:   newResults,
		CurrentOutput:  result.Output,
		OriginalTokens: s.OriginalTokens,
		Budget:         s.Budget,
		Query:          s.Query,
		Mode:           s.Mode,
	}
}

// WithSkippedLayer returns a new state with a skipped layer recorded.
func (s *PipelineState) WithSkippedLayer(layerName, reason string) *PipelineState {
	return s.WithLayerResult(StateLayerResult{
		LayerName:  layerName,
		Input:      s.CurrentOutput,
		Output:     s.CurrentOutput,
		Skipped:    true,
		SkipReason: reason,
	})
}

// TotalSaved computes total tokens saved across all layers.
func (s *PipelineState) TotalSaved() int {
	total := 0
	for _, r := range s.LayerResults {
		if !r.Skipped {
			total += r.TokensSaved
		}
	}
	return total
}

// FinalTokens returns the current token count.
func (s *PipelineState) FinalTokens() int {
	return EstimateTokens(s.CurrentOutput)
}

// ReductionPercent returns the reduction percentage.
func (s *PipelineState) ReductionPercent() float64 {
	if s.OriginalTokens == 0 {
		return 0
	}
	return float64(s.TotalSaved()) / float64(s.OriginalTokens) * 100
}

// SkippedLayers returns names of skipped layers.
func (s *PipelineState) SkippedLayers() []string {
	var skipped []string
	for _, r := range s.LayerResults {
		if r.Skipped {
			skipped = append(skipped, r.LayerName)
		}
	}
	return skipped
}

// AppliedLayers returns names of applied layers.
func (s *PipelineState) AppliedLayers() []string {
	var applied []string
	for _, r := range s.LayerResults {
		if !r.Skipped {
			applied = append(applied, r.LayerName)
		}
	}
	return applied
}
