package output

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var summaryCmd = &cobra.Command{
	Use:   "summary <command> [args...]",
	Short: "Run command and show heuristic summary",
	Long: `Execute a command and provide a condensed heuristic summary.

Analyzes output type and provides appropriate summarization for tests,
builds, logs, lists, JSON, or generic output.

Examples:
  tokman summary npm test
  tokman summary cargo build
  tokman summary cat largefile.txt`,
	DisableFlagParsing: true,
	RunE:               runSummary,
}

func init() {
	registry.Add(func() { registry.Register(summaryCmd) })
}

func runSummary(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("summary requires a command to execute")
	}

	timer := tracking.Start()
	command := strings.Join(args, " ")

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running and summarizing: %s\n", command)
	}

	execCmd := exec.Command(args[0], args[1:]...)

	output, err := execCmd.CombinedOutput()
	raw := string(output)
	success := err == nil

	summary := summarizeOutput(raw, command, success)
	fmt.Println(summary)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(summary)
	timer.Track(command, "tokman summary", originalTokens, filteredTokens)

	return nil
}

func summarizeOutput(output, command string, success bool) string {
	lines := strings.Split(output, "\n")
	var result []string

	statusIcon := "✅"
	if !success {
		statusIcon = "❌"
	}
	result = append(result, fmt.Sprintf("%s Command: %s", statusIcon, shared.TruncateLine(command, 60)))
	result = append(result, fmt.Sprintf("   %d lines of output", len(lines)))
	result = append(result, "")

	outputType := detectOutputType(output, command)

	switch outputType {
	case shared.OutputTypeTest:
		summarizeTests(output, &result)
	case shared.OutputTypeBuild:
		summarizeBuild(output, &result)
	case shared.OutputTypeLog:
		summarizeLogs(output, &result)
	case shared.OutputTypeList:
		summarizeList(output, &result)
	case shared.OutputTypeJSON:
		summarizeJSONOutput(output, &result)
	default:
		summarizeGeneric(output, &result)
	}

	return strings.Join(result, "\n")
}

func detectOutputType(output, command string) shared.OutputType {
	cmdLower := strings.ToLower(command)
	outLower := strings.ToLower(output)

	if strings.Contains(cmdLower, "test") ||
		(strings.Contains(outLower, "passed") && strings.Contains(outLower, "failed")) {
		return shared.OutputTypeTest
	}
	if strings.Contains(cmdLower, "build") ||
		strings.Contains(cmdLower, "compile") ||
		strings.Contains(outLower, "compiling") {
		return shared.OutputTypeBuild
	}
	if strings.Contains(outLower, "error:") ||
		strings.Contains(outLower, "warn:") ||
		strings.Contains(outLower, "[info]") {
		return shared.OutputTypeLog
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") ||
		strings.HasPrefix(strings.TrimSpace(output), "[") {
		return shared.OutputTypeJSON
	}
	return shared.OutputTypeGeneric
}

func summarizeTests(output string, result *[]string) {
	*result = append(*result, "📋 Test Results:")

	var passed, failed, skipped int
	var failures []string

	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "passed") || strings.Contains(lower, "✓") || strings.Contains(lower, "ok") {
			if n := extractNumber(lower, "passed"); n > 0 {
				passed = n
			} else {
				passed++
			}
		}
		if strings.Contains(lower, "failed") || strings.Contains(lower, "✗") {
			if n := extractNumber(lower, "failed"); n > 0 {
				failed = n
			}
			if !strings.Contains(line, "0 failed") {
				failures = append(failures, line)
			}
		}
		if strings.Contains(lower, "skipped") || strings.Contains(lower, "ignored") {
			if n := extractNumber(lower, "skipped"); n > 0 {
				skipped = n
			}
		}
	}

	*result = append(*result, fmt.Sprintf("   ✅ %d passed", passed))
	if failed > 0 {
		*result = append(*result, fmt.Sprintf("   ❌ %d failed", failed))
	}
	if skipped > 0 {
		*result = append(*result, fmt.Sprintf("   ⏭️  %d skipped", skipped))
	}

	if len(failures) > 0 {
		*result = append(*result, "")
		*result = append(*result, "   Failures:")
		for _, f := range failures {
			if len(f) > 5 {
				*result = append(*result, fmt.Sprintf("   • %s", shared.TruncateLine(f, 70)))
			}
		}
	}
}

