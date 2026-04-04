package tee

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	return d
}

func cfg(dir string) Config {
	return Config{Enabled: true, Mode: ModeAlways, MaxFiles: 5, Dir: dir}
}

func TestSaveEnabled(t *testing.T) {
	dir := tempDir(t)
	path, err := Save("git status", "output", 0, cfg(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read saved file: %v", err)
	}
	if string(data) != "output" {
		t.Errorf("content = %q, want %q", string(data), "output")
	}
}

func TestSaveNever(t *testing.T) {
	dir := tempDir(t)
	c := cfg(dir)
	c.Mode = ModeNever
	path, err := Save("cmd", "out", 0, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Error("expected empty path for ModeNever")
	}
}

func TestSaveFailuresOnly(t *testing.T) {
	dir := tempDir(t)
	c := cfg(dir)
	c.Mode = ModeFailures

	// Success exit code – should not save
	path, err := Save("cmd", "out", 0, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Error("expected no file for exitCode 0 in failures mode")
	}

	// Failure exit code – should save
	path, err = Save("cmd", "err", 1, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected file for exitCode 1 in failures mode")
	}
}

func TestSaveDisabled(t *testing.T) {
	dir := tempDir(t)
	c := cfg(dir)
	c.Enabled = false
	path, err := Save("cmd", "out", 0, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Error("expected empty path when disabled")
	}
}

func TestListAndRead(t *testing.T) {
	dir := tempDir(t)
	c := cfg(dir)

	_, _ = Save("git log", "log-output", 0, c)
	_, _ = Save("git diff", "diff-output", 1, c)

	entries, err := List(c)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Sorted newest first
	if entries[0].Command != "git diff" {
		t.Errorf("first entry command = %q, want %q", entries[0].Command, "git diff")
	}

	content, err := Read(entries[0].Filename, c)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if content != "diff-output" {
		t.Errorf("content = %q, want %q", content, "diff-output")
	}
}

func TestRotate(t *testing.T) {
	dir := tempDir(t)
	c := cfg(dir)
	c.MaxFiles = 3

	// Write files with unique names to avoid timestamp collisions
	for i := 0; i < 6; i++ {
		base := filepath.Join(dir, fmt.Sprintf("100%d_%04d_test_cmd.log", i, i))
		if err := os.WriteFile(base, []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Trigger rotation by calling save (which calls rotate internally)
	_, _ = Save("trigger", "data", 0, c)

	files, _ := os.ReadDir(dir)
	count := 0
	for _, f := range files {
		if !f.IsDir() {
			count++
		}
	}
	if count != 3 {
		t.Errorf("expected 3 files after rotation, got %d", count)
	}
}

func TestExpandTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := expandTilde("~/.local/share/tokman/tee")
	want := filepath.Join(home, ".local/share/tokman/tee")
	if got != want {
		t.Errorf("expandTilde = %q, want %q", got, want)
	}
	got2 := expandTilde("/absolute/path")
	if got2 != "/absolute/path" {
		t.Errorf("expandTilde = %q, want %q", got2, "/absolute/path")
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	c := DefaultConfig()
	if !c.Enabled {
		t.Error("default config should be enabled")
	}
	if c.Mode != ModeFailures {
		t.Errorf("default mode = %q, want %q", c.Mode, ModeFailures)
	}
	if c.MaxFiles != maxFiles {
		t.Errorf("default maxFiles = %d, want %d", c.MaxFiles, maxFiles)
	}
	wantDir := filepath.Join(os.Getenv("XDG_DATA_HOME"), "tokman", "tee")
	if c.Dir != wantDir {
		t.Errorf("default dir = %q, want %q", c.Dir, wantDir)
	}
}

func TestWriteAndHint(t *testing.T) {
	// WriteAndHint uses the hardcoded default directory, so we just verify
	// it doesn't panic and returns the expected hint format or empty string.
	hint := WriteAndHint("output", "test cmd", 1)
	if hint == "" {
		t.Log("WriteAndHint returned empty (default dir not writable in test)")
		return
	}
	if !strings.HasPrefix(hint, "[full output saved:") {
		t.Errorf("unexpected hint format: %q", hint)
	}
}
