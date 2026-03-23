package build

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

var golangciCmd = &cobra.Command{
	Use:   "golangci-lint [args...]",
	Short: "golangci-lint with compact output",
	Long: `Execute golangci-lint with token-optimized output.

Groups issues by linter and provides compact summary.

Examples:
  tokman golangci-lint run ./...
  tokman golangci-lint run --timeout 5m`,
	DisableFlagParsing: true,
	RunE:               runGolangci,
}

func init() {
	registry.Add(func() { registry.Register(golangciCmd) })
}

func runGolangci(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"run"}
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: golangci-lint %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("golangci-lint", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGolangciOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("golangci-lint %s", strings.Join(args, " ")), "tokman golangci-lint", originalTokens, filteredTokens)

	return err
}

func filterGolangciOutput(raw string) string {
	if raw == "" {
		return "✅ No linting issues found"
	}

	lines := strings.Split(raw, "\n")

	byLinter := make(map[string][]string)
	var errors []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if idx := strings.LastIndex(line, "("); idx > 0 {
			if idx2 := strings.LastIndex(line, ")"); idx2 > idx {
				linter := line[idx+1 : idx2]
				msg := strings.TrimSpace(line[:idx])
				byLinter[linter] = append(byLinter[linter], shared.TruncateLine(msg, 80))
				continue
			}
		}

		errors = append(errors, shared.TruncateLine(line, 100))
	}

	var result []string

	if len(byLinter) > 0 {
		for linter, issues := range byLinter {
			result = append(result, fmt.Sprintf("🔍 %s (%d):", linter, len(issues)))
			for i, issue := range issues {
				if i >= 5 {
					result = append(result, fmt.Sprintf("   ... +%d more", len(issues)-5))
					break
				}
				result = append(result, fmt.Sprintf("   %s", issue))
			}
		}
	}

	if len(errors) > 0 {
		result = append(result, fmt.Sprintf("❌ Other Issues (%d):", len(errors)))
		for i, e := range errors {
			if i >= 10 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(errors)-10))
				break
			}
			result = append(result, fmt.Sprintf("   %s", e))
		}
	}

	if len(result) == 0 {
		return "✅ No linting issues found"
	}
	return strings.Join(result, "\n")
}
