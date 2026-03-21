package filter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// MultiFileOptimizer optimizes token usage across multiple files
// by deduplicating shared context (imports, types, patterns)
type MultiFileOptimizer struct {
	mu sync.RWMutex

	// Shared context extracted from files
	SharedImports  map[string]int // import -> occurrence count
	SharedTypes    map[string]int // type -> occurrence count
	SharedPatterns map[string]int // pattern -> occurrence count

	// Per-file context cache
	FileCache map[string]*FileContext

	// Session tracking
	SessionSeen map[string]bool // files seen in current session
}

// FileContext represents extracted context from a file
type FileContext struct {
	Path         string
	Imports      []string
	Types        []string
	Functions    []string
	Constants    []string
	Patterns     []string
	TokenCount   int
	LastModified int64
}

// OptimizerConfig holds configuration for MultiFileOptimizer
type OptimizerConfig struct {
	MinOccurrences int  // Minimum occurrences to consider "shared"
	PreserveOrder  bool // Keep original order of declarations
	DenseNotation  bool // Use compact notation for output
}

// NewMultiFileOptimizer creates a new multi-file optimizer
func NewMultiFileOptimizer() *MultiFileOptimizer {
	return &MultiFileOptimizer{
		SharedImports:  make(map[string]int),
		SharedTypes:    make(map[string]int),
		SharedPatterns: make(map[string]int),
		FileCache:      make(map[string]*FileContext),
		SessionSeen:    make(map[string]bool),
	}
}

// Optimize optimizes multiple files by extracting and deduplicating shared context
func (m *MultiFileOptimizer) Optimize(files map[string]string, cfg OptimizerConfig) map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.MinOccurrences == 0 {
		cfg.MinOccurrences = 2
	}

	// Phase 1: Extract context from all files
	for path, content := range files {
		ctx := m.extractContext(path, content)
		m.FileCache[path] = ctx
		m.SessionSeen[path] = true

		// Accumulate shared context
		for _, imp := range ctx.Imports {
			m.SharedImports[imp]++
		}
		for _, typ := range ctx.Types {
			m.SharedTypes[typ]++
		}
	}

	// Phase 2: Identify truly shared elements
	sharedImports := m.getSharedElements(m.SharedImports, cfg.MinOccurrences)
	sharedTypes := m.getSharedElements(m.SharedTypes, cfg.MinOccurrences)

	// Phase 3: Optimize each file by removing redundant shared context
	result := make(map[string]string)
	for path, content := range files {
		result[path] = m.optimizeFile(content, sharedImports, sharedTypes, cfg)
	}

	return result
}

// extractContext extracts structural context from a file
func (m *MultiFileOptimizer) extractContext(path, content string) *FileContext {
	ctx := &FileContext{
		Path:       path,
		TokenCount: estimateTokens(content),
	}

	// Extract imports (language-agnostic patterns)
	ctx.Imports = m.extractImports(content)

	// Extract type definitions
	ctx.Types = m.extractTypes(content)

	// Extract function signatures
	ctx.Functions = m.extractFunctions(content)

	// Extract constants
	ctx.Constants = m.extractConstants(content)

	return ctx
}

// extractImports extracts import statements
func (m *MultiFileOptimizer) extractImports(content string) []string {
	var imports []string

	// Go imports
	goImport := regexp.MustCompile(`import\s+(?:\w+\s+)?"([^"]+)"`)
	for _, match := range goImport.FindAllStringSubmatch(content, -1) {
		imports = append(imports, match[1])
	}

	// Python imports
	pyImport := regexp.MustCompile(`(?:from\s+(\S+)\s+)?import\s+([^\n]+)`)
	for _, match := range pyImport.FindAllStringSubmatch(content, -1) {
		if match[1] != "" {
			imports = append(imports, match[1])
		} else {
			imports = append(imports, strings.Split(match[2], ",")[0])
		}
	}

	// JS/TS imports
	jsImport := regexp.MustCompile(`import\s+.*?from\s+['"]([^'"]+)['"]`)
	for _, match := range jsImport.FindAllStringSubmatch(content, -1) {
		imports = append(imports, match[1])
	}

	// Rust imports
	rustImport := regexp.MustCompile(`use\s+([^;]+);`)
	for _, match := range rustImport.FindAllStringSubmatch(content, -1) {
		imports = append(imports, match[1])
	}

	return uniqueStrings(imports)
}

// extractTypes extracts type definitions
func (m *MultiFileOptimizer) extractTypes(content string) []string {
	var types []string

	// Go types
	goType := regexp.MustCompile(`type\s+(\w+)\s+(?:struct|interface)`)
	for _, match := range goType.FindAllStringSubmatch(content, -1) {
		types = append(types, match[1])
	}

	// Python classes
	pyClass := regexp.MustCompile(`class\s+(\w+)`)
	for _, match := range pyClass.FindAllStringSubmatch(content, -1) {
		types = append(types, match[1])
	}

	// JS/TS classes and interfaces
	jsClass := regexp.MustCompile(`(?:class|interface)\s+(\w+)`)
	for _, match := range jsClass.FindAllStringSubmatch(content, -1) {
		types = append(types, match[1])
	}

	// Rust structs and enums
	rustType := regexp.MustCompile(`(?:struct|enum)\s+(\w+)`)
	for _, match := range rustType.FindAllStringSubmatch(content, -1) {
		types = append(types, match[1])
	}

	return uniqueStrings(types)
}

