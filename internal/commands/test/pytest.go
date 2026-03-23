package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
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
	registry.Add(func() { registry.Register(pytestCmd) })
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

	pytestPath, err := exec.LookPath("pytest")
	if err != nil {
		pytestPath = ""
	}

	var c *exec.Cmd
	if pytestPath != "" {
		c = exec.Command(pytestPath)
	} else {
		c = exec.Command("python3", "-m", "pytest")
	}

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

	if strings.TrimSpace(stderr.String()) != "" {
		fmt.Fprint(os.Stderr, strings.TrimSpace(stderr.String()))
	}

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("pytest %s", strings.Join(args, " ")), "tokman pytest", originalTokens, filteredTokens)

	if shared.Verbose > 0 {
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

		if strings.HasPrefix(trimmed, "===") && strings.Contains(trimmed, "test session starts") {
			state = StateHeader
			continue
		} else if strings.HasPrefix(trimmed, "===") && strings.Contains(trimmed, "FAILURES") {
			state = StateFailures
			continue
		} else if strings.HasPrefix(trimmed, "===") && strings.Contains(trimmed, "short test summary") {
			state = StateSummary
			if len(currentFailure) > 0 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
				currentFailure = nil
			}
			continue
		} else if strings.HasPrefix(trimmed, "===") && (strings.Contains(trimmed, "passed") || strings.Contains(trimmed, "failed")) {
			summaryLine = trimmed
			continue
		}

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

	if len(currentFailure) > 0 {
		failures = append(failures, strings.Join(currentFailure, "\n"))
	}

	return buildPytestSummary(summaryLine, testFiles, failures)
}

func buildPytestSummary(summary string, testFiles []string, failures []string) string {
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

	result.WriteString("\nFailures:\n")

	for i := 0; i < 5 && i < len(failures); i++ {
		failure := failures[i]
		lines := strings.Split(failure, "\n")

		if len(lines) > 0 {
			firstLine := lines[0]
			if strings.HasPrefix(firstLine, "___") {
				testName := strings.Trim(firstLine, "_ ")
				result.WriteString(fmt.Sprintf("%d. ❌ %s\n", i+1, testName))
			} else if strings.HasPrefix(firstLine, "FAILED") {
				parts := strings.SplitN(firstLine, " - ", 2)
				if len(parts) > 0 {
					testName := strings.TrimPrefix(parts[0], "FAILED ")
					result.WriteString(fmt.Sprintf("%d. ❌ %s\n", i+1, testName))
				}
				if len(parts) > 1 {
					result.WriteString(fmt.Sprintf("     %s\n", shared.Truncate(parts[1], 100)))
				}
				continue
			}
		}

		relevantLines := 0
		for _, line := range lines[1:] {
			lineLower := strings.ToLower(line)
			isRelevant := strings.HasPrefix(strings.TrimSpace(line), ">") ||
				strings.HasPrefix(strings.TrimSpace(line), "E") ||
				strings.Contains(lineLower, "assert") ||
				strings.Contains(lineLower, "error") ||
				strings.Contains(line, ".py:")

			if isRelevant && relevantLines < 3 {
				result.WriteString(fmt.Sprintf("     %s\n", shared.Truncate(line, 100)))
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
	parts := strings.Split(summary, ",")
	for _, part := range parts {
		words := strings.Fields(part)
		for i, word := range words {
			if i > 0 {
			if strings.Contains(word, "passed") {
				if _, err := fmt.Sscanf(words[i-1], "%d", &passed); err != nil {
					passed = 0
				}
			} else if strings.Contains(word, "failed") {
				if _, err := fmt.Sscanf(words[i-1], "%d", &failed); err != nil {
					failed = 0
				}
			} else if strings.Contains(word, "skipped") {
				if _, err := fmt.Sscanf(words[i-1], "%d", &skipped); err != nil {
					skipped = 0
				}
			}
			}
		}
	}
	return
}
