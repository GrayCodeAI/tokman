package filter

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// RealTimeDashboard tracks live compression metrics and provides formatted views.
// Task #89: Real-time compression ratio dashboard.
type RealTimeDashboard struct {
	mu       sync.RWMutex
	metrics  []DashboardMetric
	maxItems int
	startAt  time.Time
}

// DashboardMetric is a single compression event recorded by the dashboard.
type DashboardMetric struct {
	Timestamp    time.Time
	FilterName   string
	Mode         string
	OrigTokens   int
	FinalTokens  int
	Saved        int
	ReductionPct float64
	LatencyMs    int64
}

// NewRealTimeDashboard creates a dashboard that retains the last maxItems events.
func NewRealTimeDashboard(maxItems int) *RealTimeDashboard {
	if maxItems <= 0 {
		maxItems = 1000
	}
	return &RealTimeDashboard{
		metrics:  make([]DashboardMetric, 0, maxItems),
		maxItems: maxItems,
		startAt:  time.Now(),
	}
}

// Record adds a compression metric to the dashboard.
func (d *RealTimeDashboard) Record(m DashboardMetric) {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	d.mu.Lock()
	d.metrics = append(d.metrics, m)
	if len(d.metrics) > d.maxItems {
		d.metrics = d.metrics[len(d.metrics)-d.maxItems:]
	}
	d.mu.Unlock()
}

// RecordFilter is a convenience method to record a filter's Apply result.
func (d *RealTimeDashboard) RecordFilter(filterName string, mode Mode, orig, final int, latency time.Duration) {
	var pct float64
	if orig > 0 {
		pct = float64(orig-final) / float64(orig) * 100
	}
	d.Record(DashboardMetric{
		FilterName:   filterName,
		Mode:         string(mode),
		OrigTokens:   orig,
		FinalTokens:  final,
		Saved:        orig - final,
		ReductionPct: pct,
		LatencyMs:    latency.Milliseconds(),
	})
}

// DashboardSummary is an aggregate view of the dashboard.
type DashboardSummary struct {
	TotalRuns       int
	TotalOrigTokens int
	TotalSaved      int
	AvgReductionPct float64
	AvgLatencyMs    float64
	TopFilters      []FilterSummary
	Uptime          time.Duration
}

// FilterSummary is per-filter aggregated stats.
type FilterSummary struct {
	Name         string
	Runs         int
	TotalSaved   int
	AvgReduction float64
}

// Summary computes aggregate statistics across all recorded metrics.
func (d *RealTimeDashboard) Summary() DashboardSummary {
	d.mu.RLock()
	metrics := make([]DashboardMetric, len(d.metrics))
	copy(metrics, d.metrics)
	startAt := d.startAt
	d.mu.RUnlock()

	s := DashboardSummary{
		TotalRuns: len(metrics),
		Uptime:    time.Since(startAt),
	}

	filterMap := make(map[string]*FilterSummary)
	for _, m := range metrics {
		s.TotalOrigTokens += m.OrigTokens
		s.TotalSaved += m.Saved
		s.AvgReductionPct += m.ReductionPct
		s.AvgLatencyMs += float64(m.LatencyMs)

		fs := filterMap[m.FilterName]
		if fs == nil {
			fs = &FilterSummary{Name: m.FilterName}
			filterMap[m.FilterName] = fs
		}
		fs.Runs++
		fs.TotalSaved += m.Saved
		fs.AvgReduction += m.ReductionPct
	}

	if s.TotalRuns > 0 {
		s.AvgReductionPct /= float64(s.TotalRuns)
		s.AvgLatencyMs /= float64(s.TotalRuns)
	}

	for _, fs := range filterMap {
		if fs.Runs > 0 {
			fs.AvgReduction /= float64(fs.Runs)
		}
		s.TopFilters = append(s.TopFilters, *fs)
	}

	return s
}

// FormatTable renders the dashboard summary as an ASCII table.
func (d *RealTimeDashboard) FormatTable() string {
	s := d.Summary()
	var sb strings.Builder
	sb.WriteString("╔══════════════════════════════════════════════════════════╗\n")
	sb.WriteString(fmt.Sprintf("║ tokman Real-Time Dashboard    uptime: %-20s ║\n",
		s.Uptime.Round(time.Second).String()))
	sb.WriteString("╠══════════════════════════════════════════════════════════╣\n")
	sb.WriteString(fmt.Sprintf("║ Runs: %6d  Orig: %8d  Saved: %8d  Avg: %5.1f%% ║\n",
		s.TotalRuns, s.TotalOrigTokens, s.TotalSaved, s.AvgReductionPct))
	sb.WriteString("╠══════════════════════════════════════════════════════════╣\n")
	sb.WriteString(fmt.Sprintf("║ %-28s %6s %10s %7s ║\n", "Filter", "Runs", "Saved", "AvgPct"))
	sb.WriteString("╠══════════════════════════════════════════════════════════╣\n")
	for _, f := range s.TopFilters {
		sb.WriteString(fmt.Sprintf("║ %-28s %6d %10d %6.1f%% ║\n",
			f.Name, f.Runs, f.TotalSaved, f.AvgReduction))
	}
	sb.WriteString("╚══════════════════════════════════════════════════════════╝\n")
	return sb.String()
}
