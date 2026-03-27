package filter

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ABTestFramework runs A/B experiments comparing two compression configurations.
// Task #179: Compression effectiveness A/B testing framework.

// ABVariant names one arm of an A/B test.
type ABVariant string

const (
	ABVariantControl    ABVariant = "control"
	ABVariantTreatment  ABVariant = "treatment"
)

// ABExperiment defines an A/B test between two filter configurations.
type ABExperiment struct {
	Name            string
	Control         Filter
	Treatment       Filter
	// SplitPct is the probability [0,100] of routing to treatment. Default: 50.
	SplitPct        int
	rng             *rand.Rand
}

// ABResult records one observation from an A/B test.
type ABResult struct {
	Variant       ABVariant
	InputTokens   int
	OutputTokens  int
	TokensSaved   int
	ReductionPct  float64
	Latency       time.Duration
}

// ABSummary aggregates A/B results across many observations.
type ABSummary struct {
	Variant          ABVariant
	Observations     int
	AvgReductionPct  float64
	AvgLatencyMs     float64
	TotalTokensSaved int
}

// ABRunner executes the experiment and collects results.
type ABRunner struct {
	mu         sync.Mutex
	experiment *ABExperiment
	results    []ABResult
}

// NewABRunner creates a runner for the given experiment.
func NewABRunner(exp *ABExperiment) *ABRunner {
	if exp.SplitPct == 0 {
		exp.SplitPct = 50
	}
	exp.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &ABRunner{experiment: exp}
}

// Apply runs the appropriate variant for this observation and records the result.
// Satisfies the Filter interface so it can slot into any pipeline.
func (r *ABRunner) Apply(input string, mode Mode) (string, int) {
	r.mu.Lock()
	roll := r.experiment.rng.Intn(100)
	r.mu.Unlock()

	var (
		output  string
		saved   int
		variant ABVariant
		filter  Filter
	)
	if roll < r.experiment.SplitPct {
		variant = ABVariantTreatment
		filter = r.experiment.Treatment
	} else {
		variant = ABVariantControl
		filter = r.experiment.Control
	}

	start := time.Now()
	output, saved = filter.Apply(input, mode)
	elapsed := time.Since(start)

	origTokens := core.EstimateTokens(input)
	finalTokens := origTokens - saved
	var pct float64
	if origTokens > 0 {
		pct = float64(saved) / float64(origTokens) * 100
	}

	r.mu.Lock()
	r.results = append(r.results, ABResult{
		Variant:      variant,
		InputTokens:  origTokens,
		OutputTokens: finalTokens,
		TokensSaved:  saved,
		ReductionPct: pct,
		Latency:      elapsed,
	})
	r.mu.Unlock()

	return output, saved
}

// Name returns the experiment name.
func (r *ABRunner) Name() string { return "ab:" + r.experiment.Name }

// Summary returns per-variant statistics.
func (r *ABRunner) Summary() map[ABVariant]ABSummary {
	r.mu.Lock()
	results := make([]ABResult, len(r.results))
	copy(results, r.results)
	r.mu.Unlock()

	sums := map[ABVariant]*ABSummary{
		ABVariantControl:   {Variant: ABVariantControl},
		ABVariantTreatment: {Variant: ABVariantTreatment},
	}
	for _, res := range results {
		s := sums[res.Variant]
		s.Observations++
		s.TotalTokensSaved += res.TokensSaved
		s.AvgReductionPct += res.ReductionPct
		s.AvgLatencyMs += float64(res.Latency.Milliseconds())
	}
	out := make(map[ABVariant]ABSummary, 2)
	for v, s := range sums {
		if s.Observations > 0 {
			s.AvgReductionPct /= float64(s.Observations)
			s.AvgLatencyMs /= float64(s.Observations)
		}
		out[v] = *s
	}
	return out
}

// FormatSummary returns a human-readable A/B test report.
func (r *ABRunner) FormatSummary() string {
	sums := r.Summary()
	ctrl := sums[ABVariantControl]
	trt := sums[ABVariantTreatment]
	return fmt.Sprintf(
		"A/B Test: %s\n"+
			"  Control   (%4d obs): %.1f%% avg reduction, %.1fms avg latency\n"+
			"  Treatment (%4d obs): %.1f%% avg reduction, %.1fms avg latency\n",
		r.experiment.Name,
		ctrl.Observations, ctrl.AvgReductionPct, ctrl.AvgLatencyMs,
		trt.Observations, trt.AvgReductionPct, trt.AvgLatencyMs,
	)
}
