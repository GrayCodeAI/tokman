package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

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
	rootCmd.AddCommand(curlCmd)
}

func runCurl(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Build curl command with silent mode
	curlArgs := []string{"-s"}
	curlArgs = append(curlArgs, args...)

	if verbose > 0 {
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

// filterCurlOutput applies JSON schema extraction or truncation based on content.
func filterCurlOutput(output string, jsonDepth, maxLines, maxLineLen int) string {
	trimmed := strings.TrimSpace(output)

	// Try JSON detection: starts with { or [
	if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) &&
		(strings.HasSuffix(trimmed, "}") || strings.HasSuffix(trimmed, "]")) {
		schema := tryJSONSchema(trimmed, jsonDepth)
		// Only use schema if it's actually shorter than original
		if schema != "" && len(schema) <= len(trimmed) {
			return schema
		}
	}

	// Not JSON: truncate long output
	lines := strings.Split(trimmed, "\n")
	if len(lines) > maxLines {
		var result []string
		for i := 0; i < maxLines; i++ {
			result = append(result, truncateLine(lines[i], maxLineLen))
		}
		msg := fmt.Sprintf("\n... (%d more lines, %d bytes total)", len(lines)-maxLines, len(trimmed))
		return strings.Join(result, "\n") + msg
	}

	// Short output: return as-is but truncate long lines
	var result []string
	for _, line := range lines {
		result = append(result, truncateLine(line, maxLineLen))
	}
	return strings.Join(result, "\n")
}

// tryJSONSchema attempts to generate a schema from JSON content.
func tryJSONSchema(jsonStr string, maxDepth int) string {
	var v interface{}
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return ""
	}
	return generateSchemaFromJSON(v, 0, maxDepth)
}

// generateSchemaFromJSON recursively generates a schema representation for curl output.
func generateSchemaFromJSON(v interface{}, depth, maxDepth int) string {
	if depth > maxDepth {
		return "..."
	}

	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		elemType := generateSchema(val[0], depth+1, maxDepth)
		return fmt.Sprintf("[%s, ...]", elemType)
	case map[string]interface{}:
		if len(val) == 0 {
			return "{}"
		}
		var parts []string
		for k, v := range val {
			schema := generateSchema(v, depth+1, maxDepth)
			parts = append(parts, fmt.Sprintf("%s: %s", k, schema))
		}
		indent := strings.Repeat("  ", depth)
		return fmt.Sprintf("{\n%s  %s\n%s}", indent, strings.Join(parts, ",\n"+indent+"  "), indent)
	default:
		return fmt.Sprintf("%T", v)
	}
}

// truncateLine truncates a line to maxLen characters.
func truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen-3] + "..."
}
