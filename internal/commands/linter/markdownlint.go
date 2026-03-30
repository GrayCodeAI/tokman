package linter

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

var markdownlintCmd = &cobra.Command{
	Use:   "markdownlint [files...]",
	Short: "Markdown linting with compact output",
	Long: `Execute Markdownlint with token-optimized output.

Specialized filters for:
  - Error summary
  - File-based grouping

Examples:
  tokman markdownlint .
  tokman markdownlint README.md
  tokman markdownlint --fix .`,
	DisableFlagParsing: true,
	RunE:               runMarkdownlint,
}

func init() {
	registry.Add(func() { registry.Register(markdownlintCmd) })
}

func runMarkdownlint(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: markdownlint %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("markdownlint", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMarkdownlintOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "markdownlint", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	if filtered == "" && err == nil {
		fmt.Println("✅ No markdown issues found")
	} else {
		fmt.Println(filtered)
	}

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("markdownlint", "tokman markdownlint", originalTokens, filteredTokens)

	return err
}

func filterMarkdownlintOutput(raw string) string {
	if raw == "" {
		return ""
	}

	lines := strings.Split(raw, "\n")
	var result []string
	fileErrors := make(map[string][]string)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Parse markdownlint output: "file:line:col rule description"
		parts := strings.SplitN(trimmed, ":", 4)
		if len(parts) >= 3 {
			file := parts[0]
			fileErrors[file] = append(fileErrors[file], trimmed)
		} else {
			result = append(result, line)
		}
	}

	// Group by file in ultra-compact mode
	if shared.UltraCompact && len(fileErrors) > 0 {
		for file, errors := range fileErrors {
			if len(errors) == 1 {
				result = append(result, fmt.Sprintf("%s: 1 issue", file))
			} else {
				result = append(result, fmt.Sprintf("%s: %d issues", file, len(errors)))
			}
		}
		return strings.Join(result, "\n")
	}

	// Normal mode - show all errors, truncated
	for _, errors := range fileErrors {
		for _, err := range errors {
			result = append(result, shared.TruncateLine(err, 120))
		}
	}

	return strings.Join(result, "\n")
}
