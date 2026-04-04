package commands

import (
	"os"
	"os/exec"
	"testing"
)

func TestExitCodeForErrorNil(t *testing.T) {
	if got := exitCodeForError(nil); got != 0 {
		t.Fatalf("exitCodeForError(nil) = %d, want 0", got)
	}
}

func TestExitCodeForErrorExitError(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 7")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected command to fail")
	}

	if got := exitCodeForError(err); got != 7 {
		t.Fatalf("exitCodeForError(exit 7) = %d, want 7", got)
	}
}

func TestExitCodeForErrorGenericError(t *testing.T) {
	if got := exitCodeForError(exec.ErrNotFound); got != 1 {
		t.Fatalf("exitCodeForError(generic) = %d, want 1", got)
	}
}

func TestExtractUnknownCommandArgsFallsBackToOSArgs(t *testing.T) {
	origFallbackArgs := fallbackArgs
	origArgs := os.Args
	t.Cleanup(func() {
		fallbackArgs = origFallbackArgs
		os.Args = origArgs
	})

	fallbackArgs = nil
	os.Args = []string{"tokman", "echo", "hi"}

	got := extractUnknownCommandArgs()
	if len(got) != 2 || got[0] != "echo" || got[1] != "hi" {
		t.Fatalf("extractUnknownCommandArgs() = %v, want [echo hi]", got)
	}
}
