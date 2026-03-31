package shared

import (
	"fmt"
	"os"
	"sync"
)

// AppState encapsulates all CLI flag state in a single struct.
// This replaces the global variable pattern and enables:
// - Testability (pass different state to different tests)
// - Concurrency (multiple commands with different configs)
// - Dependency injection (pass state explicitly)
type AppState struct {
	mu sync.RWMutex

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

	// Remote mode flags (Phase 4)
	RemoteMode      bool
	CompressionAddr string
	AnalyticsAddr   string
	RemoteTimeout   int // seconds

	// Compaction flags
	CompactionEnabled    bool
	CompactionThreshold  int
	CompactionPreserve   int
	CompactionMaxTokens  int
	CompactionSnapshot   bool
	CompactionAutoDetect bool

	// Reversible mode
	ReversibleEnabled bool

	// Custom layer configuration
	EnableLayers  []string
	DisableLayers []string
	StreamMode    bool
}

// Version is set at build time.
var Version string = "dev"

// Global instance for backward compatibility.
// New code should use explicit AppState instances.
var globalState = &AppState{}

// Global returns the global AppState instance.
// Deprecated: Pass AppState explicitly where possible.
func Global() *AppState {
	return globalState
}

// SetFlags sets all flag values atomically on the global state.
func SetFlags(cfg FlagConfig) {
	globalState.Set(cfg)
	globalState.syncGlobals()
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
	RemoteMode           bool
	CompressionAddr      string
	AnalyticsAddr        string
	RemoteTimeout        int
	CompactionEnabled    bool
	CompactionThreshold  int
	CompactionPreserve   int
	CompactionMaxTokens  int
	CompactionSnapshot   bool
	CompactionAutoDetect bool
	ReversibleEnabled    bool
	EnableLayers         []string
	DisableLayers        []string
	StreamMode           bool
}

// Set sets all flag values atomically.
func (s *AppState) Set(cfg FlagConfig) {
	s.mu.Lock()
	s.Verbose = cfg.Verbose
	s.DryRun = cfg.DryRun
	s.UltraCompact = cfg.UltraCompact
	s.SkipEnv = cfg.SkipEnv
	s.QueryIntent = cfg.QueryIntent
	s.LLMEnabled = cfg.LLMEnabled
	s.TokenBudget = cfg.TokenBudget
	s.FallbackArgs = cfg.FallbackArgs
	s.LayerPreset = cfg.LayerPreset
	s.OutputFile = cfg.OutputFile
	s.QuietMode = cfg.QuietMode
	s.JSONOutput = cfg.JSONOutput
	s.RemoteMode = cfg.RemoteMode
	s.CompressionAddr = cfg.CompressionAddr
	s.AnalyticsAddr = cfg.AnalyticsAddr
	s.RemoteTimeout = cfg.RemoteTimeout
	s.CompactionEnabled = cfg.CompactionEnabled
	s.CompactionThreshold = cfg.CompactionThreshold
	s.CompactionPreserve = cfg.CompactionPreserve
	s.CompactionMaxTokens = cfg.CompactionMaxTokens
	s.CompactionSnapshot = cfg.CompactionSnapshot
	s.CompactionAutoDetect = cfg.CompactionAutoDetect
	s.ReversibleEnabled = cfg.ReversibleEnabled
	s.EnableLayers = cfg.EnableLayers
	s.DisableLayers = cfg.DisableLayers
	s.StreamMode = cfg.StreamMode
	s.mu.Unlock()
}

// IsVerbose returns true if verbose mode is enabled.
func (s *AppState) IsVerbose() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Verbose > 0
}

// IsUltraCompact returns true if ultra-compact mode is enabled.
func (s *AppState) IsUltraCompact() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.UltraCompact
}

// GetQueryIntent returns the query intent from flag or environment.
func (s *AppState) GetQueryIntent() string {
	s.mu.RLock()
	intent := s.QueryIntent
	s.mu.RUnlock()
	if intent != "" {
		return intent
	}
	return os.Getenv("TOKMAN_QUERY")
}

// IsLLMEnabled returns true if LLM compression is enabled.
func (s *AppState) IsLLMEnabled() bool {
	s.mu.RLock()
	enabled := s.LLMEnabled
	s.mu.RUnlock()
	return enabled || os.Getenv("TOKMAN_LLM") == "true"
}

