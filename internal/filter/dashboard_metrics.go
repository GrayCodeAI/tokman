package filter

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CompressionDashboard provides real-time compression analytics.
// Integrates with the TokMan web dashboard for live metrics.
type CompressionDashboard struct {
	config  DashboardConfig
	metrics *DashboardMetrics
	mu      sync.RWMutex
}

// DashboardConfig holds dashboard configuration
type DashboardConfig struct {
	// Enabled controls whether dashboard tracking is active
	Enabled bool

	// MaxRecords to keep in memory
	MaxRecords int

	// Port for the dashboard web server
	Port int
}

// DashboardMetrics holds real-time compression metrics
type DashboardMetrics struct {
	// TotalCompressions is the total number of compressions performed
	TotalCompressions int64 `json:"total_compressions"`

	// TotalTokensSaved is the cumulative tokens saved
	TotalTokensSaved int64 `json:"total_tokens_saved"`

	// AvgCompressionRatio is the average compression percentage
	AvgCompressionRatio float64 `json:"avg_compression_ratio"`

	// AvgLatencyMs is the average processing latency in ms
	AvgLatencyMs float64 `json:"avg_latency_ms"`

	// LayerStats tracks per-layer contribution
	LayerStats map[string]*LayerMetrics `json:"layer_stats"`

	// ContentTypeStats tracks per-content-type metrics
	ContentTypeStats map[string]*ContentMetrics `json:"content_type_stats"`

	// RecentCompressions holds the last N compression records
	RecentCompressions []CompressionRecord `json:"recent_compressions"`

	// StartTime is when metrics collection began
	StartTime time.Time `json:"start_time"`

	// Uptime is the duration since metrics collection started
	Uptime string `json:"uptime"`
}

// LayerMetrics tracks metrics for a single layer
type LayerMetrics struct {
	Name          string  `json:"name"`
	CallCount     int64   `json:"call_count"`
	TotalSaved    int64   `json:"total_saved"`
	AvgSaved      float64 `json:"avg_saved"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	MaxSaved      int     `json:"max_saved"`
	SkipCount     int64   `json:"skip_count"`
	Effectiveness float64 `json:"effectiveness"` // saved / call_count
}

// ContentMetrics tracks metrics per content type
type ContentMetrics struct {
	ContentType    string  `json:"content_type"`
	Count          int64   `json:"count"`
	TotalSaved     int64   `json:"total_saved"`
	AvgCompression float64 `json:"avg_compression"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
}

// CompressionRecord captures a single compression event
type CompressionRecord struct {
	Timestamp      time.Time      `json:"timestamp"`
	ContentType    string         `json:"content_type"`
	InputTokens    int            `json:"input_tokens"`
	OutputTokens   int            `json:"output_tokens"`
	SavedTokens    int            `json:"saved_tokens"`
	CompressionPct float64        `json:"compression_pct"`
	LatencyMs      float64        `json:"latency_ms"`
	LayersUsed     int            `json:"layers_used"`
	LayerContribs  map[string]int `json:"layer_contribs"`
}

// defaultDashboardConfig returns default configuration
func defaultDashboardConfig() DashboardConfig {
	return DashboardConfig{
		Enabled:    true,
		MaxRecords: 100,
		Port:       8080,
	}
}

// newCompressionDashboard creates a new dashboard
func newCompressionDashboard() *CompressionDashboard {
	return newCompressionDashboardWithConfig(defaultDashboardConfig())
}

// newCompressionDashboardWithConfig creates a dashboard with custom config
func newCompressionDashboardWithConfig(cfg DashboardConfig) *CompressionDashboard {
	return &CompressionDashboard{
		config: cfg,
		metrics: &DashboardMetrics{
			LayerStats:         make(map[string]*LayerMetrics),
			ContentTypeStats:   make(map[string]*ContentMetrics),
			RecentCompressions: make([]CompressionRecord, 0, cfg.MaxRecords),
			StartTime:          time.Now(),
		},
	}
}

