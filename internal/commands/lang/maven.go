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

var mavenCmd = &cobra.Command{
	Use:   "mvn [command] [args...]",
	Short: "Maven build commands with compact output",
	Long: `Execute Maven build commands with token-optimized output.

Specialized filters for:
  - compile: Compact compilation output
  - test: Compact test results
  - dependency: Compact dependency tree

Examples:
  tokman mvn compile
  tokman mvn test
  tokman mvn dependency:tree`,
	DisableFlagParsing: true,
	RunE:               runMaven,
}

func init() {
	registry.Add(func() { registry.Register(mavenCmd) })
}

func runMaven(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	// Check for specific Maven goals
	switch {
	case args[0] == "compile" || args[0] == "install" || args[0] == "package":
		return runMavenBuild(args)
	case args[0] == "test":
		return runMavenTest(args[1:])
	case strings.HasPrefix(args[0], "dependency:"):
		return runMavenDependency(args)
	default:
		return runMavenPassthrough(args)
	}
}

func runMavenBuild(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mvn %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("mvn", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMavenBuildOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "maven_build", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mvn", "tokman mvn", originalTokens, filteredTokens)

	return err
}

func filterMavenBuildOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var inModule bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Keep module separators
		if strings.HasPrefix(trimmed, "---") && strings.Contains(line, " ---") {
			if shared.UltraCompact {
				// Extract module name
				parts := strings.Split(trimmed, " ")
				if len(parts) >= 3 {
					result = append(result, fmt.Sprintf("📦 %s", parts[2]))
				}
			} else {
				result = append(result, shared.TruncateLine(line, 100))
			}
			inModule = true
			continue
		}

		// Keep BUILD status
		if strings.HasPrefix(trimmed, "[INFO] BUILD") {
			result = append(result, line)
			continue
		}

		// Keep errors and failures
		if strings.Contains(trimmed, "[ERROR]") || strings.Contains(trimmed, "FAILURE") {
			result = append(result, line)
			continue
		}

		// Keep warnings unless ultra-compact
		if !shared.UltraCompact && strings.Contains(trimmed, "[WARNING]") {
			result = append(result, shared.TruncateLine(line, 100))
			continue
		}

		// Skip verbose info messages in ultra-compact mode
		if shared.UltraCompact {
			continue
		}

		// Keep important info messages
		if inModule && strings.HasPrefix(trimmed, "[INFO]") {
			// Filter out less important info
			if !strings.Contains(trimmed, "Downloading") && !strings.Contains(trimmed, "Downloaded") {
				result = append(result, shared.TruncateLine(line, 100))
			}
		}
	}

	return strings.Join(result, "\n")
}

func runMavenTest(args []string) error {
	timer := tracking.Start()

	fullArgs := append([]string{"test"}, args...)
	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mvn %s\n", strings.Join(fullArgs, " "))
	}

	execCmd := exec.Command("mvn", fullArgs...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMavenTestOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "maven_test", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mvn test", "tokman mvn test", originalTokens, filteredTokens)

	return err
}

func filterMavenTestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Keep test summary lines
		if strings.Contains(trimmed, "Tests run:") {
			if shared.UltraCompact {
				// Extract just numbers
				result = append(result, line)
			} else {
				result = append(result, shared.TruncateLine(line, 120))
			}
			continue
		}

		// Keep module separators
		if strings.HasPrefix(trimmed, "---") && strings.Contains(line, " ---") {
			result = append(result, shared.TruncateLine(line, 100))
			continue
		}

		// Keep BUILD status
		if strings.HasPrefix(trimmed, "[INFO] BUILD") {
			result = append(result, line)
			continue
		}

		// Keep errors and test failures
		if strings.Contains(trimmed, "[ERROR]") || strings.Contains(trimmed, "<<< FAIL") ||
			strings.Contains(trimmed, "Failed tests:") {
			result = append(result, line)
			continue
		}

		// Skip verbose output in ultra-compact mode
		if shared.UltraCompact {
			continue
		}

		// Keep running test indicators
		if strings.Contains(trimmed, "Running ") {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	return strings.Join(result, "\n")
}

func runMavenDependency(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mvn %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("mvn", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMavenDependencyOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mvn dependency", "tokman mvn dependency", originalTokens, filteredTokens)

	return err
}

func filterMavenDependencyOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var depth int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Skip INFO lines unless they're the dependency tree
		if strings.HasPrefix(trimmed, "[INFO]") {
			// Check if it's a dependency line (contains group:artifact:version pattern)
			if !strings.Contains(trimmed, ":") || strings.Contains(trimmed, "Downloading") {
				continue
			}

			// Calculate depth based on pipe characters or indentation
			depth = strings.Count(line, "|")

			// In ultra-compact mode, show only top-level dependencies
			if shared.UltraCompact {
				if depth == 0 {
					// Extract artifact name
					parts := strings.Split(trimmed, ":")
					if len(parts) >= 2 {
						artifact := strings.TrimPrefix(parts[0], "[INFO] ")
						result = append(result, artifact)
					}
				}
				continue
			}

			// Truncate long dependency lines
			result = append(result, shared.TruncateLine(line, 120))
		}
	}

	return strings.Join(result, "\n")
}

func runMavenPassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mvn %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("mvn", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMavenBasicOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mvn", "tokman mvn", originalTokens, filteredTokens)

	return err
}

func filterMavenBasicOutput(raw string) string {
	if shared.UltraCompact {
		lines := strings.Split(raw, "\n")
		var result []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Keep only critical lines
			if strings.Contains(trimmed, "[ERROR]") ||
				strings.HasPrefix(trimmed, "[INFO] BUILD") ||
				strings.Contains(trimmed, "FAILURE") {
				result = append(result, shared.TruncateLine(line, 100))
			}
		}
		return strings.Join(result, "\n")
	}
	return raw
}
