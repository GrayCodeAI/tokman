package filter

import (
	"regexp"
	"strings"
	"unicode"
)

// NormalizationConfig holds configuration for the IdentifierNormalizer filter.
type NormalizationConfig struct {
	// TargetStyle is the desired output naming convention.
	// Valid values: "snake_case", "camelCase", "kebab-case".
	TargetStyle string
	// OnlyInComments restricts normalization to comment text only (not
	// prose strings or other content).
	OnlyInComments bool
}

// IdentifierNormalizer is a Filter that normalises identifier naming
// conventions inside non-code prose (comments and string literals).
// It converts between camelCase, snake_case, and kebab-case.
type IdentifierNormalizer struct {
	cfg NormalizationConfig
}

// NewIdentifierNormalizer creates an IdentifierNormalizer with the given config.
func NewIdentifierNormalizer(cfg NormalizationConfig) *IdentifierNormalizer {
	return &IdentifierNormalizer{cfg: cfg}
}

// Name implements Filter.
func (n *IdentifierNormalizer) Name() string {
	return "identifier-normalizer"
}

// Apply implements Filter. It normalises identifiers in comments and string
// literals according to cfg.TargetStyle.
func (n *IdentifierNormalizer) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	output := transformContent(input, n.cfg)
	saved := CalculateTokensSaved(input, output)
	return output, saved
}

// ─── Public conversion helpers ────────────────────────────────────────────────

// NormalizeToSnake converts a camelCase or PascalCase identifier to snake_case.
// Examples: "fooBar" → "foo_bar", "FooBar" → "foo_bar", "XMLParser" → "xml_parser".
func NormalizeToSnake(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i == 0 {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		prev := runes[i-1]
		// Insert underscore before an uppercase letter that either follows a
		// lowercase letter, or starts a new word in an acronym (followed by lower).
		if unicode.IsUpper(r) {
			if unicode.IsLower(prev) {
				b.WriteRune('_')
			} else if i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(prev) {
				b.WriteRune('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeToCamel converts a snake_case identifier to camelCase.
// Examples: "foo_bar" → "fooBar", "foo_bar_baz" → "fooBarBaz".
func NormalizeToCamel(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "_")
	var b strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			b.WriteString(strings.ToLower(part))
		} else {
			runes := []rune(part)
			b.WriteRune(unicode.ToUpper(runes[0]))
			b.WriteString(string(runes[1:]))
		}
	}
	return b.String()
}

// NormalizeToKebab converts a snake_case identifier to kebab-case.
// Examples: "foo_bar" → "foo-bar", "foo_bar_baz" → "foo-bar-baz".
func NormalizeToKebab(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}

// ─── Internal transformation logic ────────────────────────────────────────────

// region describes a byte range within a string that should be transformed.
type region struct {
	start, end int
	inString   bool
}

// reComment matches both // line comments and /* block comments */ (simplified).
var reComment = regexp.MustCompile(`//[^\n]*|/\*[\s\S]*?\*/`)

// reString matches double-quoted or single-quoted string literals (simplified).
var reString = regexp.MustCompile(`"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'`)

// reIdentifierLike matches word-like tokens that could be identifiers:
// camelCase, PascalCase (heuristic: mixed case, no underscores/hyphens already),
// snake_case (underscores), or kebab-case (hyphens).
var reIdentifierLike = regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9]*(?:[_\-][a-zA-Z0-9]+)+|[a-z][a-z0-9]*[A-Z][a-zA-Z0-9]*|[A-Z][a-zA-Z0-9]*[A-Z][a-zA-Z0-9]*)\b`)

// convertIdentifier converts a single identifier token to the target style.
func convertIdentifier(tok string, style string) string {
	// Detect source style.
	hasUnderscore := strings.Contains(tok, "_")
	hasHyphen := strings.Contains(tok, "-")

	var snake string
	switch {
	case hasUnderscore:
		snake = strings.ToLower(tok)
	case hasHyphen:
		snake = strings.ReplaceAll(tok, "-", "_")
		snake = strings.ToLower(snake)
	default:
		// Assume camelCase / PascalCase.
		snake = NormalizeToSnake(tok)
	}

	switch style {
	case "camelCase":
		return NormalizeToCamel(snake)
	case "kebab-case":
		return NormalizeToKebab(snake)
	default: // "snake_case"
		return snake
	}
}

// transformSegment replaces identifier-like tokens within a prose segment.
func transformSegment(text string, style string) string {
	return reIdentifierLike.ReplaceAllStringFunc(text, func(tok string) string {
		return convertIdentifier(tok, style)
	})
}

// transformContent applies the normalisation to comments and (optionally)
// string literals in the full input.
func transformContent(input string, cfg NormalizationConfig) string {
	style := cfg.TargetStyle
	if style == "" {
		style = "snake_case"
	}

	var result strings.Builder
	pos := 0

	// Collect all regions to transform (comments always; strings when not
	// OnlyInComments).
	var regions []region

	for _, loc := range reComment.FindAllStringIndex(input, -1) {
		regions = append(regions, region{loc[0], loc[1], false})
	}
	if !cfg.OnlyInComments {
		for _, loc := range reString.FindAllStringIndex(input, -1) {
			regions = append(regions, region{loc[0], loc[1], true})
		}
	}

	// Sort regions by start position using a simple insertion approach
	// (regions are generally small in number).
	sortRegions(regions)

	// Walk through input, replacing content inside identified regions.
	for _, r := range regions {
		if r.start < pos {
			// Overlapping region (e.g., string inside comment already processed); skip.
			continue
		}
		// Emit the gap before this region verbatim.
		result.WriteString(input[pos:r.start])

		segment := input[r.start:r.end]
		if r.inString {
			// Preserve quotes; transform the content inside.
			quote := string(segment[0])
			inner := segment[1 : len(segment)-1]
			result.WriteString(quote + transformSegment(inner, style) + quote)
		} else {
			// Transform the comment body (keep markers).
			result.WriteString(transformSegment(segment, style))
		}

		pos = r.end
	}
	result.WriteString(input[pos:])
	return result.String()
}

// sortRegions sorts regions by start index (simple insertion sort; N is small).
func sortRegions(regions []region) {
	for i := 1; i < len(regions); i++ {
		key := regions[i]
		j := i - 1
		for j >= 0 && regions[j].start > key.start {
			regions[j+1] = regions[j]
			j--
		}
		regions[j+1] = key
	}
}
