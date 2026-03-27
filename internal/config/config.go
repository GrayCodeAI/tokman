package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/viper"
)

// Config represents the main configuration structure.
type Config struct {
	Tracking  TrackingConfig  `mapstructure:"tracking"`
	Filter    FilterConfig    `mapstructure:"filter"`
	Pipeline  PipelineConfig  `mapstructure:"pipeline"`
	Hooks     HooksConfig     `mapstructure:"hooks"`
	Dashboard DashboardConfig `mapstructure:"dashboard"`
	Alerts    AlertsConfig    `mapstructure:"alerts"`
	Export    ExportConfig    `mapstructure:"export"`
}

// PipelineConfig controls the 20-layer compression pipeline.
// Supports contexts up to 2M tokens with streaming processing.
// Based on 120+ research papers from top institutions worldwide.
type PipelineConfig struct {
	// Context limits
	MaxContextTokens int `mapstructure:"max_context_tokens"` // Max input context (default: 2M)
	ChunkSize        int `mapstructure:"chunk_size"`         // Processing chunk size for large inputs

	// Layer enable/disable
	EnableEntropy      bool `mapstructure:"enable_entropy"`
	EnablePerplexity   bool `mapstructure:"enable_perplexity"`
	EnableGoalDriven   bool `mapstructure:"enable_goal_driven"`
	EnableAST          bool `mapstructure:"enable_ast"`
	EnableContrastive  bool `mapstructure:"enable_contrastive"`
	EnableNgram        bool `mapstructure:"enable_ngram"`
	EnableEvaluator    bool `mapstructure:"enable_evaluator"`
	EnableGist         bool `mapstructure:"enable_gist"`
	EnableHierarchical bool `mapstructure:"enable_hierarchical"`
	EnableBudget       bool `mapstructure:"enable_budget"`

	// Layer thresholds (tunable)
	EntropyThreshold      float64 `mapstructure:"entropy_threshold"`       // Layer 1: 0.0-1.0 (default: 0.3)
	PerplexityThreshold   float64 `mapstructure:"perplexity_threshold"`    // Layer 2: 0.0-1.0 (default: 0.5)
	GoalDrivenThreshold   float64 `mapstructure:"goal_driven_threshold"`   // Layer 3: 0.0-1.0 (default: 0.4)
	ASTPreserveThreshold  float64 `mapstructure:"ast_preserve_threshold"`  // Layer 4: 0.0-1.0 (default: 0.6)
	ContrastiveThreshold  float64 `mapstructure:"contrastive_threshold"`   // Layer 5: 0.0-1.0 (default: 0.5)
	NgramMinOccurrences   int     `mapstructure:"ngram_min_occurrences"`   // Layer 6: min repeats (default: 3)
	EvaluatorThreshold    float64 `mapstructure:"evaluator_threshold"`     // Layer 7: 0.0-1.0 (default: 0.4)
	GistMinChunkSize      int     `mapstructure:"gist_min_chunk_size"`     // Layer 8: chars (default: 100)
	HierarchicalMaxLevels int     `mapstructure:"hierarchical_max_levels"` // Layer 9: depth (default: 3)
	HierarchicalRatio     float64 `mapstructure:"hierarchical_ratio"`      // Layer 9: 0.0-1.0 (default: 0.3)

	// Budget enforcement
	DefaultBudget      int    `mapstructure:"default_budget"`       // Default token budget (0 = unlimited)
	HardBudgetLimit    bool   `mapstructure:"hard_budget_limit"`    // Strict enforcement
	BudgetOverflowFile string `mapstructure:"budget_overflow_file"` // File to save overflow content

	// Resilience
	TeeOnFailure       bool   `mapstructure:"tee_on_failure"`       // Save raw output on failure
	TeeDir             string `mapstructure:"tee_dir"`              // Directory for tee files
	FailSafeMode       bool   `mapstructure:"failsafe_mode"`        // Return original on corruption
	ValidateOutput     bool   `mapstructure:"validate_output"`      // Check output validity
	ShortCircuitBudget bool   `mapstructure:"short_circuit_budget"` // Skip layers if budget met

	// Performance
	ParallelLayers  bool `mapstructure:"parallel_layers"`  // Run independent layers in parallel
	CacheEnabled    bool `mapstructure:"cache_enabled"`    // Cache compression results
	CacheMaxSize    int  `mapstructure:"cache_max_size"`   // Max cache entries
	StreamThreshold int  `mapstructure:"stream_threshold"` // Stream if input > N tokens

	// LLM Compaction (Layer 11) - Semantic compression
	EnableCompaction        bool   `mapstructure:"enable_compaction"`         // Enable LLM-based compaction
	CompactionThreshold     int    `mapstructure:"compaction_threshold"`      // Minimum tokens to trigger
	CompactionPreserveTurns int    `mapstructure:"compaction_preserve_turns"` // Recent turns to keep verbatim
	CompactionMaxTokens     int    `mapstructure:"compaction_max_tokens"`     // Max summary tokens
	CompactionStateSnapshot bool   `mapstructure:"compaction_state_snapshot"` // Use state snapshot format
	CompactionAutoDetect    bool   `mapstructure:"compaction_auto_detect"`    // Auto-detect conversation content
	LLMProvider             string `mapstructure:"llm_provider"`              // ollama, lmstudio, openai
	LLMModel                string `mapstructure:"llm_model"`                 // Model name
	LLMBaseURL              string `mapstructure:"llm_base_url"`              // API endpoint

	// Attribution Filter (Layer 12) - ProCut-style pruning
	EnableAttribution     bool    `mapstructure:"enable_attribution"`     // Enable attribution filtering
	AttributionThreshold  float64 `mapstructure:"attribution_threshold"`  // Importance threshold (0.0-1.0)
	AttributionPositional bool    `mapstructure:"attribution_positional"` // Use positional bias
	AttributionFrequency  bool    `mapstructure:"attribution_frequency"`  // Use frequency bias
	AttributionSemantic   bool    `mapstructure:"attribution_semantic"`   // Use semantic preservation

	// H2O Filter (Layer 13) - Heavy-Hitter Oracle
	EnableH2O          bool `mapstructure:"enable_h2o"`            // Enable H2O compression
	H2OSinkSize        int  `mapstructure:"h2o_sink_size"`         // Attention sink tokens to preserve
	H2ORecentSize      int  `mapstructure:"h2o_recent_size"`       // Recent tokens to preserve
	H2OHeavyHitterSize int  `mapstructure:"h2o_heavy_hitter_size"` // Heavy hitter tokens to preserve

	// Attention Sink Filter (Layer 14) - StreamingLLM-style
	EnableAttentionSink  bool `mapstructure:"enable_attention_sink"`  // Enable attention sink filtering
	AttentionSinkCount   int  `mapstructure:"attention_sink_count"`   // Initial tokens to preserve as sinks
	AttentionRecentCount int  `mapstructure:"attention_recent_count"` // Recent lines to preserve

	// Meta-Token Compression (Layer 15) - LZ77-style lossless compression
	EnableMetaToken   bool `mapstructure:"enable_meta_token"`    // Enable meta-token compression
	MetaTokenWindow   int  `mapstructure:"meta_token_window"`    // Sliding window size for token matching
	MetaTokenMinMatch int  `mapstructure:"meta_token_min_match"` // Minimum match length

	// Semantic Chunking (Layer 16) - Dynamic boundary detection
	EnableSemanticChunk bool    `mapstructure:"enable_semantic_chunk"` // Enable semantic chunking
	SemanticThreshold   float64 `mapstructure:"semantic_threshold"`    // Semantic shift threshold (0.0-1.0)
	ChunkMinSize        int     `mapstructure:"chunk_min_size"`        // Minimum chunk size in tokens
	ChunkMaxSize        int     `mapstructure:"chunk_max_size"`        // Maximum chunk size in tokens

	// Sketch Store (Layer 17) - Reversible compression with KVReviver
	EnableSketchStore bool `mapstructure:"enable_sketch_store"` // Enable sketch-based storage
	SketchMemoryRatio int  `mapstructure:"sketch_memory_ratio"` // Memory reduction ratio (default: 90%)
	SketchOnDemand    bool `mapstructure:"sketch_on_demand"`    // Reconstruct on-demand when needed

	// Lazy Pruner (Layer 18) - Budget-aware dynamic pruning
	EnableLazyPruner bool    `mapstructure:"enable_lazy_pruner"` // Enable lazy pruning
	LazyBudgetRatio  float64 `mapstructure:"lazy_budget_ratio"`  // Ratio of budget to use for lazy pruning
	LazyLayerDecay   float64 `mapstructure:"lazy_layer_decay"`   // Decay factor for layer-wise pruning

	// Semantic Anchor (Layer 19) - Attention gradient detection
	EnableSemanticAnchor bool    `mapstructure:"enable_semantic_anchor"` // Enable semantic anchor preservation
	AnchorThreshold      float64 `mapstructure:"anchor_threshold"`       // Gradient threshold for anchors
	AnchorMinPreserve    int     `mapstructure:"anchor_min_preserve"`    // Minimum anchors to preserve

	// Agent Memory (Layer 20) - Knowledge graph extraction
	EnableAgentMemory    bool   `mapstructure:"enable_agent_memory"`     // Enable agent memory extraction
	AgentMemoryMaxNodes  int    `mapstructure:"agent_memory_max_nodes"`  // Max nodes in knowledge graph
	AgentMemoryMaxEdges  int    `mapstructure:"agent_memory_max_edges"`  // Max edges in knowledge graph
	AgentMemoryExtractFn string `mapstructure:"agent_memory_extract_fn"` // Extraction function type
}

