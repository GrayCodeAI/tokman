package lang

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func runRspecCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rspec %s\n", strings.Join(args, " "))
	}

	// Use --format json for structured output
	jsonArgs := append([]string{"--format", "json"}, args...)
	execCmd := exec.Command("rspec", jsonArgs...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRspecOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rspec", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("rspec %s", strings.Join(args, " ")), "tokman ruby rspec", originalTokens, filteredTokens)

	return err
}

// RSpecJSON represents the JSON output from rspec --format json
type RSpecJSON struct {
	Version  string         `json:"version"`
	Examples []RSpecExample `json:"examples"`
	Summary  RSpecSummary   `json:"summary"`
}

type RSpecExample struct {
	ID              string          `json:"id"`
	Description     string          `json:"description"`
	FullDescription string          `json:"full_description"`
	Status          string          `json:"status"`
	FilePath        string          `json:"file_path"`
	LineNumber      int             `json:"line_number"`
	Exception       *RSpecException `json:"exception,omitempty"`
}

type RSpecException struct {
	Class     string   `json:"class"`
	Message   string   `json:"message"`
	Backtrace []string `json:"backtrace"`
}

type RSpecSummary struct {
	Duration     float64 `json:"duration"`
	ExampleCount int     `json:"example_count"`
	FailureCount int     `json:"failure_count"`
	PendingCount int     `json:"pending_count"`
}

func filterRspecOutput(raw string) string {
	// Try to parse as JSON
	var rspec RSpecJSON
	if err := json.Unmarshal([]byte(raw), &rspec); err != nil {
		// Fall back to text parsing
		return filterRspecTextOutput(raw)
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		return filterRspecOutputUltraCompact(rspec)
	}

	var result []string
	result = append(result, "📋 RSpec Results:")
	result = append(result, fmt.Sprintf("   ✅ %d passed", rspec.Summary.ExampleCount-rspec.Summary.FailureCount-rspec.Summary.PendingCount))
	if rspec.Summary.FailureCount > 0 {
		result = append(result, fmt.Sprintf("   ❌ %d failed", rspec.Summary.FailureCount))
	}
	if rspec.Summary.PendingCount > 0 {
		result = append(result, fmt.Sprintf("   ⏸️  %d pending", rspec.Summary.PendingCount))
	}
	result = append(result, fmt.Sprintf("   ⏱️  %.2fs", rspec.Summary.Duration))

	// Show failures
	failures := 0
	for _, ex := range rspec.Examples {
		if ex.Status == "failed" && ex.Exception != nil {
			if failures == 0 {
				result = append(result, "")
				result = append(result, "Failures:")
			}
			failures++
			if failures <= 10 {
				result = append(result, fmt.Sprintf("   - %s:%d: %s",
					shared.TruncateLine(ex.FilePath, 40),
					ex.LineNumber,
					shared.TruncateLine(ex.Description, 50)))
				result = append(result, fmt.Sprintf("     %s",
					shared.TruncateLine(ex.Exception.Message, 80)))
			}
		}
	}
	if failures > 10 {
		result = append(result, fmt.Sprintf("   ... +%d more failures", failures-10))
	}

	return strings.Join(result, "\n")
}

func filterRspecOutputUltraCompact(rspec RSpecJSON) string {
	passed := rspec.Summary.ExampleCount - rspec.Summary.FailureCount - rspec.Summary.PendingCount
	var parts []string

	parts = append(parts, fmt.Sprintf("P:%d", passed))
	if rspec.Summary.FailureCount > 0 {
		parts = append(parts, fmt.Sprintf("F:%d", rspec.Summary.FailureCount))
	}
	if rspec.Summary.PendingCount > 0 {
		parts = append(parts, fmt.Sprintf("S:%d", rspec.Summary.PendingCount))
	}

	var result []string
	result = append(result, strings.Join(parts, " "))

	// Show up to 5 failures
	failures := 0
	for _, ex := range rspec.Examples {
		if ex.Status == "failed" && failures < 5 {
			failures++
			shortFile := ex.FilePath
			if idx := strings.LastIndex(ex.FilePath, "/"); idx >= 0 {
				shortFile = ex.FilePath[idx+1:]
			}
			result = append(result, fmt.Sprintf("FAIL: %s:%d", shortFile, ex.LineNumber))
		}
	}
	if rspec.Summary.FailureCount > 5 {
		result = append(result, fmt.Sprintf("... +%d more", rspec.Summary.FailureCount-5))
	}

	return strings.Join(result, "\n")
}

func filterRspecTextOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var failures []string
	var passed, failed, pending int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for summary patterns
		if strings.Contains(line, " examples, ") {
			result = append(result, "📋 RSpec: "+line)
		} else if strings.Contains(line, "FAILED -") || strings.Contains(line, "Failure/Error") {
			failures = append(failures, shared.TruncateLine(line, 100))
			failed++
		} else if strings.Contains(line, "pending") && strings.Contains(line, "(PENDING:") {
			pending++
		} else if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "F") {
			// Progress dots - count them
			for _, c := range line {
				if c == '.' {
					passed++
				} else if c == 'F' {
					failed++
				}
			}
		}
	}

	if len(failures) > 0 {
		result = append(result, "")
		result = append(result, "Failures:")
		for i, f := range failures {
			if i >= 10 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(failures)-10))
				break
			}
			result = append(result, fmt.Sprintf("   - %s", f))
		}
	}

	if len(result) == 0 {
		return raw
	}
	return strings.Join(result, "\n")
}
