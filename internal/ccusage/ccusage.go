// Package ccusage provides integration with the ccusage CLI for fetching
// Claude Code API usage metrics. Handles subprocess execution, JSON parsing,
// and graceful degradation when ccusage is unavailable.
package ccusage

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Metrics represents usage metrics from ccusage for a single period
type Metrics struct {
	InputTokens         uint64  `json:"inputTokens"`
	OutputTokens        uint64  `json:"outputTokens"`
	CacheCreationTokens uint64  `json:"cacheCreationTokens"`
	CacheReadTokens     uint64  `json:"cacheReadTokens"`
	TotalTokens         uint64  `json:"totalTokens"`
	TotalCost           float64 `json:"totalCost"`
}

// Period represents period data with key (date/month/week) and metrics
type Period struct {
	Key     string // "2026-01-30" (daily), "2026-01" (monthly), "2026-01-20" (weekly ISO monday)
	Metrics Metrics
}

// Granularity represents time granularity for ccusage reports
type Granularity int

const (
	Daily Granularity = iota
	Weekly
	Monthly
)

// Internal types for JSON deserialization
type dailyResponse struct {
	Daily []dailyEntry `json:"daily"`
}

type dailyEntry struct {
	Date string `json:"date"`
	Metrics
}

type weeklyResponse struct {
	Weekly []weeklyEntry `json:"weekly"`
}

type weeklyEntry struct {
	Week string `json:"week"`
	Metrics
}

type monthlyResponse struct {
	Monthly []monthlyEntry `json:"monthly"`
}

type monthlyEntry struct {
	Month string `json:"month"`
	Metrics
}

// binaryExists checks if ccusage binary exists in PATH
func binaryExists() bool {
	_, err := exec.LookPath("ccusage")
	return err == nil
}

// buildCommand builds the ccusage command, falling back to npx if binary not in PATH
func buildCommand() (*exec.Cmd, bool) {
	if binaryExists() {
		return exec.Command("ccusage"), true
	}

	// Fallback: try npx
	npxCheck := exec.Command("npx", "ccusage", "--help")
	npxCheck.Stdout = nil
	npxCheck.Stderr = nil
	if err := npxCheck.Run(); err == nil {
		return exec.Command("npx", "ccusage"), true
	}

	return nil, false
}

// IsAvailable checks if ccusage CLI is available (binary or via npx)
func IsAvailable() bool {
	_, ok := buildCommand()
	return ok
}

// Fetch retrieves usage data from ccusage.
//
// Returns nil if ccusage is unavailable (graceful degradation).
// Returns parsed data on success.
// Returns error only on unexpected failures (JSON parse, etc.).
func Fetch(granularity Granularity) ([]Period, error) {
	cmd, ok := buildCommand()
	if !ok {
		fmt.Fprintln(stderr, "⚠️  ccusage not found. Install: npm i -g ccusage (or use npx ccusage)")
		return nil, nil
	}

	subcommand := "daily"
	switch granularity {
	case Weekly:
		subcommand = "weekly"
	case Monthly:
		subcommand = "monthly"
	}

	cmd.Args = append(cmd.Args, subcommand, "--json", "--since", "20250101")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(stderr, "⚠️  ccusage exited with %s\n", exitErr.Error())
		} else {
			fmt.Fprintf(stderr, "⚠️  ccusage execution failed: %s\n", err)
		}
		return nil, nil
	}

	return parseJSON(string(output), granularity)
}

// stderrWriter is used for writing warnings to stderr
type stderrWriter struct{}

func (w *stderrWriter) Write(p []byte) (int, error) {
	return os.Stderr.Write(p)
}

// stderr is a variable for testing
var stderr = &stderrWriter{}

// parseJSON parses the ccusage JSON output
func parseJSON(jsonStr string, granularity Granularity) ([]Period, error) {
	switch granularity {
	case Daily:
		var resp dailyResponse
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			return nil, fmt.Errorf("invalid JSON structure for daily data: %w", err)
		}
		periods := make([]Period, len(resp.Daily))
		for i, e := range resp.Daily {
			periods[i] = Period{Key: e.Date, Metrics: e.Metrics}
		}
		return periods, nil

	case Weekly:
		var resp weeklyResponse
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			return nil, fmt.Errorf("invalid JSON structure for weekly data: %w", err)
		}
		periods := make([]Period, len(resp.Weekly))
		for i, e := range resp.Weekly {
			periods[i] = Period{Key: e.Week, Metrics: e.Metrics}
		}
		return periods, nil

	case Monthly:
		var resp monthlyResponse
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			return nil, fmt.Errorf("invalid JSON structure for monthly data: %w", err)
		}
		periods := make([]Period, len(resp.Monthly))
		for i, e := range resp.Monthly {
			periods[i] = Period{Key: e.Month, Metrics: e.Metrics}
		}
		return periods, nil
	}

	return nil, fmt.Errorf("unknown granularity: %d", granularity)
}
