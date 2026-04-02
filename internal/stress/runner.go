// Package stress provides stress testing capabilities for TokMan
package stress

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// TestType defines the type of stress test
type TestType string

const (
	TypeLoad      TestType = "load"
	TypeSpike     TestType = "spike"
	TypeSoak      TestType = "soak"
	TypeStress    TestType = "stress"
	TypeBreakdown TestType = "breakdown"
)

// Runner executes stress tests
type Runner struct {
	config    Config
	scenarios map[string]*Scenario
	results   []Result
	mu        sync.RWMutex
	stopCh    chan struct{}
}

// Config holds stress test configuration
type Config struct {
	Duration         time.Duration
	WarmupDuration   time.Duration
	RampUpDuration   time.Duration
	CooldownDuration time.Duration
	TargetRPS        int
	MaxConcurrency   int
	Timeout          time.Duration
}

// DefaultConfig returns default stress test configuration
func DefaultConfig() Config {
	return Config{
		Duration:         5 * time.Minute,
		WarmupDuration:   30 * time.Second,
		RampUpDuration:   60 * time.Second,
		CooldownDuration: 30 * time.Second,
		TargetRPS:        100,
		MaxConcurrency:   1000,
		Timeout:          30 * time.Second,
	}
}

// Result holds stress test results
type Result struct {
	Scenario         string
	Type             TestType
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	TotalRequests    int64
	SuccessCount     int64
	ErrorCount       int64
	TimeoutCount     int64
	LatencyHistogram map[string]time.Duration
	LatencyP50       time.Duration
	LatencyP95       time.Duration
	LatencyP99       time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	AvgLatency       time.Duration
	ThroughputRPS    float64
	SuccessRate      float64
	ErrorRate        float64
	Errors           map[string]int
	ResourceUsage    ResourceMetrics
}

// ResourceMetrics tracks resource utilization
type ResourceMetrics struct {
	CPUPercent  float64
	MemoryMB    float64
	Goroutines  int
	GCRuns      uint32
	HeapAllocMB float64
}

// Scenario defines a stress test scenario
type Scenario struct {
	Name        string
	Type        TestType
	Description string
	Fn          func(ctx context.Context) error
	Weight      int
}

// NewRunner creates a stress test runner
func NewRunner(config Config) *Runner {
	return &Runner{
		config:    config,
		scenarios: make(map[string]*Scenario),
		results:   make([]Result, 0),
		stopCh:    make(chan struct{}),
	}
}

// RegisterScenario adds a scenario
func (r *Runner) RegisterScenario(s *Scenario) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scenarios[s.Name] = s
}

// Run executes a stress test for a scenario
func (r *Runner) Run(ctx context.Context, scenarioName string) (*Result, error) {
	r.mu.RLock()
	scenario, exists := r.scenarios[scenarioName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("scenario %s not found", scenarioName)
	}

	result := &Result{
		Scenario:         scenarioName,
		Type:             scenario.Type,
		StartTime:        time.Now(),
		LatencyHistogram: make(map[string]time.Duration),
		Errors:           make(map[string]int),
	}

	// Warmup phase
	if r.config.WarmupDuration > 0 {
		fmt.Printf("Warmup phase: %v\n", r.config.WarmupDuration)
		r.runPhase(ctx, scenario, r.config.WarmupDuration, true)
	}

	// Ramp-up phase
	if r.config.RampUpDuration > 0 {
		fmt.Printf("Ramp-up phase: %v\n", r.config.RampUpDuration)
		r.runRampUp(ctx, scenario, r.config.RampUpDuration)
	}

	// Main test phase
	fmt.Printf("Main test phase: %v\n", r.config.Duration)
	metrics := r.runPhase(ctx, scenario, r.config.Duration, false)

	// Cooldown phase
	if r.config.CooldownDuration > 0 {
		fmt.Printf("Cooldown phase: %v\n", r.config.CooldownDuration)
		r.runPhase(ctx, scenario, r.config.CooldownDuration, true)
	}

	// Calculate results
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.TotalRequests = metrics.totalRequests
	result.SuccessCount = metrics.successCount
	result.ErrorCount = metrics.errorCount
	result.TimeoutCount = metrics.timeoutCount

	if result.TotalRequests > 0 {
		result.AvgLatency = time.Duration(int64(metrics.totalLatency) / result.TotalRequests)
		result.ThroughputRPS = float64(result.TotalRequests) / result.Duration.Seconds()
		result.SuccessRate = float64(result.SuccessCount) / float64(result.TotalRequests) * 100
		result.ErrorRate = float64(result.ErrorCount) / float64(result.TotalRequests) * 100
	}

	result.LatencyP50 = metrics.calculatePercentile(0.5)
	result.LatencyP95 = metrics.calculatePercentile(0.95)
	result.LatencyP99 = metrics.calculatePercentile(0.99)
	result.MinLatency = metrics.minLatency
	result.MaxLatency = metrics.maxLatency

	r.mu.Lock()
	r.results = append(r.results, *result)
	r.mu.Unlock()

	return result, nil
}

