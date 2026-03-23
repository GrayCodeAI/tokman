package analysis

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
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
	registry.Add(func() { registry.Register(smartCmd) })
	smartCmd.Flags().StringVarP(&smartModel, "model", "m", "heuristic", "Model to use (heuristic)")
}

func runSmart(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	if shared.Verbose > 0 {
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

func detectSmartLanguage(filePath string) filter.Language {
	ext := strings.ToLower(filePath)
	if idx := strings.LastIndex(ext, "."); idx >= 0 {
		ext = ext[idx+1:]
	}

	switch ext {
	case "rs":
		return filter.LangRust
	case "py":
		return filter.LangPython
	case "js", "jsx", "mjs":
		return filter.LangJavaScript
	case "ts", "tsx":
		return filter.LangTypeScript
	case "go":
		return filter.LangGo
	case "c":
		return filter.LangC
	case "cpp", "cc", "cxx":
		return filter.LangCpp
	case "java":
		return filter.LangJava
	case "rb":
		return filter.LangRuby
	case "sh", "bash", "zsh":
		return filter.LangShell
	case "sql":
		return filter.LangSQL
	default:
		return filter.LangUnknown
	}
}

func analyzeCode(content string, lang filter.Language) CodeSummary {
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

func extractImports(content string, lang filter.Language) []string {
	var pattern string
	switch lang {
	case filter.LangRust:
		pattern = `(?m)^use\s+([a-zA-Z_][a-zA-Z0-9_]*(?:::[a-zA-Z_][a-zA-Z0-9_]*)?)`
	case filter.LangPython:
		pattern = `(?m)^(?:from\s+(\S+)|import\s+(\S+))`
	case filter.LangJavaScript, filter.LangTypeScript:
		pattern = `(?:import.*from\s+['"]([^'"]+)['"]|require\(['"]([^'"]+)['"]\))`
	case filter.LangGo:
		pattern = `(?m)^\s*"([^"]+)"$`
	case filter.LangJava:
		pattern = `(?m)^import\s+`
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

func isStdImport(name string, lang filter.Language) bool {
	switch lang {
	case filter.LangRust:
		return name == "std" || name == "core" || name == "alloc"
	case filter.LangPython:
		return name == "os" || name == "sys" || name == "re" || name == "json" || name == "typing"
	case filter.LangJava:
		return strings.HasPrefix(name, "java.") || strings.HasPrefix(name, "javax.") || strings.HasPrefix(name, "jak.")
	}
	return false
}

func extractFunctions(content string, lang filter.Language) []string {
	var pattern string
	switch lang {
	case filter.LangRust:
		pattern = `(?:pub\s+)?(?:async\s+)?fn\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case filter.LangPython:
		pattern = `def\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case filter.LangJavaScript, filter.LangTypeScript:
		pattern = `(?:async\s+)?function\s+([a-zA-Z_][a-zA-Z0-9_]*)|(?:const|let|var)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(?:async\s+)?\(`
	case filter.LangGo:
		pattern = `func\s+(?:\([^)]+\)\s+)?([a-zA-Z_][a-zA-Z0-9_]*)`
	case filter.LangJava:
		pattern = `(?:(?:public|private|protected)\s+)?(?:static\s+)?(?:synchronized\s+)?(?:abstract\s+)?(?:final\s+)?([a-zA-Z_][a-zA-Z0-9_]*)\s+\w+\s*\(`
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

func extractStructs(content string, lang filter.Language) []string {
	var pattern string
	switch lang {
	case filter.LangRust:
		pattern = `(?:pub\s+)?(?:struct|enum)\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case filter.LangPython:
		pattern = `class\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case filter.LangTypeScript:
		pattern = `(?:interface|class|type)\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case filter.LangGo:
		pattern = `type\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+struct`
	case filter.LangJava:
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

func extractTraits(content string, lang filter.Language) []string {
	var pattern string
	switch lang {
	case filter.LangRust:
		pattern = `(?:pub\s+)?trait\s+([a-zA-Z_][a-zA-Z0-9_]*)`
	case filter.LangTypeScript:
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

func detectPatterns(content string, lang filter.Language) []string {
	var patterns []string

	// Common patterns
	if strings.Contains(content, "async") && strings.Contains(content, "await") {
		patterns = append(patterns, "async")
	}

	switch lang {
	case filter.LangRust:
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
	case filter.LangPython:
		if strings.Contains(content, "@dataclass") {
			patterns = append(patterns, "dataclass")
		}
		if strings.Contains(content, "def __init__") {
			patterns = append(patterns, "OOP")
		}
	case filter.LangJavaScript, filter.LangTypeScript:
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
