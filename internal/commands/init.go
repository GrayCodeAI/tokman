package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Patel230/tokman/internal/config"
	"github.com/Patel230/tokman/internal/tracking"
	"github.com/Patel230/tokman/internal/utils"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TokMan",
	Long: `Initialize TokMan by creating the necessary directories,
configuration files, and database.`,
	Run: func(cmd *cobra.Command, args []string) {
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		fmt.Printf("\n%s\n", green("🌸 Initializing TokMan..."))
		fmt.Println()

		// Create config directory
		configDir := filepath.Dir(config.ConfigPath())
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  %s Config directory: %s\n", green("✓"), cyan(configDir))

		// Create data directory
		dataDir := config.DataPath()
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  %s Data directory: %s\n", green("✓"), cyan(dataDir))

		// Create hooks directory
		hooksDir := config.HooksPath()
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating hooks directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  %s Hooks directory: %s\n", green("✓"), cyan(hooksDir))

		// Initialize database
		dbPath := config.DatabasePath()
		tracker, err := tracking.NewTracker(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
			os.Exit(1)
		}
		tracker.Close()
		fmt.Printf("  %s Database: %s\n", green("✓"), cyan(dbPath))

		// Create default config if it doesn't exist
		cfgPath := config.ConfigPath()
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			cfg := config.Defaults()
			if err := cfg.Save(cfgPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("  %s Config file: %s\n", green("✓"), cyan(cfgPath))
		} else {
			fmt.Printf("  %s Config file exists: %s\n", green("✓"), cyan(cfgPath))
		}

		// Create log file
		logPath := config.LogPath()
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			if err := utils.InitLogger(logPath, utils.LevelInfo); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create log file: %v\n", err)
			} else {
				fmt.Printf("  %s Log file: %s\n", green("✓"), cyan(logPath))
			}
		}

		fmt.Println()
		fmt.Println(green("✓ TokMan initialized successfully!"))
		fmt.Println()
		fmt.Println("To enable shell hooks, add to your .bashrc or .zshrc:")
		fmt.Printf("  %s\n", cyan("source ~/.local/share/tokman/hooks/tokman-rewrite.sh"))
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