func summarizeBuild(output string, result *[]string) {
	*result = append(*result, "🔨 Build Summary:")

	var errors, warnings, compiled int
	var errorMsgs []string

	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") && !strings.Contains(lower, "0 error") {
			errors++
			if len(errorMsgs) < 5 {
				errorMsgs = append(errorMsgs, line)
			}
		}
		if strings.Contains(lower, "warning") && !strings.Contains(lower, "0 warning") {
			warnings++
		}
		if strings.Contains(lower, "compiling") || strings.Contains(lower, "compiled") {
			compiled++
		}
	}

	if compiled > 0 {
		*result = append(*result, fmt.Sprintf("   📦 %d crates/files compiled", compiled))
	}
	if errors > 0 {
		*result = append(*result, fmt.Sprintf("   ❌ %d errors", errors))
	}
	if warnings > 0 {
		*result = append(*result, fmt.Sprintf("   ⚠️  %d warnings", warnings))
	}
	if errors == 0 && warnings == 0 {
		*result = append(*result, "   ✅ Build successful")
	}

	if len(errorMsgs) > 0 {
		*result = append(*result, "")
		*result = append(*result, "   Errors:")
		for _, e := range errorMsgs {
			*result = append(*result, fmt.Sprintf("   • %s", shared.TruncateLine(e, 70)))
		}
	}
}

func summarizeLogs(output string, result *[]string) {
	*result = append(*result, "📝 Log Summary:")

	var errors, warnings, info int

	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "fatal") {
			errors++
		} else if strings.Contains(lower, "warn") {
			warnings++
		} else if strings.Contains(lower, "info") {
			info++
		}
	}

	*result = append(*result, fmt.Sprintf("   ❌ %d errors", errors))
	*result = append(*result, fmt.Sprintf("   ⚠️  %d warnings", warnings))
	*result = append(*result, fmt.Sprintf("   ℹ️  %d info", info))
}

func summarizeList(output string, result *[]string) {
	lines := strings.Split(output, "\n")
	var nonEmpty []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}

	*result = append(*result, fmt.Sprintf("📋 List (%d items):", len(nonEmpty)))

	for i, line := range nonEmpty {
		if i >= 10 {
			break
		}
		*result = append(*result, fmt.Sprintf("   • %s", shared.TruncateLine(line, 70)))
	}
	if len(nonEmpty) > 10 {
		*result = append(*result, fmt.Sprintf("   ... +%d more", len(nonEmpty)-10))
	}
}

func summarizeJSONOutput(output string, result *[]string) {
	*result = append(*result, "📋 JSON Output:")

	schema := shared.TryJSONSchema(strings.TrimSpace(output), 5)
	if schema != "" {
		*result = append(*result, "   Structure:")
		for _, line := range strings.Split(schema, "\n") {
			*result = append(*result, "   "+line)
		}
	} else {
		*result = append(*result, "   (Invalid JSON)")
	}
}

func summarizeGeneric(output string, result *[]string) {
	lines := strings.Split(output, "\n")

	*result = append(*result, "📋 Output:")

	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			*result = append(*result, fmt.Sprintf("   %s", shared.TruncateLine(line, 75)))
			count++
			if count >= 5 {
				break
			}
		}
	}

	if len(lines) > 10 {
		*result = append(*result, "   ...")
		lastLines := lines
		if len(lastLines) > 3 {
			lastLines = lastLines[len(lastLines)-3:]
		}
		for _, line := range lastLines {
			if strings.TrimSpace(line) != "" {
				*result = append(*result, fmt.Sprintf("   %s", shared.TruncateLine(line, 75)))
			}
		}
	}
}

func extractNumber(text, after string) int {
	re := regexp.MustCompile(`(\d+)\s*` + after)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}
	return 0
}
