package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
)

var searchCmd = &cobra.Command{
	Use:   "search <tool>",
	Short: "Search available filters for a command",
	Long:  `Find which TOML filter applies to a given command or tool.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func init() {
	registry.Add(func() { registry.Register(searchCmd) })
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(args[0])

	fmt.Printf("Searching filters for '%s'...\n\n", query)

	// Search built-in filters
	builtinDir := filepath.Join(getTokmanSourceDir(), "internal", "toml", "builtin")
	found := 0

	if entries, err := os.ReadDir(builtinDir); err == nil {
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".toml") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".toml")
			if strings.Contains(strings.ToLower(name), query) {
				fmt.Printf("  ✓ %s (built-in)\n", name)
				found++
			}
		}
	}

	// Search user filters
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if home != "" {
		userDir := filepath.Join(home, ".config", "tokman", "filters")
		if entries, err := os.ReadDir(userDir); err == nil {
			for _, e := range entries {
				if !strings.HasSuffix(e.Name(), ".toml") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".toml")
				if strings.Contains(strings.ToLower(name), query) {
					fmt.Printf("  ✓ %s (user)\n", name)
					found++
				}
			}
		}
	}

	if found == 0 {
		fmt.Printf("No filters found for '%s'.\n", query)
		fmt.Println("\nTip: Use 'tokman marketplace search <query>' to find community filters.")
		fmt.Println("     Or create a custom filter in ~/.config/tokman/filters/<name>.toml")
	}

	return nil
}

func getTokmanSourceDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	dir := filepath.Dir(filepath.Dir(exe))
	builtinDir := filepath.Join(dir, "internal", "toml", "builtin")
	if _, err := os.Stat(builtinDir); err == nil {
		return dir
	}
	dir = filepath.Dir(exe)
	builtinDir = filepath.Join(dir, "internal", "toml", "builtin")
	if _, err := os.Stat(builtinDir); err == nil {
		return dir
	}
	return "."
}
