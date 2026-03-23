package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Metrics holds server observability data
type Metrics struct {
	mu sync.RWMutex

	// Counters
	TotalRequests     int64
	TotalCompressions int64
	TotalTokensIn     int64
	TotalTokensOut    int64
	TotalTokensSaved  int64
	TotalErrors       int64

	// Histograms (simplified as maps)
	ProcessingTimes   []time.Duration
	ReductionPercents []float64

	// Gauges
	ActiveConnections int64

	// Content type distribution
	ContentTypeCounts map[string]int64

	// Start time for uptime
	StartTime time.Time
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime:         time.Now(),
		ContentTypeCounts: make(map[string]int64),
		ProcessingTimes:   make([]time.Duration, 0, 1000),
		ReductionPercents: make([]float64, 0, 1000),
	}
}

// RecordCompression tracks a compression operation
func (m *Metrics) RecordCompression(original, final int, duration time.Duration, contentType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalCompressions++
	m.TotalTokensIn += int64(original)
	m.TotalTokensOut += int64(final)
	m.TotalTokensSaved += int64(original - final)
	m.TotalRequests++

	// Track processing times (keep last 1000)
	m.ProcessingTimes = append(m.ProcessingTimes, duration)
	if len(m.ProcessingTimes) > 1000 {
		m.ProcessingTimes = m.ProcessingTimes[1:]
	}

	// Track reduction percents
	reduction := 0.0
	if original > 0 {
		reduction = float64(original-final) / float64(original) * 100
	}
	m.ReductionPercents = append(m.ReductionPercents, reduction)
	if len(m.ReductionPercents) > 1000 {
		m.ReductionPercents = m.ReductionPercents[1:]
	}

	// Track content types (capped at 1000 entries)
	if len(m.ContentTypeCounts) < 1000 {
		m.ContentTypeCounts[contentType]++
	}
}

// RecordError tracks an error
func (m *Metrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalErrors++
	m.TotalRequests++
}

// Snapshot returns a point-in-time copy of metrics
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate averages
	avgProcessing := time.Duration(0)
	if len(m.ProcessingTimes) > 0 {
		var total time.Duration
		for _, t := range m.ProcessingTimes {
			total += t
		}
		avgProcessing = total / time.Duration(len(m.ProcessingTimes))
	}

	avgReduction := 0.0
	if len(m.ReductionPercents) > 0 {
		var total float64
		for _, r := range m.ReductionPercents {
			total += r
		}
		avgReduction = total / float64(len(m.ReductionPercents))
	}

	// Copy content type counts
	ctCounts := make(map[string]int64)
	for k, v := range m.ContentTypeCounts {
		ctCounts[k] = v
	}

	return MetricsSnapshot{
		Uptime:            time.Since(m.StartTime),
		TotalRequests:     m.TotalRequests,
		TotalCompressions: m.TotalCompressions,
		TotalTokensIn:     m.TotalTokensIn,
		TotalTokensOut:    m.TotalTokensOut,
		TotalTokensSaved:  m.TotalTokensSaved,
		TotalErrors:       m.TotalErrors,
		AvgProcessingMs:   avgProcessing.Milliseconds(),
		AvgReductionPct:   avgReduction,
		ContentTypeCounts: ctCounts,
	}
}

// MetricsSnapshot is a point-in-time view of metrics
type MetricsSnapshot struct {
	Uptime            time.Duration    `json:"uptime"`
	TotalRequests     int64            `json:"total_requests"`
	TotalCompressions int64            `json:"total_compressions"`
	TotalTokensIn     int64            `json:"total_tokens_in"`
	TotalTokensOut    int64            `json:"total_tokens_out"`
	TotalTokensSaved  int64            `json:"total_tokens_saved"`
	TotalErrors       int64            `json:"total_errors"`
	AvgProcessingMs   int64            `json:"avg_processing_ms"`
	AvgReductionPct   float64          `json:"avg_reduction_pct"`
	ContentTypeCounts map[string]int64 `json:"content_type_counts"`
}

