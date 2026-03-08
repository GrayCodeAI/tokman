package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var pytestCmd = &cobra.Command{
	Use:   "pytest [args...]",
	Short: "Pytest runner with filtered output",
	Long: `Pytest runner with token-optimized output.

Summarizes test results and highlights failures.

Examples:
  tokman pytest tests/
  tokman pytest -v
  tokman pytest --tb=short`,
	RunE: runPytest,
}

func init() {
	rootCmd.AddCommand(pytestCmd)
}

type ParseState int

const (
	StateHeader ParseState = iota
	StateTestProgress
	StateFailures
	StateSummary
)

func runPytest(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Try pytest directly, fallback to python -m pytest
	pytestPath, err := exec.LookPath("pytest")
	if err != nil {
		pytestPath = "" // Will use python -m pytest
	}

	var c *exec.Cmd
	if pytestPath != "" {
		c = exec.Command(pytestPath)
	} else {
		c = exec.Command("python3", "-m", "pytest")
	}

	// Force short traceback and quiet mode for compact output
	hasTbFlag := false
	hasQuietFlag := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "--tb") {
			hasTbFlag = true
		}
		if arg == "-q" || arg == "--quiet" {
			hasQuietFlag = true
		}
	}

	if !hasTbFlag {
		c.Args = append(c.Args, "--tb=short")
	}
	if !hasQuietFlag {
		c.Args = append(c.Args, "-q")
	}
	c.Args = append(c.Args, args...)

	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err = c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterPytestOutput(stdout.String())

	fmt.Print(filtered)

	// Include stderr if present (import errors, etc.)
	if strings.TrimSpace(stderr.String()) != "" {
		fmt.Fprint(os.Stderr, strings.TrimSpace(stderr.String()))
	}

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("pytest %s", strings.Join(args, " ")), "tokman pytest", originalTokens, filteredTokens)

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	return nil
}

func filterPytestOutput(output string) string {
	state := StateHeader
	var testFiles []string
	var failures []string
	var currentFailure []string
	var summaryLine string

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		// State transitions
		if strings.HasPrefix(trimmed, "===") && strings.Contains(trimmed, "test session starts") {
			state = StateHeader
			continue
		} else if strings.HasPrefix(trimmed, "===") && strings.Contains(trimmed, "FAILURES") {
			state = StateFailures
			continue
		} else if strings.HasPrefix(trimmed, "===") && strings.Contains(trimmed, "short test summary") {
			state = StateSummary
			// Save current failure if any
			if len(currentFailure) > 0 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
				currentFailure = nil
			}
			continue
		} else if strings.HasPrefix(trimmed, "===") && (strings.Contains(trimmed, "passed") || strings.Contains(trimmed, "failed")) {
			summaryLine = trimmed
			continue
		}

		// Process based on state
		switch state {
		case StateHeader:
			if strings.HasPrefix(trimmed, "collected") {
				state = StateTestProgress
			}
		case StateTestProgress:
			if trimmed != "" && !strings.HasPrefix(trimmed, "===") && (strings.Contains(trimmed, ".py") || strings.Contains(trimmed, "%]")) {
				testFiles = append(testFiles, trimmed)
			}
		case StateFailures:
			if strings.HasPrefix(trimmed, "___") {
				// New failure section
				if len(currentFailure) > 0 {
					failures = append(failures, strings.Join(currentFailure, "\n"))
					currentFailure = nil
				}
				currentFailure = append(currentFailure, trimmed)
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "===") {
				currentFailure = append(currentFailure, trimmed)
			}
		case StateSummary:
			if strings.HasPrefix(trimmed, "FAILED") || strings.HasPrefix(trimmed, "ERROR") {
				failures = append(failures, trimmed)
			}
		}
	}

	// Save last failure if any
	if len(currentFailure) > 0 {
		failures = append(failures, strings.Join(currentFailure, "\n"))
	}

	return buildPytestSummary(summaryLine, testFiles, failures)
}

func buildPytestSummary(summary string, testFiles []string, failures []string) string {
	// Parse summary line
	passed, failed, skipped := parseSummaryLine(summary)

	if failed == 0 && passed > 0 {
		return fmt.Sprintf("✓ Pytest: %d passed\n", passed)
	}

	if passed == 0 && failed == 0 {
		return "Pytest: No tests collected\n"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Pytest: %d passed, %d failed", passed, failed))
	if skipped > 0 {
		result.WriteString(fmt.Sprintf(", %d skipped", skipped))
	}
	result.WriteString("\n")
	result.WriteString("═══════════════════════════════════════\n")

	if len(failures) == 0 {
		return result.String()
	}

	// Show failures (limit to key information)
	result.WriteString("\nFailures:\n")

	for i := 0; i < 5 && i < len(failures); i++ {
		failure := failures[i]
		lines := strings.Split(failure, "\n")

		// First line is usually test name (after ___)
		if len(lines) > 0 {
			firstLine := lines[0]
			if strings.HasPrefix(firstLine, "___") {
				// Extract test name between ___
				testName := strings.Trim(firstLine, "_ ")
				result.WriteString(fmt.Sprintf("%d. ❌ %s\n", i+1, testName))
			} else if strings.HasPrefix(firstLine, "FAILED") {
				// Summary format: "FAILED tests/test_foo.py::test_bar - AssertionError"
				parts := strings.SplitN(firstLine, " - ", 2)
				if len(parts) > 0 {
					testName := strings.TrimPrefix(parts[0], "FAILED ")
					result.WriteString(fmt.Sprintf("%d. ❌ %s\n", i+1, testName))
				}
				if len(parts) > 1 {
					result.WriteString(fmt.Sprintf("     %s\n", truncate(parts[1], 100)))
				}
				continue
			}
		}

		// Show relevant error lines (assertions, errors, file locations)
		relevantLines := 0
		for _, line := range lines[1:] {
			lineLower := strings.ToLower(line)
			isRelevant := strings.HasPrefix(strings.TrimSpace(line), ">") ||
				strings.HasPrefix(strings.TrimSpace(line), "E") ||
				strings.Contains(lineLower, "assert") ||
				strings.Contains(lineLower, "error") ||
				strings.Contains(line, ".py:")

			if isRelevant && relevantLines < 3 {
				result.WriteString(fmt.Sprintf("     %s\n", truncate(line, 100)))
				relevantLines++
			}
		}

		if i < len(failures)-1 {
			result.WriteString("\n")
		}
	}

	if len(failures) > 5 {
		result.WriteString(fmt.Sprintf("\n... +%d more failures\n", len(failures)-5))
	}

	return result.String()
}

func parseSummaryLine(summary string) (passed, failed, skipped int) {
	// Parse lines like "=== 4 passed, 1 failed in 0.50s ==="
	parts := strings.Split(summary, ",")
	for _, part := range parts {
		words := strings.Fields(part)
		for i, word := range words {
			if i > 0 {
				if strings.Contains(word, "passed") {
					fmt.Sscanf(words[i-1], "%d", &passed)
				} else if strings.Contains(word, "failed") {
					fmt.Sscanf(words[i-1], "%d", &failed)
				} else if strings.Contains(word, "skipped") {
					fmt.Sscanf(words[i-1], "%d", &skipped)
				}
			}
		}
	}
	return
}