type phaseMetrics struct {
	totalRequests int64
	successCount  int64
	errorCount    int64
	timeoutCount  int64
	totalLatency  time.Duration
	latencies     []time.Duration
	minLatency    time.Duration
	maxLatency    time.Duration
	mu            sync.Mutex
}

func (m *phaseMetrics) record(latency time.Duration, err error, timeout bool) {
	atomic.AddInt64(&m.totalRequests, 1)

	if timeout {
		atomic.AddInt64(&m.timeoutCount, 1)
		atomic.AddInt64(&m.errorCount, 1)
	} else if err != nil {
		atomic.AddInt64(&m.errorCount, 1)
	} else {
		atomic.AddInt64(&m.successCount, 1)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalLatency += latency
	m.latencies = append(m.latencies, latency)

	if m.minLatency == 0 || latency < m.minLatency {
		m.minLatency = latency
	}
	if latency > m.maxLatency {
		m.maxLatency = latency
	}
}

func (m *phaseMetrics) calculatePercentile(p float64) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.latencies) == 0 {
		return 0
	}

	// Simple sort (for production, use a more efficient algorithm)
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

func (r *Runner) runPhase(ctx context.Context, scenario *Scenario, duration time.Duration, warmup bool) *phaseMetrics {
	metrics := &phaseMetrics{}
	deadline := time.Now().Add(duration)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, r.config.MaxConcurrency)

	requestInterval := time.Second / time.Duration(r.config.TargetRPS)
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return metrics
		case <-r.stopCh:
			return metrics
		case <-ticker.C:
			wg.Add(1)
			semaphore <- struct{}{}

			go func() {
				defer wg.Done()
				defer func() { <-semaphore }()

				start := time.Now()
				reqCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
				defer cancel()

				err := scenario.Fn(reqCtx)
				latency := time.Since(start)

				timeout := false
				if reqCtx.Err() == context.DeadlineExceeded {
					timeout = true
				}

				if !warmup {
					metrics.record(latency, err, timeout)
				}
			}()
		}
	}

	wg.Wait()
	return metrics
}

func (r *Runner) runRampUp(ctx context.Context, scenario *Scenario, duration time.Duration) {
	startRPS := 1
	endRPS := r.config.TargetRPS
	steps := 10

	stepDuration := duration / time.Duration(steps)
	rpsIncrement := (endRPS - startRPS) / steps

	for i := 0; i < steps; i++ {
		targetRPS := startRPS + (rpsIncrement * i)
		stepConfig := r.config
		stepConfig.TargetRPS = targetRPS

		stepRunner := &Runner{config: stepConfig}
		_ = stepRunner.runPhase(ctx, scenario, stepDuration, true)
	}
}

// Stop gracefully stops the stress test
func (r *Runner) Stop() {
	close(r.stopCh)
}

// GenerateReport creates a comprehensive stress test report
func (r *Result) GenerateReport() string {
	report := fmt.Sprintf("Stress Test Report: %s\n", r.Scenario)
	report += "==================================================\n"
	report += fmt.Sprintf("Type: %s\n", r.Type)
	report += fmt.Sprintf("Duration: %v\n", r.Duration)
	report += fmt.Sprintf("Total Requests: %d\n", r.TotalRequests)
	report += fmt.Sprintf("Success: %d (%.2f%%)\n", r.SuccessCount, float64(r.SuccessCount)/float64(r.TotalRequests)*100)
	report += fmt.Sprintf("Errors: %d (%.2f%%)\n", r.ErrorCount, float64(r.ErrorCount)/float64(r.TotalRequests)*100)
	report += fmt.Sprintf("Timeouts: %d\n", r.TimeoutCount)
	report += fmt.Sprintf("\nLatency Stats:\n")
	report += fmt.Sprintf("  Min: %v\n", r.MinLatency)
	report += fmt.Sprintf("  Avg: %v\n", r.AvgLatency)
	report += fmt.Sprintf("  P50: %v\n", r.LatencyP50)
	report += fmt.Sprintf("  P95: %v\n", r.LatencyP95)
	report += fmt.Sprintf("  P99: %v\n", r.LatencyP99)
	report += fmt.Sprintf("  Max: %v\n", r.MaxLatency)
	report += fmt.Sprintf("\nThroughput: %.2f RPS\n", r.ThroughputRPS)

	return report
}

// StandardScenarios returns common stress test scenarios
func StandardScenarios() []*Scenario {
	return []*Scenario{
		{
			Name:        "basic_load",
			Type:        TypeLoad,
			Description: "Basic load test with steady traffic",
			Fn: func(ctx context.Context) error {
				// Simulate work
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				return nil
			},
			Weight: 1,
		},
		{
			Name:        "spike_test",
			Type:        TypeSpike,
			Description: "Sudden traffic spike simulation",
			Fn: func(ctx context.Context) error {
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				return nil
			},
			Weight: 2,
		},
		{
			Name:        "soak_test",
			Type:        TypeSoak,
			Description: "Extended duration stability test",
			Fn: func(ctx context.Context) error {
				time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
				return nil
			},
			Weight: 1,
		},
	}
}
