package filter

import (
	"context"
	"time"
)

// TimeoutGuard wraps a filter and enforces a per-call timeout.
// If the inner filter's Apply() exceeds the deadline, the guard
// returns the input unchanged (pass-through) and records the timeout.
//
// This prevents one slow filter (e.g., LLM-based or semantic) from
// blocking the entire pipeline.
type TimeoutGuard struct {
	inner   Filter
	timeout time.Duration

	// TimeoutCount tracks how many times this guard has timed out.
	TimeoutCount int
}

// NewTimeoutGuard wraps a filter with the given timeout.
func NewTimeoutGuard(inner Filter, timeout time.Duration) *TimeoutGuard {
	return &TimeoutGuard{inner: inner, timeout: timeout}
}

// Name returns the wrapped filter's name.
func (g *TimeoutGuard) Name() string {
	return g.inner.Name()
}

// Apply runs the inner filter within the timeout. If the timeout is
// exceeded, input is returned unchanged and TimeoutCount is incremented.
func (g *TimeoutGuard) Apply(input string, mode Mode) (string, int) {
	if g.timeout <= 0 {
		return g.inner.Apply(input, mode)
	}

	type result struct {
		output string
		saved  int
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	resultCh := make(chan result, 1)

	go func() {
		o, s := g.inner.Apply(input, mode)
		select {
		case resultCh <- result{output: o, saved: s}:
		case <-ctx.Done():
			// Discard result if context expired
		}
	}()

	select {
	case r := <-resultCh:
		return r.output, r.saved
	case <-ctx.Done():
		g.TimeoutCount++
		return input, 0
	}
}

// DefaultTimeouts provides recommended timeouts for filter categories.
var DefaultTimeouts = map[string]time.Duration{
	"llm_compress":       2 * time.Second,
	"semantic_cache":     200 * time.Millisecond,
	"importance_scoring": 50 * time.Millisecond,
	"ast_skeleton":       30 * time.Millisecond,
	"boilerplate":        10 * time.Millisecond,
	"rle_compress":       5 * time.Millisecond,
}

// WrapWithTimeouts wraps each filter with a timeout guard.
// Filters not in the timeouts map get a default 500ms timeout.
func WrapWithTimeouts(filters []Filter, timeouts map[string]time.Duration) []Filter {
	defaultTimeout := 500 * time.Millisecond
	result := make([]Filter, len(filters))
	for i, f := range filters {
		timeout := defaultTimeout
		if t, ok := timeouts[f.Name()]; ok {
			timeout = t
		}
		result[i] = NewTimeoutGuard(f, timeout)
	}
	return result
}
