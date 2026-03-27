package filter

// otel_trace.go — Task #90: OpenTelemetry tracing for pipeline stages.
//
// go.opentelemetry.io/otel is NOT in go.mod, so this file implements a
// zero-dependency shim that mirrors the otel tracing interface.  When the
// project eventually adds the real otel packages the shim can be swapped out
// without changing call-sites.
//
// Three implementations are provided:
//   - NoopTracer  — does nothing (safe default; zero allocation on hot path)
//   - LogTracer   — writes structured timing lines to stderr
//   - PipelineTracer — wraps either of the above and is the type callers hold

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// SpanEnder — the handle returned by every Start* call.
// ---------------------------------------------------------------------------

// SpanEnder is the handle returned by StartPipelineSpan and StartFilterSpan.
// Callers must call End() exactly once (defer is idiomatic).
type SpanEnder interface {
	// End marks the span as finished and records its duration.
	End()
	// SetError records an error on the span.
	SetError(err error)
	// SetAttribute attaches a key/value pair to the span.
	SetAttribute(key string, value any)
}

// ---------------------------------------------------------------------------
// Tracer — the low-level interface both NoopTracer and LogTracer satisfy.
// ---------------------------------------------------------------------------

// Tracer is the interface that NoopTracer and LogTracer implement.
// PipelineTracer delegates to whichever Tracer is configured.
type Tracer interface {
	// StartSpan begins a new span with the given name and returns
	// an updated context and a SpanEnder.  The context may carry
	// parent-span state so that child spans nest correctly.
	StartSpan(ctx context.Context, spanName string) (context.Context, SpanEnder)
}

// ---------------------------------------------------------------------------
// spanKey — unexported context key for nesting spans.
// ---------------------------------------------------------------------------

type spanKey struct{}

// ---------------------------------------------------------------------------
// noopSpan — SpanEnder that does absolutely nothing.
// ---------------------------------------------------------------------------

type noopSpan struct{}

func (noopSpan) End()                            {}
func (noopSpan) SetError(_ error)                {}
func (noopSpan) SetAttribute(_ string, _ any)    {}

// ---------------------------------------------------------------------------
// NoopTracer — zero-cost tracer (default).
// ---------------------------------------------------------------------------

// NoopTracer is a Tracer that performs no work and allocates nothing on the
// hot path.  It is the default implementation used by PipelineTracer.
type NoopTracer struct{}

// StartSpan returns the context unchanged and a no-op SpanEnder.
func (NoopTracer) StartSpan(ctx context.Context, _ string) (context.Context, SpanEnder) {
	return ctx, noopSpan{}
}

// ---------------------------------------------------------------------------
// logSpan — SpanEnder that records timing to stderr.
// ---------------------------------------------------------------------------

// spanID is a monotonically increasing counter used to correlate start/end
// log lines without importing a UUID library.
var spanIDCounter atomic.Uint64

type logSpan struct {
	id        uint64
	name      string
	parentID  uint64
	startedAt time.Time
	attrs     []string // key=value pairs accumulated via SetAttribute
	spanErr   error
}

func newLogSpan(name string, parentID uint64) *logSpan {
	return &logSpan{
		id:        spanIDCounter.Add(1),
		name:      name,
		parentID:  parentID,
		startedAt: time.Now(),
	}
}

func (s *logSpan) End() {
	elapsed := time.Since(s.startedAt)
	errStr := "<nil>"
	if s.spanErr != nil {
		errStr = s.spanErr.Error()
	}
	attrStr := ""
	if len(s.attrs) > 0 {
		attrStr = " attrs=["
		for i, a := range s.attrs {
			if i > 0 {
				attrStr += " "
			}
			attrStr += a
		}
		attrStr += "]"
	}
	parentStr := ""
	if s.parentID != 0 {
		parentStr = fmt.Sprintf(" parent=%d", s.parentID)
	}
	fmt.Fprintf(os.Stderr,
		"[trace] span=%d%s name=%q duration=%s err=%s%s\n",
		s.id, parentStr, s.name, elapsed, errStr, attrStr,
	)
}

func (s *logSpan) SetError(err error) {
	s.spanErr = err
}

func (s *logSpan) SetAttribute(key string, value any) {
	s.attrs = append(s.attrs, fmt.Sprintf("%s=%v", key, value))
}

// ---------------------------------------------------------------------------
// LogTracer — writes structured timing lines to stderr.
// ---------------------------------------------------------------------------

// LogTracer is a Tracer that emits one "[trace]" line per span to stderr.
// It supports simple parent-child nesting via context propagation.
type LogTracer struct{}

// StartSpan begins a new span, stores it in the context for child nesting,
// and returns a *logSpan as the SpanEnder.
func (LogTracer) StartSpan(ctx context.Context, name string) (context.Context, SpanEnder) {
	var parentID uint64
	if p, ok := ctx.Value(spanKey{}).(*logSpan); ok && p != nil {
		parentID = p.id
	}
	span := newLogSpan(name, parentID)
	fmt.Fprintf(os.Stderr,
		"[trace] span=%d name=%q started\n",
		span.id, span.name,
	)
	return context.WithValue(ctx, spanKey{}, span), span
}

// ---------------------------------------------------------------------------
// PipelineTracer — the public facade callers use.
// ---------------------------------------------------------------------------

// PipelineTracer wraps a Tracer and exposes domain-specific helpers for the
// tokman compression pipeline.  The zero value uses NoopTracer (safe default).
//
// Example — wrapping a single filter stage:
//
//	ctx, span := pt.StartFilterSpan(ctx, "entropy")
//	defer span.End()
//	output, saved := entropyFilter.Apply(input, mode)
//	span.SetAttribute("tokens_saved", saved)
type PipelineTracer struct {
	inner Tracer
}

// NewPipelineTracer creates a PipelineTracer backed by the given Tracer.
// Pass NoopTracer{} for production (zero overhead) or LogTracer{} for
// development / debugging.
func NewPipelineTracer(t Tracer) *PipelineTracer {
	if t == nil {
		t = NoopTracer{}
	}
	return &PipelineTracer{inner: t}
}

// StartPipelineSpan begins a top-level span representing a full pipeline run.
// pipelineID should be a short identifier (e.g. a request ID or "default").
//
// Returns an updated context (propagate this to sub-calls) and a SpanEnder.
// Callers must call End() on the SpanEnder when the pipeline completes.
func (pt *PipelineTracer) StartPipelineSpan(ctx context.Context, pipelineID string) (context.Context, SpanEnder) {
	name := "pipeline/" + pipelineID
	return pt.inner.StartSpan(ctx, name)
}

// StartFilterSpan begins a child span representing a single filter stage.
// filterName should match Filter.Name() (e.g. "entropy", "perplexity").
//
// Returns an updated context and a SpanEnder.
// Callers must call End() on the SpanEnder when the filter stage completes.
func (pt *PipelineTracer) StartFilterSpan(ctx context.Context, filterName string) (context.Context, SpanEnder) {
	name := "filter/" + filterName
	return pt.inner.StartSpan(ctx, name)
}
