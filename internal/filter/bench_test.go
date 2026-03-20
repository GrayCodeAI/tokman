package filter

import (
	"strings"
	"testing"
)

// Benchmark data generators

func benchmarkInput(size int) string {
	return strings.Repeat("the quick brown fox jumps over the lazy dog. Error: connection failed at 192.168.1.1:8080. INFO: retrying in 5 seconds. ", size)
}

func benchmarkCodeInput(size int) string {
	base := `func main() {
	// Initialize the service
	svc := NewService(config)
	if err := svc.Start(); err != nil {
		log.Fatal("failed to start:", err)
	}
	defer svc.Close()
	
	// Handle requests
	for req := range svc.Requests() {
		resp, err := svc.Process(req)
		if err != nil {
			log.Error("processing failed:", err)
			continue
		}
		req.Reply(resp)
	}
}
`
	return strings.Repeat(base, size)
}

func benchmarkJSONInput(size int) string {
	base := `{"name": "test", "value": 42, "nested": {"key": "value", "items": [1, 2, 3]}, "status": "ok"}`
	return strings.Repeat(base, size)
}

// Pipeline benchmarks

func BenchmarkPipelineMinimal(b *testing.B) {
	input := benchmarkInput(50)
	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(input)
	}
}

func BenchmarkPipelineAggressive(b *testing.B) {
	input := benchmarkInput(50)
	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeAggressive})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(input)
	}
}

func BenchmarkPipelineWithBudget(b *testing.B) {
	input := benchmarkInput(100)
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:   ModeMinimal,
		Budget: 500,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(input)
	}
}

func BenchmarkPipelineFast(b *testing.B) {
	input := benchmarkInput(50)
	cfg := PresetConfig(PresetFast, ModeMinimal)
	p := NewPipelineCoordinator(cfg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(input)
	}
}

func BenchmarkPipelineBalanced(b *testing.B) {
	input := benchmarkInput(50)
	cfg := PresetConfig(PresetBalanced, ModeMinimal)
	p := NewPipelineCoordinator(cfg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(input)
	}
}

func BenchmarkPipelineFullNew(b *testing.B) {
	input := benchmarkInput(50)
	cfg := PresetConfig(PresetFull, ModeMinimal)
	p := NewPipelineCoordinator(cfg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(input)
	}
}

// Individual layer benchmarks

func BenchmarkEntropyFilter(b *testing.B) {
	input := benchmarkInput(100)
	f := NewEntropyFilter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Apply(input, ModeMinimal)
	}
}

func BenchmarkEntropyFilterAggressive(b *testing.B) {
	input := benchmarkInput(100)
	f := NewEntropyFilter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Apply(input, ModeAggressive)
	}
}

func BenchmarkBM25Scoring(b *testing.B) {
	lines := strings.Split(benchmarkInput(50), "\n")
	scorer := NewBM25Scorer()
	scorer.Fit(lines)
	query := "error connection failed"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, line := range lines {
			scorer.Score(line, query)
		}
	}
}

func BenchmarkAttentionSink(b *testing.B) {
	input := benchmarkInput(200)
	f := NewAttentionSinkFilter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Apply(input, ModeMinimal)
	}
}

func BenchmarkCompressJSON(b *testing.B) {
	input := benchmarkJSONInput(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompressJSON(input)
	}
}

func BenchmarkIsJSON(b *testing.B) {
	input := benchmarkJSONInput(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsJSON(input)
	}
}

func BenchmarkFingerprint(b *testing.B) {
	input := benchmarkInput(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Fingerprint(input)
	}
}

func BenchmarkEstimateTokensNew(b *testing.B) {
	input := benchmarkInput(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(input)
	}
}

// Scaling benchmarks

func BenchmarkPipelineScaling10(b *testing.B) {
	benchmarkScaling(b, 10)
}
func BenchmarkPipelineScaling50(b *testing.B) {
	benchmarkScaling(b, 50)
}
func BenchmarkPipelineScaling100(b *testing.B) {
	benchmarkScaling(b, 100)
}
func BenchmarkPipelineScaling500(b *testing.B) {
	benchmarkScaling(b, 500)
}

func benchmarkScaling(b *testing.B, size int) {
	input := benchmarkInput(size)
	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(input)
	}
}