// GetTokenBudget returns the token budget from flag or environment.
func (s *AppState) GetTokenBudget() int {
	s.mu.RLock()
	budget := s.TokenBudget
	s.mu.RUnlock()
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

// GetLayerPreset returns the layer preset from flag or environment.
func (s *AppState) GetLayerPreset() string {
	s.mu.RLock()
	preset := s.LayerPreset
	s.mu.RUnlock()
	if preset != "" {
		return preset
	}
	return os.Getenv("TOKMAN_PRESET")
}

// IsQuietMode returns true if quiet mode is enabled.
func (s *AppState) IsQuietMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.QuietMode
}

// IsReversibleEnabled returns true if reversible mode is enabled.
func (s *AppState) IsReversibleEnabled() bool {
	s.mu.RLock()
	enabled := s.ReversibleEnabled
	s.mu.RUnlock()
	return enabled || os.Getenv("TOKMAN_REVERSIBLE") == "true"
}

// IsRemoteMode returns true if remote mode is enabled.
func (s *AppState) IsRemoteMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RemoteMode || os.Getenv("TOKMAN_REMOTE") == "true"
}

// GetCompressionAddr returns the compression service address.
func (s *AppState) GetCompressionAddr() string {
	s.mu.RLock()
	addr := s.CompressionAddr
	s.mu.RUnlock()
	if addr != "" {
		return addr
	}
	return os.Getenv("TOKMAN_COMPRESSION_ADDR")
}

// GetAnalyticsAddr returns the analytics service address.
func (s *AppState) GetAnalyticsAddr() string {
	s.mu.RLock()
	addr := s.AnalyticsAddr
	s.mu.RUnlock()
	if addr != "" {
		return addr
	}
	return os.Getenv("TOKMAN_ANALYTICS_ADDR")
}

// GetRemoteTimeout returns the remote operation timeout in seconds.
func (s *AppState) GetRemoteTimeout() int {
	s.mu.RLock()
	timeout := s.RemoteTimeout
	s.mu.RUnlock()
	if timeout > 0 {
		return timeout
	}
	return 30
}

// GetEnableLayers returns layers to explicitly enable.
func (s *AppState) GetEnableLayers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.EnableLayers
}

// GetDisableLayers returns layers to explicitly disable.
func (s *AppState) GetDisableLayers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DisableLayers
}

// IsStreamMode returns true if streaming mode is enabled for large inputs.
func (s *AppState) IsStreamMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.StreamMode
}

// Global accessor functions for backward compatibility.
// These delegate to the global AppState instance and also sync package-level globals.

var (
	CfgFile              string
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
	RemoteMode           bool
	CompressionAddr      string
	AnalyticsAddr        string
	RemoteTimeout        int
	CompactionEnabled    bool
	CompactionThreshold  int
	CompactionPreserve   int
	CompactionMaxTokens  int
	CompactionSnapshot   bool
	CompactionAutoDetect bool
	ReversibleEnabled    bool
	EnableLayers         []string
	DisableLayers        []string
	StreamMode           bool
)

// syncGlobals copies AppState fields to package-level globals.
func (s *AppState) syncGlobals() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	Verbose = s.Verbose
	DryRun = s.DryRun
	UltraCompact = s.UltraCompact
	SkipEnv = s.SkipEnv
	QueryIntent = s.QueryIntent
	LLMEnabled = s.LLMEnabled
	TokenBudget = s.TokenBudget
	FallbackArgs = s.FallbackArgs
	LayerPreset = s.LayerPreset
	OutputFile = s.OutputFile
	QuietMode = s.QuietMode
	JSONOutput = s.JSONOutput
	RemoteMode = s.RemoteMode
	CompressionAddr = s.CompressionAddr
	AnalyticsAddr = s.AnalyticsAddr
	RemoteTimeout = s.RemoteTimeout
	CompactionEnabled = s.CompactionEnabled
	CompactionThreshold = s.CompactionThreshold
	CompactionPreserve = s.CompactionPreserve
	CompactionMaxTokens = s.CompactionMaxTokens
	CompactionSnapshot = s.CompactionSnapshot
	CompactionAutoDetect = s.CompactionAutoDetect
	ReversibleEnabled = s.ReversibleEnabled
	EnableLayers = s.EnableLayers
	DisableLayers = s.DisableLayers
	StreamMode = s.StreamMode
}

