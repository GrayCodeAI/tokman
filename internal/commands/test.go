package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [args...]",
	Short: "Run Go tests with aggregated output",
	Long: `Run go test with output aggregation:
- Aggregates multiple test suites into single summary
- Shows full output only on failures
- Tracks token savings from condensed output`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		originalOutput, filteredOutput, err := runGoTest(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			// Test failure - show full output
			fmt.Print(originalOutput)
			os.Exit(1)
		}

		fmt.Print(filteredOutput)

		// Record to tracker
		if err := recordCommand("go test", originalOutput, filteredOutput, execTime, true); err != nil && verbose > 0 {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

var buildCmd = &cobra.Command{
	Use:   "build [args...]",
	Short: "Run Go build with filtered output",
	Long: `Run go build with output filtering:
- Strips verbose build output
- Shows errors and warnings only`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		originalOutput, filteredOutput, err := runGoBuild(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Print(filteredOutput)
			os.Exit(1)
		}

		fmt.Print(filteredOutput)

		// Record to tracker
		if err := recordCommand("go build", originalOutput, filteredOutput, execTime, err == nil); err != nil && verbose > 0 {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(buildCmd)
}

// TestSuite represents aggregated test results
type TestSuite struct {
	Package     string
	Passed      int
	Failed      int
	Skipped     int
	Duration    time.Duration
	FailedTests []string
}

// runGoTest executes go test and aggregates output
func runGoTest(args []string) (string, string, error) {
	// Build command with verbose flag for parsing
	testArgs := []string{"test", "-v"}
	testArgs = append(testArgs, args...)

	cmd := exec.Command("go", testArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()

	// Parse and aggregate test results
	suites := parseTestOutput(output)
	filtered := formatTestSummary(suites, err != nil)

	return output, filtered, err
}

// parseTestOutput parses go test -v output
func parseTestOutput(output string) []TestSuite {
	lines := strings.Split(output, "\n")

	var suites []TestSuite
	var currentSuite *TestSuite
	var currentPkg string

	// Regex patterns for test output
	passRe := regexp.MustCompile(`^--- PASS:\s+(\S+)\s+\(([^)]+)\)`)
	failRe := regexp.MustCompile(`^--- FAIL:\s+(\S+)\s+\(([^)]+)\)`)
	skipRe := regexp.MustCompile(`^--- SKIP:\s+(\S+)\s+\(([^)]+)\)`)
	pkgRe := regexp.MustCompile(`^PASS\s+(.+)$|^FAIL\s+(.+)$`)
	summaryRe := regexp.MustCompile(`^(PASS|FAIL)\s+([^ ]+)\s+([\d.]+s)$`)

	for _, line := range lines {
		// Detect package start
		if matches := pkgRe.FindStringSubmatch(line); matches != nil {
			pkg := matches[1]
			if matches[2] != "" {
				pkg = matches[2]
			}
			if pkg != currentPkg && pkg != "" {
				if currentSuite != nil {
					suites = append(suites, *currentSuite)
				}
				currentSuite = &TestSuite{Package: pkg}
				currentPkg = pkg
			}
		}

		// Parse test results
		if matches := passRe.FindStringSubmatch(line); matches != nil {
			if currentSuite == nil {
				currentSuite = &TestSuite{Package: "unknown"}
			}
			currentSuite.Passed++
			if dur, err := time.ParseDuration(matches[2]); err == nil {
				currentSuite.Duration += dur
			}
		}

		if matches := failRe.FindStringSubmatch(line); matches != nil {
			if currentSuite == nil {
				currentSuite = &TestSuite{Package: "unknown"}
			}
			currentSuite.Failed++
			currentSuite.FailedTests = append(currentSuite.FailedTests, matches[1])
			if dur, err := time.ParseDuration(matches[2]); err == nil {
				currentSuite.Duration += dur
			}
		}

		if matches := skipRe.FindStringSubmatch(line); matches != nil {
			if currentSuite == nil {
				currentSuite = &TestSuite{Package: "unknown"}
			}
			currentSuite.Skipped++
		}

		// Check for summary line: PASS pkg X.XXXs
		if matches := summaryRe.FindStringSubmatch(line); matches != nil {
			if currentSuite != nil {
				if dur, err := time.ParseDuration(matches[3]); err == nil {
					currentSuite.Duration = dur
				}
			}
		}
	}

	// Add last suite
	if currentSuite != nil {
		suites = append(suites, *currentSuite)
	}

	return suites
}

// formatTestSummary formats aggregated test results
func formatTestSummary(suites []TestSuite, hasFailure bool) string {
	if len(suites) == 0 {
		return "No tests run\n"
	}

	var result strings.Builder

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	// If failures, show full details
	if hasFailure {
		result.WriteString(bold("\n❌ Test Failures:\n"))
		result.WriteString(strings.Repeat("─", 40) + "\n")

		for _, suite := range suites {
			if suite.Failed > 0 {
				result.WriteString(fmt.Sprintf("\n%s %s\n", red("Package:"), cyan(suite.Package)))
				for _, test := range suite.FailedTests {
					result.WriteString(fmt.Sprintf("  ✗ %s\n", test))
				}
			}
		}
		result.WriteString("\n")
	}

	// Summary header
	result.WriteString(bold("📊 Test Summary\n"))
	result.WriteString(strings.Repeat("─", 40) + "\n")

	// Aggregate totals
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	totalDuration := time.Duration(0)

	for _, suite := range suites {
		totalPassed += suite.Passed
		totalFailed += suite.Failed
		totalSkipped += suite.Skipped
		totalDuration += suite.Duration
	}

	// Overall status
	if totalFailed == 0 {
		result.WriteString(green(fmt.Sprintf("\n✓ %d tests passed", totalPassed)))
	} else {
		result.WriteString(red(fmt.Sprintf("\n✗ %d passed, %d failed", totalPassed, totalFailed)))
	}

	if totalSkipped > 0 {
		result.WriteString(yellow(fmt.Sprintf(", %d skipped", totalSkipped)))
	}

	result.WriteString(fmt.Sprintf(" (%d packages, %s)\n", len(suites), totalDuration.Round(time.Millisecond)))

	// Per-package breakdown (compact)
	if len(suites) > 1 {
		result.WriteString("\n")
		for _, suite := range suites {
			status := green("✓")
			if suite.Failed > 0 {
				status = red("✗")
			}
			result.WriteString(fmt.Sprintf("  %s %s: %d passed", status, cyan(suite.Package), suite.Passed))
			if suite.Failed > 0 {
				result.WriteString(red(fmt.Sprintf(", %d failed", suite.Failed)))
			}
			if suite.Skipped > 0 {
				result.WriteString(yellow(fmt.Sprintf(", %d skipped", suite.Skipped)))
			}
			result.WriteString("\n")
		}
	}

	return result.String()
}

// runGoBuild executes go build and filters output
func runGoBuild(args []string) (string, string, error) {
	buildArgs := []string{"build"}
	buildArgs = append(buildArgs, args...)

	cmd := exec.Command("go", buildArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()
	filtered := filterBuildOutput(output)

	return output, filtered, err
}

// filterBuildOutput filters verbose build output
func filterBuildOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Keep errors and warnings
		if strings.Contains(line, "error") ||
			strings.Contains(line, "warning") ||
			strings.Contains(line, "Error") ||
			strings.Contains(line, "Warning") {
			result = append(result, line)
		}

		// Keep package build failures
		if strings.Contains(line, "cannot find package") ||
			strings.Contains(line, "undefined:") ||
			strings.Contains(line, "not used") {
			result = append(result, line)
		}
	}

	if len(result) == 0 && len(output) > 0 {
		// Build succeeded with no errors
		green := color.New(color.FgGreen).SprintFunc()
		return green("✓ Build successful\n")
	}

	return strings.Join(result, "\n")
}
