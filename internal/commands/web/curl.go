package web

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

var curlCmd = &cobra.Command{
	Use:   "curl [args...]",
	Short: "Curl with auto-JSON detection and schema output",
	Long: `Execute curl with automatic JSON detection and schema extraction.

Automatically detects JSON responses and shows schema instead of values,
reducing token consumption for large API responses.

Examples:
  tokman curl https://api.example.com/data
  tokman curl -H "Authorization: Bearer token" https://api.example.com/users`,
	DisableFlagParsing: true,
	RunE:               runCurl,
}

func init() {
	registry.Add(func() { registry.Register(curlCmd) })
}

func runCurl(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	curlArgs := []string{"-s"}
	curlArgs = append(curlArgs, args...)

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: curl %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("curl", curlArgs...)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: curl %s\n", string(output))
		return err
	}

	raw := string(output)
	filtered := filterCurlOutput(raw, 5, 30, 200)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(strings.Join(args, " "), "tokman curl", originalTokens, filteredTokens)

	return nil
}

func filterCurlOutput(output string, jsonDepth, maxLines, maxLineLen int) string {
	trimmed := strings.TrimSpace(output)

	if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) &&
		(strings.HasSuffix(trimmed, "}") || strings.HasSuffix(trimmed, "]")) {
		schema := shared.TryJSONSchema(trimmed, jsonDepth)
		if schema != "" && len(schema) <= len(trimmed) {
			return schema
		}
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) > maxLines {
		var result []string
		for i := 0; i < maxLines; i++ {
			result = append(result, shared.TruncateLine(lines[i], maxLineLen))
		}
		msg := fmt.Sprintf("\n... (%d more lines, %d bytes total)", len(lines)-maxLines, len(trimmed))
		return strings.Join(result, "\n") + msg
	}

	var result []string
	for _, line := range lines {
		result = append(result, shared.TruncateLine(line, maxLineLen))
	}
	return strings.Join(result, "\n")
}


