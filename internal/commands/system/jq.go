package system

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

var jqCmd = &cobra.Command{
	Use:   "jq [filter] [file...]",
	Short: "JSON processor with compact output",
	Long: `Execute jq commands with token-optimized output.

Specialized filters for:
  - Large JSON: Truncated output
  - Arrays: Compact representation

Examples:
  tokman jq '.[] | .name' data.json
  tokman jq 'keys' package.json
  cat data.json | tokman jq '.items'`,
	DisableFlagParsing: true,
	RunE:               runJq,
}

func init() {
	registry.Add(func() { registry.Register(jqCmd) })
}

func runJq(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: jq %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("jq", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterJqOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "jq", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("jq", "tokman jq", originalTokens, filteredTokens)

	return err
}

func filterJqOutput(raw string) string {
	if raw == "" {
		return ""
	}

	// Check for jq errors
	if strings.HasPrefix(raw, "jq:") || strings.Contains(raw, "error:") {
		return raw
	}

	// In ultra-compact mode, truncate large outputs
	if shared.UltraCompact {
		lines := strings.Split(raw, "\n")
		if len(lines) > 50 {
			// Show first 20 and last 10 lines
			var result []string
			result = append(result, lines[:20]...)
			result = append(result, "... (truncated)")
			result = append(result, lines[len(lines)-10:]...)
			return strings.Join(result, "\n")
		}
		return raw
	}

	// In normal mode, just truncate very large outputs
	lines := strings.Split(raw, "\n")
	if len(lines) > 200 {
		var result []string
		result = append(result, lines[:100]...)
		result = append(result, "... (truncated)")
		result = append(result, lines[len(lines)-50:]...)
		return strings.Join(result, "\n")
	}

	return raw
}
