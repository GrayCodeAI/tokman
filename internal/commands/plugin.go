package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage custom filter plugins",
	Long: `Manage custom filter plugins for TokMan.

Plugins allow you to define custom filtering rules in JSON format.
Each plugin can hide or replace patterns in command output.

Example plugin (save as ~/.config/tokman/plugins/hide-npm-warnings.json):
  {
    "name": "hide-npm-warnings",
    "description": "Hide npm deprecation warnings",
    "enabled": true,
    "patterns": ["npm WARN deprecated"],
    "mode": "hide"
  }`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all loaded plugins",
	RunE:  runPluginList,
}

var pluginCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new plugin template",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginCreate,
}

var pluginExamplesCmd = &cobra.Command{
	Use:   "examples",
	Short: "Generate example plugins",
	RunE:  runPluginExamples,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginEnable,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginDisable,
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginCreateCmd)
	pluginCmd.AddCommand(pluginExamplesCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
}

func getPluginsDir() string {
	return filepath.Join(filepath.Dir(config.ConfigPath()), "plugins")
}

func runPluginList(cmd *cobra.Command, args []string) error {
	pluginsDir := getPluginsDir()
	pm := filter.NewPluginManager(pluginsDir)

	if err := pm.LoadPlugins(); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	filters := pm.GetFilters()
	if len(filters) == 0 {
		fmt.Println("No plugins loaded.")
		fmt.Println("\nRun 'tokman plugin examples' to generate example plugins.")
		return nil
	}

	fmt.Println("Loaded Plugins:")
	fmt.Println()
	for _, f := range filters {
		status := "disabled"
		if f.Name() != "" {
			// Check if enabled by trying to cast
			status = "enabled"
		}
		fmt.Printf("  • %s (%s)\n", f.Name(), status)
		if f.Description() != "" {
			fmt.Printf("    %s\n", f.Description())
		}
	}

	return nil
}

func runPluginCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	pluginsDir := getPluginsDir()

	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	pluginPath := filepath.Join(pluginsDir, name+".json")

	// Check if already exists
	if _, err := os.Stat(pluginPath); err == nil {
		return fmt.Errorf("plugin %q already exists", name)
	}

	template := filter.PluginConfig{
		Name:        name,
		Description: "Description of what this plugin filters",
		Enabled:     true,
		Patterns:    []string{"pattern to match"},
		Mode:        "hide",
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(pluginPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}

	fmt.Printf("Created plugin: %s\n", pluginPath)
	fmt.Println("\nEdit the file to customize your filter patterns.")
	return nil
}

func runPluginExamples(cmd *cobra.Command, args []string) error {
	pluginsDir := getPluginsDir()

	if err := filter.SaveExamplePlugins(pluginsDir); err != nil {
		return fmt.Errorf("failed to save examples: %w", err)
	}

	fmt.Printf("Generated example plugins in: %s\n", pluginsDir)
	fmt.Println("\nExamples created:")
	fmt.Println("  • hide-npm-warnings.json")
	fmt.Println("  • shorten-paths.json")
	fmt.Println("  • hide-test-output.json")
	return nil
}

func runPluginEnable(cmd *cobra.Command, args []string) error {
	return togglePlugin(args[0], true)
}

func runPluginDisable(cmd *cobra.Command, args []string) error {
	return togglePlugin(args[0], false)
}

func togglePlugin(name string, enabled bool) error {
	pluginsDir := getPluginsDir()
	pluginPath := filepath.Join(pluginsDir, name+".json")

	data, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin %q not found", name)
	}

	var cfg filter.PluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid plugin format: %w", err)
	}

	cfg.Enabled = enabled

	newData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(pluginPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to update plugin: %w", err)
	}

	action := "enabled"
	if !enabled {
		action = "disabled"
	}
	fmt.Printf("Plugin %q %s.\n", name, action)
	return nil
}
