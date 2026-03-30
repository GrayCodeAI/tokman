package lang

import (
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

var gradleCmd = &cobra.Command{
	Use:   "gradle [command] [args...]",
	Short: "Gradle build commands with compact output",
	Long: `Execute Gradle build commands with token-optimized output.

Specialized filters for:
  - build: Compact build summary
  - test: Compact test results
  - dependencies: Compact dependency tree

Examples:
  tokman gradle build
  tokman gradle test
  tokman gradle dependencies`,
	DisableFlagParsing: true,
	RunE:               runGradle,
}

func init() {
	registry.Add(func() { registry.Register(gradleCmd) })
}

func runGradle(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"tasks"}
	}

	switch args[0] {
	case "build":
		return runGradleBuild(args[1:])
	case "test":
		return runGradleTest(args[1:])
	case "dependencies", "deps":
		return runGradleDependencies(args[1:])
	case "tasks":
		return runGradleTasks(args[1:])
	default:
		return runGradlePassthrough(args)
	}
}

func runGradleBuild(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: gradle build %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gradle", append([]string{"build"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGradleBuildOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "gradle_build", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("gradle build", "tokman gradle build", originalTokens, filteredTokens)

	return err
}

func filterGradleBuildOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var inTask bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Keep task header lines
		if strings.HasPrefix(trimmed, "> Task ") {
			inTask = true
			if shared.UltraCompact {
				// Extract task name only
				taskName := strings.TrimPrefix(trimmed, "> Task ")
				taskName = strings.Split(taskName, " ")[0]
				result = append(result, taskName)
			} else {
				result = append(result, shared.TruncateLine(line, 100))
			}
			continue
		}

		// Keep BUILD status
		if strings.HasPrefix(trimmed, "BUILD") {
			result = append(result, line)
			continue
		}

		// Keep failures
		if strings.Contains(line, "FAILED") || strings.Contains(line, "ERROR") {
			result = append(result, line)
			continue
		}

		// Skip verbose output in ultra-compact mode
		if shared.UltraCompact {
			continue
		}

		// Keep action descriptions in normal mode
		if inTask && strings.HasPrefix(line, "> ") {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func runGradleTest(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: gradle test %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gradle", append([]string{"test"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGradleTestOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "gradle_test", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("gradle test", "tokman gradle test", originalTokens, filteredTokens)

	return err
}

func filterGradleTestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var failCount int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Keep test summary
		if strings.Contains(line, "test") && strings.Contains(line, "passed") {
			result = append(result, line)
			continue
		}

		// Keep failure details
		if strings.Contains(line, "FAILED") || strings.Contains(line, "test >") {
			result = append(result, line)
			failCount++
			continue
		}

		// Keep BUILD status
		if strings.HasPrefix(trimmed, "BUILD") {
			result = append(result, line)
			continue
		}

		// In ultra-compact mode, skip most output
		if shared.UltraCompact {
			if strings.Contains(line, "tests completed") {
				result = append(result, line)
			}
			continue
		}

		// Keep task lines
		if strings.HasPrefix(trimmed, "> Task ") {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	return strings.Join(result, "\n")
}

func runGradleDependencies(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: gradle dependencies %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gradle", append([]string{"dependencies"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGradleDependenciesOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("gradle dependencies", "tokman gradle dependencies", originalTokens, filteredTokens)

	return err
}

func filterGradleDependenciesOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var depth int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Calculate depth based on leading characters
		if strings.HasPrefix(line, "+---") || strings.HasPrefix(line, "\\---") {
			depth = 0
		} else if strings.HasPrefix(line, "|    ") || strings.HasPrefix(line, "     ") {
			depth = strings.Count(line[:strings.IndexFunc(line, func(r rune) bool { return r != ' ' && r != '|' })], "   ")
		}

		// In ultra-compact mode, show only top-level dependencies
		if shared.UltraCompact {
			if depth == 0 && (strings.HasPrefix(line, "+---") || strings.HasPrefix(line, "\\---")) {
				// Extract dependency name
				dep := strings.TrimLeft(line, "+\\--- ")
				dep = strings.Split(dep, ":")[0] // Just the group/name
				result = append(result, dep)
			}
			continue
		}

		// In normal mode, truncate long lines but keep structure
		result = append(result, shared.TruncateLine(line, 120))
	}

	return strings.Join(result, "\n")
}

func runGradleTasks(args []string) error {
	timer := tracking.Start()

	execCmd := exec.Command("gradle", append([]string{"tasks"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGradleTasksOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("gradle tasks", "tokman gradle tasks", originalTokens, filteredTokens)

	return err
}

func filterGradleTasksOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var inGroup bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			inGroup = false
			continue
		}

		// Keep group headers
		if strings.HasSuffix(trimmed, "tasks") || strings.HasSuffix(trimmed, "---------") {
			if !strings.HasPrefix(trimmed, "---") {
				result = append(result, "")
				result = append(result, line)
				inGroup = true
			}
			continue
		}

		// Keep task names in groups
		if inGroup && !shared.UltraCompact {
			result = append(result, shared.TruncateLine(line, 80))
		} else if inGroup && shared.UltraCompact {
			// Extract just the task name
			taskName := strings.Fields(trimmed)[0]
			result = append(result, taskName)
		}
	}

	return strings.Join(result, "\n")
}

func runGradlePassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: gradle %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gradle", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	// Basic filtering even for passthrough
	filtered := filterGradleBasicOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("gradle", "tokman gradle", originalTokens, filteredTokens)

	return err
}

func filterGradleBasicOutput(raw string) string {
	if shared.UltraCompact {
		lines := strings.Split(raw, "\n")
		var result []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "> Task ") || strings.HasPrefix(trimmed, "BUILD") ||
				strings.Contains(line, "FAILED") || strings.Contains(line, "ERROR") {
				result = append(result, shared.TruncateLine(line, 80))
			}
		}
		return strings.Join(result, "\n")
	}
	return raw
}
