package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
)

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable global TokMan interception",
	Long: `Enable TokMan to automatically intercept and compress all CLI output.

When enabled, shell hooks will intercept commands and route them through
TokMan's compression pipeline. Use 'tokman disable' to turn off.

Examples:
  tokman enable        # Turn on automatic compression
  tokman disable       # Turn off automatic compression
  tokman status        # Check if TokMan is enabled`,
	Run: func(cmd *cobra.Command, args []string) {
		green := color.New(color.FgGreen).SprintFunc()

		markerPath := getEnabledMarkerPath()
		markerDir := filepath.Dir(markerPath)

		// Ensure directory exists
		if err := os.MkdirAll(markerDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Check if already enabled
		if isEnabled() {
			fmt.Printf("%s TokMan is already enabled\n", green("✓"))
			return
		}

		// Create marker file
		if err := os.WriteFile(markerPath, []byte("enabled\n"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error enabling TokMan: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s TokMan enabled globally\n", green("✓"))
		fmt.Println()
		fmt.Println("All commands will now be automatically compressed.")
		fmt.Println("Run 'tokman disable' to turn off.")
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable global TokMan interception",
	Long: `Disable TokMan interception. Commands will run normally without compression.

Use 'tokman enable' to turn interception back on.`,
	Run: func(cmd *cobra.Command, args []string) {
		red := color.New(color.FgRed).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()

		markerPath := getEnabledMarkerPath()

		if !isEnabled() {
			fmt.Printf("%s TokMan is already disabled\n", green("✓"))
			return
		}

		if err := os.Remove(markerPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error disabling TokMan: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s TokMan disabled\n", red("✗"))
		fmt.Println()
		fmt.Println("Commands will run normally without compression.")
		fmt.Println("Run 'tokman enable' to turn back on.")
	},
}

func init() {
	registry.Add(func() { registry.Register(enableCmd) })
	registry.Add(func() { registry.Register(disableCmd) })
}

// getEnabledMarkerPath returns the path to the enabled marker file.
func getEnabledMarkerPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "tokman", ".enabled")
}

// isEnabled checks if TokMan is globally enabled.
func isEnabled() bool {
	markerPath := getEnabledMarkerPath()
	_, err := os.Stat(markerPath)
	return err == nil
}
