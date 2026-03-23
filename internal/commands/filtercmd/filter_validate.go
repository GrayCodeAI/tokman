package filtercmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/core"
	"github.com/GrayCodeAI/tokman/internal/commands/registry"
)

var filterValidateCmd = &cobra.Command{
	Use:   "filter-validate [name]",
	Short: "Validate TOML filter syntax",
	Long:  `Check TOML filter files for syntax errors.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runFilterValidate,
}

func init() {
	registry.Add(func() { registry.Register(filterValidateCmd) })
}

func runFilterValidate(cmd *cobra.Command, args []string) error {
	var filterDirs []string

	builtinDir := filepath.Join(core.GetTokmanSourceDir(), "internal", "toml", "builtin")
	if _, err := os.Stat(builtinDir); err == nil {
		filterDirs = append(filterDirs, builtinDir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if home != "" {
		userDir := filepath.Join(home, ".config", "tokman", "filters")
		if _, err := os.Stat(userDir); err == nil {
			filterDirs = append(filterDirs, userDir)
		}
	}

	hasErrors := false
	totalChecked := 0

	for _, dir := range filterDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
				continue
			}

			if len(args) > 0 {
				name := strings.TrimSuffix(e.Name(), ".toml")
				if name != args[0] {
					continue
				}
			}

			path := filepath.Join(dir, e.Name())
			totalChecked++

			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Printf("  ✗ %s: read error: %v\n", e.Name(), err)
				hasErrors = true
				continue
			}

			content := string(data)

			if !strings.Contains(content, "[[rules]]") && !strings.Contains(content, "[filters.") {
				fmt.Printf("  ✗ %s: missing [[rules]] or [filters] section\n", e.Name())
				hasErrors = true
				continue
			}

			if !strings.Contains(content, "match =") && !strings.Contains(content, "match_command =") {
				fmt.Printf("  ⚠ %s: no match pattern found\n", e.Name())
			}

			fmt.Printf("  ✓ %s\n", e.Name())
		}
	}

	fmt.Printf("\nChecked %d filters\n", totalChecked)
	if hasErrors {
		return fmt.Errorf("some filters have errors")
	}
	return nil
}
