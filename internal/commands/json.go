package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var jsonDepth int

var jsonCmd = &cobra.Command{
	Use:   "json <file>",
	Short: "Show JSON structure without values",
	Long: `Show JSON structure/schema without actual values.

Useful for understanding large JSON responses without consuming tokens on values.

Examples:
  tokman json response.json
  tokman json response.json --depth 3`,
	Args: cobra.ExactArgs(1),
	RunE: runJSON,
}

func init() {
	rootCmd.AddCommand(jsonCmd)
	jsonCmd.Flags().IntVarP(&jsonDepth, "depth", "d", 5, "Max depth to show")
}

func runJSON(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	filePath := args[0]
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Generate schema
	filtered := generateSchema(v, 0, jsonDepth)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(string(data))
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(filePath, "tokman json", originalTokens, filteredTokens)

	if verbose {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d (%.1f%%)\n", 
			originalTokens-filteredTokens, 
			float64(originalTokens-filteredTokens)/float64(originalTokens)*100)
	}

	return nil
}

func generateSchema(v interface{}, depth, maxDepth int) string {
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
