package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	envFilter  string
	envShowAll bool
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Show environment variables (filtered, sensitive masked)",
	Long: `Show environment variables with sensitive values masked.

Filters common noise variables and masks secrets (API keys, tokens, passwords).

Examples:
  tokman env
  tokman env --filter AWS
  tokman env --show-all`,
	RunE: runEnv,
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().StringVarP(&envFilter, "filter", "f", "", "Filter by name (e.g. PATH, AWS)")
	envCmd.Flags().BoolVar(&envShowAll, "show-all", false, "Show all (include sensitive)")
}

func runEnv(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Get environment variables
	env := os.Environ()
	sort.Strings(env)

	// Filter and mask
	var output strings.Builder
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name, value := parts[0], parts[1]

		// Apply filter
		if envFilter != "" && !strings.Contains(strings.ToUpper(name), strings.ToUpper(envFilter)) {
			continue
		}

		// Mask sensitive values
		if !envShowAll && isSensitive(name) {
			value = maskSensitive(value)
		}

		output.WriteString(fmt.Sprintf("%s=%s\n", name, value))
	}

	result := output.String()
	fmt.Print(result)

	originalTokens := filter.EstimateTokens(strings.Join(env, "\n"))
	filteredTokens := filter.EstimateTokens(result)
	timer.Track("env", "tokman env", originalTokens, filteredTokens)

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Variables: %d -> %d\n", len(env), len(strings.Split(result, "\n"))-1)
	}

	return nil
}

func isSensitive(name string) bool {
	name = strings.ToUpper(name)
	sensitive := []string{"KEY", "SECRET", "TOKEN", "PASSWORD", "PASS", "API", "AUTH", "CRED", "PRIVATE"}
	for _, s := range sensitive {
		if strings.Contains(name, s) {
			return true
		}
	}
	return false
}

func maskSensitive(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "***" + value[len(value)-2:]
}
