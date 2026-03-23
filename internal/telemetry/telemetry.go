package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Configuration
const (
	PingIntervalSecs = 23 * 3600 // 23 hours between pings
)

// TelemetryURL and Token can be set at build time via ldflags.
var (
	TelemetryURL   string
	TelemetryToken string
)

// StatsProvider defines the interface for getting tracking stats.
type StatsProvider interface {
	CountCommandsSince(since time.Time) (int64, error)
	TopCommands(limit int) ([]string, error)
	OverallSavingsPct() (float64, error)
	TokensSaved24h() (int64, error)
	TokensSavedTotal() (int64, error)
}

// Client handles telemetry ping operations.
type Client struct {
	url     string
	token   string
	dataDir string
	enabled bool
	stats   StatsProvider
}

// NewClient creates a new telemetry client.
func NewClient(stats StatsProvider) *Client {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dataDir := filepath.Join(home, ".local", "share", "tokman")

	return &Client{
		url:     TelemetryURL,
		token:   TelemetryToken,
		dataDir: dataDir,
		enabled: TelemetryURL != "",
		stats:   stats,
	}
}

// MaybePing sends a telemetry ping if enabled and not already sent today.
// This is fire-and-forget: errors are silently ignored.
func (c *Client) MaybePing() {
	// No URL configured → telemetry disabled
	if !c.enabled || c.url == "" {
		return
	}

	// Check opt-out: environment variable
	if os.Getenv("TOKMAN_TELEMETRY_DISABLED") == "1" {
		return
	}

	// Check opt-out: config file (handled by caller)

	// Check last ping time
	marker := c.markerPath()
	if info, err := os.Stat(marker); err == nil {
		if time.Since(info.ModTime()).Seconds() < PingIntervalSecs {
			return // Already pinged recently
		}
	}

	// Touch marker immediately (before sending) to avoid double-ping
	c.touchMarker(marker)

	// Send ping in background (never block the CLI)
	go c.sendPing()
}

func (c *Client) sendPing() {
	deviceHash := generateDeviceHash()
	version := getVersion()
	osName := runtime.GOOS
	arch := runtime.GOARCH
	installMethod := detectInstallMethod()

	// Get stats from tracking DB
	var commands24h int64
	var topCommands []string
	var savingsPct float64
	var tokensSaved24h, tokensSavedTotal int64

	if c.stats != nil {
		commands24h, _ = c.stats.CountCommandsSince(time.Now().Add(-24 * time.Hour))
		topCommands, _ = c.stats.TopCommands(5)
		savingsPct, _ = c.stats.OverallSavingsPct()
		tokensSaved24h, _ = c.stats.TokensSaved24h()
		tokensSavedTotal, _ = c.stats.TokensSavedTotal()
	}

	// Build payload using proper JSON marshaling to prevent injection
	telemetryPayload := struct {
		DeviceHash       string   `json:"device_hash"`
		Version          string   `json:"version"`
		OS               string   `json:"os"`
		Arch             string   `json:"arch"`
		InstallMethod    string   `json:"install_method"`
		Commands24h      int64    `json:"commands_24h"`
		TopCommands      []string `json:"top_commands"`
		SavingsPct       float64  `json:"savings_pct"`
		TokensSaved24h   int64    `json:"tokens_saved_24h"`
		TokensSavedTotal int64    `json:"tokens_saved_total"`
	}{
		DeviceHash:       deviceHash,
		Version:          version,
		OS:               osName,
		Arch:             arch,
		InstallMethod:    installMethod,
		Commands24h:      commands24h,
		TopCommands:      topCommands,
		SavingsPct:       savingsPct,
		TokensSaved24h:   tokensSaved24h,
		TokensSavedTotal: tokensSavedTotal,
	}
	payloadBytes, err := json.Marshal(telemetryPayload)
	if err != nil {
		return
	}
	payload := string(payloadBytes)

	// Send HTTP POST (with 2-second timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, strings.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TokMan-Token", c.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func (c *Client) markerPath() string {
	return filepath.Join(c.dataDir, ".telemetry_last_ping")
}

func (c *Client) touchMarker(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}
	if err := os.WriteFile(path, []byte{}, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", path, err)
	}
}

// generateDeviceHash creates an anonymous SHA-256 hash of hostname:username.
func generateDeviceHash() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	data := fmt.Sprintf("%s:%s", hostname, username)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func getVersion() string {
	// This could be set via ldflags at build time
	if v := os.Getenv("TOKMAN_VERSION"); v != "" {
		return v
	}
	return "dev"
}

// detectInstallMethod determines how tokman was installed.
func detectInstallMethod() string {
	// Check if running from GOPATH/bin (go install)
	execPath, err := os.Executable()
	if err == nil {
		// Check common install locations
		home, err := os.UserHomeDir()
		if err != nil {
			return "unknown"
		}
		goBinPath := filepath.Join(home, "go", "bin")
		if strings.HasPrefix(execPath, goBinPath) {
			return "go-install"
		}
		// Check for Homebrew on macOS
		if strings.HasPrefix(execPath, "/usr/local/Cellar/") || strings.HasPrefix(execPath, "/opt/homebrew/Cellar/") {
			return "homebrew"
		}
		// Check for Linux package manager paths
		if strings.HasPrefix(execPath, "/usr/bin/") || strings.HasPrefix(execPath, "/usr/local/bin/") {
			return "package-manager"
		}
	}
	return "unknown"
}

// Global client
var (
	defaultClient *Client
	clientOnce    sync.Once
)

// Init initializes the global telemetry client.
func Init(stats StatsProvider) {
	clientOnce.Do(func() {
		defaultClient = NewClient(stats)
	})
}

// MaybePing sends a telemetry ping using the global client.
func MaybePing() {
	if defaultClient != nil {
		defaultClient.MaybePing()
	}
}

// IsEnabled returns whether telemetry is configured.
func IsEnabled() bool {
	return TelemetryURL != ""
}

// IsOptedOut checks if telemetry is explicitly disabled.
func IsOptedOut() bool {
	return os.Getenv("TOKMAN_TELEMETRY_DISABLED") == "1"
}
