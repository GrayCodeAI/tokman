package filter

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamingProcessor_Basic(t *testing.T) {
	config := PipelineConfig{
		Mode:   ModeMinimal,
		Budget: 1000,
	}

	sp := newStreamingProcessor(config)

	// Write content in chunks
	content := strings.Repeat("This is a test line with some content.\n", 50)

	n, err := sp.Write([]byte(content))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(content) {
		t.Errorf("Write returned %d, expected %d", n, len(content))
	}

	// Flush and get compressed output
	output := sp.Flush()
	if len(output) == 0 {
		t.Error("Expected non-empty output after flush")
	}

	t.Logf("Original: %d bytes, Compressed: %d bytes", len(content), len(output))
}

func TestStreamingWriter(t *testing.T) {
	config := PipelineConfig{
		Mode:   ModeMinimal,
		Budget: 1000,
	}

	var buf bytes.Buffer
	sw := newStreamingWriter(&buf, config)

	content := strings.Repeat("Test content for streaming compression.\n", 30)

	n, err := sw.Write([]byte(content))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(content) {
		t.Errorf("Write returned %d, expected %d", n, len(content))
	}

	// Close to flush
	if err := sw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}

	t.Logf("Original: %d, Output: %d", len(content), len(output))
}

func TestStreamChannel(t *testing.T) {
	config := PipelineConfig{
		Mode:   ModeMinimal,
		Budget: 1000,
	}

	input, output := streamChannel(config)

	// Send chunks
	go func() {
		input <- strings.Repeat("Chunk 1 content.\n", 20)
		input <- strings.Repeat("Chunk 2 content.\n", 20)
		input <- strings.Repeat("Chunk 3 content.\n", 20)
		close(input)
	}()

	// Collect results
	var results []StreamChunk
	for chunk := range output {
		results = append(results, chunk)
	}

	if len(results) == 0 {
		t.Error("Expected at least one chunk")
	}

	totalSaved := 0
	for _, r := range results {
		totalSaved += r.TokensSaved
		t.Logf("Chunk: compressed=%v, saved=%d", r.IsCompressed, r.TokensSaved)
	}

	t.Logf("Total chunks: %d, Total tokens saved: %d", len(results), totalSaved)
}

func TestStreamingProcessor_MultipleWrites(t *testing.T) {
	config := PipelineConfig{
		Mode:   ModeMinimal,
		Budget: 100, // Small budget to trigger processing
	}

	sp := newStreamingProcessor(config)

	// Write in small chunks
	for i := 0; i < 10; i++ {
		chunk := strings.Repeat("Test chunk content.\n", 10)
		n, err := sp.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
		t.Logf("Write %d: %d bytes, buffer size: %d", i, n, sp.GetCurrentSize())
	}

	output := sp.Flush()
	t.Logf("Final output: %d bytes", len(output))
}
