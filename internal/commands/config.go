package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or create configuration file",
	Long: `Display the current TokMan configuration or create a default config file.

The configuration file is stored at ~/.config/tokman/config.toml and controls:
- Token tracking behavior
- Output filtering settings
- Shell hook exclusions`,
	Run: func(cmd *cobra.Command, args []string) {
		create, _ := cmd.Flags().GetBool("create")

		if create {
			path, err := createDefaultConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created: %s\n", path)
			return
		}

		// Show current config
		showConfig()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().Bool("create", false, "Create default config file")
}

func createDefaultConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "tokman", "config.toml")

	cfg := config.Defaults()
	if err := cfg.Save(configPath); err != nil {
		return "", fmt.Errorf("failed to save config: %w", err)
	}

	return configPath, nil
}

func showConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(home, ".config", "tokman", "config.toml")
	fmt.Printf("Config: %s\n\n", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("(default config, file not created)")
		fmt.Println()
		cfg := config.Defaults()
		printConfig(cfg)
		return
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	printConfig(cfg)
}

func printConfig(cfg *config.Config) {
	fmt.Println("[tracking]")
	fmt.Printf("enabled = %v\n", cfg.Tracking.Enabled)
	if cfg.Tracking.DatabasePath != "" {
		fmt.Printf("database_path = %q\n", cfg.Tracking.DatabasePath)
	}
	fmt.Printf("telemetry = %v\n", cfg.Tracking.Telemetry)
	fmt.Println()

	fmt.Println("[filter]")
	fmt.Printf("mode = %q\n", cfg.Filter.Mode)
	fmt.Printf("noise_dirs = %v\n", cfg.Filter.NoiseDirs)
	fmt.Println()

	fmt.Println("[hooks]")
	if len(cfg.Hooks.ExcludedCommands) > 0 {
		fmt.Printf("excluded_commands = %v\n", cfg.Hooks.ExcludedCommands)
	} else {
		fmt.Println("excluded_commands = []")
	}
}
