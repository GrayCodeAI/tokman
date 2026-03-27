package filter

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// DegradationLevel controls how aggressively the pipeline degrades under pressure.
// Task #104: Graceful degradation under memory pressure.
type DegradationLevel int

const (
	// DegradeLevelNone — full pipeline, no degradation.
	DegradeLevelNone DegradationLevel = iota
	// DegradeLevelLight — skip expensive filters (LLM, neural scoring).
	DegradeLevelLight
	// DegradeLevelModerate — minimal filter set: whitespace, dedup, ANSI strip.
	DegradeLevelModerate
	// DegradeLevelEmergency — pass-through only; no compression.
	DegradeLevelEmergency
)

func (d DegradationLevel) String() string {
	switch d {
	case DegradeLevelLight:
		return "light"
	case DegradeLevelModerate:
		return "moderate"
	case DegradeLevelEmergency:
		return "emergency"
	default:
		return "none"
	}
}

// DegradationPolicy defines thresholds that trigger automatic degradation.
type DegradationPolicy struct {
	// MaxHeapMB: if current heap allocation exceeds this, degrade.
	// 0 means disabled.
	MaxHeapMB uint64

	// MaxInputTokens: skip expensive filters for large inputs.
	// 0 means disabled.
	MaxInputTokens int

	// FilterTimeoutMs: max time (ms) allowed per filter before skipping the rest.
	// 0 means disabled.
	FilterTimeoutMs int64
}

// DefaultDegradationPolicy returns a policy tuned for typical LLM workloads.
func DefaultDegradationPolicy() DegradationPolicy {
	return DegradationPolicy{
		MaxHeapMB:       512,
		MaxInputTokens:  200_000,
		FilterTimeoutMs: 2000,
	}
}

// GracefulDegrader monitors runtime health and returns the current degradation level.
type GracefulDegrader struct {
	mu       sync.Mutex
	policy   DegradationPolicy
	current  DegradationLevel
	lastSnap time.Time
	snapMS   stats
}

type stats struct {
	heapMB uint64
}

// NewGracefulDegrader creates a degrader with the given policy.
func NewGracefulDegrader(policy DegradationPolicy) *GracefulDegrader {
	return &GracefulDegrader{policy: policy}
}

// Level returns the current degradation level, refreshing stats at most every second.
func (g *GracefulDegrader) Level() DegradationLevel {
	g.mu.Lock()
	defer g.mu.Unlock()
	if time.Since(g.lastSnap) > time.Second {
		g.refresh()
	}
	return g.current
}

// LevelForInput returns the degradation level appropriate for the given input.
func (g *GracefulDegrader) LevelForInput(input string) DegradationLevel {
	base := g.Level()
	if g.policy.MaxInputTokens > 0 {
		tokens := core.EstimateTokens(input)
		if tokens > g.policy.MaxInputTokens {
			if base < DegradeLevelModerate {
				base = DegradeLevelModerate
			}
		} else if tokens > g.policy.MaxInputTokens/2 {
			if base < DegradeLevelLight {
				base = DegradeLevelLight
			}
		}
	}
	return base
}

func (g *GracefulDegrader) refresh() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	heapMB := ms.HeapInuse / (1024 * 1024)
	g.snapMS.heapMB = heapMB
	g.lastSnap = time.Now()

	if g.policy.MaxHeapMB > 0 {
		pct := float64(heapMB) / float64(g.policy.MaxHeapMB)
		switch {
		case pct >= 1.0:
			g.current = DegradeLevelEmergency
		case pct >= 0.85:
			g.current = DegradeLevelModerate
		case pct >= 0.70:
			g.current = DegradeLevelLight
		default:
			g.current = DegradeLevelNone
		}
	}
}

// Stats returns a human-readable degradation status string.
func (g *GracefulDegrader) Stats() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("degradation=%s heap=%dMB", g.current, g.snapMS.heapMB)
}

// ShouldSkipFilter returns true if a given filter should be skipped at this level.
// expensiveFilters is the list of filter names considered "expensive" at each tier.
func ShouldSkipFilter(filterName string, level DegradationLevel) bool {
	switch level {
	case DegradeLevelNone:
		return false
	case DegradeLevelLight:
		// Skip LLM-dependent and neural filters
		expensive := []string{"llm_compress", "llm_aware", "attention_predictor",
			"evaluator_heads", "h2o", "attention_sink", "contrastive"}
		return containsString(expensive, filterName)
	case DegradeLevelModerate:
		// Keep only cheap filters
		cheap := []string{"whitespace", "ansi_strip", "comment_strip",
			"dedup", "error_dedup", "line_truncate"}
		return !containsString(cheap, filterName)
	case DegradeLevelEmergency:
		// Skip everything
		return true
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
