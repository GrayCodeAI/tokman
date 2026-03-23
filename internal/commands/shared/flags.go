package shared

import (
	"fmt"
	"os"
	"sync"
)

// Global flags and their thread-safe accessors.
// This file contains only flag state - no external dependencies.

var (
	configMu sync.RWMutex

	// Global flags set by CLI
	CfgFile      string
	Verbose      int
	DryRun       bool
	UltraCompact bool
	SkipEnv      bool
	QueryIntent  string
	LLMEnabled   bool
	TokenBudget  int
	FallbackArgs []string
	LayerPreset  string
	OutputFile   string
	QuietMode    bool
	JSONOutput   bool

	// Compaction flags
	CompactionEnabled    bool
	CompactionThreshold  int
	CompactionPreserve   int
	CompactionMaxTokens  int
	CompactionSnapshot   bool
	CompactionAutoDetect bool

	// Reversible mode
	ReversibleEnabled bool

	// Version (set at build time)
	Version string = "dev"
)

// IsVerbose returns true if verbose mode is enabled.
func IsVerbose() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return Verbose > 0
}

// VerbosityLevel returns the verbosity level (0-3).
func VerbosityLevel() int {
	configMu.RLock()
	defer configMu.RUnlock()
	return Verbose
}

// IsUltraCompact returns true if ultra-compact mode is enabled.
func IsUltraCompact() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return UltraCompact
}

// IsSkipEnv returns true if environment sanitization is skipped.
func IsSkipEnv() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return SkipEnv
}

// IsDryRun returns true if dry-run mode is enabled.
func IsDryRun() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return DryRun
}

// GetQueryIntent returns the query intent from flag or environment.
func GetQueryIntent() string {
	configMu.RLock()
	intent := QueryIntent
	configMu.RUnlock()
	if intent != "" {
		return intent
	}
	return os.Getenv("TOKMAN_QUERY")
}

// IsLLMEnabled returns true if LLM compression is enabled.
func IsLLMEnabled() bool {
	configMu.RLock()
	enabled := LLMEnabled
	configMu.RUnlock()
	return enabled || os.Getenv("TOKMAN_LLM") == "true"
}

// GetTokenBudget returns the token budget from flag or environment.
func GetTokenBudget() int {
	configMu.RLock()
	budget := TokenBudget
	configMu.RUnlock()
	if budget > 0 {
		return budget
	}
	envBudget := os.Getenv("TOKMAN_BUDGET")
	if envBudget != "" {
		var b int
		if _, err := fmt.Sscanf(envBudget, "%d", &b); err == nil {
			return b
		}
	}
	return 0
}

// IsCompactionEnabled returns true if compaction is enabled.
func IsCompactionEnabled() bool {
	configMu.RLock()
	enabled := CompactionEnabled
	configMu.RUnlock()
	return enabled || os.Getenv("TOKMAN_COMPACTION") == "true"
}

// GetCompactionThreshold returns the compaction token threshold.
func GetCompactionThreshold() int {
	configMu.RLock()
	threshold := CompactionThreshold
	configMu.RUnlock()
	if threshold > 0 {
		return threshold
	}
	return 500
}

// GetCompactionPreserveTurns returns the number of recent turns to preserve.
func GetCompactionPreserveTurns() int {
	configMu.RLock()
	preserve := CompactionPreserve
	configMu.RUnlock()
	if preserve > 0 {
		return preserve
	}
	return 10
}

// GetCompactionMaxTokens returns the max summary tokens for compaction.
func GetCompactionMaxTokens() int {
	configMu.RLock()
	maxTokens := CompactionMaxTokens
	configMu.RUnlock()
	if maxTokens > 0 {
		return maxTokens
	}
	return 5000
}

// IsCompactionSnapshotEnabled returns true if state snapshot format is enabled.
func IsCompactionSnapshotEnabled() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return CompactionSnapshot
}

// IsCompactionAutoDetect returns true if auto-detection is enabled.
func IsCompactionAutoDetect() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return CompactionAutoDetect
}

// GetLayerPreset returns the layer preset from flag or environment.
func GetLayerPreset() string {
	configMu.RLock()
	preset := LayerPreset
	configMu.RUnlock()
	if preset != "" {
		return preset
	}
	return os.Getenv("TOKMAN_PRESET")
}

// GetOutputFile returns the output file path.
func GetOutputFile() string {
	configMu.RLock()
	defer configMu.RUnlock()
	return OutputFile
}

// IsQuietMode returns true if quiet mode is enabled.
func IsQuietMode() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return QuietMode
}

// IsJSONOutput returns true if JSON output is enabled.
func IsJSONOutput() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return JSONOutput
}

// IsReversibleEnabled returns true if reversible mode is enabled.
func IsReversibleEnabled() bool {
	configMu.RLock()
	enabled := ReversibleEnabled
	configMu.RUnlock()
	return enabled || os.Getenv("TOKMAN_REVERSIBLE") == "true"
}

// FlagConfig holds all flag values for atomic setting.
type FlagConfig struct {
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
}

// SetFlags sets all flag values atomically under a single lock.
func SetFlags(cfg FlagConfig) {
	configMu.Lock()
	Verbose = cfg.Verbose
	DryRun = cfg.DryRun
	UltraCompact = cfg.UltraCompact
	SkipEnv = cfg.SkipEnv
	QueryIntent = cfg.QueryIntent
	LLMEnabled = cfg.LLMEnabled
	TokenBudget = cfg.TokenBudget
	FallbackArgs = cfg.FallbackArgs
	LayerPreset = cfg.LayerPreset
	OutputFile = cfg.OutputFile
	QuietMode = cfg.QuietMode
	JSONOutput = cfg.JSONOutput
	CompactionEnabled = cfg.CompactionEnabled
	CompactionThreshold = cfg.CompactionThreshold
	CompactionPreserve = cfg.CompactionPreserve
	CompactionMaxTokens = cfg.CompactionMaxTokens
	CompactionSnapshot = cfg.CompactionSnapshot
	CompactionAutoDetect = cfg.CompactionAutoDetect
	ReversibleEnabled = cfg.ReversibleEnabled
	configMu.Unlock()
}
