package filter

import (
	"io"
	"strings"
	"sync"
)

// StreamingProcessor provides real-time compression for streaming content.
// Designed for chat agents and long-running sessions where content arrives
// incrementally and needs to be compressed on-the-fly.
type StreamingProcessor struct {
	config      PipelineConfig
	coordinator *PipelineCoordinator
	buffer      strings.Builder
	mu          sync.Mutex
	chunkSize   int // Process when buffer reaches this size
}

// NewStreamingProcessor creates a streaming processor for incremental compression
func NewStreamingProcessor(config PipelineConfig) *StreamingProcessor {
	return &StreamingProcessor{
		config:      config,
		coordinator: NewPipelineCoordinator(config),
		chunkSize:   config.Budget * 4, // ~4 chars per token
	}
}

// Write adds content to the buffer and returns compressed output if threshold reached
func (s *StreamingProcessor) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, err := s.buffer.Write(data)
	if err != nil {
		return n, err
	}

	// Check if we should process
	if s.buffer.Len() >= s.chunkSize {
		// Process and clear buffer
		content := s.buffer.String()
		s.buffer.Reset()

		// Apply compression
		compressed, _ := s.coordinator.Process(content)
		// Return compressed content via callback or channel
		// Caller should check HasPendingOutput()
		s.buffer.WriteString(compressed)
	}

	return n, nil
}

// Flush processes remaining buffer content and returns final compressed output
func (s *StreamingProcessor) Flush() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.buffer.Len() == 0 {
		return ""
	}

	content := s.buffer.String()
	s.buffer.Reset()
	compressed, _ := s.coordinator.Process(content)
	return compressed
}

// GetCurrentSize returns current buffer size in bytes
func (s *StreamingProcessor) GetCurrentSize() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buffer.Len()
}

// StreamingWriter wraps an io.Writer and compresses output in chunks
type StreamingWriter struct {
	processor *StreamingProcessor
	target    io.Writer
}

// NewStreamingWriter creates a writer that compresses before writing
func NewStreamingWriter(target io.Writer, config PipelineConfig) *StreamingWriter {
	return &StreamingWriter{
		processor: NewStreamingProcessor(config),
		target:    target,
	}
}

// Write implements io.Writer
func (w *StreamingWriter) Write(p []byte) (int, error) {
	return w.processor.Write(p)
}

// Close flushes remaining content and writes to target
func (w *StreamingWriter) Close() error {
	compressed := w.processor.Flush()
	if compressed != "" {
		_, err := w.target.Write([]byte(compressed))
		return err
	}
	return nil
}

// StreamChunk represents a chunk of streaming content with metadata
type StreamChunk struct {
	Content      string
	IsCompressed bool
	TokensSaved  int
}

// StreamChannel creates a channel-based streaming processor
func StreamChannel(config PipelineConfig) (chan<- string, <-chan StreamChunk) {
	input := make(chan string, 100)
	output := make(chan StreamChunk, 100)

	processor := NewStreamingProcessor(config)

	go func() {
		defer close(output)

		for content := range input {
			compressed, stats := processor.coordinator.Process(content)
			output <- StreamChunk{
				Content:      compressed,
				IsCompressed: stats.ReductionPercent > 0,
				TokensSaved:  stats.TotalSaved,
			}
		}

		// Flush remaining
		if remaining := processor.Flush(); remaining != "" {
			output <- StreamChunk{
				Content:      remaining,
				IsCompressed: false,
			}
		}
	}()

	return input, output
}
