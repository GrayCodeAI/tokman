package config

import (
	"os"
	"path/filepath"
)

// HierarchicalConfig supports system → user → project → command-specific overrides.
// R45: Config precedence follows standard CLI tool patterns.
type HierarchicalConfig struct {
	SystemConfig  *Config // /etc/tokman/config.toml
	UserConfig    *Config // ~/.config/tokman/config.toml
	ProjectConfig *Config // .tokman.toml in project root
	CommandConfig *Config // per-command overrides
}

// LoadHierarchical loads config with proper precedence.
// Priority: command > project > user > system > defaults
func LoadHierarchical() (*HierarchicalConfig, error) {
	hc := &HierarchicalConfig{}

	// 1. System config (lowest priority)
	systemPaths := []string{
		"/etc/tokman/config.toml",
		"/usr/local/etc/tokman/config.toml",
	}
	for _, p := range systemPaths {
		if cfg, err := LoadFromFile(p); err == nil {
			hc.SystemConfig = cfg
			break
		}
	}

	// 2. User config
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	if home != "" {
		userPaths := []string{
			filepath.Join(home, ".config", "tokman", "config.toml"),
			filepath.Join(home, ".tokman.toml"),
		}
		for _, p := range userPaths {
			if cfg, err := LoadFromFile(p); err == nil {
				hc.UserConfig = cfg
				break
			}
		}
	}

	// 3. Project config (.tokman.toml in current or parent dirs)
	cwd, _ := os.Getwd()
	if cwd != "" {
		projectConfig := findProjectConfig(cwd)
		if projectConfig != "" {
			if cfg, err := LoadFromFile(projectConfig); err == nil {
				hc.ProjectConfig = cfg
			}
		}
	}

	// 4. Merge with defaults (command config = defaults)
	hc.CommandConfig = Defaults()

	return hc, nil
}

// Merged returns the final merged configuration.
// Later configs override earlier ones.
func (hc *HierarchicalConfig) Merged() *Config {
	result := Defaults()

	// Apply in reverse priority order (lowest first)
	if hc.SystemConfig != nil {
		mergeConfig(result, hc.SystemConfig)
	}
	if hc.UserConfig != nil {
		mergeConfig(result, hc.UserConfig)
	}
	if hc.ProjectConfig != nil {
		mergeConfig(result, hc.ProjectConfig)
	}
	if hc.CommandConfig != nil {
		mergeConfig(result, hc.CommandConfig)
	}

	return result
}

// findProjectConfig searches for .tokman.toml in current and parent directories.
func findProjectConfig(startDir string) string {
	dir := startDir
	for {
		configPath := filepath.Join(dir, ".tokman.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached filesystem root
		}
		dir = parent
	}
	return ""
}

