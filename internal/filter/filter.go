package filter

import "strings"

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
	filters []Filter
	mode    Mode
}

// NewEngine creates a new filter engine with all registered filters.
func NewEngine(mode Mode) *Engine {
	filters := []Filter{
		NewANSIFilter(),
		NewCommentFilter(),
		NewImportFilter(),
		NewLogAggregator(),
	}
	if mode == ModeAggressive {
		filters = append(filters, NewBodyFilter())
	}

	return &Engine{
		filters: filters,
		mode:    mode,
	}
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

// DetectLanguageFromInput detects language from input content
func DetectLanguageFromInput(input string) Language {
	if strings.Contains(input, "package ") || strings.Contains(input, "func ") {
		return LangGo
	}
	if strings.Contains(input, "fn ") || strings.Contains(input, "pub fn") {
		return LangRust
	}
	if strings.Contains(input, "def ") || strings.Contains(input, "class ") {
		if strings.Contains(input, ":") && !strings.Contains(input, "{") {
			return LangPython
		}
		if strings.Contains(input, "import ") {
			return LangPython
		}
	}
	if strings.Contains(input, "function ") || strings.Contains(input, "const ") {
		if strings.Contains(input, ":") {
			return LangTypeScript
		}
		return LangJavaScript
	}
	if strings.Contains(input, "SELECT") || strings.Contains(input, "FROM") || strings.Contains(input, "WHERE") || strings.Contains(input, "INSERT") || strings.Contains(input, "UPDATE") {
		return LangSQL
	}
	return LangUnknown
}

// SetMode changes the filter mode.
func (e *Engine) SetMode(mode Mode) {
	e.mode = mode
}

// EstimateTokens provides a heuristic token count.
// Uses the formula: ceil(text.length / 4.0)
func EstimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// CalculateTokensSaved computes token savings between original and filtered.
func CalculateTokensSaved(original, filtered string) int {
	originalTokens := EstimateTokens(original)
	filteredTokens := EstimateTokens(filtered)
	if originalTokens > filteredTokens {
		return originalTokens - filteredTokens
	}
	return 0
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
	if strings.Contains(output, "def ") || strings.Contains(output, "import ") {
		if strings.Contains(output, ":") && !strings.Contains(output, "{") {
			return "python"
		}
	}
	if strings.Contains(output, "function ") || strings.Contains(output, "const ") {
		return "javascript"
	}
	return "unknown"
}
