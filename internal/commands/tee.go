package commands

import (
	"os/exec"
	
	"github.com/GrayCodeAI/tokman/internal/tee"
)

// TeeOnFailure writes raw output to a file when a command fails.
// Returns a hint string to append to filtered output if tee was written.
func TeeOnFailure(raw string, commandSlug string, err error) string {
	if err == nil {
		return ""
	}
	
	// Get exit code from exec.ExitError if possible
	exitCode := 1 // Default to failure
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() > 0 {
			exitCode = exitErr.ExitCode()
		}
	}
	
	return tee.WriteAndHint(raw, commandSlug, exitCode)
}

// TeeOnFailureWithCode writes raw output to a file when exit code is non-zero.
func TeeOnFailureWithCode(raw string, commandSlug string, exitCode int) string {
	if exitCode == 0 {
		return ""
	}
	return tee.WriteAndHint(raw, commandSlug, exitCode)
}