// syncFromGlobals copies package-level globals into AppState.
func (s *AppState) syncFromGlobals() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Verbose = Verbose
	s.DryRun = DryRun
	s.UltraCompact = UltraCompact
	s.SkipEnv = SkipEnv
	s.QueryIntent = QueryIntent
	s.LLMEnabled = LLMEnabled
	s.TokenBudget = TokenBudget
	s.FallbackArgs = FallbackArgs
	s.LayerPreset = LayerPreset
	s.OutputFile = OutputFile
	s.QuietMode = QuietMode
	s.JSONOutput = JSONOutput
	s.RemoteMode = RemoteMode
	s.CompressionAddr = CompressionAddr
	s.AnalyticsAddr = AnalyticsAddr
	s.RemoteTimeout = RemoteTimeout
	s.CompactionEnabled = CompactionEnabled
	s.CompactionThreshold = CompactionThreshold
	s.CompactionPreserve = CompactionPreserve
	s.CompactionMaxTokens = CompactionMaxTokens
	s.CompactionSnapshot = CompactionSnapshot
	s.CompactionAutoDetect = CompactionAutoDetect
	s.ReversibleEnabled = ReversibleEnabled
	s.EnableLayers = EnableLayers
	s.DisableLayers = DisableLayers
	s.StreamMode = StreamMode
}

// IsVerbose returns true if verbose mode is enabled.
func IsVerbose() bool {
	globalState.syncFromGlobals()
	return globalState.IsVerbose()
}

// IsUltraCompact returns true if ultra-compact mode is enabled.
func IsUltraCompact() bool {
	globalState.syncFromGlobals()
	return globalState.IsUltraCompact()
}

// GetQueryIntent returns the query intent from flag or environment.
func GetQueryIntent() string {
	globalState.syncFromGlobals()
	return globalState.GetQueryIntent()
}

// IsLLMEnabled returns true if LLM compression is enabled.
func IsLLMEnabled() bool {
	globalState.syncFromGlobals()
	return globalState.IsLLMEnabled()
}

// GetTokenBudget returns the token budget from flag or environment.
func GetTokenBudget() int {
	globalState.syncFromGlobals()
	return globalState.GetTokenBudget()
}

// GetLayerPreset returns the layer preset from flag or environment.
func GetLayerPreset() string {
	globalState.syncFromGlobals()
	return globalState.GetLayerPreset()
}

// IsQuietMode returns true if quiet mode is enabled.
func IsQuietMode() bool {
	globalState.syncFromGlobals()
	return globalState.IsQuietMode()
}

// IsReversibleEnabled returns true if reversible mode is enabled.
func IsReversibleEnabled() bool {
	globalState.syncFromGlobals()
	return globalState.IsReversibleEnabled()
}

// IsRemoteMode returns true if remote mode is enabled.
func IsRemoteMode() bool {
	globalState.syncFromGlobals()
	return globalState.IsRemoteMode()
}

// GetCompressionAddr returns the compression service address.
func GetCompressionAddr() string {
	globalState.syncFromGlobals()
	return globalState.GetCompressionAddr()
}

// GetAnalyticsAddr returns the analytics service address.
func GetAnalyticsAddr() string {
	globalState.syncFromGlobals()
	return globalState.GetAnalyticsAddr()
}

// GetRemoteTimeout returns the remote operation timeout in seconds.
func GetRemoteTimeout() int {
	globalState.syncFromGlobals()
	return globalState.GetRemoteTimeout()
}

// GetEnableLayers returns layers to explicitly enable.
func GetEnableLayers() []string {
	globalState.syncFromGlobals()
	return globalState.GetEnableLayers()
}

// GetDisableLayers returns layers to explicitly disable.
func GetDisableLayers() []string {
	globalState.syncFromGlobals()
	return globalState.GetDisableLayers()
}

// IsStreamMode returns true if streaming mode is enabled for large inputs.
func IsStreamMode() bool {
	globalState.syncFromGlobals()
	return globalState.IsStreamMode()
}
