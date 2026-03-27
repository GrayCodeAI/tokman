package filter

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// FilterWeights maps filter name to an importance weight in [0, 1].
// Higher weight = filter is applied with higher priority / more aggressively.
// Task #184: Automatic filter weight tuning via gradient-free optimization.
type FilterWeights map[string]float64

// WeightTuner optimizes filter weights using coordinate-descent with
// random perturbation (a gradient-free method similar to Nelder-Mead simplex).
type WeightTuner struct {
	mu       sync.Mutex
	weights  FilterWeights
	history  []tunerRecord
	rng      *rand.Rand
	// Objective: maximize token reduction while preserving content quality
	// (measured by contentPreservationRatio).
	qualityWeight float64 // how much to penalize quality loss (default 0.5)
}

type tunerRecord struct {
	weights FilterWeights
	score   float64
}

// NewWeightTuner creates a tuner with equal starting weights.
func NewWeightTuner(filterNames []string) *WeightTuner {
	w := make(FilterWeights, len(filterNames))
	for _, name := range filterNames {
		w[name] = 0.5
	}
	return &WeightTuner{
		weights:       w,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
		qualityWeight: 0.5,
	}
}

// Weights returns the current best weights (safe to call concurrently).
func (t *WeightTuner) Weights() FilterWeights {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(FilterWeights, len(t.weights))
	for k, v := range t.weights {
		out[k] = v
	}
	return out
}

// Observe records a filter's actual performance to update weights.
// Call this after each Apply() with the actual token reduction ratio.
func (t *WeightTuner) Observe(filterName string, reductionRatio, qualityRatio float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Exponential moving average update: weight ← 0.9·weight + 0.1·reward
	score := reductionRatio*(1-t.qualityWeight) + qualityRatio*t.qualityWeight
	if w, ok := t.weights[filterName]; ok {
		t.weights[filterName] = clampWeight(0.9*w + 0.1*score)
	}
}

// Optimize runs N iterations of coordinate-descent optimization over the
// provided corpus of sample inputs. Each iteration perturbs one weight and
// evaluates the objective; better configurations are accepted.
func (t *WeightTuner) Optimize(filters []Filter, corpus []string, mode Mode, iterations int) FilterWeights {
	if len(corpus) == 0 || len(filters) == 0 {
		return t.Weights()
	}

	t.mu.Lock()
	best := copyWeights(t.weights)
	t.mu.Unlock()

	bestScore := t.evaluate(filters, best, corpus, mode)

	for iter := 0; iter < iterations; iter++ {
		// Pick a random filter to perturb
		candidate := copyWeights(best)
		names := filterWeightNames(candidate)
		name := names[t.rng.Intn(len(names))]

		// Gaussian perturbation σ=0.1
		delta := t.rng.NormFloat64() * 0.1
		candidate[name] = clampWeight(candidate[name] + delta)

		score := t.evaluate(filters, candidate, corpus, mode)
		if score > bestScore {
			bestScore = score
			best = candidate
			t.mu.Lock()
			t.history = append(t.history, tunerRecord{weights: copyWeights(best), score: score})
			t.mu.Unlock()
		}
	}

	t.mu.Lock()
	t.weights = best
	t.mu.Unlock()
	return best
}

// evaluate scores a weight configuration on the corpus.
func (t *WeightTuner) evaluate(filters []Filter, weights FilterWeights, corpus []string, mode Mode) float64 {
	var totalScore float64
	for _, sample := range corpus {
		origTokens := core.EstimateTokens(sample)
		if origTokens == 0 {
			continue
		}
		output := sample
		for _, f := range filters {
			w := weights[f.Name()]
			// Weight acts as a probability gate: if rand > weight, skip this filter.
			if t.rng.Float64() > w {
				continue
			}
			compressed, _ := f.Apply(output, mode)
			output = compressed
		}
		compTokens := core.EstimateTokens(output)
		reduction := float64(origTokens-compTokens) / float64(origTokens)
		quality := contentPreservationRatio(sample, output)
		totalScore += reduction*(1-t.qualityWeight) + quality*t.qualityWeight
	}
	if len(corpus) > 0 {
		totalScore /= float64(len(corpus))
	}
	return totalScore
}

// BestHistory returns the top N weight configurations found during optimization.
func (t *WeightTuner) BestHistory(n int) []tunerRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	sorted := make([]tunerRecord, len(t.history))
	copy(sorted, t.history)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].score > sorted[j].score })
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// Report returns a human-readable summary of current weights.
func (t *WeightTuner) Report() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out string
	names := filterWeightNames(t.weights)
	sort.Strings(names)
	for _, name := range names {
		bars := int(math.Round(t.weights[name] * 20))
		out += fmt.Sprintf("  %-30s %.2f [%s%s]\n",
			name, t.weights[name],
			repeatChar('█', bars), repeatChar('░', 20-bars))
	}
	return out
}

func clampWeight(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

func copyWeights(w FilterWeights) FilterWeights {
	out := make(FilterWeights, len(w))
	for k, v := range w {
		out[k] = v
	}
	return out
}

func filterWeightNames(w FilterWeights) []string {
	names := make([]string, 0, len(w))
	for k := range w {
		names = append(names, k)
	}
	return names
}

func repeatChar(ch rune, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]rune, n)
	for i := range out {
		out[i] = ch
	}
	return string(out)
}