// PrometheusFormat returns metrics in Prometheus text format
func (m *Metrics) PrometheusFormat() string {
	snap := m.Snapshot()

	var output string
	output += fmt.Sprintf("# HELP tokman_requests_total Total number of requests\n")
	output += fmt.Sprintf("# TYPE tokman_requests_total counter\n")
	output += fmt.Sprintf("tokman_requests_total %d\n", snap.TotalRequests)

	output += fmt.Sprintf("\n# HELP tokman_compressions_total Total number of compressions\n")
	output += fmt.Sprintf("# TYPE tokman_compressions_total counter\n")
	output += fmt.Sprintf("tokman_compressions_total %d\n", snap.TotalCompressions)

	output += fmt.Sprintf("\n# HELP tokman_tokens_in_total Total input tokens\n")
	output += fmt.Sprintf("# TYPE tokman_tokens_in_total counter\n")
	output += fmt.Sprintf("tokman_tokens_in_total %d\n", snap.TotalTokensIn)

	output += fmt.Sprintf("\n# HELP tokman_tokens_out_total Total output tokens\n")
	output += fmt.Sprintf("# TYPE tokman_tokens_out_total counter\n")
	output += fmt.Sprintf("tokman_tokens_out_total %d\n", snap.TotalTokensOut)

	output += fmt.Sprintf("\n# HELP tokman_tokens_saved_total Total tokens saved\n")
	output += fmt.Sprintf("# TYPE tokman_tokens_saved_total counter\n")
	output += fmt.Sprintf("tokman_tokens_saved_total %d\n", snap.TotalTokensSaved)

	output += fmt.Sprintf("\n# HELP tokman_errors_total Total errors\n")
	output += fmt.Sprintf("# TYPE tokman_errors_total counter\n")
	output += fmt.Sprintf("tokman_errors_total %d\n", snap.TotalErrors)

	output += fmt.Sprintf("\n# HELP tokman_avg_processing_ms Average processing time in ms\n")
	output += fmt.Sprintf("# TYPE tokman_avg_processing_ms gauge\n")
	output += fmt.Sprintf("tokman_avg_processing_ms %d\n", snap.AvgProcessingMs)

	output += fmt.Sprintf("\n# HELP tokman_avg_reduction_pct Average reduction percentage\n")
	output += fmt.Sprintf("# TYPE tokman_avg_reduction_pct gauge\n")
	output += fmt.Sprintf("tokman_avg_reduction_pct %.2f\n", snap.AvgReductionPct)

	output += fmt.Sprintf("\n# HELP tokman_uptime_seconds Server uptime in seconds\n")
	output += fmt.Sprintf("# TYPE tokman_uptime_seconds gauge\n")
	output += fmt.Sprintf("tokman_uptime_seconds %.0f\n", snap.Uptime.Seconds())

	// Content type metrics
	if len(snap.ContentTypeCounts) > 0 {
		output += fmt.Sprintf("\n# HELP tokman_content_type_total Requests by content type\n")
		output += fmt.Sprintf("# TYPE tokman_content_type_total counter\n")
		for ct, count := range snap.ContentTypeCounts {
			output += fmt.Sprintf("tokman_content_type_total{type=\"%s\"} %d\n", ct, count)
		}
	}

	return output
}

// Logger provides structured logging
type Logger struct {
	level  string
	output io.Writer
}

// NewLogger creates a new logger
func NewLogger(level string) *Logger {
	return &Logger{
		level:  level,
		output: os.Stdout,
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Time    string                 `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// Info logs at INFO level
func (l *Logger) Info(msg string, fields map[string]any) {
	l.log("INFO", msg, fields)
}

// Error logs at ERROR level
func (l *Logger) Error(msg string, fields map[string]any) {
	l.log("ERROR", msg, fields)
}

// Debug logs at DEBUG level
func (l *Logger) Debug(msg string, fields map[string]any) {
	if l.level == "debug" {
		l.log("DEBUG", msg, fields)
	}
}

func (l *Logger) log(level, msg string, fields map[string]any) {
	entry := LogEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		Message: msg,
		Fields:  fields,
	}

	data, _ := json.Marshal(entry)
	fmt.Fprintf(l.output, "%s\n", data)
}
