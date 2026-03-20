package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file",
	RunE:  runConfigInit,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg := config.Defaults()
	fmt.Println("Current Configuration:")
	fmt.Println("=====================")
	fmt.Printf("  Pipeline:\n")
	fmt.Printf("    max_context_tokens: %d\n", cfg.Pipeline.MaxContextTokens)
	fmt.Printf("    default_budget: %d\n", cfg.Pipeline.DefaultBudget)
	fmt.Printf("    entropy_threshold: %.2f\n", cfg.Pipeline.EntropyThreshold)
	fmt.Printf("  Filter:\n")
	fmt.Printf("    mode: %s\n", cfg.Filter.Mode)
	fmt.Printf("    max_width: %d\n", cfg.Filter.MaxWidth)
	fmt.Printf("  Tracking:\n")
	fmt.Printf("    enabled: %v\n", cfg.Tracking.Enabled)
	fmt.Printf("    database_path: %s\n", cfg.Tracking.DatabasePath)
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "tokman")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		return nil
	}

	defaultConfig := `# TokMan Configuration
[pipeline]
max_context_tokens = 100000
default_budget = 0
entropy_threshold = 2.0

[filter]
mode = "minimal"
max_width = 0

[tracking]
enabled = true
retention_days = 90
`
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return err
	}

	fmt.Printf("Created config at %s\n", configPath)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// For now, just print the instruction
	fmt.Printf("To set %s = %s, edit your config file:\n", key, value)
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "tokman", "config.toml")
	fmt.Printf("  %s\n", configPath)
	return nil
}
