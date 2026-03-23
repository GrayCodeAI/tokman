package test

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var testCmd = &cobra.Command{
	Use:   "test [args...]",
	Short: "Run Go tests with aggregated output",
	Long: `Run go test with output aggregation:
- Aggregates multiple test suites into single summary
- Shows full output only on failures
- Tracks token savings from condensed output`,
	Run: func(cmd *cobra.Command, args []string) {
		shared.ExecuteAndRecord("go test", func() (string, string, error) {
			return runGoTest(args)
		})
	},
}

var buildCmd = &cobra.Command{
	Use:   "build [args...]",
	Short: "Run Go build with filtered output",
	Long: `Run go build with output filtering:
- Strips verbose build output
- Shows errors and warnings only`,
	Run: func(cmd *cobra.Command, args []string) {
		shared.ExecuteAndRecord("go build", func() (string, string, error) {
			return runGoBuild(args)
		})
	},
}

func init() {
	registry.Add(func() { registry.Register(testCmd) })
	registry.Add(func() { registry.Register(buildCmd) })
}

type TestSuite struct {
	Package     string
	Passed      int
	Failed      int
	Skipped     int
	Duration    time.Duration
	FailedTests []string
}

func runGoTest(args []string) (string, string, error) {
	testArgs := []string{"test", "-v"}
	testArgs = append(testArgs, args...)

	cmd := exec.Command("go", testArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()

	suites := parseTestOutput(output)
	filtered := formatTestSummary(suites, err != nil)

	return output, filtered, err
}

func parseTestOutput(output string) []TestSuite {
	lines := strings.Split(output, "\n")

	var suites []TestSuite
	var currentSuite *TestSuite
	var currentPkg string

	passRe := regexp.MustCompile(`^--- PASS:\s+(\S+)\s+\(([^)]+)\)`)
	failRe := regexp.MustCompile(`^--- FAIL:\s+(\S+)\s+\(([^)]+)\)`)
	skipRe := regexp.MustCompile(`^--- SKIP:\s+(\S+)\s+\(([^)]+)\)`)
	pkgRe := regexp.MustCompile(`^PASS\s+(.+)$|^FAIL\s+(.+)$`)
	summaryRe := regexp.MustCompile(`^(PASS|FAIL)\s+([^ ]+)\s+([\d.]+s)$`)

	for _, line := range lines {
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

		if matches := summaryRe.FindStringSubmatch(line); matches != nil {
			if currentSuite != nil {
				if dur, err := time.ParseDuration(matches[3]); err == nil {
					currentSuite.Duration = dur
				}
			}
		}
	}

	if currentSuite != nil {
		suites = append(suites, *currentSuite)
	}

	return suites
}

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

	result.WriteString(bold("📊 Test Summary\n"))
	result.WriteString(strings.Repeat("─", 40) + "\n")

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

	if totalFailed == 0 {
		result.WriteString(green(fmt.Sprintf("\n✓ %d tests passed", totalPassed)))
	} else {
		result.WriteString(red(fmt.Sprintf("\n✗ %d passed, %d failed", totalPassed, totalFailed)))
	}

	if totalSkipped > 0 {
		result.WriteString(yellow(fmt.Sprintf(", %d skipped", totalSkipped)))
	}

	result.WriteString(fmt.Sprintf(" (%d packages, %s)\n", len(suites), totalDuration.Round(time.Millisecond)))

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

func filterBuildOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.Contains(line, "error") ||
			strings.Contains(line, "warning") ||
			strings.Contains(line, "Error") ||
			strings.Contains(line, "Warning") {
			result = append(result, line)
		}

		if strings.Contains(line, "cannot find package") ||
			strings.Contains(line, "undefined:") ||
			strings.Contains(line, "not used") {
			result = append(result, line)
		}
	}

	if len(result) == 0 && len(output) > 0 {
		green := color.New(color.FgGreen).SprintFunc()
		return green("✓ Build successful\n")
	}

	return strings.Join(result, "\n")
}