// CommandContext provides metadata about the command being executed.
// Used for intelligent filtering decisions.
type CommandContext struct {
	Command    string `mapstructure:"command"`    // "git", "npm", "cargo", etc.
	Subcommand string `mapstructure:"subcommand"` // "status", "test", "build"
	ExitCode   int    `mapstructure:"exit_code"`  // Non-zero = likely has errors
	Intent     string `mapstructure:"intent"`     // "debug", "review", "deploy", "search"
	IsTest     bool   `mapstructure:"is_test"`    // Test output detection
	IsBuild    bool   `mapstructure:"is_build"`   // Build output detection
	IsError    bool   `mapstructure:"is_error"`   // Error output detection
}

// TrackingConfig controls token tracking behavior.
type TrackingConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	DatabasePath string `mapstructure:"database_path"`
	Telemetry    bool   `mapstructure:"telemetry"`
}

// FilterConfig controls output filtering behavior.
type FilterConfig struct {
	NoiseDirs   []string `mapstructure:"noise_dirs"`
	IgnoreFiles []string `mapstructure:"ignore_files"` // File patterns to ignore (e.g., "*.lock", "*.min.js")
	Mode        string   `mapstructure:"mode"`         // "minimal" or "aggressive"
	MaxWidth    int      `mapstructure:"max_width"`    // Max display width (0 = auto)
}

