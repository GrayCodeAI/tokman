package filter

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// HealthStatus represents the overall health of the filter system.
// Task #124: Health check endpoint with component status.
type HealthStatus struct {
	Status     string                     `json:"status"` // "ok", "degraded", "error"
	Components map[string]ComponentStatus `json:"components"`
	Timestamp  time.Time                  `json:"timestamp"`
}

// ComponentStatus represents the health of a single component.
type ComponentStatus struct {
	Status  string        `json:"status"` // "ok", "degraded", "error"
	Message string        `json:"message"`
	Latency time.Duration `json:"latency_ns"`
}

// ToJSON serializes the HealthStatus to a JSON string.
func (h HealthStatus) ToJSON() string {
	b, err := json.Marshal(h)
	if err != nil {
		return fmt.Sprintf(`{"status":"error","error":%q}`, err.Error())
	}
	return string(b)
}

// HealthChecker checks the health of filter system components.
type HealthChecker struct{}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// Check performs health checks on all components and returns a HealthStatus.
func (hc *HealthChecker) Check() HealthStatus {
	components := make(map[string]ComponentStatus)

	components["tokenizer"] = hc.checkTokenizer()
	components["pipeline"] = hc.checkPipeline()
	components["memory"] = hc.checkMemory()

	overall := "ok"
	for _, cs := range components {
		if cs.Status == "error" {
			overall = "error"
			break
		}
		if cs.Status == "degraded" && overall != "error" {
			overall = "degraded"
		}
	}

	return HealthStatus{
		Status:     overall,
		Components: components,
		Timestamp:  time.Now(),
	}
}

// checkTokenizer verifies core.EstimateTokens works and measures latency.
func (hc *HealthChecker) checkTokenizer() ComponentStatus {
	start := time.Now()
	count := core.EstimateTokens("test")
	latency := time.Since(start)

	if count <= 0 {
		return ComponentStatus{
			Status:  "error",
			Message: "tokenizer returned non-positive token count",
			Latency: latency,
		}
	}
	return ComponentStatus{
		Status:  "ok",
		Message: fmt.Sprintf("estimated %d token(s) for test input", count),
		Latency: latency,
	}
}

// checkPipeline verifies the PipelineCoordinator can compress a minimal input.
func (hc *HealthChecker) checkPipeline() ComponentStatus {
	start := time.Now()

	cfg := PipelineConfig{
		Mode:          ModeMinimal,
		EnableEntropy: true,
	}
	p := NewPipelineCoordinator(cfg)
	output, stats := p.Process("hello world")

	latency := time.Since(start)

	if output == "" {
		return ComponentStatus{
			Status:  "error",
			Message: "pipeline returned empty output",
			Latency: latency,
		}
	}
	return ComponentStatus{
		Status:  "ok",
		Message: fmt.Sprintf("compressed %d->%d tokens", stats.OriginalTokens, stats.FinalTokens),
		Latency: latency,
	}
}

// checkMemory reads runtime.MemStats and reports heap usage.
func (hc *HealthChecker) checkMemory() ComponentStatus {
	start := time.Now()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	latency := time.Since(start)

	heapMB := float64(ms.HeapAlloc) / (1024 * 1024)
	sysMB := float64(ms.Sys) / (1024 * 1024)

	status := "ok"
	msg := fmt.Sprintf("heap=%.2f MB, sys=%.2f MB", heapMB, sysMB)

	// Warn if heap exceeds 1 GB
	if ms.HeapAlloc > 1<<30 {
		status = "degraded"
		msg = fmt.Sprintf("high heap usage: %.2f MB (sys=%.2f MB)", heapMB, sysMB)
	}

	return ComponentStatus{
		Status:  status,
		Message: msg,
		Latency: latency,
	}
}
