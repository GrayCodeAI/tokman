package filter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// BenchmarkResult holds the measured performance of a single filter or pipeline run.
type BenchmarkResult struct {
	FilterName   string
	Mode         Mode
	InputTokens  int
	OutputTokens int
	TokensSaved  int
	Reduction    float64 // fraction saved: 0.0–1.0
	LatencyNs    int64   // nanoseconds for Apply() call
	ThroughputMB float64 // MB/s based on input size and latency
}

// BenchmarkReport aggregates results across multiple filters and content types.
type BenchmarkReport struct {
	Results      []BenchmarkResult
	TotalInputT  int
	TotalOutputT int
	OverallReduction float64
	P50LatencyNs int64
	P95LatencyNs int64
	P99LatencyNs int64
}

// FilterBenchmarkHarness runs filters against a fixed set of test inputs
// and collects performance and quality metrics.
type FilterBenchmarkHarness struct {
	filters []Filter
	inputs  []BenchmarkInput
	modes   []Mode
}

// BenchmarkInput is a labeled test input for the harness.
type BenchmarkInput struct {
	Name    string
	Content string
}

// NewFilterBenchmarkHarness creates a benchmark harness.
func NewFilterBenchmarkHarness() *FilterBenchmarkHarness {
	return &FilterBenchmarkHarness{
		modes: []Mode{ModeMinimal, ModeAggressive},
	}
}

// AddFilter registers a filter to benchmark.
func (h *FilterBenchmarkHarness) AddFilter(f Filter) *FilterBenchmarkHarness {
	h.filters = append(h.filters, f)
	return h
}

// AddInput registers a named test input.
func (h *FilterBenchmarkHarness) AddInput(name, content string) *FilterBenchmarkHarness {
	h.inputs = append(h.inputs, BenchmarkInput{Name: name, Content: content})
	return h
}

// AddDefaultFilters adds the standard set of production filters.
func (h *FilterBenchmarkHarness) AddDefaultFilters() *FilterBenchmarkHarness {
	h.filters = append(h.filters,
		NewRLECompressFilter(),
		NewHTMLCompressFilter(),
		NewShellOutputFilter(),
		NewKeywordExtractFilter(),
		NewImportanceScoringFilter(),
		NewNumericCompressFilter(),
		NewErrorDedupFilter(),
		NewASTSkeletonFilter(),
		NewProtoCompressFilter(),
		NewSmartTruncateFilter(),
	)
	return h
}

// Run executes all filters against all inputs and returns the full report.
func (h *FilterBenchmarkHarness) Run() *BenchmarkReport {
	var results []BenchmarkResult
	var latencies []int64

	for _, inp := range h.inputs {
		for _, f := range h.filters {
			for _, mode := range h.modes {
				r := h.runOne(f, inp.Content, mode)
				r.FilterName = f.Name() + "/" + inp.Name
				results = append(results, r)
				latencies = append(latencies, r.LatencyNs)
			}
		}
	}

	report := &BenchmarkReport{Results: results}
	if len(results) > 0 {
		totalIn, totalOut := 0, 0
		for _, r := range results {
			totalIn += r.InputTokens
			totalOut += r.OutputTokens
		}
		report.TotalInputT = totalIn
		report.TotalOutputT = totalOut
		if totalIn > 0 {
			report.OverallReduction = 1.0 - float64(totalOut)/float64(totalIn)
		}
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		report.P50LatencyNs = latencies[len(latencies)*50/100]
		report.P95LatencyNs = latencies[len(latencies)*95/100]
		report.P99LatencyNs = latencies[min99(len(latencies)*99/100, len(latencies)-1)]
	}
	return report
}

// runOne benchmarks a single filter/input/mode combination.
func (h *FilterBenchmarkHarness) runOne(f Filter, input string, mode Mode) BenchmarkResult {
	inputTokens := core.EstimateTokens(input)
	inputBytes := len(input)

	start := time.Now()
	output, _ := f.Apply(input, mode)
	elapsed := time.Since(start).Nanoseconds()

	outputTokens := core.EstimateTokens(output)
	saved := inputTokens - outputTokens
	reduction := 0.0
	if inputTokens > 0 {
		reduction = float64(saved) / float64(inputTokens)
	}
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(inputBytes) / float64(elapsed) * 1e9 / (1024 * 1024) // MB/s
	}

	return BenchmarkResult{
		Mode:         mode,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TokensSaved:  saved,
		Reduction:    reduction,
		LatencyNs:    elapsed,
		ThroughputMB: throughput,
	}
}

// FormatReport formats a BenchmarkReport as a human-readable table.
func (r *BenchmarkReport) FormatReport() string {
	var sb strings.Builder
	sb.WriteString("=== Pipeline Benchmark Report ===\n\n")
	sb.WriteString(fmt.Sprintf("%-40s  %-10s  %8s  %8s  %7s  %9s  %9s\n",
		"Filter/Input", "Mode", "In(tok)", "Out(tok)", "Saved%", "Lat(µs)", "MB/s"))
	sb.WriteString(strings.Repeat("-", 100) + "\n")

	for _, res := range r.Results {
		sb.WriteString(fmt.Sprintf("%-40s  %-10s  %8d  %8d  %6.1f%%  %9.1f  %9.1f\n",
			truncStr(res.FilterName, 40),
			string(res.Mode),
			res.InputTokens,
			res.OutputTokens,
			res.Reduction*100,
			float64(res.LatencyNs)/1000,
			res.ThroughputMB,
		))
	}

	sb.WriteString(strings.Repeat("-", 100) + "\n")
	sb.WriteString(fmt.Sprintf("\nOverall token reduction: %.1f%%\n", r.OverallReduction*100))
	sb.WriteString(fmt.Sprintf("Latency p50/p95/p99: %.1f / %.1f / %.1f µs\n",
		float64(r.P50LatencyNs)/1000,
		float64(r.P95LatencyNs)/1000,
		float64(r.P99LatencyNs)/1000,
	))
	return sb.String()
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func min99(a, b int) int {
	if a < b {
		return a
	}
	return b
}