// HooksConfig controls shell hook behavior.
type HooksConfig struct {
	ExcludedCommands []string `mapstructure:"excluded_commands"`
	AuditDir         string   `mapstructure:"audit_dir"` // Directory for hook audit logs
	TeeDir           string   `mapstructure:"tee_dir"`   // Directory for failure tee logs
}

// DashboardConfig controls dashboard behavior.
type DashboardConfig struct {
	Port           int    `mapstructure:"port"`
	Bind           string `mapstructure:"bind"`
	UpdateInterval int    `mapstructure:"update_interval"`
	Theme          string `mapstructure:"theme"`
	EnableExport   bool   `mapstructure:"enable_export"`
}

// AlertsConfig controls alert thresholds.
type AlertsConfig struct {
	Enabled             bool    `mapstructure:"enabled"`
	DailyTokenLimit     int64   `mapstructure:"daily_token_limit"`
	WeeklyTokenLimit    int64   `mapstructure:"weekly_token_limit"`
	UsageSpikeThreshold float64 `mapstructure:"usage_spike_threshold"`
}

// ExportConfig controls export behavior.
type ExportConfig struct {
	DefaultFormat    string `mapstructure:"default_format"`
	IncludeTimestamp bool   `mapstructure:"include_timestamps"`
	MaxRecords       int    `mapstructure:"max_records"`
}

