package telemetry

import (
	"os"
	"testing"
)

func TestGenerateDeviceHash(t *testing.T) {
	h1 := generateDeviceHash()
	h2 := generateDeviceHash()

	// Hash should be stable
	if h1 != h2 {
		t.Error("Device hash should be stable across calls")
	}

	// Hash should be SHA-256 hex (64 chars)
	if len(h1) != 64 {
		t.Errorf("Device hash should be 64 chars (SHA-256 hex), got %d", len(h1))
	}
}

func TestGetVersion(t *testing.T) {
	// Default version when env not set
	orig := os.Getenv("TOKMAN_VERSION")
	defer os.Setenv("TOKMAN_VERSION", orig)

	os.Unsetenv("TOKMAN_VERSION")
	v := getVersion()
	if v == "" {
		t.Error("getVersion() should return non-empty string")
	}

	os.Setenv("TOKMAN_VERSION", "1.2.3")
	v = getVersion()
	if v != "1.2.3" {
		t.Errorf("getVersion() = %s, want 1.2.3", v)
	}
}

func TestNewClient(t *testing.T) {
	// Save original values
	origURL := TelemetryURL
	origToken := TelemetryToken
	defer func() {
		TelemetryURL = origURL
		TelemetryToken = origToken
	}()

	// Test with no URL
	TelemetryURL = ""
	TelemetryToken = ""
	client := NewClient(nil)
	if client.enabled {
		t.Error("Client should be disabled when no URL is set")
	}
}

func TestIsOptedOut(t *testing.T) {
	orig := os.Getenv("TOKMAN_TELEMETRY_DISABLED")
	defer os.Setenv("TOKMAN_TELEMETRY_DISABLED", orig)

	os.Unsetenv("TOKMAN_TELEMETRY_DISABLED")
	if IsOptedOut() {
		t.Error("IsOptedOut() should be false when env not set")
	}

	os.Setenv("TOKMAN_TELEMETRY_DISABLED", "1")
	if !IsOptedOut() {
		t.Error("IsOptedOut() should be true when env is 1")
	}
}