// RecordCompression records a compression event for dashboard display
func (d *CompressionDashboard) RecordCompression(
	contentType string,
	inputTokens, outputTokens, savedTokens int,
	latencyMs float64,
	layerContribs map[string]int,
) {
	if !d.config.Enabled {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	compressionPct := 0.0
	if inputTokens > 0 {
		compressionPct = float64(savedTokens) / float64(inputTokens) * 100
	}

	record := CompressionRecord{
		Timestamp:      time.Now(),
		ContentType:    contentType,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		SavedTokens:    savedTokens,
		CompressionPct: compressionPct,
		LatencyMs:      latencyMs,
		LayersUsed:     len(layerContribs),
		LayerContribs:  layerContribs,
	}

	// Update totals
	d.metrics.TotalCompressions++
	d.metrics.TotalTokensSaved += int64(savedTokens)

	// Update average compression ratio
	n := float64(d.metrics.TotalCompressions)
	d.metrics.AvgCompressionRatio = (d.metrics.AvgCompressionRatio*(n-1) + compressionPct) / n
	d.metrics.AvgLatencyMs = (d.metrics.AvgLatencyMs*(n-1) + latencyMs) / n

	// Update layer stats
	for layer, saved := range layerContribs {
		if d.metrics.LayerStats[layer] == nil {
			d.metrics.LayerStats[layer] = &LayerMetrics{Name: layer}
		}
		lm := d.metrics.LayerStats[layer]
		lm.CallCount++
		lm.TotalSaved += int64(saved)
		lm.AvgSaved = float64(lm.TotalSaved) / float64(lm.CallCount)
		if saved > lm.MaxSaved {
			lm.MaxSaved = saved
		}
		lm.Effectiveness = float64(lm.TotalSaved) / float64(lm.CallCount)
	}

	// Update content type stats
	if d.metrics.ContentTypeStats[contentType] == nil {
		d.metrics.ContentTypeStats[contentType] = &ContentMetrics{ContentType: contentType}
	}
	cm := d.metrics.ContentTypeStats[contentType]
	cm.Count++
	cm.TotalSaved += int64(savedTokens)
	cm.AvgCompression = float64(cm.TotalSaved) / float64(cm.Count)
	cm.AvgLatencyMs = (cm.AvgLatencyMs*float64(cm.Count-1) + latencyMs) / float64(cm.Count)

	// Add to recent records
	d.metrics.RecentCompressions = append(d.metrics.RecentCompressions, record)
	if len(d.metrics.RecentCompressions) > d.config.MaxRecords {
		d.metrics.RecentCompressions = d.metrics.RecentCompressions[1:]
	}

	// Update uptime
	d.metrics.Uptime = time.Since(d.metrics.StartTime).Round(time.Second).String()
}

// GetMetrics returns current dashboard metrics
func (d *CompressionDashboard) GetMetrics() *DashboardMetrics {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy
	m := *d.metrics
	m.LayerStats = make(map[string]*LayerMetrics, len(d.metrics.LayerStats))
	for k, v := range d.metrics.LayerStats {
		lm := *v
		m.LayerStats[k] = &lm
	}
	m.ContentTypeStats = make(map[string]*ContentMetrics, len(d.metrics.ContentTypeStats))
	for k, v := range d.metrics.ContentTypeStats {
		cm := *v
		m.ContentTypeStats[k] = &cm
	}

	return &m
}

// GetMetricsJSON returns metrics as JSON
func (d *CompressionDashboard) GetMetricsJSON() ([]byte, error) {
	metrics := d.GetMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

// GetSummary returns a formatted summary string
func (d *CompressionDashboard) GetSummary() string {
	metrics := d.GetMetrics()

	return fmt.Sprintf(`
╔════════════════════════════════════════════════════════╗
║           TokMan Compression Dashboard                 ║
╠════════════════════════════════════════════════════════╣
║ Uptime:          %s
║ Compressions:    %d
║ Total Saved:     %d tokens
║ Avg Compression: %.1f%%
║ Avg Latency:     %.2f ms
╠════════════════════════════════════════════════════════╣
║ Top Layers by Effectiveness:
`,
		metrics.Uptime,
		metrics.TotalCompressions,
		metrics.TotalTokensSaved,
		metrics.AvgCompressionRatio,
		metrics.AvgLatencyMs,
	) + d.formatTopLayers(metrics) + `
╠════════════════════════════════════════════════════════╣
║ Content Types:
` + d.formatContentTypes(metrics) + `
╚════════════════════════════════════════════════════════╝
`
}

func (d *CompressionDashboard) formatTopLayers(metrics *DashboardMetrics) string {
	type kv struct {
		name  string
		score float64
	}

	var sorted []kv
	for _, lm := range metrics.LayerStats {
		sorted = append(sorted, kv{lm.Name, lm.Effectiveness})
	}

	// Sort descending
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].score > sorted[i].score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	result := ""
	limit := 10
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		result += fmt.Sprintf("║   %-25s: %.1f tokens/call\n", sorted[i].name, sorted[i].score)
	}
	return result
}

func (d *CompressionDashboard) formatContentTypes(metrics *DashboardMetrics) string {
	result := ""
	for _, cm := range metrics.ContentTypeStats {
		result += fmt.Sprintf("║   %-20s: %d calls, %.1f%% avg\n",
			cm.ContentType, cm.Count, cm.AvgCompression)
	}
	return result
}

// Reset clears all metrics
func (d *CompressionDashboard) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.metrics = &DashboardMetrics{
		LayerStats:         make(map[string]*LayerMetrics),
		ContentTypeStats:   make(map[string]*ContentMetrics),
		RecentCompressions: make([]CompressionRecord, 0, d.config.MaxRecords),
		StartTime:          time.Now(),
	}
}
