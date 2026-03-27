package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// JSONYAMLCompressFilter reduces token usage in JSON and YAML content by:
//  1. Collapsing short arrays/objects onto single lines
//  2. Removing null/empty values in aggressive mode
//  3. Abbreviating long string values (preserving first/last chars)
//  4. Removing comments from YAML
type JSONYAMLCompressFilter struct{}

var (
	// YAML comment lines
	yamlCommentRe = regexp.MustCompile(`(?m)^\s*#.*$`)
	// Empty JSON value: "key": null or "key": ""
	jsonNullRe  = regexp.MustCompile(`(?m),?\s*"[^"]+"\s*:\s*(?:null|""|0|false)\s*`)
	// Long JSON string values (>80 chars)
	longStringRe = regexp.MustCompile(`"([^"]{80,})"`)
	// YAML null/empty values
	yamlNullRe = regexp.MustCompile(`(?m)^(\s*\w[\w-]*):\s*(?:null|~|''|""|)\s*$`)
	// Compact JSON arrays: multiline array that fits on one line
	multilineEmptyArrayRe = regexp.MustCompile(`\[\s*\n\s*\]`)
	multilineEmptyObjRe   = regexp.MustCompile(`\{\s*\n\s*\}`)
)

// NewJSONYAMLCompressFilter creates a new JSON/YAML compression filter.
func NewJSONYAMLCompressFilter() *JSONYAMLCompressFilter {
	return &JSONYAMLCompressFilter{}
}

// Name returns the filter name.
func (f *JSONYAMLCompressFilter) Name() string {
	return "json_yaml_compress"
}

// Apply compresses JSON and YAML content.
func (f *JSONYAMLCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Detect if this looks like JSON or YAML
	trimmed := strings.TrimSpace(input)
	isJSON := len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
	isYAML := !isJSON && (strings.Contains(input, ":\n") || strings.Contains(input, ": "))
	if !isJSON && !isYAML {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input

	if isYAML {
		// Remove YAML comments
		output = yamlCommentRe.ReplaceAllString(output, "")
		if mode == ModeAggressive {
			// Remove null/empty YAML values
			output = yamlNullRe.ReplaceAllStringFunc(output, func(m string) string {
				return "" // Remove the line entirely
			})
		}
	}

	if isJSON {
		// Collapse empty arrays and objects
		output = multilineEmptyArrayRe.ReplaceAllString(output, "[]")
		output = multilineEmptyObjRe.ReplaceAllString(output, "{}")

		if mode == ModeAggressive {
			// Remove null/empty values
			output = jsonNullRe.ReplaceAllString(output, "")
		}
	}

	// Abbreviate very long string values in both formats
	if mode == ModeAggressive {
		output = longStringRe.ReplaceAllStringFunc(output, func(m string) string {
			// m includes the surrounding quotes
			inner := m[1 : len(m)-1]
			if len(inner) > 80 {
				runes := []rune(inner)
				return `"` + string(runes[:20]) + `…` + string(runes[len(runes)-10:]) + `"`
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

// collapseBlankLines collapses runs of blank lines to at most max consecutive blanks.
func collapseBlankLines(input string, max int) string {
	lines := strings.Split(input, "\n")
	var result []string
	blanks := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks <= max {
				result = append(result, line)
			}
		} else {
			blanks = 0
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}
