package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/integrity"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

var (
	cfgFile      string
	verbose      int // Count-based: -v, -vv, -vvv
	dryRun       bool
	ultraCompact bool
	skipEnv      bool
)

// Version is set via ldflags during build
var Version = "dev"

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "tokman",
	Short: "Token-aware CLI proxy",
	Long: `TokMan intercepts CLI commands and filters verbose output
to reduce token usage in LLM interactions.

It acts as a transparent proxy that executes commands, captures their
output, applies intelligent filtering, and tracks token savings.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set SKIP_ENV_VALIDATION for child processes if requested
		if skipEnv {
			os.Setenv("SKIP_ENV_VALIDATION", "1")
		}
		// Skip integrity check for meta commands
		if isOperationalCommand(cmd) {
			if err := integrity.RuntimeCheck(); err != nil {
				return err
			}
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("TokMan {{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is ~/.config/tokman/config.toml)")
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v",
		"verbosity level (-v, -vv, -vvv)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false,
		"show what would be filtered without executing")
	rootCmd.PersistentFlags().BoolVarP(&ultraCompact, "ultra-compact", "u", false,
		"ultra-compact mode: ASCII icons, inline format")
	rootCmd.PersistentFlags().BoolVar(&skipEnv, "skip-env", false,
		"set SKIP_ENV_VALIDATION=1 for child processes")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME/.config/tokman")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("TOKMAN")

	// Read config file if exists
	if err := viper.ReadInConfig(); err == nil && verbose > 0 {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Initialize logger
	logLevel := utils.LevelInfo
	if verbose > 0 {
		logLevel = utils.LevelDebug
	}
	if err := utils.InitLogger(config.LogPath(), logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize logger: %v\n", err)
	}
}

// GetConfig returns the current configuration.
func GetConfig() (*config.Config, error) {
	return config.Load(cfgFile)
}

// IsVerbose returns whether verbose mode is enabled.
func IsVerbose() bool {
	return verbose > 0
}

// VerbosityLevel returns the verbosity level (0-3).
func VerbosityLevel() int {
	return verbose
}

// IsUltraCompact returns whether ultra-compact mode is enabled.
func IsUltraCompact() bool {
	return ultraCompact
}

// IsSkipEnv returns whether SKIP_ENV_VALIDATION should be set.
func IsSkipEnv() bool {
	return skipEnv
}

// IsDryRun returns whether dry-run mode is enabled.
func IsDryRun() bool {
	return dryRun
}

// isOperationalCommand returns true for commands that process CLI output
// and need runtime integrity verification. Meta commands (init, verify,
// config, economics, status, report, summary) are excluded.
func isOperationalCommand(cmd *cobra.Command) bool {
	// Meta commands that don't need integrity checks
	metaCommands := map[string]bool{
		"init":       true,
		"verify":     true,
		"config":     true,
		"economics":  true,
		"status":     true,
		"report":     true,
		"summary":    true,
		"ccusage":    true,
		"help":       true,
		"version":    true,
		"rewrite":    true,
		"deps":       true,
		"gain":       true,
		"hook-audit": true,
		"discover":   true,
		"learn":      true,
		"err":        true,
	}

	// Get the called command name
	name := cmd.Name()
	if metaCommands[name] {
		return false
	}

	// Check parent command for subcommands
	for p := cmd.Parent(); p != nil; p = p.Parent() {
		if metaCommands[p.Name()] {
			return false
		}
	}

	return true
}
