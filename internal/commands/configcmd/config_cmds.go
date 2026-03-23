package configcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
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
	ConfigCmd.AddCommand(configShowCmd)
	ConfigCmd.AddCommand(configInitCmd)
	ConfigCmd.AddCommand(configSetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := shared.GetConfig()
	if err != nil {
		cfg = config.Defaults()
	}
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
	configDir := config.ConfigDir()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

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
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return err
	}

	fmt.Printf("Created config at %s\n", configPath)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	configDir := config.ConfigDir()
	configPath := filepath.Join(configDir, "config.toml")

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	var lines []string
	if data, err := os.ReadFile(configPath); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	parts := strings.SplitN(key, ".", 2)
	section := ""
	field := key
	if len(parts) == 2 {
		section = parts[0]
		field = parts[1]
	}

	found := false
	inSection := section == ""
	newLines := make([]string, 0, len(lines)+2)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			secName := strings.TrimPrefix(strings.TrimSuffix(trimmed, "]"), "[")
			inSection = secName == section
			newLines = append(newLines, line)
			continue
		}

		if inSection && strings.Contains(trimmed, "=") {
			kv := strings.SplitN(trimmed, "=", 2)
			existingKey := strings.TrimSpace(kv[0])
			if existingKey == field {
				newLines = append(newLines, fmt.Sprintf("%s = %s", field, value))
				found = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	if !found {
		if section != "" {
			sectionExists := false
			for _, line := range newLines {
				if strings.TrimSpace(line) == fmt.Sprintf("[%s]", section) {
					sectionExists = true
					break
				}
			}
			if !sectionExists {
				newLines = append(newLines, "", fmt.Sprintf("[%s]", section))
			}
			if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
				newLines = append(newLines, "")
			}
			newLines = append(newLines, fmt.Sprintf("%s = %s", field, value))
		} else {
			if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
				newLines = append(newLines, "")
			}
			newLines = append(newLines, fmt.Sprintf("%s = %s", field, value))
		}
	}

	content := strings.Join(newLines, "\n")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	fmt.Printf("Config: %s\n", configPath)
	return nil
}
