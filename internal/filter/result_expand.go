package filter

import (
	"regexp"
	"strings"
)

// ResultExpander attempts to reverse lossless compression transforms that were
// applied by filters in this package.
//
// Supported reversals:
//   - var-shorten: parses the "// var-shorten: X=Y" legend at the top of the
//     text and substitutes short names back to their original long names.
//
// Non-reversible transforms:
//   - ANSI strip: lossy — colour codes cannot be recovered.
//   - Whitespace normalization: lossy — original spacing cannot be fully restored.
//
// Task #190: Compression result decompression/expansion.
type ResultExpander struct{}

// NewResultExpander creates a ResultExpander.
func NewResultExpander() *ResultExpander {
	return &ResultExpander{}
}

// ExpandResult describes what was done during an expansion attempt.
type ExpandResult struct {
	// Expanded is the (partially) restored text.
	Expanded string
	// Reversible is true when every applied transform could be fully reversed.
	Reversible bool
	// Applied lists the names of transforms that were successfully reversed.
	Applied []string
}

// varShortenLegendRe matches the header line written by VarShortenFilter.
// Example: // var-shorten: cpm=connectionPoolManager, ub=userBuffer
var varShortenLegendRe = regexp.MustCompile(`^// var-shorten: (.+)$`)

// varShortenEntryRe matches a single short=long entry inside the legend.
var varShortenEntryRe = regexp.MustCompile(`(\w+)=(\w+)`)

// Expand attempts to reverse compression transforms present in compressed.
// It detects which transforms were applied based on header markers and
// performs the reversals it can. Lossy transforms are noted but not reversed.
func (e *ResultExpander) Expand(compressed string) (string, ExpandResult) {
	output := compressed
	var applied []string
	reversible := true

	// --- Detect and note lossy transforms (cannot reverse) ---
	// We cannot detect ANSI strip or whitespace normalisation from the output
	// alone, but we can document that they are not reversible.
	// (Nothing to do here — they leave no marker.)

	// --- Reverse var-shorten ---
	varShortenApplied, expanded := reverseVarShorten(output)
	if varShortenApplied {
		output = expanded
		applied = append(applied, "var-shorten")
	}

	// If there were any lossy transforms we cannot reverse them; however,
	// since we have no way to detect them from the text alone, we leave
	// reversible=true unless we know otherwise.
	// Callers can set their own expectations based on the pipeline they used.
	_ = reversible // kept for clarity; may be extended in the future

	return output, ExpandResult{
		Expanded:   output,
		Reversible: len(applied) > 0,
		Applied:    applied,
	}
}

// reverseVarShorten parses the var-shorten legend line (if present) and
// replaces every short identifier back to its original long name.
// Returns (true, expanded) when the legend was found, or (false, original)
// when no legend is present.
func reverseVarShorten(text string) (bool, string) {
	// The legend is always the first line if present.
	firstNL := strings.IndexByte(text, '\n')
	if firstNL < 0 {
		return false, text
	}
	firstLine := text[:firstNL]
	rest := text[firstNL+1:]

	m := varShortenLegendRe.FindStringSubmatch(firstLine)
	if m == nil {
		return false, text
	}

	legendBody := m[1]
	// Parse comma-separated short=long entries.
	entries := varShortenEntryRe.FindAllStringSubmatch(legendBody, -1)
	if len(entries) == 0 {
		return false, text
	}

	// Build mapping: short -> long, sorted longest-short-first to avoid
	// partial replacements when one abbreviation is a prefix of another.
	type mapping struct{ short, long string }
	mappings := make([]mapping, 0, len(entries))
	for _, entry := range entries {
		mappings = append(mappings, mapping{short: entry[1], long: entry[2]})
	}
	// Sort by descending short name length.
	for i := 0; i < len(mappings); i++ {
		for j := i + 1; j < len(mappings); j++ {
			if len(mappings[j].short) > len(mappings[i].short) {
				mappings[i], mappings[j] = mappings[j], mappings[i]
			}
		}
	}

	// Apply substitutions on rest (body without legend line).
	expanded := rest
	for _, mp := range mappings {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(mp.short) + `\b`)
		expanded = re.ReplaceAllString(expanded, mp.long)
	}

	return true, expanded
}
