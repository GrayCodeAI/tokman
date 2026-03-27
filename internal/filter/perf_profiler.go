package filter

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"
)

// FilterProfiler wraps a list of filters and collects detailed performance
// metrics: CPU time, memory allocations, and GC pressure per filter.
//
// Usage:
//
//	p := NewFilterProfiler(filters...)
//	results := p.Profile(input, mode, runs)
//	fmt.Print(results.FormatReport())
type FilterProfiler struct {
	filters []Filter
}

// ProfileSample is one filter's metrics from a single run.
type ProfileSample struct {
	FilterName  string
	LatencyNs   int64
	AllocBytes  uint64
	AllocCount  uint64
	TokensSaved int
}

// ProfileReport aggregates samples across multiple runs.
type ProfileReport struct {
	Samples  []ProfileAggregate
	TotalRuns int
}

// ProfileAggregate holds aggregated stats for one filter across N runs.
type ProfileAggregate struct {
	FilterName       string
	AvgLatencyUs     float64
	P95LatencyUs     float64
	TotalAllocBytes  uint64
	TotalAllocCount  uint64
	AvgAllocBytesRun uint64
	TotalTokensSaved int
}

// NewFilterProfiler creates a profiler for the given filters.
func NewFilterProfiler(filters ...Filter) *FilterProfiler {
	return &FilterProfiler{filters: filters}
}

// Profile runs each filter `runs` times against `input` and aggregates metrics.
func (p *FilterProfiler) Profile(input string, mode Mode, runs int) ProfileReport {
	if runs <= 0 {
		runs = 5
	}

	// Per-filter samples across runs
	allSamples := make(map[string][]ProfileSample, len(p.filters))

	for _, f := range p.filters {
		name := f.Name()
		samples := make([]ProfileSample, 0, runs)
		for r := 0; r < runs; r++ {
			s := p.profileOne(f, input, mode)
			s.FilterName = name
			samples = append(samples, s)
		}
		allSamples[name] = samples
	}

	// Aggregate
	var aggregates []ProfileAggregate
	for _, f := range p.filters {
		name := f.Name()
		samples := allSamples[name]
		agg := aggregate(name, samples)
		aggregates = append(aggregates, agg)
	}

	return ProfileReport{Samples: aggregates, TotalRuns: runs}
}

// profileOne profiles a single filter run using runtime.ReadMemStats.
func (p *FilterProfiler) profileOne(f Filter, input string, mode Mode) ProfileSample {
	var msBefore, msAfter runtime.MemStats
	runtime.GC() // stabilize GC before measurement
	runtime.ReadMemStats(&msBefore)

	start := time.Now()
	_, saved := f.Apply(input, mode)
	elapsed := time.Since(start).Nanoseconds()

	runtime.ReadMemStats(&msAfter)

	return ProfileSample{
		LatencyNs:   elapsed,
		AllocBytes:  msAfter.TotalAlloc - msBefore.TotalAlloc,
		AllocCount:  msAfter.Mallocs - msBefore.Mallocs,
		TokensSaved: saved,
	}
}

func aggregate(name string, samples []ProfileSample) ProfileAggregate {
	if len(samples) == 0 {
		return ProfileAggregate{FilterName: name}
	}

	latencies := make([]int64, len(samples))
	totalAlloc := uint64(0)
	totalAllocCount := uint64(0)
	totalSaved := 0

	for i, s := range samples {
		latencies[i] = s.LatencyNs
		totalAlloc += s.AllocBytes
		totalAllocCount += s.AllocCount
		totalSaved += s.TokensSaved
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	sumLat := int64(0)
	for _, l := range latencies {
		sumLat += l
	}

	n := len(samples)
	return ProfileAggregate{
		FilterName:       name,
		AvgLatencyUs:     float64(sumLat) / float64(n) / 1000,
		P95LatencyUs:     float64(latencies[n*95/100]) / 1000,
		TotalAllocBytes:  totalAlloc,
		TotalAllocCount:  totalAllocCount,
		AvgAllocBytesRun: totalAlloc / uint64(n),
		TotalTokensSaved: totalSaved,
	}
}

// FormatReport returns a human-readable performance table.
func (r ProfileReport) FormatReport() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Filter Performance Profile (%d runs) ===\n\n", r.TotalRuns))
	sb.WriteString(fmt.Sprintf("%-30s  %10s  %10s  %12s  %10s  %10s\n",
		"Filter", "AvgLat(µs)", "P95Lat(µs)", "AllocBytes/run", "AllocCount", "TokSaved"))
	sb.WriteString(strings.Repeat("-", 90) + "\n")

	// Sort by avg latency descending (slowest first)
	sorted := make([]ProfileAggregate, len(r.Samples))
	copy(sorted, r.Samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AvgLatencyUs > sorted[j].AvgLatencyUs
	})

	for _, agg := range sorted {
		sb.WriteString(fmt.Sprintf("%-30s  %10.1f  %10.1f  %12d  %10d  %10d\n",
			truncStr(agg.FilterName, 30),
			agg.AvgLatencyUs,
			agg.P95LatencyUs,
			agg.AvgAllocBytesRun,
			agg.TotalAllocCount,
			agg.TotalTokensSaved,
		))
	}

	// Top-5 slowest
	sb.WriteString("\n--- Top 5 Slowest Filters ---\n")
	for i := 0; i < 5 && i < len(sorted); i++ {
		sb.WriteString(fmt.Sprintf("  %d. %s (avg %.1f µs)\n", i+1, sorted[i].FilterName, sorted[i].AvgLatencyUs))
	}

	// Top-5 most allocating (sort by alloc)
	allocSorted := make([]ProfileAggregate, len(r.Samples))
	copy(allocSorted, r.Samples)
	sort.Slice(allocSorted, func(i, j int) bool {
		return allocSorted[i].AvgAllocBytesRun > allocSorted[j].AvgAllocBytesRun
	})
	sb.WriteString("\n--- Top 5 Most Allocating Filters ---\n")
	for i := 0; i < 5 && i < len(allocSorted); i++ {
		sb.WriteString(fmt.Sprintf("  %d. %s (avg %d B/run)\n",
			i+1, allocSorted[i].FilterName, allocSorted[i].AvgAllocBytesRun))
	}

	return sb.String()
}
