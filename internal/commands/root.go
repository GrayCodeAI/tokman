package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/integrity"
	"github.com/GrayCodeAI/tokman/internal/utils"

	// Sub-package imports (blank imports for side effects - init() registration)
	_ "github.com/GrayCodeAI/tokman/internal/commands/agents"
	_ "github.com/GrayCodeAI/tokman/internal/commands/analysis"
	_ "github.com/GrayCodeAI/tokman/internal/commands/build"
	_ "github.com/GrayCodeAI/tokman/internal/commands/cloud"
	_ "github.com/GrayCodeAI/tokman/internal/commands/configcmd"
	_ "github.com/GrayCodeAI/tokman/internal/commands/container"
	_ "github.com/GrayCodeAI/tokman/internal/commands/core"
	_ "github.com/GrayCodeAI/tokman/internal/commands/filtercmd"
	_ "github.com/GrayCodeAI/tokman/internal/commands/hooks"
	_ "github.com/GrayCodeAI/tokman/internal/commands/init"
	_ "github.com/GrayCodeAI/tokman/internal/commands/lang"
	_ "github.com/GrayCodeAI/tokman/internal/commands/linter"
	_ "github.com/GrayCodeAI/tokman/internal/commands/output"
	_ "github.com/GrayCodeAI/tokman/internal/commands/pkgmgr"
	_ "github.com/GrayCodeAI/tokman/internal/commands/sessioncmd"
	_ "github.com/GrayCodeAI/tokman/internal/commands/system"
	_ "github.com/GrayCodeAI/tokman/internal/commands/test"
	_ "github.com/GrayCodeAI/tokman/internal/commands/vcs"
	_ "github.com/GrayCodeAI/tokman/internal/commands/web"
)

var (
	cfgFile      string
	verbose      int // Count-based: -v, -vv, -vvv
	dryRun       bool
	ultraCompact bool
	skipEnv      bool
	queryIntent  string   // Query intent for query-aware compression
	llmEnabled   bool     // Enable LLM-based compression
	tokenBudget  int      // Token budget for compression (0 = unlimited)
	fallbackArgs []string // Args for fallback handler
	layerPreset  string   // Pipeline preset: fast/balanced/full (T90)
	outputFile   string   // R35: Write output to file
	quietMode    bool     // R36: Suppress all non-essential output
	jsonOutput   bool     // R37: Machine-readable JSON output

	// Compaction flags (Layer 11)
	compactionEnabled    bool
	compactionThreshold  int
	compactionPreserve   int
	compactionMaxTokens  int
	compactionSnapshot   bool
	compactionAutoDetect bool

	// Reversible compression (R1: claw-compactor style)
	reversibleEnabled bool
)

// Version is set via ldflags during build
var Version = "dev"

