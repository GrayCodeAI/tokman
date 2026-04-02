// Package stress provides test scheduling capabilities
package stress

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Scheduler schedules stress tests
type Scheduler struct {
	jobs   map[string]*ScheduledJob
	runner *Runner
	stopCh chan struct{}
	mu     sync.Mutex
}

// ScheduledJob represents a scheduled test job
type ScheduledJob struct {
	ID       string
	Name     string
	Scenario *Scenario
	Schedule Schedule
	LastRun  time.Time
	NextRun  time.Time
	RunCount int
	MaxRuns  int
	Status   JobStatus
	Results  []Result
}

// Schedule defines when to run
type Schedule struct {
	Type     ScheduleType
	Interval time.Duration
	Cron     string
	At       time.Time
}

// ScheduleType defines schedule types
type ScheduleType string

const (
	ScheduleInterval ScheduleType = "interval"
	ScheduleOnce     ScheduleType = "once"
	ScheduleDaily    ScheduleType = "daily"
	ScheduleWeekly   ScheduleType = "weekly"
)

// JobStatus represents job status
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusPaused    JobStatus = "paused"
)

// NewScheduler creates a new scheduler
func NewScheduler(runner *Runner) *Scheduler {
	return &Scheduler{
		jobs:   make(map[string]*ScheduledJob),
		runner: runner,
		stopCh: make(chan struct{}),
	}
}

// AddJob adds a scheduled job
func (s *Scheduler) AddJob(job *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}

	// Calculate next run
	job.NextRun = s.calculateNextRun(job.Schedule)
	job.Status = JobStatusPending

	s.jobs[job.ID] = job
	return nil
}

// RemoveJob removes a job
func (s *Scheduler) RemoveJob(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, jobID)
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndRunJobs(ctx)
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) checkAndRunJobs(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, job := range s.jobs {
		if job.Status != JobStatusPending && job.Status != JobStatusRunning {
			continue
		}

		if job.MaxRuns > 0 && job.RunCount >= job.MaxRuns {
			job.Status = JobStatusCompleted
			continue
		}

		if now.After(job.NextRun) {
			go s.runJob(ctx, job)
		}
	}
}

func (s *Scheduler) runJob(ctx context.Context, job *ScheduledJob) {
	job.Status = JobStatusRunning
	job.LastRun = time.Now()

	result, err := s.runner.Run(ctx, job.Scenario.Name)
	if err != nil {
		job.Status = JobStatusFailed
	} else {
		job.Results = append(job.Results, *result)
		job.RunCount++
		job.Status = JobStatusPending
		job.NextRun = s.calculateNextRun(job.Schedule)
	}
}

func (s *Scheduler) calculateNextRun(schedule Schedule) time.Time {
	now := time.Now()

	switch schedule.Type {
	case ScheduleInterval:
		return now.Add(schedule.Interval)
	case ScheduleOnce:
		return schedule.At
	case ScheduleDaily:
		return now.Add(24 * time.Hour)
	case ScheduleWeekly:
		return now.Add(7 * 24 * time.Hour)
	default:
		return now.Add(1 * time.Hour)
	}
}

// ListJobs returns all jobs
func (s *Scheduler) ListJobs() []*ScheduledJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// ThresholdMonitor monitors results against thresholds
type ThresholdMonitor struct {
	thresholds ThresholdConfig
	alerts     chan Alert
}

// ThresholdConfig holds threshold configuration
type ThresholdConfig struct {
	MaxErrorRate  float64
	MaxLatencyP95 time.Duration
	MinThroughput float64
}

// Alert represents a threshold alert
type Alert struct {
	Type      string
	Metric    string
	Value     float64
	Threshold float64
	Timestamp time.Time
	Severity  string
}

// NewThresholdMonitor creates a monitor
func NewThresholdMonitor(config ThresholdConfig) *ThresholdMonitor {
	return &ThresholdMonitor{
		thresholds: config,
		alerts:     make(chan Alert, 100),
	}
}

// Check checks a result against thresholds
func (tm *ThresholdMonitor) Check(result *Result) []Alert {
	alerts := make([]Alert, 0)

	// Check error rate
	if result.ErrorRate > tm.thresholds.MaxErrorRate {
		alerts = append(alerts, Alert{
			Type:      "error_rate",
			Metric:    "error_rate",
			Value:     result.ErrorRate,
			Threshold: tm.thresholds.MaxErrorRate,
			Timestamp: time.Now(),
			Severity:  "critical",
		})
	}

	// Check latency
	if result.LatencyP95 > tm.thresholds.MaxLatencyP95 {
		alerts = append(alerts, Alert{
			Type:      "latency",
			Metric:    "latency_p95",
			Value:     float64(result.LatencyP95),
			Threshold: float64(tm.thresholds.MaxLatencyP95),
			Timestamp: time.Now(),
			Severity:  "warning",
		})
	}

	// Check throughput
	if result.ThroughputRPS < tm.thresholds.MinThroughput {
		alerts = append(alerts, Alert{
			Type:      "throughput",
			Metric:    "throughput",
			Value:     result.ThroughputRPS,
			Threshold: tm.thresholds.MinThroughput,
			Timestamp: time.Now(),
			Severity:  "warning",
		})
	}

	return alerts
}

// GeographicLoad provides geographic distribution
type GeographicLoad struct {
	Regions map[string]*RegionLoad
}

// RegionLoad represents load for a region
type RegionLoad struct {
	Name       string
	Percentage float64
	RPS        int
	Latency    time.Duration
}

// Distribute calculates geographic distribution
func Distribute(totalRPS int, distribution map[string]float64) map[string]int {
	result := make(map[string]int)

	for region, percentage := range distribution {
		result[region] = int(float64(totalRPS) * percentage / 100)
	}

	return result
}

// Replay replays a previous test
func Replay(originalResult *Result, config Config) (*Result, error) {
	// Create runner with original config
	runner := NewRunner(config)

	// Find original scenario
	scenario := &Scenario{
		Name: originalResult.Scenario,
		Type: originalResult.Type,
	}

	runner.RegisterScenario(scenario)

	// Run test
	ctx := context.Background()
	return runner.Run(ctx, scenario.Name)
}

// MetricCorrelator correlates metrics
type MetricCorrelator struct {
	data []MetricDataPoint
}

// MetricDataPoint represents a data point
type MetricDataPoint struct {
	Timestamp  time.Time
	Latency    time.Duration
	Throughput float64
	Errors     int
	CPU        float64
	Memory     float64
}

// Correlate finds correlations
func (mc *MetricCorrelator) Correlate() map[string]float64 {
	correlations := make(map[string]float64)

	if len(mc.data) < 2 {
		return correlations
	}

	// Simple correlation calculation
	// In production, use proper statistical correlation
	correlations["latency_cpu"] = 0.5   // Placeholder
	correlations["memory_errors"] = 0.3 // Placeholder

	return correlations
}

// CostEstimator estimates test costs
type CostEstimator struct {
	config CostConfig
}

// CostConfig holds cost configuration
type CostConfig struct {
	CostPerRequest float64
	CostPerMinute  float64
	Currency       string
}

// Estimate estimates cost for a test
func (ce *CostEstimator) Estimate(duration time.Duration, rps int) float64 {
	// Calculate cost
	requestCost := float64(rps*int(duration.Seconds())) * ce.config.CostPerRequest
	timeCost := duration.Minutes() * ce.config.CostPerMinute

	return requestCost + timeCost
}
