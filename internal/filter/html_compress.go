package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// HTMLCompressFilter removes noise from HTML/XML content to reduce token usage.
// Focuses on removing boilerplate that doesn't contribute semantic value:
//  1. HTML comments
//  2. Script and style tag contents
//  3. Long attribute values (inline styles, data-* attrs with large values)
//  4. DOCTYPE declarations and XML processing instructions
//  5. Blank lines within tags
type HTMLCompressFilter struct{}

var (
	// HTML comment
	htmlCommentRe = regexp.MustCompile(`<!--[\s\S]*?-->`)
	// Script/style tag contents
	scriptTagRe = regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	styleTagRe  = regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	// DOCTYPE and XML processing instructions
	doctypeRe = regexp.MustCompile(`(?i)<!doctype[^>]*>`)
	xmlProcRe = regexp.MustCompile(`<\?xml[^?]*\?>`)
	// Long inline styles: style="...very long..."
	inlineStyleRe = regexp.MustCompile(`\s+style="[^"]{40,}"`)
	// Long data attributes: data-foo="...very long..."
	longDataAttrRe = regexp.MustCompile(`\s+data-[a-z-]+="[^"]{60,}"`)
	// Long class lists
	longClassRe = regexp.MustCompile(`\s+class="([^"]{80,})"`)
	// SVG path data (very long)
	svgPathRe = regexp.MustCompile(`\s+d="[^"]{100,}"`)
	// Base64 src attributes
	base64SrcRe = regexp.MustCompile(`\s+src="data:[^"]{50,}"`)
)

// NewHTMLCompressFilter creates a new HTML/XML compression filter.
func NewHTMLCompressFilter() *HTMLCompressFilter {
	return &HTMLCompressFilter{}
}

// Name returns the filter name.
func (f *HTMLCompressFilter) Name() string {
	return "html_compress"
}

// Apply compresses HTML/XML content.
func (f *HTMLCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Only apply to HTML/XML content
	if !strings.Contains(input, "<") || !strings.Contains(input, ">") {
		return input, 0
	}
	// Require at least some tag-like structure
	if !strings.Contains(input, "</") && !strings.Contains(input, "/>") {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input

	// Always: remove HTML comments
	output = htmlCommentRe.ReplaceAllString(output, "<!-- ... -->")

	// Always: remove DOCTYPE and XML PIs
	output = doctypeRe.ReplaceAllString(output, "")
	output = xmlProcRe.ReplaceAllString(output, "")

	// Always: remove base64 inline images (huge waste)
	output = base64SrcRe.ReplaceAllString(output, ` src="[base64-data]"`)

	// Always: remove SVG path data
	output = svgPathRe.ReplaceAllString(output, ` d="[path-data]"`)

	// Always: remove long inline styles
	output = inlineStyleRe.ReplaceAllString(output, ` style="[...]"`)

	// Always: remove long data attributes
	output = longDataAttrRe.ReplaceAllString(output, "")

	if mode == ModeAggressive {
		// Aggressive: also remove script/style contents
		output = scriptTagRe.ReplaceAllString(output, "<script>[...]</script>")
		output = styleTagRe.ReplaceAllString(output, "<style>[...]</style>")

		// Truncate long class lists
		output = longClassRe.ReplaceAllStringFunc(output, func(m string) string {
			runes := []rune(m)
			if len(runes) > 40 {
				return ` class="[...]"`
			}
			return m
		})
	}

	// Clean up blank lines
	output = collapseBlankLines(output, 1)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}
