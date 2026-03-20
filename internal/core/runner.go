package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// OSCommandRunner executes real shell commands using os/exec.
type OSCommandRunner struct {
	Env []string
}

// NewOSCommandRunner creates a command runner with the current environment.
func NewOSCommandRunner() *OSCommandRunner {
	return &OSCommandRunner{
		Env: os.Environ(),
	}
}

// Run executes a command and captures combined stdout+stderr.
func (r *OSCommandRunner) Run(ctx context.Context, args []string) (string, int, error) {
	if len(args) == 0 {
		return "", 0, nil
	}

	cmdPath, err := exec.LookPath(args[0])
	if err != nil {
		hint := fmt.Sprintf("command not found: %s\n\nDid you install it? Try:\n  apt install %s  # Debian/Ubuntu\n  brew install %s  # macOS\n  dnf install %s  # Fedora", args[0], args[0], args[0], args[0])
		return hint, 127, err
	}

	cmd := exec.CommandContext(ctx, cmdPath, args[1:]...)
	cmd.Env = r.Env

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return string(output), exitCode, err
}

// RunCombined executes a command and returns output, exit code, and duration.
func (r *OSCommandRunner) RunCombined(ctx context.Context, args []string) (string, int, error) {
	return r.Run(ctx, args)
}

// LookPath resolves a command name to its full path.
func (r *OSCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// MockCommandRunner is a test double for CommandRunner.
type MockCommandRunner struct {
	Outputs   map[string]string
	ExitCodes map[string]int
	Errors    map[string]error
	Calls     []MockCall
}

// MockCall records a command invocation.
type MockCall struct {
	Args     []string
	Duration time.Duration
}

// NewMockCommandRunner creates a mock runner for testing.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		Outputs:   make(map[string]string),
		ExitCodes: make(map[string]int),
		Errors:    make(map[string]error),
	}
}

// Run returns a pre-configured output for testing.
func (m *MockCommandRunner) Run(ctx context.Context, args []string) (string, int, error) {
	key := args[0]
	m.Calls = append(m.Calls, MockCall{Args: args})
	return m.Outputs[key], m.ExitCodes[key], m.Errors[key]
}

// RunCombined delegates to Run for the mock.
func (m *MockCommandRunner) RunCombined(ctx context.Context, args []string) (string, int, error) {
	return m.Run(ctx, args)
}

// LookPath returns the command name as-is for the mock.
func (m *MockCommandRunner) LookPath(name string) (string, error) {
	return name, nil
}
