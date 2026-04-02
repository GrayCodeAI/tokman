// Package chaos provides additional chaos experiments
package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// NetworkPartition creates network partition between services
func NetworkPartition(ctx context.Context, config PartitionConfig) error {
	// Simulate network partition by dropping packets
	duration := config.Duration
	if duration == 0 {
		duration = 30 * time.Second
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// PartitionConfig holds partition configuration
type PartitionConfig struct {
	Source      string
	Destination string
	Duration    time.Duration
	Direction   string // one-way or two-way
}

// DiskFailure simulates disk failures
func DiskFailure(ctx context.Context, config DiskConfig) error {
	// Simulate disk errors by creating temp files that fail
	duration := config.Duration
	if duration == 0 {
		duration = 30 * time.Second
	}

	start := time.Now()
	for time.Since(start) < duration {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate disk write failure
		if rand.Float64() < config.FailureRate {
			// Would fail in real implementation
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// DiskConfig holds disk failure configuration
type DiskConfig struct {
	Path        string
	FailureRate float64
	Duration    time.Duration
}

// DNSChaos creates DNS resolution failures
func DNSChaos(ctx context.Context, config DNSConfig) error {
	// Simulate DNS failures
	duration := config.Duration
	if duration == 0 {
		duration = 30 * time.Second
	}

	// Temporarily modify /etc/hosts or use resolver override
	// In real implementation, would use iptables or custom resolver

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// DNSConfig holds DNS chaos configuration
type DNSConfig struct {
	Domains     []string
	FailureRate float64
	Duration    time.Duration
}

// TimeDrift simulates clock skew
func TimeDrift(ctx context.Context, config TimeConfig) error {
	// Simulate time drift by offsetting system time
	duration := config.Duration
	if duration == 0 {
		duration = 30 * time.Second
	}

	offset := config.Offset
	if offset == 0 {
		offset = 5 * time.Second
	}

	// In real implementation, would use libfaketime or similar
	_ = offset

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// TimeConfig holds time drift configuration
type TimeConfig struct {
	Offset   time.Duration
	Duration time.Duration
}

// DependencyFailure simulates external dependency failures
func DependencyFailure(ctx context.Context, config DependencyConfig) error {
	duration := config.Duration
	if duration == 0 {
		duration = 30 * time.Second
	}

	// Block connections to dependency
	for _, dep := range config.Dependencies {
		// Would use firewall rules or proxy in real implementation
		_ = dep
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// DependencyConfig holds dependency failure configuration
type DependencyConfig struct {
	Dependencies []string
	Duration     time.Duration
	FailureType  string // timeout, error, slow
}

// CascadingFailure simulates cascading failures
func CascadingFailure(ctx context.Context, config CascadeConfig) error {
	// Simulate failure that propagates through services
	services := config.Services
	if len(services) == 0 {
		return fmt.Errorf("no services specified")
	}

	delay := config.DelayBetween
	if delay == 0 {
		delay = 5 * time.Second
	}

	for i, service := range services {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Inject failure into service
		_ = service

		// Wait before cascading to next service
		if i < len(services)-1 {
			time.Sleep(delay)
		}
	}

	return nil
}

// CascadeConfig holds cascading failure configuration
type CascadeConfig struct {
	Services     []string
	DelayBetween time.Duration
	FailureType  string
}

// ChaosTemplate represents reusable chaos experiment templates
type ChaosTemplate struct {
	Name          string
	Description   string
	Type          ExperimentType
	DefaultConfig map[string]interface{}
	Setup         func() error
	Execute       func(ctx context.Context, config map[string]interface{}) error
	Teardown      func() error
}

// StandardTemplates returns standard chaos templates
func StandardTemplates() []ChaosTemplate {
	return []ChaosTemplate{
		{
			Name:        "database-outage",
			Description: "Simulates database connectivity issues",
			Type:        TypeError,
			DefaultConfig: map[string]interface{}{
				"duration":   "30s",
				"error_rate": 0.5,
			},
		},
		{
			Name:        "slow-network",
			Description: "Adds latency to network connections",
			Type:        TypeLatency,
			DefaultConfig: map[string]interface{}{
				"duration": "60s",
				"latency":  "200ms",
			},
		},
		{
			Name:        "memory-pressure",
			Description: "Consumes available memory",
			Type:        TypeMemory,
			DefaultConfig: map[string]interface{}{
				"duration": "120s",
				"size_mb":  512,
			},
		},
		{
			Name:        "cpu-throttle",
			Description: "Consumes CPU resources",
			Type:        TypeCPU,
			DefaultConfig: map[string]interface{}{
				"duration":  "60s",
				"intensity": 0.8,
			},
		},
	}
}

// ExperimentSchedule schedules chaos experiments
type ExperimentSchedule struct {
	Experiments []ScheduledExperiment
}

// ScheduledExperiment represents a scheduled experiment
type ScheduledExperiment struct {
	Experiment *Experiment
	Schedule   time.Time
	Repeat     bool
	Interval   time.Duration
}

// Schedule creates a schedule
func Schedule(experiments []*Experiment, interval time.Duration) *ExperimentSchedule {
	schedule := &ExperimentSchedule{
		Experiments: make([]ScheduledExperiment, 0, len(experiments)),
	}

	now := time.Now()
	for _, exp := range experiments {
		schedule.Experiments = append(schedule.Experiments, ScheduledExperiment{
			Experiment: exp,
			Schedule:   now.Add(interval),
			Repeat:     true,
			Interval:   interval,
		})
	}

	return schedule
}

// GameDay runs a chaos game day
func GameDay(ctx context.Context, config GameDayConfig) (*GameDayResult, error) {
	result := &GameDayResult{
		StartTime: time.Now(),
		Scenarios: make([]ScenarioResult, 0),
	}

	engine := NewEngine()

	// Register standard handlers
	for t, h := range StandardFaultHandlers() {
		engine.RegisterHandler(t, h)
	}

	// Run each scenario
	for _, scenario := range config.Scenarios {
		scenarioResult := ScenarioResult{
			Name:      scenario.Name,
			StartTime: time.Now(),
		}

		// Run experiment
		if err := engine.StartExperiment(ctx, scenario.Experiment.ID); err != nil {
			scenarioResult.Success = false
			scenarioResult.Error = err.Error()
		} else {
			scenarioResult.Success = true
		}

		scenarioResult.EndTime = time.Now()
		result.Scenarios = append(result.Scenarios, scenarioResult)
	}

	result.EndTime = time.Now()
	return result, nil
}

// GameDayConfig holds game day configuration
type GameDayConfig struct {
	Name      string
	Scenarios []GameDayScenario
	Duration  time.Duration
}

// GameDayScenario represents a game day scenario
type GameDayScenario struct {
	Name         string
	Experiment   *Experiment
	ExpectedMTTR time.Duration
}

// GameDayResult holds game day results
type GameDayResult struct {
	StartTime time.Time
	EndTime   time.Time
	Scenarios []ScenarioResult
}

// ScenarioResult represents scenario results
type ScenarioResult struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	Error     string
	MTTR      time.Duration
}

// BlastRadius calculates blast radius
func BlastRadius(experiment *Experiment, services []string) map[string]float64 {
	radius := make(map[string]float64)

	// Calculate impact based on scope
	for _, service := range services {
		// Simple calculation - would be more complex in production
		if contains(experiment.Scope.Services, service) {
			radius[service] = experiment.Scope.Percentage
		}
	}

	return radius
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
