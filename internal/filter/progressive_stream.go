package filter

import (
	"io"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ProgressiveStreamer delivers compressed output progressively as chunks are
// ready, allowing consumers to start processing before the full pipeline completes.
// Task #140: Output streaming with progressive delivery.
type ProgressiveStreamer struct {
	pipeline    *PipelineCoordinator
	chunkSize   int  // chars per chunk delivered to writer
	flushDelay  time.Duration
}

// ProgressEvent is a single delivery event from the streamer.
type ProgressEvent struct {
	// Chunk is the text delivered in this event.
	Chunk string
	// Cumulative is total text delivered so far.
	CumulativeTokens int
	// Done is true for the final event.
	Done bool
	// TotalSaved is the token reduction achieved (set only on final event).
	TotalSaved int
}

// NewProgressiveStreamer creates a streamer backed by the given pipeline.
// chunkSize is the approximate character count per delivery event (0 = line-by-line).
func NewProgressiveStreamer(pipeline *PipelineCoordinator, chunkSize int) *ProgressiveStreamer {
	if chunkSize <= 0 {
		chunkSize = 0 // signals line-by-line mode
	}
	return &ProgressiveStreamer{
		pipeline:   pipeline,
		chunkSize:  chunkSize,
		flushDelay: 10 * time.Millisecond,
	}
}

// Stream compresses input and writes output progressively to w.
// It returns when all output has been written or w returns an error.
func (s *ProgressiveStreamer) Stream(input string, mode Mode, w io.Writer) error {
	// Run the full pipeline first (pipeline is not truly incremental today).
	compressed, _ := s.pipeline.Process(input)

	// Deliver the compressed output progressively.
	if s.chunkSize > 0 {
		return s.streamBySize(compressed, w)
	}
	return s.streamByLine(compressed, w)
}

// StreamEvents compresses input and sends ProgressEvent values to ch.
// Close of ch signals completion. The caller should drain ch until it's closed.
func (s *ProgressiveStreamer) StreamEvents(input string, mode Mode) <-chan ProgressEvent {
	ch := make(chan ProgressEvent, 16)
	go func() {
		defer close(ch)
		origTokens := core.EstimateTokens(input)
		compressed, _ := s.pipeline.Process(input)

		lines := strings.Split(compressed, "\n")
		cumulative := 0
		var delivered strings.Builder

		for i, line := range lines {
			delivered.WriteString(line)
			if i < len(lines)-1 {
				delivered.WriteByte('\n')
			}
			lineTokens := core.EstimateTokens(line)
			cumulative += lineTokens

			isDone := i == len(lines)-1
			saved := 0
			if isDone {
				saved = origTokens - core.EstimateTokens(compressed)
			}
			ch <- ProgressEvent{
				Chunk:            line + "\n",
				CumulativeTokens: cumulative,
				Done:             isDone,
				TotalSaved:       saved,
			}
			if !isDone {
				time.Sleep(s.flushDelay)
			}
		}
	}()
	return ch
}

func (s *ProgressiveStreamer) streamByLine(text string, w io.Writer) error {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if _, err := io.WriteString(w, line+"\n"); err != nil {
			return err
		}
		time.Sleep(s.flushDelay)
	}
	return nil
}

func (s *ProgressiveStreamer) streamBySize(text string, w io.Writer) error {
	for len(text) > 0 {
		end := s.chunkSize
		if end > len(text) {
			end = len(text)
		}
		if _, err := io.WriteString(w, text[:end]); err != nil {
			return err
		}
		text = text[end:]
		if len(text) > 0 {
			time.Sleep(s.flushDelay)
		}
	}
	return nil
}