// mergeConfig merges src into dst (non-zero values override).
func mergeConfig(dst, src *Config) {
	if src.Pipeline.MaxContextTokens > 0 {
		dst.Pipeline.MaxContextTokens = src.Pipeline.MaxContextTokens
	}
	if src.Pipeline.ChunkSize > 0 {
		dst.Pipeline.ChunkSize = src.Pipeline.ChunkSize
	}
	if src.Pipeline.DefaultBudget > 0 {
		dst.Pipeline.DefaultBudget = src.Pipeline.DefaultBudget
	}
	if src.Pipeline.EntropyThreshold > 0 {
		dst.Pipeline.EntropyThreshold = src.Pipeline.EntropyThreshold
	}
	if src.Pipeline.PerplexityThreshold > 0 {
		dst.Pipeline.PerplexityThreshold = src.Pipeline.PerplexityThreshold
	}
	if src.Pipeline.GoalDrivenThreshold > 0 {
		dst.Pipeline.GoalDrivenThreshold = src.Pipeline.GoalDrivenThreshold
	}
	if src.Pipeline.ASTPreserveThreshold > 0 {
		dst.Pipeline.ASTPreserveThreshold = src.Pipeline.ASTPreserveThreshold
	}
	if src.Pipeline.ContrastiveThreshold > 0 {
		dst.Pipeline.ContrastiveThreshold = src.Pipeline.ContrastiveThreshold
	}
	if src.Pipeline.NgramMinOccurrences > 0 {
		dst.Pipeline.NgramMinOccurrences = src.Pipeline.NgramMinOccurrences
	}
	if src.Pipeline.EvaluatorThreshold > 0 {
		dst.Pipeline.EvaluatorThreshold = src.Pipeline.EvaluatorThreshold
	}
	if src.Pipeline.GistMinChunkSize > 0 {
		dst.Pipeline.GistMinChunkSize = src.Pipeline.GistMinChunkSize
	}
	if src.Pipeline.HierarchicalMaxLevels > 0 {
		dst.Pipeline.HierarchicalMaxLevels = src.Pipeline.HierarchicalMaxLevels
	}
	if src.Pipeline.HierarchicalRatio > 0 {
		dst.Pipeline.HierarchicalRatio = src.Pipeline.HierarchicalRatio
	}
	if src.Pipeline.CompactionThreshold > 0 {
		dst.Pipeline.CompactionThreshold = src.Pipeline.CompactionThreshold
	}
	if src.Pipeline.CompactionPreserveTurns > 0 {
		dst.Pipeline.CompactionPreserveTurns = src.Pipeline.CompactionPreserveTurns
	}
	if src.Pipeline.CompactionMaxTokens > 0 {
		dst.Pipeline.CompactionMaxTokens = src.Pipeline.CompactionMaxTokens
	}
	if src.Pipeline.AttributionThreshold > 0 {
		dst.Pipeline.AttributionThreshold = src.Pipeline.AttributionThreshold
	}
	if src.Pipeline.H2OSinkSize > 0 {
		dst.Pipeline.H2OSinkSize = src.Pipeline.H2OSinkSize
	}
	if src.Pipeline.H2ORecentSize > 0 {
		dst.Pipeline.H2ORecentSize = src.Pipeline.H2ORecentSize
	}
	if src.Pipeline.H2OHeavyHitterSize > 0 {
		dst.Pipeline.H2OHeavyHitterSize = src.Pipeline.H2OHeavyHitterSize
	}
	if src.Pipeline.AttentionSinkCount > 0 {
		dst.Pipeline.AttentionSinkCount = src.Pipeline.AttentionSinkCount
	}
	if src.Pipeline.AttentionRecentCount > 0 {
		dst.Pipeline.AttentionRecentCount = src.Pipeline.AttentionRecentCount
	}
	if src.Pipeline.CacheMaxSize > 0 {
		dst.Pipeline.CacheMaxSize = src.Pipeline.CacheMaxSize
	}
	if src.Pipeline.StreamThreshold > 0 {
		dst.Pipeline.StreamThreshold = src.Pipeline.StreamThreshold
	}
	if src.Pipeline.LLMProvider != "" {
		dst.Pipeline.LLMProvider = src.Pipeline.LLMProvider
	}
	if src.Pipeline.LLMModel != "" {
		dst.Pipeline.LLMModel = src.Pipeline.LLMModel
	}
	if src.Pipeline.LLMBaseURL != "" {
		dst.Pipeline.LLMBaseURL = src.Pipeline.LLMBaseURL
	}
	if src.Pipeline.TeeDir != "" {
		dst.Pipeline.TeeDir = src.Pipeline.TeeDir
	}
	if src.Pipeline.BudgetOverflowFile != "" {
		dst.Pipeline.BudgetOverflowFile = src.Pipeline.BudgetOverflowFile
	}
	// Boolean layer toggles - unconditionally propagate src values so that explicit
	// false (disable) is honoured, not silently ignored.
	dst.Pipeline.EnableEntropy = src.Pipeline.EnableEntropy
	dst.Pipeline.EnablePerplexity = src.Pipeline.EnablePerplexity
	dst.Pipeline.EnableGoalDriven = src.Pipeline.EnableGoalDriven
	dst.Pipeline.EnableAST = src.Pipeline.EnableAST
	dst.Pipeline.EnableContrastive = src.Pipeline.EnableContrastive
	dst.Pipeline.EnableNgram = src.Pipeline.EnableNgram
	dst.Pipeline.EnableEvaluator = src.Pipeline.EnableEvaluator
	dst.Pipeline.EnableGist = src.Pipeline.EnableGist
	dst.Pipeline.EnableHierarchical = src.Pipeline.EnableHierarchical
	dst.Pipeline.EnableBudget = src.Pipeline.EnableBudget
	dst.Pipeline.EnableCompaction = src.Pipeline.EnableCompaction
	dst.Pipeline.EnableAttribution = src.Pipeline.EnableAttribution
	dst.Pipeline.EnableH2O = src.Pipeline.EnableH2O
	dst.Pipeline.EnableAttentionSink = src.Pipeline.EnableAttentionSink
	dst.Pipeline.HardBudgetLimit = src.Pipeline.HardBudgetLimit
	dst.Pipeline.TeeOnFailure = src.Pipeline.TeeOnFailure
	dst.Pipeline.FailSafeMode = src.Pipeline.FailSafeMode
	dst.Pipeline.ValidateOutput = src.Pipeline.ValidateOutput
	dst.Pipeline.ShortCircuitBudget = src.Pipeline.ShortCircuitBudget
	dst.Pipeline.ParallelLayers = src.Pipeline.ParallelLayers
	dst.Pipeline.CacheEnabled = src.Pipeline.CacheEnabled
	dst.Pipeline.CompactionStateSnapshot = src.Pipeline.CompactionStateSnapshot
	dst.Pipeline.CompactionAutoDetect = src.Pipeline.CompactionAutoDetect
	dst.Pipeline.AttributionPositional = src.Pipeline.AttributionPositional
	dst.Pipeline.AttributionFrequency = src.Pipeline.AttributionFrequency
	dst.Pipeline.AttributionSemantic = src.Pipeline.AttributionSemantic

	if src.Filter.Mode != "" {
		dst.Filter.Mode = src.Filter.Mode
	}
	if src.Filter.MaxWidth > 0 {
		dst.Filter.MaxWidth = src.Filter.MaxWidth
	}
	if len(src.Filter.NoiseDirs) > 0 {
		dst.Filter.NoiseDirs = src.Filter.NoiseDirs
	}
	if len(src.Filter.IgnoreFiles) > 0 {
		dst.Filter.IgnoreFiles = src.Filter.IgnoreFiles
	}

	if src.Tracking.DatabasePath != "" {
		dst.Tracking.DatabasePath = src.Tracking.DatabasePath
	}
	if src.Hooks.AuditDir != "" {
		dst.Hooks.AuditDir = src.Hooks.AuditDir
	}
	if src.Hooks.TeeDir != "" {
		dst.Hooks.TeeDir = src.Hooks.TeeDir
	}
	if src.Dashboard.Port > 0 {
		dst.Dashboard.Port = src.Dashboard.Port
	}
}
