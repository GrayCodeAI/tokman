package commands

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// Language represents a programming language.
type Language string

const (
	LangRust       Language = "Rust"
	LangPython     Language = "Python"
	LangJavaScript Language = "JavaScript"
	LangTypeScript Language = "TypeScript"
	LangGo         Language = "Go"
	LangC          Language = "C"
	LangCpp        Language = "C++"
	LangJava       Language = "Java"
	LangRuby       Language = "Ruby"
	LangShell      Language = "Shell"
	LangUnknown    Language = "Code"
)

var smartModel string

// CodeSummary holds the 2-line summary.
type CodeSummary struct {
	line1 string
	line2 string
}

var smartCmd = &cobra.Command{
	Use:   "smart <file>",
	Short: "Generate 2-line technical summary (heuristic-based)",
	Long: `Analyze a source code file and generate a concise 2-line summary.

Uses heuristic-based analysis to identify:
- Language and file type
- Main components (functions, structs, classes)
- Key imports and dependencies
- Detected patterns (async, React hooks, etc.)

Examples:
  tokman smart main.go
  tokman smart src/index.ts`,
	Args: cobra.ExactArgs(1),
	RunE: runSmart,
}

func init() {
	rootCmd.AddCommand(smartCmd)
	smartCmd.Flags().StringVarP(&smartModel, "model", "m", "heuristic", "Model to use (heuristic)")
}

func runSmart(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Analyzing: %s\n", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lang := detectSmartLanguage(filePath)
	summary := analyzeCode(string(content), lang)

	fmt.Println(summary.line1)
	fmt.Println(summary.line2)

	return nil
}

func detectSmartLanguage(filePath string) Language {
	ext := strings.ToLower(filePath)
	if idx := strings.LastIndex(ext, "."); idx >= 0 {
		ext = ext[idx+1:]
	}

	switch ext {
	case "rs":
		return LangRust
	case "py":
		return LangPython
	case "js", "jsx", "mjs":
		return LangJavaScript
	case "ts", "tsx":
		return LangTypeScript
	case "go":
		return LangGo
	case "c":
		return LangC
	case "cpp", "cc", "cxx":
		return LangCpp
	case "java":
		return LangJava
	case "rb":
		return LangRuby
	case "sh", "bash", "zsh":
		return LangShell
	default:
		return LangUnknown
	}
}

func analyzeCode(content string, lang Language) CodeSummary {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Extract components
	imports := extractImports(content, lang)
	functions := extractFunctions(content, lang)
	structs := extractStructs(content, lang)
	traits := extractTraits(content, lang)

	// Detect patterns
	patterns := detectPatterns(content, lang)

	// Build line 1: What it is
	mainType := fmt.Sprintf("%s code", lang)
	if len(structs) > 0 && len(functions) > 0 {
		mainType = fmt.Sprintf("%s module", lang)
	} else if len(structs) > 0 {
		mainType = fmt.Sprintf("%s data structures", lang)
	} else if len(functions) > 0 {
		mainType = fmt.Sprintf("%s functions", lang)
	}

	var components []string
	if len(functions) > 0 {
		components = append(components, fmt.Sprintf("%d fn", len(functions)))
	}
	if len(structs) > 0 {
		components = append(components, fmt.Sprintf("%d struct", len(structs)))
	}
	if len(traits) > 0 {
		components = append(components, fmt.Sprintf("%d trait", len(traits)))
	}

	line1 := fmt.Sprintf("%s (%d lines)", mainType, totalLines)
	if len(components) > 0 {
		line1 = fmt.Sprintf("%s (%s) - %d lines", mainType, strings.Join(components, ", "), totalLines)
	}

	// Build line 2: Key details
	var details []string

	// Main imports/dependencies
	if len(imports) > 0 {
		keyImports := imports
		if len(keyImports) > 3 {
			keyImports = keyImports[:3]
		}
		details = append(details, fmt.Sprintf("uses: %s", strings.Join(keyImports, ", ")))
	}

	// Key patterns detected
	if len(patterns) > 0 {
		details = append(details, fmt.Sprintf("patterns: %s", strings.Join(patterns, ", ")))
	}

	// Main functions/structs
	if len(functions) > 0 && len(details) == 0 {
		keyFns := functions
		if len(keyFns) > 3 {
			keyFns = keyFns[:3]
		}
		details = append(details, fmt.Sprintf("defines: %s", strings.Join(keyFns, ", ")))
	}

	line2 := "General purpose code file"
	if len(details) > 0 {
		line2 = strings.Join(details, " | ")
	}

	return CodeSummary{line1: line1, line2: line2}
}

