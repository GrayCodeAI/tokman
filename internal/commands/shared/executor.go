package shared

import (
	"fmt"
	"os"
	"time"

	"github.com/GrayCodeAI/tokman/internal/config"
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
// Returns an error instead of calling os.Exit so callers control exit behavior.
func ExecuteAndRecord(name string, fn func() (string, string, error)) error {
	startTime := time.Now()
	raw, filtered, err := fn()
	execTime := time.Since(startTime).Milliseconds()

	if err != nil {
		return err
	}

	fmt.Print(filtered)

	if rerr := RecordCommand(name, raw, filtered, execTime, true); rerr != nil && Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", rerr)
	}
	return nil
}
