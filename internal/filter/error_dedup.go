package filter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// errorKeywords are the prefix strings that mark a line as an error line.
var errorKeywords = []string{
	"error:", "Error:", "ERROR:",
	"failed:", "FAILED:",
	"panic:",
	"exception:",
	"refused", "timeout", "unreachable",
}

// ErrorDedupFilter groups repeated error messages — both exact duplicates and
// template-based duplicates (same error message, different hosts/IDs/paths).
//
// Examples:
//
//	"connection refused to 10.0.0.1:8080" ×3
//	→ connection refused to [...] [×3: 10.0.0.1:8080, 10.0.0.2:8080, 10.0.0.3:8080]
//
//	"error: file not found: /tmp/a.txt" ×2
//	→ error: file not found: [...] [×2: /tmp/a.txt, /tmp/b.txt]
type ErrorDedupFilter struct{}

var (
	// IP:port or hostname:port
	errIPPortRe = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?::\d+)?\b|\blocalhost(?::\d+)?\b`)
	// Absolute file paths
	errFilePathRe = regexp.MustCompile(`(?:^|[\s"'(])(/[\w./\-_]+)`)
	// Large integers (IDs, timestamps, offsets)
	errBigIntRe = regexp.MustCompile(`\b\d{5,}\b`)
	// UUIDs
	errUUIDRe = regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	// Hex addresses (e.g. 0x7f3a2b1c)
	errHexRe = regexp.MustCompile(`\b0x[0-9a-fA-F]{4,}\b`)
)

// NewErrorDedupFilter creates a new ErrorDedupFilter.
func NewErrorDedupFilter() *ErrorDedupFilter {
	return &ErrorDedupFilter{}
}

// Name returns the filter name.
func (f *ErrorDedupFilter) Name() string {
	return "error_dedup"
}

// Apply deduplicates repeated error lines and returns token savings.
func (f *ErrorDedupFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)
	lines := strings.Split(input, "\n")

	minCount := 2
	if mode == ModeAggressive {
		minCount = 2
	}

	type group struct {
		template  string
		firstIdx  int
		count     int
		variables []string // unique variable parts
	}

	templateOrder := []string{}
	groups := make(map[string]*group)

	for i, line := range lines {
		if !errDedupIsErrorLine(line) {
			continue
		}
		tmpl, variable := errExtractTemplate(line)
		if g, ok := groups[tmpl]; ok {
			g.count++
			if variable != "" && !errContains(g.variables, variable) {
				g.variables = append(g.variables, variable)
			}
		} else {
			vars := []string{}
			if variable != "" {
				vars = append(vars, variable)
			}
			groups[tmpl] = &group{
				template: tmpl,
				firstIdx: i,
				count:    1,
				variables: vars,
			}
			templateOrder = append(templateOrder, tmpl)
		}
	}

	// Find templates that meet the minimum count threshold
	dedup := make(map[string]bool)
	for _, tmpl := range templateOrder {
		if groups[tmpl].count >= minCount {
			dedup[tmpl] = true
		}
	}

	if len(dedup) == 0 {
		return input, 0
	}

	// Second pass: build output
	emitted := make(map[string]bool)
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		if !errDedupIsErrorLine(line) {
			out = append(out, line)
			continue
		}
		tmpl, _ := errExtractTemplate(line)
		if !dedup[tmpl] {
			out = append(out, line)
			continue
		}
		if !emitted[tmpl] {
			g := groups[tmpl]
			out = append(out, errBuildSummary(line, tmpl, g.variables, g.count))
			emitted[tmpl] = true
		}
		// Subsequent occurrences silently dropped.
	}

	output := strings.Join(out, "\n")
	saved := originalTokens - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// errDedupIsErrorLine returns true when line contains one of the known error keywords.
func errDedupIsErrorLine(line string) bool {
	for _, kw := range errorKeywords {
		if strings.Contains(line, kw) {
			return true
		}
	}
	return false
}

// errExtractTemplate replaces variable parts of an error line with placeholders,
// returning the canonical template and the extracted variable string.
func errExtractTemplate(line string) (template, variable string) {
	// Collect variables before replacing
	addrs := errIPPortRe.FindAllString(line, -1)
	paths := errFilePathRe.FindAllStringSubmatch(line, -1)
	uuids := errUUIDRe.FindAllString(line, -1)
	hexes := errHexRe.FindAllString(line, -1)

	tmpl := errIPPortRe.ReplaceAllString(line, "<addr>")
	tmpl = errFilePathRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		if len(m) > 1 && (m[0] == ' ' || m[0] == '"' || m[0] == '\'' || m[0] == '(') {
			return string(m[0]) + "<path>"
		}
		return "<path>"
	})
	tmpl = errUUIDRe.ReplaceAllString(tmpl, "<uuid>")
	tmpl = errHexRe.ReplaceAllString(tmpl, "<hex>")
	tmpl = errBigIntRe.ReplaceAllString(tmpl, "<N>")

	// Collect unique variable values
	var vars []string
	vars = append(vars, addrs...)
	for _, p := range paths {
		if len(p) > 1 {
			vars = append(vars, p[1])
		}
	}
	vars = append(vars, uuids...)
	vars = append(vars, hexes...)

	variable = strings.Join(vars, ", ")
	return tmpl, variable
}

// errBuildSummary builds a compact summary for a grouped error.
func errBuildSummary(firstLine, _ string, vars []string, count int) string {
	// Use first line as display base, add grouping annotation
	sort.Strings(vars)
	if len(vars) == 0 {
		return fmt.Sprintf("%s (×%d)", firstLine, count)
	}
	shown := vars
	extra := 0
	if len(shown) > 5 {
		shown = vars[:5]
		extra = len(vars) - 5
	}
	suffix := fmt.Sprintf(" [×%d: %s", count, strings.Join(shown, ", "))
	if extra > 0 {
		suffix += fmt.Sprintf(", +%d more", extra)
	}
	suffix += "]"
	return firstLine + suffix
}

func errContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