// Defaults returns the default configuration.
func Defaults() *Config {
	return &Config{
		Tracking: TrackingConfig{
			Enabled:      true,
			DatabasePath: "",
			Telemetry:    false,
		},
		Filter: FilterConfig{
			NoiseDirs: []string{
				// Version control
				".git",
				// Dependencies
				"node_modules",
				"vendor",
				// Build outputs
				"target",
				"dist",
				"build",
				".next",
				".turbo",
				".vercel",
				".output",
				// Python
				"__pycache__",
				".venv",
				"venv",
				".pytest_cache",
				".mypy_cache",
				".tox",
				".eggs",
				// IDE/Editor
				".idea",
				".vscode",
				".vs",
				// JS/TS
				"coverage",
				".cache",
				".nyc_output",
				// Framework-specific
				".svelte-kit",
				".angular",
				".parcel-cache",
				// OS files
				".DS_Store",
				"Thumbs.db",
				// Misc
				".data",
			},
			IgnoreFiles: []string{
				"*.lock",
				"*.min.js",
				"*.min.css",
				"*.map",
				"package-lock.json",
				"yarn.lock",
				"pnpm-lock.yaml",
				"Cargo.lock",
				"poetry.lock",
			},
			Mode:     "minimal",
			MaxWidth: 0,
		},
		Pipeline: PipelineConfig{
			// Context limits - supports up to 2M tokens
			MaxContextTokens: 2000000, // 2M tokens max
			ChunkSize:        100000,  // 100K tokens per chunk

			// All layers enabled by default
			EnableEntropy:      true,
			EnablePerplexity:   true,
			EnableGoalDriven:   true,
			EnableAST:          true,
			EnableContrastive:  true,
			EnableNgram:        true,
			EnableEvaluator:    true,
			EnableGist:         true,
			EnableHierarchical: true,
			EnableBudget:       true,

			// Layer thresholds (research-backed defaults)
			EntropyThreshold:      0.3,
			PerplexityThreshold:   0.5,
			GoalDrivenThreshold:   0.4,
			ASTPreserveThreshold:  0.6,
			ContrastiveThreshold:  0.5,
			NgramMinOccurrences:   3,
			EvaluatorThreshold:    0.4,
			GistMinChunkSize:      100,
			HierarchicalMaxLevels: 3,
			HierarchicalRatio:     0.3,

			// Budget
			DefaultBudget:   0,    // Unlimited by default
			HardBudgetLimit: true, // Strict enforcement when budget set

			// Resilience
			TeeOnFailure:       true,
			FailSafeMode:       true,
			ValidateOutput:     true,
			ShortCircuitBudget: true,

			// Performance
			ParallelLayers:  false, // Sequential for deterministic output
			CacheEnabled:    true,
			CacheMaxSize:    1000,
			StreamThreshold: 500000, // Stream if > 500K tokens

			// Layer 11: Compaction (enabled by default for automatic chat compression)
			EnableCompaction:        true,
			CompactionThreshold:     500,  // Trigger early for better compression
			CompactionPreserveTurns: 10,   // Keep more recent turns
			CompactionMaxTokens:     5000, // Larger summaries for big contexts
			CompactionStateSnapshot: true,
			CompactionAutoDetect:    true,

			// Layer 12: Attribution (ProCut-style pruning)
			EnableAttribution:     true,
			AttributionThreshold:  0.25, // Lower threshold = more pruning
			AttributionPositional: true, // Preserve start/end content
			AttributionFrequency:  true, // Reduce repeated content
			AttributionSemantic:   true, // Preserve keywords, numbers, code

			// Layer 13: H2O (Heavy-Hitter Oracle)
			EnableH2O:          true,
			H2OSinkSize:        4,  // First 4 tokens are attention sinks
			H2ORecentSize:      20, // Keep last 20 tokens
			H2OHeavyHitterSize: 40, // Top 40 heavy hitters

			// Layer 14: Attention Sink (StreamingLLM-style)
			EnableAttentionSink:  true,
			AttentionSinkCount:   4, // First 4 lines are attention sinks
			AttentionRecentCount: 8, // Keep last 8 lines in rolling cache

			// Layer 15: Meta-Token Compression (LZ77-style lossless)
			EnableMetaToken:   true,
			MetaTokenWindow:   512, // Sliding window for token matching
			MetaTokenMinMatch: 3,   // Minimum match length

			// Layer 16: Semantic Chunking (Dynamic boundaries)
			EnableSemanticChunk: true,
			SemanticThreshold:   0.5, // Semantic shift threshold
			ChunkMinSize:        50,  // Minimum chunk size
			ChunkMaxSize:        500, // Maximum chunk size

			// Layer 17: Sketch Store (KVReviver-style)
			EnableSketchStore: true,
			SketchMemoryRatio: 90,   // 90% memory reduction
			SketchOnDemand:    true, // Reconstruct when needed

			// Layer 18: Lazy Pruner (LazyLLM-style)
			EnableLazyPruner: true,
			LazyBudgetRatio:  0.3, // Use 30% of budget for lazy pruning
			LazyLayerDecay:   0.9, // Decay factor per layer

			// Layer 19: Semantic Anchor (Attention gradient)
			EnableSemanticAnchor: true,
			AnchorThreshold:      0.4, // Gradient threshold
			AnchorMinPreserve:    5,   // Minimum anchors

			// Layer 20: Agent Memory (Knowledge graph)
			EnableAgentMemory:    true,
			AgentMemoryMaxNodes:  100,       // Max graph nodes
			AgentMemoryMaxEdges:  200,       // Max graph edges
			AgentMemoryExtractFn: "default", // Extraction function
		},
		Hooks: HooksConfig{
			ExcludedCommands: []string{},
			AuditDir:         "",
			TeeDir:           "",
		},
		Dashboard: DashboardConfig{
			Port:           8080,
			Bind:           "localhost",
			UpdateInterval: 30000,
			Theme:          "dark",
			EnableExport:   true,
		},
		Alerts: AlertsConfig{
			Enabled:             true,
			DailyTokenLimit:     1000000,
			WeeklyTokenLimit:    5000000,
			UsageSpikeThreshold: 2.0,
		},
		Export: ExportConfig{
			DefaultFormat:    "json",
			IncludeTimestamp: true,
			MaxRecords:       0,
		},
	}
}

