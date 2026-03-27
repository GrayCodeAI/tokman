package filter

import (
	"bufio"
	"io"
	"strings"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// StreamingProcessor provides real-time compression for streaming content.
// Designed for chat agents and long-running sessions where content arrives
// incrementally and needs to be compressed on-the-fly.
type StreamingProcessor struct {
	config      PipelineConfig
	coordinator *PipelineCoordinator
	buffer      strings.Builder
	mu          sync.Mutex
	chunkSize   int         // Process when buffer reaches this size
	lineChan    chan string // For true line-by-line processing
}

// newStreamingProcessor creates a streaming processor for incremental compression
func newStreamingProcessor(config PipelineConfig) *StreamingProcessor {
	chunkSize := config.Budget * 4 // ~4 chars per token
	if chunkSize <= 0 {
		chunkSize = 4096
	}
	return &StreamingProcessor{
		config:      config,
		coordinator: NewPipelineCoordinator(config),
		chunkSize:   chunkSize,
		lineChan:    make(chan string, 100),
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

// LineProcessor processes content line-by-line for true streaming compression.
// P3.1: Instead of buffering until a threshold, each line is compressed individually
// as it arrives, providing lower latency for real-time CLI output.
type LineProcessor struct {
	config      PipelineConfig
	coordinator *PipelineCoordinator
	mu          sync.Mutex
	lineCount   int
	totalSaved  int
}

// newLineProcessor creates a line-by-line streaming processor
func newLineProcessor(config PipelineConfig) *LineProcessor {
	return &LineProcessor{
		config:      config,
		coordinator: NewPipelineCoordinator(config),
	}
}

// ProcessLine processes a single line and returns compressed output
func (lp *LineProcessor) ProcessLine(line string) (string, int) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	if strings.TrimSpace(line) == "" {
		return "\n", 0
	}

	originalTokens := core.EstimateTokens(line)

	// Apply lightweight per-line compression (skip heavy layers)
	filtered := lp.applyLightweightFilter(line)
	saved := originalTokens - core.EstimateTokens(filtered)

	lp.lineCount++
	lp.totalSaved += saved

	return filtered, saved
}

// ProcessReader reads from an io.Reader and compresses line-by-line
func (lp *LineProcessor) ProcessReader(r io.Reader) (string, int) {
	scanner := bufio.NewScanner(r)
	var result strings.Builder
	totalSaved := 0

	for scanner.Scan() {
		line := scanner.Text()
		compressed, saved := lp.ProcessLine(line)
		result.WriteString(compressed)
		result.WriteString("\n")
		totalSaved += saved
	}

	if err := scanner.Err(); err != nil {
		return strings.TrimSpace(result.String()), totalSaved
	}

	return strings.TrimSpace(result.String()), totalSaved
}

// applyLightweightFilter applies fast per-line compression
func (lp *LineProcessor) applyLightweightFilter(line string) string {
	// Only apply fast filters per-line (skip expensive pipeline)
	filtered := line

	// ANSI strip
	filtered = stripANSI(filtered)

	// Noise removal
	if isNoiseLine(filtered) {
		return ""
	}

	return filtered
}

// StreamingWriter wraps an io.Writer and compresses output in chunks
type StreamingWriter struct {
	processor *StreamingProcessor
	target    io.Writer
}

// newStreamingWriter creates a writer that compresses before writing
func newStreamingWriter(target io.Writer, config PipelineConfig) *StreamingWriter {
	return &StreamingWriter{
		processor: newStreamingProcessor(config),
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

// streamChannel creates a channel-based streaming processor
func streamChannel(config PipelineConfig) (chan<- string, <-chan StreamChunk) {
	input := make(chan string, 100)
	output := make(chan StreamChunk, 100)

	processor := newStreamingProcessor(config)

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

// GetStats returns streaming statistics
func (lp *LineProcessor) GetStats() (lines int, totalSaved int) {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	return lp.lineCount, lp.totalSaved
}

// stripANSI removes ANSI escape codes from a line
func stripANSI(line string) string {
	result := make([]byte, 0, len(line))
	i := 0
	for i < len(line) {
		if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
			// Skip ANSI escape sequence
			i += 2
			for i < len(line) && !((line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z')) {
				i++
			}
			if i < len(line) {
				i++ // Skip the letter
			}
		} else {
			result = append(result, line[i])
			i++
		}
	}
	return string(result)
}

// isNoiseLine checks if a line is noise (progress bars, spinners, etc.)
func isNoiseLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 {
		return false
	}

	// Progress bar patterns
	if strings.Contains(trimmed, "[") && strings.Contains(trimmed, "]") &&
		(strings.Contains(trimmed, "=") || strings.Contains(trimmed, "#") ||
			strings.Contains(trimmed, ">")) {
		return true
	}

	// Spinner characters
	spinners := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	for _, r := range trimmed {
		for _, s := range spinners {
			if r == s {
				return true
			}
		}
	}

	return false
}
