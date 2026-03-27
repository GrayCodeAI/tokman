package filter

import (
	"regexp"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// SymbolicCompressFilter implements MetaGlyph-style symbolic instruction compression.
// Research Source: "Semantic Compression of LLM Instructions via Symbolic Metalanguages" (Jan 2026)
// Key Innovation: Replace verbose natural-language instructions with compact symbolic notation.
// Results: 62-81% token reduction on instruction patterns while preserving semantic fidelity.
//
// This compresses common instruction patterns found in system prompts, CLI help text,
// and configuration documentation into compact symbolic representations.
type SymbolicCompressFilter struct {
	config   SymbolicConfig
	patterns []symbolicPattern
	
	// Pre-compiled patterns for compressInstructionPatterns (Phase 1 optimization)
	followingPattern    *regexp.Regexp
	thisFuncPattern     *regexp.Regexp
	returnsPattern      *regexp.Regexp
	paramPattern        *regexp.Regexp
}

// SymbolicConfig holds configuration for symbolic compression
type SymbolicConfig struct {
	// Enabled controls whether the filter is active
	Enabled bool

	// MinContentLength is the minimum character length to apply
	MinContentLength int

	// PreservesStructure keeps line breaks for readability
	PreserveStructure bool
}

// symbolicPattern maps verbose patterns to compact symbols
type symbolicPattern struct {
	pattern     *regexp.Regexp
	replacement string
	description string
}

// DefaultSymbolicConfig returns default configuration
func DefaultSymbolicConfig() SymbolicConfig {
	return SymbolicConfig{
		Enabled:           true,
		MinContentLength:  100,
		PreserveStructure: true,
	}
}

// NewSymbolicCompressFilter creates a new symbolic compression filter
func NewSymbolicCompressFilter() *SymbolicCompressFilter {
	f := &SymbolicCompressFilter{
		config: DefaultSymbolicConfig(),
		// Pre-compile patterns used in compressInstructionPatterns
		followingPattern: regexp.MustCompile(`(?i)the\s+following\s+(\w+)\s+are\s+(\w+):`),
		thisFuncPattern:  regexp.MustCompile(`(?i)This\s+(?:function|method|command|tool)\s+(?:is\s+used\s+to|will|allows\s+you\s+to)\s+`),
		returnsPattern:   regexp.MustCompile(`(?i)(?:returns?|outputs?|produces?|yields?)\s+(?:a\s+|an\s+|the\s+)?`),
		paramPattern:     regexp.MustCompile(`(?i)@(?:param|parameter)\s+(\w+)\s*[-:]\s*`),
	}
	f.patterns = f.initPatterns()
	return f
}

// Name returns the filter name
func (f *SymbolicCompressFilter) Name() string {
	return "symbolic_compress"
}

// Apply applies symbolic compression to instruction-style content
func (f *SymbolicCompressFilter) Apply(input string, mode Mode) (string, int) {
	if !f.config.Enabled || mode == ModeNone {
		return input, 0
	}

	if len(input) < f.config.MinContentLength {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	output := input

	// Apply symbolic patterns
	for _, p := range f.patterns {
		output = p.pattern.ReplaceAllString(output, p.replacement)
	}

	// Compress verbose instruction patterns
	output = f.compressInstructionPatterns(output, mode)

	finalTokens := core.EstimateTokens(output)
	saved := originalTokens - finalTokens
	if saved < 3 {
		return input, 0
	}

	return output, saved
}

// initPatterns initializes the symbolic pattern dictionary
func (f *SymbolicCompressFilter) initPatterns() []symbolicPattern {
	return []symbolicPattern{
		// Conditional patterns
		{regexp.MustCompile(`(?i)if\s+(.+?)\s+then\s+(.+?)(?:\.|$)`), "($1)→$2", "if-then"},
		{regexp.MustCompile(`(?i)when\s+(.+?)\s+do\s+(.+?)(?:\.|$)`), "[$1]→$2", "when-do"},
		{regexp.MustCompile(`(?i)unless\s+(.+?)\s*,?\s*(.+?)(?:\.|$)`), "(!$1)→$2", "unless"},

		// Permission patterns
		{regexp.MustCompile(`(?i)you\s+(?:can|may|are\s+allowed\s+to)\s+(.+?)(?:\.|$)`), "✓$1", "permission-grant"},
		{regexp.MustCompile(`(?i)you\s+(?:cannot|may\s+not|must\s+not|should\s+not)\s+(.+?)(?:\.|$)`), "✗$1", "permission-deny"},
		{regexp.MustCompile(`(?i)do\s+not\s+(.+?)(?:\.|$)`), "✗$1", "do-not"},
		{regexp.MustCompile(`(?i)never\s+(.+?)(?:\.|$)`), "✗$1", "never"},

		// Requirement patterns
		{regexp.MustCompile(`(?i)you\s+(?:must|shall|need\s+to|have\s+to|required?\s+to)\s+(.+?)(?:\.|$)`), "!$1", "requirement"},
		{regexp.MustCompile(`(?i)always\s+(.+?)(?:\.|$)`), "∀$1", "always"},

		// Example patterns
		{regexp.MustCompile(`(?i)for\s+example[,:]?\s+(.+?)(?:\.|$)`), "e.g.=$1", "example"},
		{regexp.MustCompile(`(?i)such\s+as\s+(.+?)(?:\.|$)`), "e.g.=$1", "such-as"},
		{regexp.MustCompile(`(?i)in\s+other\s+words[,:]?\s+(.+?)(?:\.|$)`), "i.e.=$1", "clarification"},

		// Sequence patterns
		{regexp.MustCompile(`(?i)first[,:]\s+(.+?)\s+then[,:]\s+(.+?)(?:\.|$)`), "1.$1 2.$2", "sequence"},
		{regexp.MustCompile(`(?i)step\s+(\d+)[,:]?\s+(.+?)(?:\.|$)`), "$1.$2", "step"},

		// Comparison patterns
		{regexp.MustCompile(`(?i)instead\s+of\s+(.+?)[,:]?\s+use\s+(.+?)(?:\.|$)`), "$1→$2", "replacement"},
		{regexp.MustCompile(`(?i)(.+?)\s+or\s+(.+?)\s+but\s+not\s+(.+?)(?:\.|$)`), "$1|$2-{$3}", "xor-pattern"},

		// Common verbose phrases
		{regexp.MustCompile(`(?i)in\s+order\s+to\s+`), "to ", "in-order-to"},
		{regexp.MustCompile(`(?i)it\s+is\s+important\s+to\s+note\s+that\s+`), "⚠ ", "important-note"},
		{regexp.MustCompile(`(?i)please\s+(?:note\s+that\s+|be\s+aware\s+that\s+|keep\s+in\s+mind\s+that\s+)?`), "ℹ ", "please-note"},
		{regexp.MustCompile(`(?i)make\s+sure\s+(?:that\s+)?`), "! ", "make-sure"},
		{regexp.MustCompile(`(?i)keep\s+in\s+mind\s+(?:that\s+)?`), "⚠ ", "keep-in-mind"},
		{regexp.MustCompile(`(?i)it\s+should\s+be\s+noted\s+that\s+`), "⚠ ", "should-be-noted"},
		{regexp.MustCompile(`(?i)as\s+mentioned\s+(?:above|earlier|before)\s*`), "↻ ", "as-mentioned"},
		{regexp.MustCompile(`(?i)with\s+respect\s+to\s+`), "re: ", "wrt"},
		{regexp.MustCompile(`(?i)in\s+the\s+context\s+of\s+`), "ctx: ", "in-context-of"},
		{regexp.MustCompile(`(?i)the\s+purpose\s+of\s+(?:this|the)\s+(?:is\s+to|function|method)\s+`), "⟹ ", "purpose-of"},
	}
}

// compressInstructionPatterns compresses common instruction-style patterns
func (f *SymbolicCompressFilter) compressInstructionPatterns(input string, mode Mode) string {
	output := input

	// Compress list item patterns (using pre-compiled pattern)
	if mode == ModeAggressive {
		// "The following X are Y:" → "X: Y"
		output = f.followingPattern.ReplaceAllString(output, "$1($2):")
	}

	// Compress verbose descriptions (using pre-compiled patterns)
	output = f.thisFuncPattern.ReplaceAllString(output, "⟹ ")

	// Compress return value descriptions
	output = f.returnsPattern.ReplaceAllString(output, "→ ")

	// Compress parameter descriptions
	output = f.paramPattern.ReplaceAllString(output, "⊤$1: ")

	return output
}
