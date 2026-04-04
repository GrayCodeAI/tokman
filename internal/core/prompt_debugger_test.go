package core

import (
	"path/filepath"
	"testing"
)

func TestNewPromptDebuggerUsesDataPathByDefault(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	debugger := NewPromptDebugger("")
	want := filepath.Join(dataHome, "tokman", "prompts")
	if debugger.dir != want {
		t.Fatalf("PromptDebugger dir = %q, want %q", debugger.dir, want)
	}
}
