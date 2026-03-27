package filter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ImportCollapseFilter collapses verbose import/require blocks into compact form.
// Reduces token usage for files with many imports by grouping and summarizing them.
type ImportCollapseFilter struct{}

var (
	// Go/Java/Python multi-line import blocks
	goImportBlockRe = regexp.MustCompile(`(?m)^import\s*\(\n((?:\s+[^\n]+\n)+)\)`)
	// JS/TS require lines: const foo = require('...')
	jsRequireRe = regexp.MustCompile(`(?m)^(?:const|let|var)\s+\{?[^}=\n]+\}?\s*=\s*require\([^\)]+\);?\s*$`)
	// ES6 import lines: import foo from '...'
	es6ImportRe = regexp.MustCompile(`(?m)^import\s+(?:\{[^}]+\}|[^'";\n]+)\s+from\s+['"][^'"]+['"];?\s*$`)
	// Python import lines
	pythonImportRe = regexp.MustCompile(`(?m)^(?:import|from)\s+\S+[^\n]*$`)
	// Rust use declarations
	rustUseRe = regexp.MustCompile(`(?m)^use\s+[^\n]+;$`)
)

// NewImportCollapseFilter creates a new import collapse filter.
func NewImportCollapseFilter() *ImportCollapseFilter {
	return &ImportCollapseFilter{}
}

// Name returns the filter name.
func (f *ImportCollapseFilter) Name() string {
	return "import_collapse"
}

// Apply collapses import/require blocks in ModeAggressive; in ModeMinimal only
// collapses blocks with 5+ entries.
func (f *ImportCollapseFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	threshold := 5
	if mode == ModeAggressive {
		threshold = 2
	}

	original := core.EstimateTokens(input)
	output := f.collapseGoImports(input, threshold)
	output = f.collapseSequentialImports(output, es6ImportRe, "import", threshold)
	output = f.collapseSequentialImports(output, jsRequireRe, "require", threshold)
	output = f.collapseSequentialImports(output, pythonImportRe, "import", threshold)
	output = f.collapseSequentialImports(output, rustUseRe, "use", threshold)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// collapseGoImports collapses Go-style import() blocks.
func (f *ImportCollapseFilter) collapseGoImports(input string, threshold int) string {
	return goImportBlockRe.ReplaceAllStringFunc(input, func(match string) string {
		lines := strings.Split(match, "\n")
		// Count actual import entries (non-empty, non-brace lines)
		count := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && trimmed != "import (" && trimmed != ")" {
				count++
			}
		}
		if count < threshold {
			return match
		}
		return "import (/* " + strconv.Itoa(count) + " packages */)"
	})
}

// collapseSequentialImports collapses runs of N+ consecutive import lines.
func (f *ImportCollapseFilter) collapseSequentialImports(input string, re *regexp.Regexp, label string, threshold int) string {
	lines := strings.Split(input, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		// Find start of a run of matching lines
		if re.MatchString(lines[i]) {
			runStart := i
			for i < len(lines) && re.MatchString(lines[i]) {
				i++
			}
			count := i - runStart
			if count >= threshold {
				result = append(result, label+" (/* "+strconv.Itoa(count)+" statements */)")
			} else {
				result = append(result, lines[runStart:i]...)
			}
		} else {
			result = append(result, lines[i])
			i++
		}
	}
	return strings.Join(result, "\n")
}

