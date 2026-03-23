package shared

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/tee"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// Command execution, recording, and tee-on-failure.
// Depends on core, tracking, tee, and config packages.

// TeeOnFailure writes output to tee file on error.
func TeeOnFailure(raw string, commandSlug string, err error) string {
	if err == nil {
		return ""
	}
	exitCode := 1
	if exitErr, ok := err.(*os.PathError); ok {
		_ = exitErr
	}
	return tee.WriteAndHint(raw, commandSlug, exitCode)
}

// TeeOnFailureWithCode writes output to tee file with explicit exit code.
func TeeOnFailureWithCode(raw string, commandSlug string, exitCode int) string {
	if exitCode == 0 {
		return ""
	}
	return tee.WriteAndHint(raw, commandSlug, exitCode)
}

// RecordCommand records command execution metrics to the tracking database.
func RecordCommand(command, originalOutput, filteredOutput string, execTimeMs int64, success bool) error {
	cfg := GetCachedConfig()

	if !cfg.Tracking.Enabled {
		return nil
	}

	tracker, err := tracking.NewTracker(cfg.GetDatabasePath())
	if err != nil {
		return err
	}
	defer tracker.Close()

	originalTokens := tracking.EstimateTokens(originalOutput)
	filteredTokens := tracking.EstimateTokens(filteredOutput)
	savedTokens := 0
	if originalTokens > filteredTokens {
		savedTokens = originalTokens - filteredTokens
	}

	record := &tracking.CommandRecord{
		Command:        command,
		OriginalOutput: originalOutput,
		FilteredOutput: filteredOutput,
		OriginalTokens: originalTokens,
		FilteredTokens: filteredTokens,
		SavedTokens:    savedTokens,
		ProjectPath:    config.ProjectPath(),
		ExecTimeMs:     execTimeMs,
		Timestamp:      time.Now(),
		ParseSuccess:   success,
	}

	return tracker.Record(record)
}

// ExecuteAndRecord runs a command function, prints output, and records metrics.
// This consolidates the common pattern of: time -> execute -> print -> record.
func ExecuteAndRecord(name string, fn func() (string, string, error)) {
	startTime := time.Now()
	raw, filtered, err := fn()
	execTime := time.Since(startTime).Milliseconds()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(filtered)

	if rerr := RecordCommand(name, raw, filtered, execTime, true); rerr != nil && Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", rerr)
	}
}

// RunAndFilter executes a command, applies a filter, prints output, and tracks tokens.
// This eliminates the boilerplate pattern duplicated across 77+ command files.
func RunAndFilter(name string, cmdArgs []string, filterFunc func(string) string, trackCmd string) error {
	timer := tracking.Start()
	if Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: %s %s\n", name, cmdArgs)
	}
	runner := core.NewOSCommandRunner()
	output, _, err := runner.Run(context.Background(), append([]string{name}, cmdArgs...))
	raw := output
	filtered := filterFunc(raw)
	fmt.Print(filtered)
	originalTokens := core.EstimateTokens(raw)
	filteredTokens := core.EstimateTokens(filtered)
	timer.Track(trackCmd, "tokman "+name, originalTokens, filteredTokens)
	return err
}
