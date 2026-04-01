package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/contextread"
)

func TestBuildContextFileAutoMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	content := "package main\n\nfunc alpha() {}\nfunc beta() {}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	out, _, _, _, err := buildContextFile(path, contextread.Options{Mode: "auto"})
	if err != nil {
		t.Fatalf("buildContextFile() error = %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestBuildContextFileDeltaMode(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte("package main\nfunc alpha() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	first, _, _, _, err := buildContextFile(path, contextread.Options{Mode: "delta", SaveSnapshot: true})
	if err != nil {
		t.Fatalf("first buildContextFile() error = %v", err)
	}
	if !strings.Contains(first, "No previous snapshot found") {
		t.Fatalf("expected initial snapshot message, got %q", first)
	}

	if err := os.WriteFile(path, []byte("package main\nfunc beta() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() second error = %v", err)
	}

	second, _, _, _, err := buildContextFile(path, contextread.Options{Mode: "delta", SaveSnapshot: true})
	if err != nil {
		t.Fatalf("second buildContextFile() error = %v", err)
	}
	if !strings.Contains(second, "Delta:") {
		t.Fatalf("expected delta output, got %q", second)
	}
}
