package filter

import (
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// WarmUpResult records the outcome of a pipeline warm-up pass.
// Task #131: Pipeline warm-up and lazy initialization.
type WarmUpResult struct {
	Duration    time.Duration
	FiltersHit  int
	TokensIn    int
	TokensOut   int
}

// warmupOnce ensures the one-time warm-up only runs once per process.
var warmupOnce sync.Once

// warmupSentinel is a representative text used to prime filters on first use.
// It is short but exercises code paths (identifiers, numbers, comments, logs).
const warmupSentinel = `// WarmUp: tokman pipeline priming pass
func example() error {
	connectionPoolManager := getPool()
	total := 0
	for i := 0; i < 100; i++ {
		total += connectionPoolManager.Size()
	}
	log.Printf("2024-01-01T00:00:00Z pool size=%d total=%d", connectionPoolManager.Size(), total)
	return nil
}
ERROR: connection refused (attempt 3/3)
ERROR: connection refused (attempt 3/3)
`

// WarmUp primes a PipelineCoordinator by running a synthetic input through it.
// This triggers any lazy initializations (BPE loader, regex compilation, etc.)
// so the first real call has minimal cold-start latency.
//
// Calling WarmUp multiple times is safe — subsequent calls are no-ops.
func WarmUp(pipeline *PipelineCoordinator) WarmUpResult {
	var result WarmUpResult
	warmupOnce.Do(func() {
		result = warmUpOnce(pipeline)
	})
	return result
}

// WarmUpFresh always runs the warm-up, regardless of whether it's been done before.
// Use this when you have a freshly-created coordinator.
func WarmUpFresh(pipeline *PipelineCoordinator) WarmUpResult {
	return warmUpOnce(pipeline)
}

func warmUpOnce(pipeline *PipelineCoordinator) WarmUpResult {
	start := time.Now()
	origTokens := core.EstimateTokens(warmupSentinel)

	_, stats := pipeline.Process(warmupSentinel)

	return WarmUpResult{
		Duration:   time.Since(start),
		FiltersHit: len(stats.LayerStats),
		TokensIn:   origTokens,
		TokensOut:  stats.FinalTokens,
	}
}

// LazyPipeline wraps a PipelineCoordinator with once-on-first-use warm-up.
// The warm-up runs in a background goroutine immediately on construction
// so it's ready by the time Process() is first called.
type LazyPipeline struct {
	coordinator *PipelineCoordinator
	warmupDone  chan struct{}
	result      WarmUpResult
}

// NewLazyPipeline creates a pipeline that warms up in the background.
func NewLazyPipeline(cfg PipelineConfig) *LazyPipeline {
	lp := &LazyPipeline{
		coordinator: NewPipelineCoordinator(cfg),
		warmupDone:  make(chan struct{}),
	}
	go func() {
		lp.result = WarmUpFresh(lp.coordinator)
		close(lp.warmupDone)
	}()
	return lp
}

// Process compresses input, waiting for warm-up to complete first if needed.
func (lp *LazyPipeline) Process(input string) (string, *PipelineStats) {
	select {
	case <-lp.warmupDone:
	default:
		// Warm-up still running: check if it finishes quickly; otherwise proceed.
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-lp.warmupDone:
		case <-timer.C:
		}
		timer.Stop()
	}
	return lp.coordinator.Process(input)
}

// WarmUpResult returns the result of the background warm-up (blocks if not done).
func (lp *LazyPipeline) WarmUpResult() WarmUpResult {
	<-lp.warmupDone
	return lp.result
}
