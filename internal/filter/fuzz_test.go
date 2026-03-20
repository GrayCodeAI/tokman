package filter

import (
	"strings"
	"testing"
)

// FuzzEntropyFilter tests entropy filter with random inputs
func FuzzEntropyFilter(f *testing.F) {
	f.Add("hello world test data")
	f.Add("")
	f.Add(strings.Repeat("a", 10000))
	f.Add("Error: connection failed at 192.168.1.1:8080")
	f.Add("func main() {\n\tfmt.Println(\"hello\")\n}")

	filter := NewEntropyFilter()
	f.Fuzz(func(t *testing.T, input string) {
		// Should never panic
		output, saved := filter.Apply(input, ModeMinimal)
		if saved < 0 {
			t.Errorf("negative tokens saved: %d", saved)
		}
		if len(output) > len(input)*2 {
			t.Errorf("output much larger than input: %d > %d", len(output), len(input)*2)
		}
	})
}

// FuzzPipeline tests pipeline with random inputs
func FuzzPipeline(f *testing.F) {
	f.Add("hello world test data for compression")
	f.Add("")
	f.Add("a")
	f.Add(strings.Repeat("test ", 5000))

	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	f.Fuzz(func(t *testing.T, input string) {
		// Should never panic
		output, stats := p.Process(input)
		if stats.FinalTokens < 0 {
			t.Errorf("negative final tokens: %d", stats.FinalTokens)
		}
		if stats.TotalSaved < 0 {
			t.Errorf("negative tokens saved: %d", stats.TotalSaved)
		}
		_ = output
	})
}

// FuzzIsJSON tests JSON detection with random inputs
func FuzzIsJSON(f *testing.F) {
	f.Add(`{"key": "value"}`)
	f.Add(`[1, 2, 3]`)
	f.Add(`not json`)
	f.Add(``)
	f.Add(`{`)

	f.Fuzz(func(t *testing.T, input string) {
		// Should never panic
		IsJSON(input)
		CompressJSON(input)
	})
}

// FuzzFingerprint tests fingerprinting with random inputs
func FuzzFingerprint(f *testing.F) {
	f.Add("hello world")
	f.Add("")
	f.Add(strings.Repeat("x", 100000))

	f.Fuzz(func(t *testing.T, input string) {
		fp := Fingerprint(input)
		if len(fp) != 16 {
			t.Errorf("fingerprint length = %d, want 16", len(fp))
		}
	})
}

// FuzzBM25 tests BM25 scoring with random inputs
func FuzzBM25(f *testing.F) {
	f.Add("error connection failed", "what went wrong?")
	f.Add("info server started", "server status")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, doc, query string) {
		scorer := NewBM25Scorer()
		// Should never panic
		score := scorer.Score(doc, query)
		if score < 0 {
			t.Errorf("negative score: %f", score)
		}
	})
}
