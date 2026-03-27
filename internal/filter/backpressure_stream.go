package filter

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// BackpressureStreamProcessor is a streaming pipeline stage that applies
// compression filters to chunks of content as they arrive. It uses a
// bounded channel between producer and consumer so that producers
// automatically slow down (block on Send) when the consumer is behind —
// providing natural backpressure.
//
// Usage:
//
//	proc := NewBackpressureStreamProcessor(cfg, 8)
//	ctx, cancel := context.WithCancel(context.Background())
//	outCh := proc.Start(ctx)
//	proc.Send(ctx, chunk1)
//	proc.Send(ctx, chunk2)
//	proc.Close()
//	for result := range outCh { ... }
//	cancel()
type BackpressureStreamProcessor struct {
	config     PipelineConfig
	bufferSize int // number of pending chunks before producer blocks
	inCh       chan string
	outCh      chan BPStreamChunk

	once sync.Once
}

// BPStreamChunk is a compressed output chunk with its associated metrics.
type BPStreamChunk struct {
	Output      string
	InputTokens int
	SavedTokens int
	Latency     time.Duration
}

// NewBackpressureStreamProcessor creates a stream processor with a bounded buffer.
// bufferSize controls how many unprocessed chunks can queue before producers block.
func NewBackpressureStreamProcessor(cfg PipelineConfig, bufferSize int) *BackpressureStreamProcessor {
	if bufferSize <= 0 {
		bufferSize = 4
	}
	return &BackpressureStreamProcessor{
		config:     cfg,
		bufferSize: bufferSize,
		inCh:       make(chan string, bufferSize),
		outCh:      make(chan BPStreamChunk, bufferSize),
	}
}

// Start launches the compression goroutine and returns the output channel.
// The goroutine exits when inCh is closed or ctx is cancelled.
// Callers MUST call cancel() to ensure the goroutine is cleaned up.
func (p *BackpressureStreamProcessor) Start(ctx context.Context) <-chan BPStreamChunk {
	p.once.Do(func() {
		go p.run(ctx)
	})
	return p.outCh
}

// Send queues a chunk for compression. Blocks when the internal buffer is full,
// providing natural backpressure. Returns false if ctx is cancelled.
func (p *BackpressureStreamProcessor) Send(ctx context.Context, chunk string) bool {
	select {
	case p.inCh <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}

// Close signals that no more input will be sent. The goroutine will drain
// remaining chunks and close outCh.
func (p *BackpressureStreamProcessor) Close() {
	close(p.inCh)
}

// run is the internal compression goroutine. It reads from inCh, applies
// the pipeline, and writes to outCh. It exits when inCh is closed or ctx
// is cancelled, always closing outCh on exit.
func (p *BackpressureStreamProcessor) run(ctx context.Context) {
	defer close(p.outCh)

	coordinator := NewPipelineCoordinator(p.config)

	for {
		select {
		case chunk, ok := <-p.inCh:
			if !ok {
				return // inCh closed — drain complete
			}
			p.processChunk(ctx, coordinator, chunk)
		case <-ctx.Done():
			return
		}
	}
}

// processChunk applies the pipeline to one chunk and sends the result.
func (p *BackpressureStreamProcessor) processChunk(ctx context.Context, coord *PipelineCoordinator, chunk string) {
	inToks := core.EstimateTokens(chunk)
	start := time.Now()

	output, stats := coord.Process(chunk)
	elapsed := time.Since(start)

	saved := 0
	if stats != nil {
		saved = stats.OriginalTokens - stats.FinalTokens
	}
	if saved < 0 {
		saved = 0
	}

	result := BPStreamChunk{
		Output:      output,
		InputTokens: inToks,
		SavedTokens: saved,
		Latency:     elapsed,
	}

	select {
	case p.outCh <- result:
	case <-ctx.Done():
	}
}

// CollectAll is a convenience function that sends all lines to the processor
// and returns the full compressed output. Useful for testing.
func CollectAll(ctx context.Context, cfg PipelineConfig, input string) (string, int) {
	proc := NewBackpressureStreamProcessor(cfg, 8)
	outCh := proc.Start(ctx)

	// Send line chunks asynchronously
	go func() {
		lines := strings.Split(input, "\n")
		chunkSize := 50
		for i := 0; i < len(lines); i += chunkSize {
			end := i + chunkSize
			if end > len(lines) {
				end = len(lines)
			}
			chunk := strings.Join(lines[i:end], "\n")
			if !proc.Send(ctx, chunk) {
				break
			}
		}
		proc.Close()
	}()

	var sb strings.Builder
	totalSaved := 0
	for result := range outCh {
		sb.WriteString(result.Output)
		totalSaved += result.SavedTokens
	}
	return sb.String(), totalSaved
}
