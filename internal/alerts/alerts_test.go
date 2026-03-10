package alerts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestManager(t *testing.T) *Manager {
	tmpDir, err := os.MkdirTemp("", "tokman-alerts-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	config := DefaultConfig()
	config.CooldownMinutes = 0 // Disable cooldown for testing

	return &Manager{
		config:    config,
		alerts:    []Alert{},
		alertFile: filepath.Join(tmpDir, "alerts.json"),
		cooldowns: make(map[AlertType]time.Time),
	}
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	m := NewManager(config)

	if m == nil {
		t.Fatal("Expected non-nil manager")
	}

	if !m.config.Enabled {
		t.Error("Expected alerts to be enabled by default")
	}
}

func TestCheckTokenLimit(t *testing.T) {
	m := newTestManager(t)

	// Below limits - no alerts
	alerts := m.CheckTokenLimit(500000, 1000000)
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts below limits, got %d", len(alerts))
	}

	// Above daily limit
	alerts = m.CheckTokenLimit(2000000, 1000000)
	if len(alerts) == 0 {
		t.Error("Expected alert for exceeding daily limit")
	}
}

func TestCheckUsageSpike(t *testing.T) {
	m := newTestManager(t)

	// Normal usage - no spike
	alerts := m.CheckUsageSpike(10, 10)
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts for normal usage, got %d", len(alerts))
	}

	// Spike detected (3x baseline)
	alerts = m.CheckUsageSpike(30, 10)
	if len(alerts) == 0 {
		t.Error("Expected alert for usage spike")
	}

	// Zero baseline - no check
	alerts = m.CheckUsageSpike(30, 0)
	if len(alerts) != 0 {
		t.Error("Expected no alert when baseline is zero")
	}
}

func TestCheckCostThreshold(t *testing.T) {
	m := newTestManager(t)

	// Below thresholds
	alerts := m.CheckCostThreshold(5.0, 25.0)
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts below thresholds, got %d", len(alerts))
	}

	// Above daily threshold
	alerts = m.CheckCostThreshold(15.0, 25.0)
	if len(alerts) == 0 {
		t.Error("Expected alert for exceeding daily cost threshold")
	}
}

func TestCheckEfficiency(t *testing.T) {
	m := newTestManager(t)

	// Good efficiency
	alerts := m.CheckEfficiency(75.0)
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts for good efficiency, got %d", len(alerts))
	}

	// Low efficiency
	alerts = m.CheckEfficiency(25.0)
	if len(alerts) == 0 {
		t.Error("Expected alert for low efficiency")
	}
}

func TestCheckParseFailureRate(t *testing.T) {
	m := newTestManager(t)

	// Low failure rate
	alerts := m.CheckParseFailureRate(0.05)
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts for low failure rate, got %d", len(alerts))
	}

	// High failure rate
	alerts = m.CheckParseFailureRate(0.15)
	if len(alerts) == 0 {
		t.Error("Expected alert for high parse failure rate")
	}
}

func TestAlertAcknowledgement(t *testing.T) {
	m := newTestManager(t)

	// Create an alert
	alerts := m.CheckTokenLimit(2000000, 1000000)
	if len(alerts) == 0 {
		t.Fatal("Expected alert to be created")
	}

	alertID := alerts[0].ID

	// Acknowledge
	err := m.Acknowledge(alertID)
	if err != nil {
		t.Errorf("Failed to acknowledge alert: %v", err)
	}

	// Check it's acknowledged
	active := m.GetActive()
	for _, a := range active {
		if a.ID == alertID {
			t.Error("Alert should not be in active list after acknowledgement")
		}
	}

	// Try to acknowledge non-existent alert
	err = m.Acknowledge("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent alert")
	}
}

func TestAlertResolve(t *testing.T) {
	m := newTestManager(t)

	// Create an alert
	alerts := m.CheckTokenLimit(2000000, 1000000)
	if len(alerts) == 0 {
		t.Fatal("Expected alert to be created")
	}

	alertID := alerts[0].ID

	// Resolve
	err := m.Resolve(alertID)
	if err != nil {
		t.Errorf("Failed to resolve alert: %v", err)
	}

	// Check it's resolved
	all := m.GetAll(0)
	found := false
	for _, a := range all {
		if a.ID == alertID {
			found = true
			if !a.Resolved {
				t.Error("Alert should be marked as resolved")
			}
		}
	}
	if !found {
		t.Error("Alert should still exist in list")
	}
}

func TestCooldown(t *testing.T) {
	config := DefaultConfig()
	config.CooldownMinutes = 1 // 1 minute cooldown
	m := newTestManager(t)
	m.config.CooldownMinutes = 1

	// First alert should trigger
	alerts := m.CheckTokenLimit(2000000, 1000000)
	if len(alerts) == 0 {
		t.Fatal("Expected first alert to trigger")
	}

	// Second alert within cooldown should NOT trigger
	alerts = m.CheckTokenLimit(2000000, 1000000)
	if len(alerts) != 0 {
		t.Error("Expected second alert to be suppressed by cooldown")
	}
}

func TestDisabledAlerts(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false
	m := newTestManager(t)
	m.config.Enabled = false

	// All checks should return no alerts when disabled
	alerts := m.CheckTokenLimit(2000000, 1000000)
	if len(alerts) != 0 {
		t.Error("Expected no alerts when disabled")
	}

	alerts = m.CheckUsageSpike(30, 10)
	if len(alerts) != 0 {
		t.Error("Expected no alerts when disabled")
	}
}

func TestStats(t *testing.T) {
	m := newTestManager(t)

	// Create some alerts
	m.CheckTokenLimit(2000000, 1000000)
	m.CheckCostThreshold(15.0, 25.0)

	stats := m.Stats()

	if stats["total_alerts"].(int) < 2 {
		t.Errorf("Expected at least 2 total alerts, got %d", stats["total_alerts"])
	}

	if stats["active_alerts"].(int) < 2 {
		t.Errorf("Expected at least 2 active alerts, got %d", stats["active_alerts"])
	}
}

func TestPersistence(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tokman-alerts-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create manager with custom path
	config := DefaultConfig()
	config.CooldownMinutes = 0
	m := &Manager{
		config:    config,
		alerts:    []Alert{},
		alertFile: filepath.Join(tmpDir, "alerts.json"),
		cooldowns: make(map[AlertType]time.Time),
	}

	// Create an alert
	m.CheckTokenLimit(2000000, 1000000)

	// Verify file was created
	if _, err := os.Stat(m.alertFile); os.IsNotExist(err) {
		t.Error("Alert file should have been created")
	}

	// Create new manager to test loading
	m2 := &Manager{
		config:    config,
		alerts:    []Alert{},
		alertFile: m.alertFile,
		cooldowns: make(map[AlertType]time.Time),
	}
	m2.load()

	if len(m2.alerts) == 0 {
		t.Error("Alerts should have been loaded from file")
	}
}

func TestClearResolved(t *testing.T) {
	m := newTestManager(t)

	// Create and resolve an alert
	alerts := m.CheckTokenLimit(2000000, 1000000)
	if len(alerts) == 0 {
		t.Fatal("Expected alert to be created")
	}

	m.Resolve(alerts[0].ID)

	// Clear resolved
	cleared := m.ClearResolved()
	if cleared != 1 {
		t.Errorf("Expected 1 alert to be cleared, got %d", cleared)
	}

	// Check it's gone
	all := m.GetAll(0)
	for _, a := range all {
		if a.ID == alerts[0].ID {
			t.Error("Resolved alert should have been cleared")
		}
	}
}