// extractFunctions extracts function signatures
func (m *MultiFileOptimizer) extractFunctions(content string) []string {
	var functions []string

	// Go functions
	goFunc := regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?(\w+)\s*\([^)]*\)`)
	for _, match := range goFunc.FindAllStringSubmatch(content, -1) {
		functions = append(functions, match[1])
	}

	// Python functions
	pyFunc := regexp.MustCompile(`def\s+(\w+)\s*\(`)
	for _, match := range pyFunc.FindAllStringSubmatch(content, -1) {
		functions = append(functions, match[1])
	}

	// JS/TS functions
	jsFunc := regexp.MustCompile(`(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\())`)
	for _, match := range jsFunc.FindAllStringSubmatch(content, -1) {
		if match[1] != "" {
			functions = append(functions, match[1])
		} else if match[2] != "" {
			functions = append(functions, match[2])
		}
	}

	// Rust functions
	rustFunc := regexp.MustCompile(`fn\s+(\w+)\s*[<\(]`)
	for _, match := range rustFunc.FindAllStringSubmatch(content, -1) {
		functions = append(functions, match[1])
	}

	return uniqueStrings(functions)
}

// extractConstants extracts constant definitions
func (m *MultiFileOptimizer) extractConstants(content string) []string {
	var constants []string

	// Go constants
	goConst := regexp.MustCompile(`const\s+(\w+)\s*=`)
	for _, match := range goConst.FindAllStringSubmatch(content, -1) {
		constants = append(constants, match[1])
	}

	// Python constants (UPPER_CASE)
	pyConst := regexp.MustCompile(`^([A-Z][A-Z0-9_]*)\s*=`)
	for _, match := range pyConst.FindAllStringSubmatch(content, -1) {
		constants = append(constants, match[1])
	}

	// JS/TS constants
	jsConst := regexp.MustCompile(`const\s+(\w+)\s*=`)
	for _, match := range jsConst.FindAllStringSubmatch(content, -1) {
		constants = append(constants, match[1])
	}

	return uniqueStrings(constants)
}

// getSharedElements returns elements appearing at least minOccur times
func (m *MultiFileOptimizer) getSharedElements(counts map[string]int, minOccur int) map[string]bool {
	shared := make(map[string]bool)
	for elem, count := range counts {
		if count >= minOccur {
			shared[elem] = true
		}
	}
	return shared
}

// optimizeFile optimizes a single file by removing redundant shared context
func (m *MultiFileOptimizer) optimizeFile(content string, sharedImports, sharedTypes map[string]bool, cfg OptimizerConfig) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip shared imports if we've seen them before
		if sharedImports != nil && m.isSharedImportLine(trimmed, sharedImports) {
			continue
		}

		// Keep the line
		result = append(result, line)

		// Use dense notation if enabled
		if cfg.DenseNotation && len(result) > 0 {
			// Could transform to dense notation here
		}
	}

	return strings.Join(result, "\n")
}

// isSharedImportLine checks if a line is a shared import
func (m *MultiFileOptimizer) isSharedImportLine(line string, sharedImports map[string]bool) bool {
	// Check various import patterns
	for imp := range sharedImports {
		if strings.Contains(line, imp) &&
			(strings.Contains(line, "import") ||
				strings.Contains(line, "use ") ||
				strings.Contains(line, "from ")) {
			return true
		}
	}
	return false
}

// GetSessionSummary returns a summary of context seen in the session
func (m *MultiFileOptimizer) GetSessionSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("## Session Context Summary\n\n")

	if len(m.SharedImports) > 0 {
		sb.WriteString("### Shared Imports\n")
		for _, sc := range m.sortByCount(m.SharedImports) {
			if sc.count >= 2 {
				sb.WriteString(fmt.Sprintf("- `%s` (%d files)\n", sc.str, sc.count))
			}
		}
		sb.WriteString("\n")
	}

	if len(m.SharedTypes) > 0 {
		sb.WriteString("### Shared Types\n")
		for _, sc := range m.sortByCount(m.SharedTypes) {
			if sc.count >= 2 {
				sb.WriteString(fmt.Sprintf("- `%s` (%d files)\n", sc.str, sc.count))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("**Files analyzed:** %d\n", len(m.SessionSeen)))

	return sb.String()
}

// sortByCount returns elements sorted by count descending
func (m *MultiFileOptimizer) sortByCount(counts map[string]int) []stringCount {
	var sc []stringCount
	for s, c := range counts {
		sc = append(sc, stringCount{s, c})
	}
	sort.Slice(sc, func(i, j int) bool {
		return sc[i].count > sc[j].count
	})
	return sc
}

type stringCount struct {
	str   string
	count int
}

// uniqueStrings returns unique strings
func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, str := range s {
		if !seen[str] && str != "" {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

// Reset clears the session state
func (m *MultiFileOptimizer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SharedImports = make(map[string]int)
	m.SharedTypes = make(map[string]int)
	m.SharedPatterns = make(map[string]int)
	m.FileCache = make(map[string]*FileContext)
	m.SessionSeen = make(map[string]bool)
}