// rootCmd represents the base command when called without any subcommands.
var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokman",
		Short: "Token-aware CLI proxy",
		Long: `TokMan intercepts CLI commands and filters verbose output
to reduce token usage in LLM interactions.

It acts as a transparent proxy that executes commands, captures their
output, applies intelligent filtering, and tracks token savings.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			shared.SetRootCmd(cmd)
			shared.Version = Version
			shared.SetConfig(struct {
				Verbose              int
				DryRun               bool
				UltraCompact         bool
				SkipEnv              bool
				QueryIntent          string
				LLMEnabled           bool
				TokenBudget          int
				FallbackArgs         []string
				LayerPreset          string
				OutputFile           string
				QuietMode            bool
				JSONOutput           bool
				CompactionEnabled    bool
				CompactionThreshold  int
				CompactionPreserve   int
				CompactionMaxTokens  int
				CompactionSnapshot   bool
				CompactionAutoDetect bool
				ReversibleEnabled    bool
			}{
				Verbose:              verbose,
				DryRun:               dryRun,
				UltraCompact:         ultraCompact,
				SkipEnv:              skipEnv,
				QueryIntent:          queryIntent,
				LLMEnabled:           llmEnabled,
				TokenBudget:          tokenBudget,
				FallbackArgs:         fallbackArgs,
				LayerPreset:          layerPreset,
				OutputFile:           outputFile,
				QuietMode:            quietMode,
				JSONOutput:           jsonOutput,
				CompactionEnabled:    compactionEnabled,
				CompactionThreshold:  compactionThreshold,
				CompactionPreserve:   compactionPreserve,
				CompactionMaxTokens:  compactionMaxTokens,
				CompactionSnapshot:   compactionSnapshot,
				CompactionAutoDetect: compactionAutoDetect,
				ReversibleEnabled:    reversibleEnabled,
			})
			shared.SetConfigFile(cfgFile)

			if skipEnv {
				os.Setenv("SKIP_ENV_VALIDATION", "1")
			}
			if isOperationalCommand(cmd) {
				if err := integrity.RuntimeCheck(); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			fallback := shared.GetFallback()
			output, handled, err := fallback.Handle(args)

			if !handled {
				return fmt.Errorf("unknown command: %s", args[0])
			}

			fmt.Print(output)
			return err
		},
	}
	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// Unknown commands are handled by the TOML filter fallback system.
func Execute() {
	// Enable unknown command handling
	rootCmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	rootCmd.TraverseChildren = true

	_, err := rootCmd.ExecuteC()
	if err != nil {
		// Check if this is an unknown command error
		if isUnknownCommandError(err) {
			// Extract the unknown command from args
			args := extractUnknownCommandArgs()
			if len(args) > 0 {
				fallback := shared.GetFallback()
				output, handled, ferr := fallback.Handle(args)
				if handled {
					fmt.Print(output)
					if ferr != nil {
						os.Exit(1)
					}
					return
				}
			}
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// isUnknownCommandError checks if the error is an unknown command error
func isUnknownCommandError(err error) bool {
	return strings.Contains(err.Error(), "unknown command") ||
		strings.Contains(err.Error(), "unknown shorthand flag")
}

// extractUnknownCommandArgs extracts args for the fallback handler
func extractUnknownCommandArgs() []string {
	return fallbackArgs
}

func init() {
	registry.Init(rootCmd)

	cobra.OnInitialize(initConfig)

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("TokMan {{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is ~/.config/tokman/config.toml)")
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v",
		"verbosity level (-v, -vv, -vvv)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false,
		"show what would be filtered without executing")
	rootCmd.PersistentFlags().BoolVarP(&ultraCompact, "ultra-compact", "u", true,
		"ultra-compact mode: ASCII icons, inline format (default: true)")
	rootCmd.PersistentFlags().BoolVar(&skipEnv, "skip-env", false,
		"set SKIP_ENV_VALIDATION=1 for child processes")
	rootCmd.PersistentFlags().StringVar(&queryIntent, "query", "",
		"query intent for compression (debug/review/deploy/search)")
	rootCmd.PersistentFlags().BoolVar(&llmEnabled, "llm", false,
		"enable LLM-based compression (requires Ollama/LM Studio)")
	rootCmd.PersistentFlags().IntVar(&tokenBudget, "budget", 0,
		"token budget for output (0 = unlimited, e.g., --budget 2000)")
	rootCmd.PersistentFlags().StringVar(&layerPreset, "preset", "",
		"pipeline preset: fast, balanced, or full (T90)")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "",
		"write output to file instead of stdout")
	rootCmd.PersistentFlags().BoolVarP(&quietMode, "quiet", "q", false,
		"suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false,
		"machine-readable JSON output")

	// Compaction flags (Layer 11 - Semantic compression)
	rootCmd.PersistentFlags().BoolVar(&compactionEnabled, "compaction", true,
		"enable semantic compaction for chat/conversation content (default: true)")
	rootCmd.PersistentFlags().IntVar(&compactionThreshold, "compaction-threshold", 500,
		"minimum tokens to trigger compaction (default: 500)")
	rootCmd.PersistentFlags().IntVar(&compactionPreserve, "compaction-preserve", 10,
		"recent conversation turns to preserve verbatim (default: 10)")
	rootCmd.PersistentFlags().IntVar(&compactionMaxTokens, "compaction-max-tokens", 5000,
		"maximum tokens for compaction summary (default: 5000)")
	rootCmd.PersistentFlags().BoolVar(&compactionSnapshot, "compaction-snapshot", true,
		"use state snapshot format (4-section XML)")
	rootCmd.PersistentFlags().BoolVar(&compactionAutoDetect, "compaction-auto-detect", true,
		"auto-detect conversation content for compaction")

	// Reversible compression flag (R1)
	rootCmd.PersistentFlags().BoolVar(&reversibleEnabled, "reversible", false,
		"store original output for later restoration (use 'tokman restore' to retrieve)")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("query", rootCmd.PersistentFlags().Lookup("query"))
	viper.BindPFlag("llm", rootCmd.PersistentFlags().Lookup("llm"))
	viper.BindPFlag("budget", rootCmd.PersistentFlags().Lookup("budget"))
	viper.BindPFlag("pipeline.enable_compaction", rootCmd.PersistentFlags().Lookup("compaction"))
	viper.BindPFlag("pipeline.compaction_threshold", rootCmd.PersistentFlags().Lookup("compaction-threshold"))
	viper.BindPFlag("pipeline.compaction_preserve_turns", rootCmd.PersistentFlags().Lookup("compaction-preserve"))
	viper.BindPFlag("pipeline.compaction_max_tokens", rootCmd.PersistentFlags().Lookup("compaction-max-tokens"))
	viper.BindPFlag("pipeline.compaction_state_snapshot", rootCmd.PersistentFlags().Lookup("compaction-snapshot"))
	viper.BindPFlag("pipeline.compaction_auto_detect", rootCmd.PersistentFlags().Lookup("compaction-auto-detect"))

	registry.RegisterAll()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "tokman"))
		}
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
