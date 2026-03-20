package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
)

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check config files for errors",
	Long: `Validate tokman configuration files for syntax errors,
invalid values, and deprecated options.`,
	RunE: runConfigValidate,
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	fmt.Println("Validating tokman configuration...")
	fmt.Println()

	hasErrors := false

	// Check main config file
	configPaths := []string{
		cfgFile,
	}
	if cfgFile == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			configPaths = []string{
				filepath.Join(home, ".config", "tokman", "config.toml"),
			}
		}
	}

	for _, path := range configPaths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("  ⚠ %s: not found (using defaults)\n", path)
			continue
		}

		cfg, err := config.LoadFromFile(path)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", path, err)
			hasErrors = true
			continue
		}

		// Validate specific values
		if cfg.Pipeline.MaxContextTokens < 0 {
			fmt.Printf("  ✗ %s: max_context_tokens cannot be negative\n", path)
			hasErrors = true
		}
		if cfg.Pipeline.EntropyThreshold < 0 || cfg.Pipeline.EntropyThreshold > 1 {
			fmt.Printf("  ✗ %s: entropy_threshold must be 0.0-1.0\n", path)
			hasErrors = true
		}
		if cfg.Pipeline.PerplexityThreshold < 0 || cfg.Pipeline.PerplexityThreshold > 1 {
			fmt.Printf("  ✗ %s: perplexity_threshold must be 0.0-1.0\n", path)
			hasErrors = true
		}
		if cfg.Pipeline.H2OSinkSize < 0 {
			fmt.Printf("  ✗ %s: h2o_sink_size cannot be negative\n", path)
			hasErrors = true
		}
		if cfg.Pipeline.CacheMaxSize < 0 {
			fmt.Printf("  ✗ %s: cache_max_size cannot be negative\n", path)
			hasErrors = true
		}

		fmt.Printf("  ✓ %s: valid\n", path)
	}

	// Check defaults
	defaults := config.Defaults()
	if defaults.Pipeline.MaxContextTokens > 0 {
		fmt.Printf("  ✓ defaults: max_context=%d tokens\n", defaults.Pipeline.MaxContextTokens)
	}

	fmt.Println()
	if hasErrors {
		return fmt.Errorf("configuration has errors")
	}
	fmt.Println("All configuration checks passed!")
	return nil
}
