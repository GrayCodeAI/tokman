package filter

// ContextWindowTier represents a size category for LLM context windows.
type ContextWindowTier string

const (
	TierSmall  ContextWindowTier = "small"  // < 32K tokens (GPT-3.5, older models)
	TierMedium ContextWindowTier = "medium" // 32K–200K tokens (GPT-4, Claude Sonnet)
	TierLarge  ContextWindowTier = "large"  // 200K–1M tokens (Claude, Gemini Pro)
	TierXLarge ContextWindowTier = "xlarge" // > 1M tokens (Gemini 1.5 Pro)
)

// ContextWindowProfile holds compression settings appropriate for a given
// context window size. Smaller windows require more aggressive compression.
type ContextWindowProfile struct {
	Tier           ContextWindowTier
	WindowTokens   int
	DefaultMode    Mode
	TargetRatio    float64  // Target output/input ratio
	Filters        []string // Ordered filter names for this tier
}

// ContextWindowProfiles defines profiles for each tier.
var ContextWindowProfiles = map[ContextWindowTier]ContextWindowProfile{
	TierSmall: {
		Tier:         TierSmall,
		WindowTokens: 16_384,
		DefaultMode:  ModeAggressive,
		TargetRatio:  0.3, // compress to 30%
		Filters: []string{
			"ast_skeleton", "import_collapse", "boilerplate",
			"rle_compress", "importance_scoring", "smart_truncate",
			"error_dedup", "comment_strip",
		},
	},
	TierMedium: {
		Tier:         TierMedium,
		WindowTokens: 128_000,
		DefaultMode:  ModeMinimal,
		TargetRatio:  0.6, // compress to 60%
		Filters: []string{
			"boilerplate", "rle_compress", "importance_scoring",
			"error_dedup", "html_compress", "shell_output",
		},
	},
	TierLarge: {
		Tier:         TierLarge,
		WindowTokens: 200_000,
		DefaultMode:  ModeMinimal,
		TargetRatio:  0.8, // compress to 80%
		Filters: []string{
			"boilerplate", "rle_compress", "error_dedup",
		},
	},
	TierXLarge: {
		Tier:         TierXLarge,
		WindowTokens: 1_000_000,
		DefaultMode:  ModeNone, // pass through — window is huge
		TargetRatio:  1.0,
		Filters:      []string{},
	},
}

// ModelContextWindows maps known model identifiers to their context window sizes.
var ModelContextWindows = map[string]int{
	"claude-opus-4.6":           200_000,
	"claude-sonnet-4.6":         200_000,
	"claude-haiku-4.5":          200_000,
	"claude-3.5-sonnet":         200_000,
	"claude-3-haiku":            200_000,
	"gpt-4o":                    128_000,
	"gpt-4o-mini":               128_000,
	"gpt-4-turbo":               128_000,
	"gpt-3.5-turbo":             16_384,
	"gemini-1.5-pro":            1_000_000,
	"gemini-1.5-flash":          1_000_000,
	"gemini-pro":                32_000,
	"llama-3-70b":               128_000,
	"llama-3-8b":                8_000,
	"mistral-large":             128_000,
	"mistral-7b":                32_000,
}

// AdaptiveContextWindowCompressor automatically scales compression aggressiveness
// based on the target model's context window size.
type AdaptiveContextWindowCompressor struct {
	// ModelName is the target model identifier. If set, overrides WindowTokens.
	ModelName string
	// WindowTokens is an explicit context window override.
	WindowTokens int
}

// NewAdaptiveContextWindowCompressor creates a compressor for the given model.
func NewAdaptiveContextWindowCompressor(modelName string) *AdaptiveContextWindowCompressor {
	return &AdaptiveContextWindowCompressor{ModelName: modelName}
}

// NewAdaptiveContextWindowCompressorWithSize creates a compressor for the given window.
func NewAdaptiveContextWindowCompressorWithSize(windowTokens int) *AdaptiveContextWindowCompressor {
	return &AdaptiveContextWindowCompressor{WindowTokens: windowTokens}
}

// Name returns the filter name.
func (a *AdaptiveContextWindowCompressor) Name() string {
	return "adaptive_context_window"
}

// Profile returns the compression profile for the configured model/window.
func (a *AdaptiveContextWindowCompressor) Profile() ContextWindowProfile {
	windowTokens := a.WindowTokens
	if windowTokens <= 0 && a.ModelName != "" {
		windowTokens = ModelContextWindows[a.ModelName]
	}
	if windowTokens <= 0 {
		windowTokens = 128_000 // default to medium
	}
	return ProfileForWindow(windowTokens)
}

// Apply compresses input using the profile appropriate for the target model.
func (a *AdaptiveContextWindowCompressor) Apply(input string, mode Mode) (string, int) {
	profile := a.Profile()
	if profile.DefaultMode == ModeNone || len(profile.Filters) == 0 {
		return input, 0
	}

	// Use adaptive ratio to hit the target
	compressor := NewAdaptiveRatioCompressor(profile.TargetRatio)
	return compressor.Apply(input, profile.DefaultMode)
}

// ProfileForWindow returns the appropriate compression profile for a given
// context window size in tokens.
func ProfileForWindow(windowTokens int) ContextWindowProfile {
	switch {
	case windowTokens >= 1_000_000:
		return ContextWindowProfiles[TierXLarge]
	case windowTokens >= 200_000:
		return ContextWindowProfiles[TierLarge]
	case windowTokens >= 32_000:
		return ContextWindowProfiles[TierMedium]
	default:
		return ContextWindowProfiles[TierSmall]
	}
}

// TierForModel returns the context window tier for a named model.
func TierForModel(modelName string) ContextWindowTier {
	windowTokens := ModelContextWindows[modelName]
	if windowTokens == 0 {
		return TierMedium // default assumption
	}
	return ProfileForWindow(windowTokens).Tier
}
