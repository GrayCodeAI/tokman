package tee

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cargo_test", "cargo_test"},
		{"cargo test", "cargo_test"},
		{"cargo-test", "cargo-test"},
		{"go/test/./pkg", "go_test___pkg"},
		{"simple", "simple"},
		{strings.Repeat("a", 50), strings.Repeat("a", 40)}, // Truncates at 40
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeSlug(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("Default config should have tee enabled")
	}
	if cfg.Mode != ModeFailures {
		t.Errorf("Default mode should be ModeFailures, got %v", cfg.Mode)
	}
	if cfg.MaxFiles != DefaultMaxFiles {
		t.Errorf("Default MaxFiles should be %d, got %d", DefaultMaxFiles, cfg.MaxFiles)
	}
}

func TestShouldTee(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		rawLen   int
		exitCode int
		want     bool
	}{
		{
			name:   "disabled",
			config: Config{Enabled: false, Mode: ModeFailures},
			rawLen: 1000, exitCode: 1, want: false,
		},
		{
			name:   "mode_never",
			config: Config{Enabled: true, Mode: ModeNever},
			rawLen: 1000, exitCode: 1, want: false,
		},
		{
			name:   "skip_small_output",
			config: Config{Enabled: true, Mode: ModeFailures},
			rawLen: 100, exitCode: 1, want: false,
		},
		{
			name:   "skip_success_in_failures_mode",
			config: Config{Enabled: true, Mode: ModeFailures},
			rawLen: 1000, exitCode: 0, want: false,
		},
		{
			name:   "proceed_on_failure",
			config: Config{Enabled: true, Mode: ModeFailures},
			rawLen: 1000, exitCode: 1, want: true,
		},
		{
			name:   "always_mode_success",
			config: Config{Enabled: true, Mode: ModeAlways},
			rawLen: 1000, exitCode: 0, want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tee := New(tt.config)
			got := tee.ShouldTee(tt.rawLen, tt.exitCode)
			if got != tt.want {
				t.Errorf("ShouldTee() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	tee := New(Config{
		Enabled:     true,
		Mode:        ModeAlways,
		MaxFiles:    20,
		MaxFileSize: DefaultMaxSize,
		Directory:   tmpDir,
	})

	content := strings.Repeat("error: test failed\n", 50)
	path := tee.Write(content, "cargo_test", 1)

	if path == "" {
		t.Fatal("Write() returned empty path")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Tee file was not created at %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read tee file: %v", err)
	}
	if !strings.Contains(string(data), "error: test failed") {
		t.Error("Tee file should contain the error message")
	}
}

func TestWriteTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	tee := New(Config{
		Enabled:     true,
		Mode:        ModeAlways,
		MaxFiles:    20,
		MaxFileSize: 1000, // Small size for testing
		Directory:   tmpDir,
	})

	bigOutput := strings.Repeat("x", 2000)
	path := tee.Write(bigOutput, "test", 1)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read tee file: %v", err)
	}

	if !strings.Contains(string(data), "truncated at 1000 bytes") {
		t.Error("Tee file should indicate truncation")
	}
}

func TestCleanupOldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 25 .log files with epoch prefixes (simulating real files)
	for i := 0; i < 25; i++ {
		epoch := 1000000 + i
		filename := filepath.Join(tmpDir, fmt.Sprintf("%d_test.log", epoch))
		os.WriteFile(filename, []byte("content"), 0644)
	}

	tee := New(Config{
		Enabled:     true,
		Mode:        ModeAlways,
		MaxFiles:    20,
		MaxFileSize: DefaultMaxSize,
		Directory:   tmpDir,
	})
	tee.cleanupOldFiles(tmpDir)

	// Count remaining files
	entries, _ := os.ReadDir(tmpDir)
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			count++
		}
	}

	if count != 20 {
		t.Errorf("Expected 20 files after cleanup, got %d", count)
	}
}
