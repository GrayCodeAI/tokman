package core

import "context"

// CommandRunner abstracts shell command execution.
type CommandRunner interface {
	Run(ctx context.Context, args []string) (output string, exitCode int, err error)
	RunCombined(ctx context.Context, args []string) (output string, exitCode int, err error)
	LookPath(name string) (string, error)
}
