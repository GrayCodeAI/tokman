package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/toml"
)

var trustCmd = &cobra.Command{
	Use:   "trust [project-path]",
	Short: "Trust a project directory for local filters",
	Long: `Mark a project directory as trusted to load local TOML filters.

This is a security measure to prevent malicious filter injection.
Project-local filters are stored in .tokman/filters.toml and are only
loaded if the project directory has been explicitly trusted.

Example:
  tokman trust /path/to/project
  tokman trust .  # Trust current directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTrust,
}

var untrustCmd = &cobra.Command{
	Use:   "untrust [project-path]",
	Short: "Remove trust from a project directory",
	Long: `Remove a project directory from the trusted list.

After untrusting, local .tokman/filters.toml files in that directory
will no longer be loaded.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUntrust,
}

var listTrustedCmd = &cobra.Command{
	Use:   "list-trusted",
	Short: "List all trusted project paths",
	Run:   runListTrusted,
}

func init() {
	registry.Add(func() { registry.Register(trustCmd) })
	registry.Add(func() { registry.Register(untrustCmd) })
	registry.Add(func() { registry.Register(listTrustedCmd) })
}

func runTrust(cmd *cobra.Command, args []string) error {
	loader := toml.GetLoader()

	// Determine project path
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Resolve to absolute path
	absPath, err := resolvePath(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if .tokman/filters.toml exists
	filtersPath := fmt.Sprintf("%s/.tokman/filters.toml", absPath)
	if _, err := os.Stat(filtersPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: %s does not exist\n", filtersPath)
		fmt.Fprintf(os.Stderr, "Create it with: mkdir -p %s/.tokman && touch %s\n", absPath, filtersPath)
	}

	// Trust the project
	if err := loader.TrustProject(absPath); err != nil {
		return fmt.Errorf("failed to trust project: %w", err)
	}

	fmt.Printf("✓ Trusted: %s\n", absPath)
	fmt.Println("Local .tokman/filters.toml will now be loaded for this project")
	return nil
}

func runUntrust(cmd *cobra.Command, args []string) error {
	loader := toml.GetLoader()

	// Determine project path
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Resolve to absolute path
	absPath, err := resolvePath(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Untrust the project
	if err := loader.UntrustProject(absPath); err != nil {
		return fmt.Errorf("failed to untrust project: %w", err)
	}

	fmt.Printf("✓ Untrusted: %s\n", absPath)
	return nil
}

func runListTrusted(cmd *cobra.Command, args []string) {
	loader := toml.GetLoader()
	paths := loader.ListTrusted()

	if len(paths) == 0 {
		fmt.Println("No trusted projects")
		return
	}

	fmt.Println("Trusted projects:")
	for _, path := range paths {
		fmt.Printf("  %s\n", path)
	}
}

func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return abs, nil
}
