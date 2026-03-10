package filter

import (
	"regexp"
	"strings"
)

// ImportFilter condenses import statements.
type ImportFilter struct {
	patterns []*regexp.Regexp
}

// NewImportFilter creates a new import filter.
func NewImportFilter() *ImportFilter {
	return &ImportFilter{
		patterns: ImportPatterns,
	}
}

// Name returns the filter name.
func (f *ImportFilter) Name() string {
	return "import"
}

// Apply condenses import statements and returns token savings.
func (f *ImportFilter) Apply(input string, mode Mode) (string, int) {
	original := len(input)

	// Check if this looks like code with imports
	if !IsCode(input) {
		return input, 0
	}

	lang := DetectLanguageFromInput(input)
	var output string

	switch lang {
	case LangGo:
		output = f.condenseGoImports(input)
	case LangRust:
		output = f.condenseRustImports(input)
	case LangPython:
		output = f.condensePythonImports(input)
	case LangJavaScript, LangTypeScript:
		output = f.condenseJSImports(input)
	default:
		output = f.condenseGenericImports(input)
	}

	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// condenseGoImports condenses Go import blocks.
func (f *ImportFilter) condenseGoImports(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	inImportBlock := false
	importCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle import block start
		if trimmed == "import (" {
			inImportBlock = true
			importCount = 0
			continue
		}

		// Handle import block end
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
			if importCount > 0 {
				result = append(result, "// imports condensed")
			}
			continue
		}

		// Skip lines inside import block
		if inImportBlock {
			importCount++
			continue
		}

		// Handle single imports
		if strings.HasPrefix(trimmed, "import ") && !strings.Contains(trimmed, "(") {
			importCount++
			continue
		}

		result = append(result, line)
	}

	// Add import count if any were condensed
	if importCount > 0 {
		result = append([]string{""}, result...)
		result = append([]string{""}, result...)
	}

	return strings.Join(result, "\n")
}

// condenseRustImports condenses Rust use statements.
func (f *ImportFilter) condenseRustImports(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	importCount := 0
	inUseBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for multi-line use statement
		if strings.HasPrefix(trimmed, "use ") {
			if strings.HasSuffix(trimmed, "{") {
				inUseBlock = true
			}
			if strings.HasSuffix(trimmed, ";") && !inUseBlock {
				importCount++
				continue
			}
			importCount++
			continue
		}

		// Handle multi-line use block
		if inUseBlock {
			if strings.HasSuffix(trimmed, "};") {
				inUseBlock = false
			}
			continue
		}

		result = append(result, line)
	}

	if importCount > 0 {
		// Insert use statement count at the top
		header := "// imports condensed"
		result = append([]string{header, ""}, result...)
	}

	return strings.Join(result, "\n")
}

// condensePythonImports condenses Python import statements.
func (f *ImportFilter) condensePythonImports(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	importCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle import statements
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			importCount++
			continue
		}

		result = append(result, line)
	}

	if importCount > 0 {
		header := "# imports condensed"
		result = append([]string{header, ""}, result...)
	}

	return strings.Join(result, "\n")
}

// condenseJSImports condenses JavaScript/TypeScript imports.
func (f *ImportFilter) condenseJSImports(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	importCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle various import styles
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "export ") ||
			strings.HasPrefix(trimmed, "require(") {
			importCount++
			continue
		}

		result = append(result, line)
	}

	if importCount > 0 {
		header := "// imports condensed"
		result = append([]string{header, ""}, result...)
	}

	return strings.Join(result, "\n")
}

// condenseGenericImports provides generic import condensation.
func (f *ImportFilter) condenseGenericImports(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	importCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isImport := false

		for _, pattern := range f.patterns {
			if pattern.MatchString(trimmed) {
				isImport = true
				break
			}
		}

		if isImport {
			importCount++
			continue
		}

		result = append(result, line)
	}

	if importCount > 0 {
		header := "// imports condensed"
		result = append([]string{header, ""}, result...)
	}

	return strings.Join(result, "\n")
}
