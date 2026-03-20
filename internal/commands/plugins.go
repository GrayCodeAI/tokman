package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage filter plugins",
	Long:  `List installed filter plugins and their sources.`,
	RunE:  runPlugins,
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
}

func runPlugins(cmd *cobra.Command, args []string) error {
	fmt.Println("TokMan Filter Plugins")
	fmt.Println("=====================")
	fmt.Println()

	// Built-in filters
	builtinDir := filepath.Join(getTokmanSourceDir(), "internal", "toml", "builtin")
	if entries, err := os.ReadDir(builtinDir); err == nil {
		fmt.Printf("Built-in filters (%d):\n", len(entries))
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".toml") {
				name := strings.TrimSuffix(e.Name(), ".toml")
				fmt.Printf("  ✓ %s\n", name)
			}
		}
	}

	fmt.Println()

	// User filters
	home, _ := os.UserHomeDir()
	if home != "" {
		userDir := filepath.Join(home, ".config", "tokman", "filters")
		if entries, err := os.ReadDir(userDir); err == nil && len(entries) > 0 {
			fmt.Printf("User filters (%d):\n", len(entries))
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".toml") {
					name := strings.TrimSuffix(e.Name(), ".toml")
					fmt.Printf("  ✓ %s (user)\n", name)
				}
			}
		} else {
			fmt.Println("User filters: none installed")
			fmt.Printf("  Place .toml filter files in %s\n", userDir)
		}
	}

	fmt.Println()
	fmt.Println("Tip: Use 'tokman explain <command>' to see which filters apply.")

	return nil
}

func getTokmanSourceDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	// Try to find source from executable path
	return filepath.Dir(filepath.Dir(exe))
}
