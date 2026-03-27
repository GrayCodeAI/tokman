package filter

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// VarShortenFilter shortens long local variable names within a code block.
// It replaces consistently-used long identifiers (>10 chars) with shorter
// abbreviations, preserving semantic meaning while reducing token count.
// This is lossless in the sense that the mapping is embedded in the output.
//
// Example: "connectionPoolManager" → "cpm" (with "// cpm=connectionPoolManager" header)
//
// Task #139: Variable name shortening for local scope.
type VarShortenFilter struct {
	// MinLength is the minimum identifier length to consider shortening.
	MinLength int
	// MinOccurrences is the minimum times an identifier must appear to shorten.
	MinOccurrences int
	// MaxMappings limits how many variables are shortened per block.
	MaxMappings int
}

// NewVarShortenFilter creates a variable shortening filter with defaults.
func NewVarShortenFilter() *VarShortenFilter {
	return &VarShortenFilter{
		MinLength:      12,
		MinOccurrences: 3,
		MaxMappings:    8,
	}
}

// Name returns the filter name.
func (f *VarShortenFilter) Name() string { return "var_shorten" }

// identRe matches Go/Rust/JS/Python identifiers (camelCase, snake_case).
var identRe = regexp.MustCompile(`\b([a-z][a-zA-Z0-9_]{11,}|[A-Z][a-zA-Z0-9_]{11,})\b`)

// Apply shortens long repeated variable names in the input.
func (f *VarShortenFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Only run on code-like content
	if !looksLikeCode(input) {
		return input, 0
	}

	original := core.EstimateTokens(input)

	// Count identifier occurrences
	counts := make(map[string]int)
	for _, match := range identRe.FindAllString(input, -1) {
		counts[match]++
	}

	// Build abbreviation map for frequent long identifiers
	type abbrevEntry struct{ long, short string }
	var abbrevs []abbrevEntry
	used := make(map[string]bool)

	for ident, count := range counts {
		if count < f.MinOccurrences || len(ident) < f.MinLength {
			continue
		}
		if len(abbrevs) >= f.MaxMappings {
			break
		}
		abbr := makeAbbreviation(ident)
		if used[abbr] {
			// Collision: append a digit
			for i := 2; i <= 9; i++ {
				candidate := abbr + itoa(i)
				if !used[candidate] {
					abbr = candidate
					break
				}
			}
		}
		if used[abbr] {
			continue // skip if still colliding
		}
		used[abbr] = true
		abbrevs = append(abbrevs, abbrevEntry{ident, abbr})
	}

	if len(abbrevs) == 0 {
		return input, 0
	}

	// Sort by longest identifier first to avoid partial replacements
	for i := 0; i < len(abbrevs); i++ {
		for j := i + 1; j < len(abbrevs); j++ {
			if len(abbrevs[j].long) > len(abbrevs[i].long) {
				abbrevs[i], abbrevs[j] = abbrevs[j], abbrevs[i]
			}
		}
	}

	// Apply substitutions
	output := input
	for _, a := range abbrevs {
		// Use word-boundary substitution
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(a.long) + `\b`)
		output = re.ReplaceAllString(output, a.short)
	}

	// Prepend abbreviation legend
	var legend strings.Builder
	legend.WriteString("// var-shorten: ")
	for i, a := range abbrevs {
		if i > 0 {
			legend.WriteString(", ")
		}
		legend.WriteString(a.short + "=" + a.long)
	}
	legend.WriteString("\n")
	output = legend.String() + output

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// makeAbbreviation creates a short abbreviation for a camelCase or snake_case identifier.
func makeAbbreviation(ident string) string {
	// snake_case: take first letter of each word
	if strings.Contains(ident, "_") {
		parts := strings.Split(ident, "_")
		var abbr []rune
		for _, p := range parts {
			if len(p) > 0 {
				abbr = append(abbr, rune(p[0]))
			}
		}
		if len(abbr) >= 2 {
			return strings.ToLower(string(abbr))
		}
	}

	// camelCase: take uppercase letters + first letter
	var abbr []rune
	for i, r := range ident {
		if i == 0 || unicode.IsUpper(r) {
			abbr = append(abbr, unicode.ToLower(r))
		}
	}
	if len(abbr) >= 2 {
		return string(abbr)
	}
	// Fallback: first 3 chars
	if len(ident) >= 3 {
		return strings.ToLower(ident[:3])
	}
	return strings.ToLower(ident)
}

// looksLikeCode checks if the input contains code patterns.
func looksLikeCode(input string) bool {
	codeMarkers := []string{":=", "func ", "def ", "class ", "var ", "const ", "let ", "return "}
	lower := strings.ToLower(input[:min(len(input), 512)])
	for _, m := range codeMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