// Load reads configuration from file and environment.
func Load(cfgFile string) (*Config, error) {
	cfg := Defaults()

	// Set up viper
	viper.SetConfigType("toml")

	if cfgFile != "" {
		viper.SetConfigFile(os.ExpandEnv(cfgFile))
	} else {
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "tokman"))
		}
		viper.SetConfigName("config")
	}

	// Environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TOKMAN")

	// Environment variable aliases (for compatibility)
	if val := os.Getenv("TOKMAN_DB_PATH"); val != "" {
		viper.Set("tracking.database_path", val)
	}
	if val := os.Getenv("TOKMAN_TELEMETRY_DISABLED"); val != "" {
		viper.Set("tracking.telemetry", val == "false")
	}
	if val := os.Getenv("TOKMAN_AUDIT_DIR"); val != "" {
		viper.Set("hooks.audit_dir", val)
	}
	if val := os.Getenv("TOKMAN_TEE_DIR"); val != "" {
		viper.Set("hooks.tee_dir", val)
	}
	if val := os.Getenv("TOKMAN_TEE"); val != "" {
		viper.Set("hooks.tee_enabled", val == "true" || val == "1")
	}
	if val := os.Getenv("TOKMAN_HOOK_AUDIT"); val != "" {
		viper.Set("hooks.audit_enabled", val == "true" || val == "1")
	}

	// T163: Pipeline environment variable overrides
	if val := os.Getenv("TOKMAN_BUDGET"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			viper.Set("pipeline.default_budget", n)
		}
	}
	if val := os.Getenv("TOKMAN_MODE"); val != "" {
		viper.Set("filter.mode", val)
	}
	if val := os.Getenv("TOKMAN_PRESET"); val != "" {
		viper.Set("pipeline.preset", val)
	}
	if val := os.Getenv("TOKMAN_MAX_CONTEXT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			viper.Set("pipeline.max_context_tokens", n)
		}
	}
	if val := os.Getenv("TOKMAN_CACHE_SIZE"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			viper.Set("pipeline.cache_max_size", n)
		}
	}
	if val := os.Getenv("TOKMAN_ENTROPY_THRESHOLD"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			viper.Set("pipeline.entropy_threshold", f)
		}
	}
	if val := os.Getenv("TOKMAN_COMPACTION"); val != "" {
		viper.Set("pipeline.enable_compaction", val == "true" || val == "1")
	}
	if val := os.Getenv("TOKMAN_H2O"); val != "" {
		viper.Set("pipeline.enable_h2o", val == "true" || val == "1")
	}
	if val := os.Getenv("TOKMAN_ATTENTION_SINK"); val != "" {
		viper.Set("pipeline.enable_attention_sink", val == "true" || val == "1")
	}

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found, use defaults
		return cfg, nil
	}

	// Unmarshal into config struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// Validate configuration values
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// LoadFromFile reads a TOML configuration file directly.
func LoadFromFile(path string) (*Config, error) {
	cfg := Defaults()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// Save writes the configuration to a TOML file.
func (c *Config) Save(path string) (retErr error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); retErr == nil {
			retErr = cerr
		}
	}()

	return toml.NewEncoder(f).Encode(c)
}

