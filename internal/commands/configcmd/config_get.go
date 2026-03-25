package configcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

func init() {
	configCmd.AddCommand(configGetCmd)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	envKey := "TOKMAN_" + key
	if val := os.Getenv(envKey); val != "" {
		fmt.Printf("%s = %s (from env)\n", key, val)
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	configPath := filepath.Join(home, ".config", "tokman", "config.toml")

	if _, err := os.Stat(configPath); err != nil {
		fmt.Printf("%s = (not set)\n", key)
		fmt.Println("Run 'tokman config init' to create a config file.")
		return nil
	}

	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, readErr)
	}
	content := string(data)

	for _, line := range splitLines(content) {
		trimmed := trimStr(line)
		if len(trimmed) == 0 || trimmed[0] == '#' || trimmed[0] == '[' {
			continue
		}

		parts := splitFirst(trimmed, "=")
		if len(parts) == 2 {
			k := trimStr(parts[0])
			v := trimStr(parts[1])
			if k == key {
				fmt.Printf("%s = %s\n", key, v)
				return nil
			}
		}
	}

	fmt.Printf("%s = (not set)\n", key)
	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitFirst(s, sep string) []string {
	idx := findStr(s, sep)
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

func trimStr(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

func findStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
