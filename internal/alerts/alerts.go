// Package alerts provides configurable alert management for TokMan.
// Supports token limits, usage spikes, cost thresholds, and custom alerts.
package alerts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeDailyTokenLimit  AlertType = "daily_token_limit"
	AlertTypeWeeklyTokenLimit AlertType = "weekly_token_limit"
	AlertTypeUsageSpike       AlertType = "usage_spike"
	AlertTypeCostThreshold    AlertType = "cost_threshold"
	AlertTypeEfficiencyDrop   AlertType = "efficiency_drop"
	AlertTypeParseFailureRate AlertType = "parse_failure_rate"
	AlertTypeCacheHitRate     AlertType = "cache_hit_rate"
)

// AlertSeverity represents the severity level
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents a triggered alert
type Alert struct {
	ID           string         `json:"id"`
	Type         AlertType      `json:"type"`
	Severity     AlertSeverity  `json:"severity"`
	Title        string         `json:"title"`
	Message      string         `json:"message"`
	Value        any            `json:"value,omitempty"`
	Threshold    any            `json:"threshold,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
	Acknowledged bool           `json:"acknowledged"`
	Resolved     bool           `json:"resolved"`
	ResolvedAt   *time.Time     `json:"resolved_at,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Config represents alert configuration
type Config struct {
	Enabled             bool    `json:"enabled"`
	DailyTokenLimit     int64   `json:"daily_token_limit"`
	WeeklyTokenLimit    int64   `json:"weekly_token_limit"`
	UsageSpikeThreshold float64 `json:"usage_spike_threshold"`
	CostThresholdDaily  float64 `json:"cost_threshold_daily"`
	CostThresholdWeekly float64 `json:"cost_threshold_weekly"`
	MinEfficiencyPct    float64 `json:"min_efficiency_pct"`
	MaxParseFailureRate float64 `json:"max_parse_failure_rate"`
	MinCacheHitRate     float64 `json:"min_cache_hit_rate"`
	CooldownMinutes     int     `json:"cooldown_minutes"`
	NotificationWebhook string  `json:"notification_webhook,omitempty"`
	NotificationEmail   string  `json:"notification_email,omitempty"`
}

// DefaultConfig returns default alert configuration
func DefaultConfig() Config {
	return Config{
		Enabled:             true,
		DailyTokenLimit:     1000000,
		WeeklyTokenLimit:    5000000,
		UsageSpikeThreshold: 2.0,
		CostThresholdDaily:  10.0,
		CostThresholdWeekly: 50.0,
		MinEfficiencyPct:    50.0,
		MaxParseFailureRate: 0.1,
		MinCacheHitRate:     0.0,
		CooldownMinutes:     60,
	}
}

// Manager handles alert management
type Manager struct {
	config    Config
	alerts    []Alert
	alertFile string
	mu        sync.RWMutex
	cooldowns map[AlertType]time.Time
}

// NewManager creates a new alert manager
func NewManager(config Config) *Manager {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	alertFile := filepath.Join(home, ".tokman", "alerts.json")

	m := &Manager{
		config:    config,
		alerts:    []Alert{},
		alertFile: alertFile,
		cooldowns: make(map[AlertType]time.Time),
	}

	m.load()
	return m
}

// CheckTokenLimit checks daily/weekly token limits
func (m *Manager) CheckTokenLimit(saved24h, savedTotal int64) []Alert {
	if !m.config.Enabled {
		return nil
	}

	var alerts []Alert

	// Check daily limit
	if saved24h > m.config.DailyTokenLimit {
		if alert := m.createAlert(
			AlertTypeDailyTokenLimit,
			SeverityWarning,
			"Daily Token Limit Exceeded",
			fmt.Sprintf("Token savings (%d) exceeded daily limit (%d)", saved24h, m.config.DailyTokenLimit),
			saved24h,
			m.config.DailyTokenLimit,
		); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	// Check weekly limit
	if savedTotal > m.config.WeeklyTokenLimit {
		if alert := m.createAlert(
			AlertTypeWeeklyTokenLimit,
			SeverityWarning,
			"Weekly Token Limit Exceeded",
			fmt.Sprintf("Total token savings (%d) exceeded weekly limit (%d)", savedTotal, m.config.WeeklyTokenLimit),
			savedTotal,
			m.config.WeeklyTokenLimit,
		); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// CheckUsageSpike detects unusual usage patterns
func (m *Manager) CheckUsageSpike(currentAvg, baselineAvg float64) []Alert {
	if !m.config.Enabled || baselineAvg == 0 {
		return nil
	}

	var alerts []Alert

	ratio := currentAvg / baselineAvg
	if ratio > m.config.UsageSpikeThreshold {
		if alert := m.createAlert(
			AlertTypeUsageSpike,
			SeverityInfo,
			"Usage Spike Detected",
			fmt.Sprintf("Current usage (%.1f) is %.1fx higher than baseline (%.1f)", currentAvg, ratio, baselineAvg),
			currentAvg,
			baselineAvg*m.config.UsageSpikeThreshold,
		); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// CheckCostThreshold monitors cost thresholds
func (m *Manager) CheckCostThreshold(dailyCost, weeklyCost float64) []Alert {
	if !m.config.Enabled {
		return nil
	}

	var alerts []Alert

	if dailyCost > m.config.CostThresholdDaily {
		if alert := m.createAlert(
			AlertTypeCostThreshold,
			SeverityWarning,
			"Daily Cost Threshold Exceeded",
			fmt.Sprintf("Daily cost ($%.2f) exceeded threshold ($%.2f)", dailyCost, m.config.CostThresholdDaily),
			dailyCost,
			m.config.CostThresholdDaily,
		); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	if weeklyCost > m.config.CostThresholdWeekly {
		if alert := m.createAlert(
			AlertTypeCostThreshold,
			SeverityCritical,
			"Weekly Cost Threshold Exceeded",
			fmt.Sprintf("Weekly cost ($%.2f) exceeded threshold ($%.2f)", weeklyCost, m.config.CostThresholdWeekly),
			weeklyCost,
			m.config.CostThresholdWeekly,
		); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// CheckEfficiency monitors filtering efficiency
func (m *Manager) CheckEfficiency(efficiencyPct float64) []Alert {
	if !m.config.Enabled {
		return nil
	}

	var alerts []Alert

	if efficiencyPct < m.config.MinEfficiencyPct {
		if alert := m.createAlert(
			AlertTypeEfficiencyDrop,
			SeverityWarning,
			"Low Filtering Efficiency",
			fmt.Sprintf("Efficiency (%.1f%%) below minimum threshold (%.1f%%)", efficiencyPct, m.config.MinEfficiencyPct),
			efficiencyPct,
			m.config.MinEfficiencyPct,
		); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// CheckParseFailureRate monitors parse failure rate
func (m *Manager) CheckParseFailureRate(failureRate float64) []Alert {
	if !m.config.Enabled {
		return nil
	}

	var alerts []Alert

	if failureRate > m.config.MaxParseFailureRate {
		if alert := m.createAlert(
			AlertTypeParseFailureRate,
			SeverityWarning,
			"High Parse Failure Rate",
			fmt.Sprintf("Parse failure rate (%.1f%%) exceeds threshold (%.1f%%)", failureRate*100, m.config.MaxParseFailureRate*100),
			failureRate,
			m.config.MaxParseFailureRate,
		); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// createAlert creates a new alert if not in cooldown
func (m *Manager) createAlert(alertType AlertType, severity AlertSeverity, title, message string, value, threshold any) *Alert {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check cooldown
	if lastTriggered, ok := m.cooldowns[alertType]; ok {
		if time.Since(lastTriggered) < time.Duration(m.config.CooldownMinutes)*time.Minute {
			return nil
		}
	}

	alert := Alert{
		ID:        fmt.Sprintf("%s-%d", alertType, time.Now().Unix()),
		Type:      alertType,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Value:     value,
		Threshold: threshold,
		Timestamp: time.Now(),
	}

	m.alerts = append(m.alerts, alert)
	m.cooldowns[alertType] = time.Now()

	// Keep only last 100 alerts
	if len(m.alerts) > 100 {
		m.alerts = m.alerts[len(m.alerts)-100:]
	}

	m.save()

	return &alert
}

// getActiveLocked returns active alerts. Caller must hold at least m.mu.RLock.
func (m *Manager) getActiveLocked() []Alert {
	var active []Alert
	for _, a := range m.alerts {
		if !a.Acknowledged && !a.Resolved {
			active = append(active, a)
		}
	}
	return active
}

// GetActive returns all active (unacknowledged) alerts.
func (m *Manager) GetActive() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getActiveLocked()
}

// GetAll returns all alerts
func (m *Manager) GetAll(limit int) []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit > 0 && len(m.alerts) > limit {
		return m.alerts[len(m.alerts)-limit:]
	}
	return m.alerts
}

// Acknowledge marks an alert as acknowledged
func (m *Manager) Acknowledge(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.alerts {
		if m.alerts[i].ID == id {
			m.alerts[i].Acknowledged = true
			m.save()
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", id)
}

// Resolve marks an alert as resolved
func (m *Manager) Resolve(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.alerts {
		if m.alerts[i].ID == id {
			m.alerts[i].Resolved = true
			now := time.Now()
			m.alerts[i].ResolvedAt = &now
			m.save()
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", id)
}

// ClearResolved removes all resolved alerts
func (m *Manager) ClearResolved() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var active []Alert
	cleared := 0
	for _, a := range m.alerts {
		if !a.Resolved {
			active = append(active, a)
		} else {
			cleared++
		}
	}

	m.alerts = active
	m.save()

	return cleared
}

// UpdateConfig updates the alert configuration
func (m *Manager) UpdateConfig(config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// save persists alerts to disk
func (m *Manager) save() {
	if m.alertFile == "" {
		return
	}

	dir := filepath.Dir(m.alertFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}

	data, err := json.MarshalIndent(m.alerts, "", "  ")
	if err != nil {
		return
	}

	if err := os.WriteFile(m.alertFile, data, 0600); err != nil {
		// Log but don't fail - alert persistence is non-critical
		_ = err
	}
}

// load restores alerts from disk
func (m *Manager) load() {
	if m.alertFile == "" {
		return
	}

	data, err := os.ReadFile(m.alertFile)
	if err != nil {
		return
	}

	if err := json.Unmarshal(data, &m.alerts); err != nil {
		// Start fresh with empty alerts on corrupt file
		m.alerts = []Alert{}
	}
}

// Stats returns alert statistics
func (m *Manager) Stats() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]any{
		"total_alerts":       len(m.alerts),
		"active_alerts":      len(m.getActiveLocked()),
		"alerts_by_type":     make(map[AlertType]int),
		"alerts_by_severity": make(map[AlertSeverity]int),
	}

	byType := stats["alerts_by_type"].(map[AlertType]int)
	bySeverity := stats["alerts_by_severity"].(map[AlertSeverity]int)

	for _, a := range m.alerts {
		byType[a.Type]++
		bySeverity[a.Severity]++
	}

	return stats
}
