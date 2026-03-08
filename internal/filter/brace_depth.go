package filter

import (
	"regexp"
	"strings"
)

// BodyFilter strips function bodies in aggressive mode.
// Preserves function signatures while removing body content.
type BodyFilter struct {
	signaturePatterns []*regexp.Regexp
}

// NewBodyFilter creates a new body filter.
func NewBodyFilter() *BodyFilter {
	return &BodyFilter{
		signaturePatterns: SignaturePatterns,
	}
}

// Name returns the filter name.
func (f *BodyFilter) Name() string {
	return "body"
}

// Apply strips function bodies and returns token savings.
func (f *BodyFilter) Apply(input string, mode Mode) (string, int) {
	// Only apply in aggressive mode
	if mode != ModeAggressive {
		return input, 0
	}

	// Check if this looks like code
	if !IsCode(input) {
		return input, 0
	}

	original := len(input)
	lang := DetectLanguage(input)
	
	var output string
	switch lang {
	case "go":
		output = f.stripGoBodies(input)
	case "rust":
		output = f.stripRustBodies(input)
	case "python":
		output = f.stripPythonBodies(input)
	case "javascript", "typescript":
		output = f.stripJSBodies(input)
	default:
		output = f.stripGenericBodies(input)
	}

	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// stripGoBodies strips function bodies from Go code.
func (f *BodyFilter) stripGoBodies(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	depth := 0
	inFunc := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this starts a function
		if f.isFunctionStart(trimmed, "go") && depth == 0 {
			inFunc = true
			result = append(result, line)
			
			// Check if function body is on same line
			if strings.Contains(line, "{") {
				depth += strings.Count(line, "{")
				depth -= strings.Count(line, "}")
				if depth == 0 {
					// Single line function, keep as-is
					inFunc = false
				} else {
					// Replace body with placeholder
					idx := strings.Index(line, "{")
					result[len(result)-1] = line[:idx+1] + " /* body stripped */ }"
					depth = 0
					inFunc = false
				}
			}
			continue
		}

		// Track brace depth
		if inFunc {
			depth += strings.Count(line, "{")
			depth -= strings.Count(line, "}")

			if depth <= 0 {
				inFunc = false
				// Output closing brace
				result = append(result, "}")
			}
			// Skip body content
			continue
		}

		// Not in function, output line
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// stripRustBodies strips function bodies from Rust code.
func (f *BodyFilter) stripRustBodies(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	depth := 0
	inFunc := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for function start
		if f.isFunctionStart(trimmed, "rust") && depth == 0 {
			inFunc = true
			result = append(result, line)
			
			// Handle brace on same line
			if strings.Contains(line, "{") {
				depth += strings.Count(line, "{")
				depth -= strings.Count(line, "}")
				
				if depth > 0 {
					// Replace with placeholder
					idx := strings.Index(line, "{")
					result[len(result)-1] = line[:idx+1] + " /* body stripped */ }"
					depth = 0
					inFunc = false
				}
			}
			continue
		}

		// Track depth in function body
		if inFunc {
			depth += strings.Count(line, "{")
			depth -= strings.Count(line, "}")
			
			// Also track braces in string contexts (simplified)
			depth += strings.Count(line, "(")
			depth -= strings.Count(line, ")")

			if depth <= 0 {
				inFunc = false
				result = append(result, "}")
			}
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// stripPythonBodies strips function bodies from Python code.
func (f *BodyFilter) stripPythonBodies(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	inFunc := false
	funcIndent := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check for function/class start
		if f.isFunctionStart(trimmed, "python") {
			inFunc = true
			funcIndent = len(line) - len(strings.TrimLeft(line, " \t"))
			result = append(result, line)
			result = append(result, strings.Repeat(" ", funcIndent+4) + "pass  # body stripped")
			continue
		}

		if inFunc {
			// Check if we're still inside the function
			if line == "" {
				continue
			}
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if currentIndent <= funcIndent && trimmed != "" {
				inFunc = false
			} else {
				continue // Skip body
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// stripJSBodies strips function bodies from JavaScript/TypeScript code.
func (f *BodyFilter) stripJSBodies(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	depth := 0
	inFunc := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if f.isFunctionStart(trimmed, "javascript") && depth == 0 {
			inFunc = true
			result = append(result, line)
			
			if strings.Contains(line, "{") {
				depth += strings.Count(line, "{")
				depth -= strings.Count(line, "}")
				
				if depth > 0 {
					idx := strings.Index(line, "{")
					result[len(result)-1] = line[:idx+1] + " /* body stripped */ }"
					depth = 0
					inFunc = false
				}
			}
			continue
		}

		if inFunc {
			depth += strings.Count(line, "{")
			depth -= strings.Count(line, "}")

			if depth <= 0 {
				inFunc = false
				result = append(result, "}")
			}
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// stripGenericBodies provides generic body stripping.
func (f *BodyFilter) stripGenericBodies(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	depth := 0
	inFunc := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if f.isFunctionStart(trimmed, "unknown") && depth == 0 {
			inFunc = true
			result = append(result, line)
			
			if strings.Contains(line, "{") {
				depth += strings.Count(line, "{")
				depth -= strings.Count(line, "}")
				
				if depth > 0 {
					idx := strings.Index(line, "{")
					if idx >= 0 {
						result[len(result)-1] = line[:idx+1] + " /* body stripped */ }"
					}
					depth = 0
					inFunc = false
				}
			}
			continue
		}

		if inFunc {
			depth += strings.Count(line, "{")
			depth -= strings.Count(line, "}")

			if depth <= 0 {
				inFunc = false
				result = append(result, "}")
			}
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// isFunctionStart checks if a line starts a function definition.
func (f *BodyFilter) isFunctionStart(line string, lang string) bool {
	for _, pattern := range f.signaturePatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

// StripBodies is a utility function to strip function bodies.
func StripBodies(code string, lang string, mode Mode) string {
	filter := NewBodyFilter()
	output, _ := filter.Apply(code, mode)
	return output
}