// GetDatabasePath returns the effective database path.
func (c *Config) GetDatabasePath() string {
	if c.Tracking.DatabasePath != "" {
		return c.Tracking.DatabasePath
	}
	return DatabasePath()
}

// Validate checks configuration values for correctness and applies corrections.
func (c *Config) Validate() error {
	var errs []string

	// Pipeline thresholds must be 0.0-1.0
	validateThreshold := func(name string, val float64) {
		if val < 0.0 || val > 1.0 {
			errs = append(errs, fmt.Sprintf("%s must be between 0.0 and 1.0, got %.2f", name, val))
		}
	}
	validateThreshold("entropy_threshold", c.Pipeline.EntropyThreshold)
	validateThreshold("perplexity_threshold", c.Pipeline.PerplexityThreshold)
	validateThreshold("goal_driven_threshold", c.Pipeline.GoalDrivenThreshold)
	validateThreshold("ast_preserve_threshold", c.Pipeline.ASTPreserveThreshold)
	validateThreshold("contrastive_threshold", c.Pipeline.ContrastiveThreshold)
	validateThreshold("evaluator_threshold", c.Pipeline.EvaluatorThreshold)
	validateThreshold("hierarchical_ratio", c.Pipeline.HierarchicalRatio)
	validateThreshold("attribution_threshold", c.Pipeline.AttributionThreshold)
	validateThreshold("semantic_threshold", c.Pipeline.SemanticThreshold)
	validateThreshold("lazy_budget_ratio", c.Pipeline.LazyBudgetRatio)
	validateThreshold("lazy_layer_decay", c.Pipeline.LazyLayerDecay)
	validateThreshold("anchor_threshold", c.Pipeline.AnchorThreshold)

	// Positive integer constraints
	if c.Pipeline.MaxContextTokens < 0 {
		errs = append(errs, "max_context_tokens must be non-negative")
	}
	if c.Pipeline.ChunkSize < 0 {
		errs = append(errs, "chunk_size must be non-negative")
	}
	if c.Pipeline.CacheMaxSize < 0 {
		errs = append(errs, "cache_max_size must be non-negative")
	}
	if c.Pipeline.NgramMinOccurrences < 0 {
		errs = append(errs, "ngram_min_occurrences must be non-negative")
	}
	if c.Pipeline.HierarchicalMaxLevels < 0 {
		errs = append(errs, "hierarchical_max_levels must be non-negative")
	}
	if c.Pipeline.DefaultBudget < 0 {
		errs = append(errs, "default_budget must be non-negative")
	}

	// Filter mode must be valid
	if c.Filter.Mode != "" && c.Filter.Mode != "minimal" && c.Filter.Mode != "aggressive" {
		errs = append(errs, fmt.Sprintf("filter.mode must be 'minimal' or 'aggressive', got '%s'", c.Filter.Mode))
	}

	// Dashboard port must be valid
	if c.Dashboard.Port < 0 || c.Dashboard.Port > 65535 {
		errs = append(errs, fmt.Sprintf("dashboard.port must be between 0 and 65535, got %d", c.Dashboard.Port))
	}

	// Alert limits must be non-negative
	if c.Alerts.DailyTokenLimit < 0 {
		errs = append(errs, "alerts.daily_token_limit must be non-negative")
	}
	if c.Alerts.WeeklyTokenLimit < 0 {
		errs = append(errs, "alerts.weekly_token_limit must be non-negative")
	}
	if c.Alerts.UsageSpikeThreshold < 0 {
		errs = append(errs, "alerts.usage_spike_threshold must be non-negative")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