func extractImports(content string, lang Language) []string {
	var pattern string
	switch lang {
	case LangRust:
		pattern = `(?m)^use\s+([a-zA-Z_][a-zA-Z0-9_]*(?:::[a-zA-Z_][a-zA-Z0-9_]*)?)`
	case LangPython:
		pattern = `(?m)^(?:from\s+(\S+)|import\s+(\S+))`
	case LangJavaScript, LangTypeScript:
		pattern = `(?:import.*from\s+['"]([^'"]+)['"]|require\(['"]([^'"]+)['"]\))`
	case LangGo:
		pattern = `(?m)^\s*"([^"]+)"$`
	default:
		return nil
	}

	re := regexp.MustCompile(pattern)
	var imports []string
	seen := make(map[string]bool)

	for _, match := range re.FindAllStringSubmatch(content, -1) {
		var imp string
		if len(match) > 1 && match[1] != "" {
			imp = match[1]
		} else if len(match) > 2 && match[2] != "" {
			imp = match[2]
		}
		if imp != "" {
			// Get base package name
			base := strings.Split(imp, "::")[0]
			base = strings.Split(base, "/")[0]
			if !seen[base] && !isStdImport(base, lang) {
				seen[base] = true
				imports = append(imports, base)
			}
		}
	}

	if len(imports) > 5 {
		imports = imports[:5]
	}
	return imports
}

func isStdImport(name string, lang Language) bool {
	switch lang {
	case LangRust:
		return name == "std" || name == "core" || name == "alloc"
	case LangPython:
		return name == "os" || name == "sys" || name == "re" || name == "json" || name == "typing"
	}
	return false
}

func extractFunctions(content string, lang Language) []string {
	var pattern string
	switch lang {
	case LangRust:
		pattern = `(?:pub\s+)?(?:async\s+)?fn\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case LangPython:
		pattern = `def\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case LangJavaScript, LangTypeScript:
		pattern = `(?:async\s+)?function\s+([a-zA-Z_][a-zA-Z0-9_]*)|(?:const|let|var)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(?:async\s+)?\(`
	case LangGo:
		pattern = `func\s+(?:\([^)]+\)\s+)?([a-zA-Z_][a-zA-Z0-9_]*)`
	default:
		return nil
	}

	re := regexp.MustCompile(pattern)
	var functions []string

	for _, match := range re.FindAllStringSubmatch(content, -1) {
		var name string
		if len(match) > 1 && match[1] != "" {
			name = match[1]
		} else if len(match) > 2 && match[2] != "" {
			name = match[2]
		}
		if name != "" && !strings.HasPrefix(name, "test_") && name != "main" && name != "new" {
			functions = append(functions, name)
		}
	}

	if len(functions) > 10 {
		functions = functions[:10]
	}
	return functions
}

func extractStructs(content string, lang Language) []string {
	var pattern string
	switch lang {
	case LangRust:
		pattern = `(?:pub\s+)?(?:struct|enum)\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case LangPython:
		pattern = `class\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case LangTypeScript:
		pattern = `(?:interface|class|type)\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case LangGo:
		pattern = `type\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+struct`
	case LangJava:
		pattern = `(?:public\s+)?class\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	default:
		return nil
	}

	re := regexp.MustCompile(pattern)
	var structs []string

	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			structs = append(structs, match[1])
		}
	}

	if len(structs) > 10 {
		structs = structs[:10]
	}
	return structs
}

func extractTraits(content string, lang Language) []string {
	var pattern string
	switch lang {
	case LangRust:
		pattern = `(?:pub\s+)?trait\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case LangTypeScript:
		pattern = `interface\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	default:
		return nil
	}

	re := regexp.MustCompile(pattern)
	var traits []string

	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			traits = append(traits, match[1])
		}
	}

	if len(traits) > 5 {
		traits = traits[:5]
	}
	return traits
}

func detectPatterns(content string, lang Language) []string {
	var patterns []string

	// Common patterns
	if strings.Contains(content, "async") && strings.Contains(content, "await") {
		patterns = append(patterns, "async")
	}

	switch lang {
	case LangRust:
		if strings.Contains(content, "impl") && strings.Contains(content, "for") {
			patterns = append(patterns, "trait impl")
		}
		if strings.Contains(content, "#[derive") {
			patterns = append(patterns, "derive")
		}
		if strings.Contains(content, "Result<") || strings.Contains(content, "anyhow::") {
			patterns = append(patterns, "error handling")
		}
		if strings.Contains(content, "#[test]") {
			patterns = append(patterns, "tests")
		}
	case LangPython:
		if strings.Contains(content, "@dataclass") {
			patterns = append(patterns, "dataclass")
		}
		if strings.Contains(content, "def __init__") {
			patterns = append(patterns, "OOP")
		}
	case LangJavaScript, LangTypeScript:
		if strings.Contains(content, "useState") || strings.Contains(content, "useEffect") {
			patterns = append(patterns, "React hooks")
		}
		if strings.Contains(content, "export default") {
			patterns = append(patterns, "ES modules")
		}
	}

	if len(patterns) > 3 {
		patterns = patterns[:3]
	}
	return patterns
}
