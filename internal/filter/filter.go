package filter

import (
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// Mode represents the filtering mode.
type Mode string

const (
	ModeNone       Mode = "none"
	ModeMinimal    Mode = "minimal"
	ModeAggressive Mode = "aggressive"
)

var allModes = []Mode{ModeNone, ModeMinimal, ModeAggressive}

// Language represents a programming language for filtering
type Language string

const (
	LangRust       Language = "rust"
	LangPython     Language = "python"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangGo         Language = "go"
	LangC          Language = "c"
	LangCpp        Language = "cpp"
	LangJava       Language = "java"
	LangRuby       Language = "ruby"
	LangShell      Language = "sh"
	LangSQL        Language = "sql"
	LangUnknown    Language = "unknown"
)

// Filter defines the interface for output filters.
type Filter interface {
	// Name returns the filter name.
	Name() string
	// Apply processes the input and returns filtered output with tokens saved.
	Apply(input string, mode Mode) (output string, tokensSaved int)
}

// Engine combines multiple filters.
type Engine struct {
	filters        []Filter
	mode           Mode
	queryIntent    string // Query intent for query-aware compression
	promptTemplate string // Prompt template name for LLM summarization
}

// EngineConfig holds configuration for the filter engine
type EngineConfig struct {
	Mode             Mode
	QueryIntent      string
	LLMEnabled       bool
	MultiFileEnabled bool
	PromptTemplate   string // Template name for LLM summarization
}

// NewEngine creates a new filter engine with all registered filters.
func NewEngine(mode Mode) *Engine {
	return NewEngineWithQuery(mode, "")
}

// NewEngineWithQuery creates a new filter engine with query-aware compression.
func NewEngineWithQuery(mode Mode, queryIntent string) *Engine {
	return NewEngineWithConfig(EngineConfig{
		Mode:        mode,
		QueryIntent: queryIntent,
	})
}

// NewEngineWithConfig creates a filter engine with full configuration options.
func NewEngineWithConfig(cfg EngineConfig) *Engine {
	filters := []Filter{
		NewANSIFilter(),
		NewCommentFilter(),
		NewImportFilter(),
		NewLogAggregator(),
	}

	// Add multi-file filter early if enabled (for cross-file optimization)
	if cfg.MultiFileEnabled {
		filters = append(filters, NewMultiFileFilter(MultiFileConfig{
			PreserveBoundaries: true,
		}))
	}

	// Add research-based semantic filters
	filters = append(filters,
		NewSemanticFilter(),      // Semantic pruning - research-based
		NewPositionAwareFilter(), // Position-bias optimization - reorders for LLM recall
		NewHierarchicalFilter(),  // Multi-level summarization for large outputs
	)

	// Add query-aware filter if intent is provided
	if cfg.QueryIntent != "" {
		filters = append(filters, NewQueryAwareFilter(cfg.QueryIntent))
	}

	if cfg.Mode == ModeAggressive {
		filters = append(filters, NewBodyFilter())
	}

	return &Engine{
		filters:        filters,
		mode:           cfg.Mode,
		queryIntent:    cfg.QueryIntent,
		promptTemplate: cfg.PromptTemplate,
	}
}

// NewEngineWithLLM creates a filter engine with LLM-aware compression enabled.
// Falls back to heuristic-based filters if LLM is unavailable.
func NewEngineWithLLM(mode Mode, queryIntent string, llmEnabled bool) *Engine {
	return NewEngineWithConfig(EngineConfig{
		Mode:        mode,
		QueryIntent: queryIntent,
		LLMEnabled:  llmEnabled,
	})
}

// NewEngineWithLLMAndConfig creates a fully configured engine with LLM support.
func NewEngineWithLLMAndConfig(cfg EngineConfig) *Engine {
	engine := NewEngineWithConfig(cfg)

	if cfg.LLMEnabled {
		// Insert LLM-aware filter after semantic filter
		llmFilter := NewLLMAwareFilter(LLMAwareConfig{
			Threshold:      2000,
			Enabled:        true,
			CacheEnabled:   true,
			PromptTemplate: cfg.PromptTemplate,
		})

		// Find position after semantic filter
		filters := make([]Filter, 0, len(engine.filters)+1)
		for _, f := range engine.filters {
			filters = append(filters, f)
			if f.Name() == "semantic" {
				filters = append(filters, llmFilter)
			}
		}
		engine.filters = filters
	}

	return engine
}

// Process applies all filters to the input.
func (e *Engine) Process(input string) (string, int) {
	output := input
	totalSaved := 0

	for _, filter := range e.filters {
		// Skip body filter in minimal mode
		if e.mode == ModeMinimal && filter.Name() == "body" {
			continue
		}

		filtered, saved := filter.Apply(output, e.mode)
		output = filtered
		totalSaved += saved
	}

	return output, totalSaved
}

// ModeNone = raw passthrough
func (e *Engine) ProcessWithLang(input string, lang string) (string, int) {
	// Language-specific processing can be added here
	return e.Process(input)
}

// DetectLanguageFromInput detects language from input content.
// Delegates to DetectLanguage and wraps the result as a Language.
func DetectLanguageFromInput(input string) Language {
	return Language(DetectLanguage(input))
}

// SetMode changes the filter mode.
func (e *Engine) SetMode(mode Mode) {
	e.mode = mode
}

// EstimateTokens provides a heuristic token count.
// Delegates to core.EstimateTokens for single source of truth (T22).
func EstimateTokens(text string) int {
	return core.EstimateTokens(text)
}

// CalculateTokensSaved computes token savings between original and filtered.
func CalculateTokensSaved(original, filtered string) int {
	return core.CalculateTokensSaved(original, filtered)
}

// IsCode checks if the output looks like source code.
func IsCode(output string) bool {
	codeIndicators := []string{
		"func ", "function ", "def ", "class ", "struct ",
		"import ", "package ", "use ", "require(",
		"pub fn", "pub struct", "pub async",
		"//", "/*", "#!", "package main",
	}

	for _, indicator := range codeIndicators {
		if strings.Contains(output, indicator) {
			return true
		}
	}

	return false
}

// DetectLanguage attempts to detect the programming language from output.
func DetectLanguage(output string) string {
	if strings.Contains(output, "package ") || strings.Contains(output, "func ") {
		return "go"
	}
	if strings.Contains(output, "fn ") || strings.Contains(output, "pub fn") {
		return "rust"
	}
	if strings.Contains(output, "def ") || strings.Contains(output, "class ") {
		if strings.Contains(output, ":") && !strings.Contains(output, "{") {
			return "python"
		}
		if strings.Contains(output, "import ") {
			return "python"
		}
	}
	if strings.Contains(output, "SELECT") || strings.Contains(output, "FROM") || strings.Contains(output, "WHERE") || strings.Contains(output, "INSERT") || strings.Contains(output, "UPDATE") {
		return "sql"
	}
	if strings.Contains(output, "function ") || strings.Contains(output, "const ") {
		if strings.Contains(output, ":") {
			return "typescript"
		}
		return "javascript"
	}
	return "unknown"
}
