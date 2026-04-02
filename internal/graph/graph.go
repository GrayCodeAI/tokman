package graph

import (
	"bufio"
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ProjectGraph represents the dependency graph of a project.
// Inspired by lean-ctx's ctx_graph.
type ProjectGraph struct {
	mu      sync.RWMutex
	nodes   map[string]*Node
	edges   map[string][]string
	rootDir string
	module  string
}

// Node represents a file in the project graph.
type Node struct {
	Path       string
	Language   string
	Size       int64
	Imports    []string
	Symbols    []string
	References []string
	ImportedBy []string
	Tags       []string
}

// NewProjectGraph creates a new project graph.
func NewProjectGraph(rootDir string) *ProjectGraph {
	return &ProjectGraph{
		nodes:   make(map[string]*Node),
		edges:   make(map[string][]string),
		rootDir: rootDir,
		module:  readModuleName(rootDir),
	}
}

// Analyze scans the project and builds the dependency graph.
func (g *ProjectGraph) Analyze() error {
	if err := filepath.Walk(g.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		lang := detectLanguage(path)
		if lang == "" {
			return nil
		}

		relPath, _ := filepath.Rel(g.rootDir, path)
		node := &Node{
			Path:     relPath,
			Language: lang,
			Size:     info.Size(),
		}

		imports := extractImports(path, lang)
		node.Imports = imports
		symbols, refs := extractSymbols(path, lang)
		node.Symbols = symbols
		node.References = refs

		g.mu.Lock()
		g.nodes[relPath] = node
		g.mu.Unlock()

		return nil
	}); err != nil {
		return err
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.edges = make(map[string][]string)
	for relPath, node := range g.nodes {
		for _, imp := range node.Imports {
			if resolved, ok := g.resolveImport(relPath, imp); ok {
				g.edges[relPath] = append(g.edges[relPath], resolved)
			}
		}
	}
	return nil
}

// FindRelatedFiles finds files related to the given file through dependencies.
func (g *ProjectGraph) FindRelatedFiles(path string, maxResults int) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	related := make(map[string]int)

	// Direct imports
	for _, imp := range g.edges[path] {
		related[imp] += 20
	}

	// Files that import this file
	for file, imports := range g.edges {
		for _, imp := range imports {
			if imp == path {
				related[file] += 16
			}
		}
	}

	// Two-hop neighbors: files imported by my imports and reverse import chains.
	for _, imp := range g.edges[path] {
		for _, second := range g.edges[imp] {
			if second != path {
				related[second] += 6
			}
		}
	}
	for file, imports := range g.edges {
		for _, imp := range imports {
			if imp == path {
				for _, second := range g.edges[file] {
					if second != path {
						related[second] += 4
					}
				}
			}
		}
	}

	// Same directory files
	dir := filepath.Dir(path)
	for file := range g.nodes {
		if filepath.Dir(file) == dir && file != path {
			related[file] += 5
		}
	}

	// Prefer files whose symbols are referenced by the target file and vice versa.
	if target, ok := g.nodes[path]; ok {
		for file, node := range g.nodes {
			if file == path {
				continue
			}
			defMatches := overlapScore(target.References, node.Symbols)
			refMatches := overlapScore(node.References, target.Symbols)
			related[file] += defMatches*4 + refMatches*3
		}
	}

	// Prefer matching filenames and matching extensions.
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	ext := filepath.Ext(path)
	for file := range g.nodes {
		if file == path {
			continue
		}
		if filepath.Ext(file) == ext {
			related[file] += 2
		}
		if strings.Contains(filepath.Base(file), base) || strings.Contains(base, strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))) {
			related[file] += 3
		}
	}

	// Sort by score
	type kv struct {
		path  string
		score int
	}
	var pairs []kv
	for k, v := range related {
		if _, ok := g.nodes[k]; ok && k != path {
			pairs = append(pairs, kv{k, v})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].score == pairs[j].score {
			return pairs[i].path < pairs[j].path
		}
		return pairs[i].score > pairs[j].score
	})

	var results []string
	for i := 0; i < len(pairs) && i < maxResults; i++ {
		results = append(results, pairs[i].path)
	}
	return results
}

// ImpactAnalysis finds all files affected by a change to the given file.
func (g *ProjectGraph) ImpactAnalysis(path string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	affected := make(map[string]bool)
	queue := []string{path}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for file, imports := range g.edges {
			for _, imp := range imports {
				if imp == current && !affected[file] {
					affected[file] = true
					queue = append(queue, file)
				}
			}
		}
	}

	var results []string
	for f := range affected {
		results = append(results, f)
	}
	return results
}

// Stats returns project statistics.
func (g *ProjectGraph) Stats() map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()

	byLang := make(map[string]int)
	totalSize := int64(0)
	totalFiles := len(g.nodes)

	for _, node := range g.nodes {
		byLang[node.Language]++
		totalSize += node.Size
	}

	return map[string]any{
		"total_files": totalFiles,
		"total_size":  totalSize,
		"by_language": byLang,
		"total_edges": len(g.edges),
	}
}

func shouldSkipDir(path string) bool {
	skipDirs := []string{
		"node_modules", ".git", "vendor", "__pycache__",
		".tox", ".venv", "dist", "build", "target",
		".next", ".nuxt", ".svelte-kit",
	}
	base := filepath.Base(path)
	for _, d := range skipDirs {
		if base == d {
			return true
		}
	}
	return false
}

func detectLanguage(path string) string {
	ext := filepath.Ext(path)
	langMap := map[string]string{
		".go":    "go",
		".rs":    "rust",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".tsx":   "typescript",
		".jsx":   "javascript",
		".rb":    "ruby",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".cs":    "csharp",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".toml":  "toml",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".md":    "markdown",
	}
	return langMap[ext]
}

func extractImports(path string, lang string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)
	var imports []string

	switch lang {
	case "go":
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import ") {
				if strings.Contains(line, "\"") {
					start := strings.Index(line, "\"")
					end := strings.LastIndex(line, "\"")
					if start < end {
						imports = append(imports, line[start+1:end])
					}
				}
			}
		}
	case "python":
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					imports = append(imports, parts[1])
				}
			}
		}
	case "javascript", "typescript":
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "require(") {
				if strings.Contains(line, "from '") {
					start := strings.Index(line, "from '") + 6
					end := strings.Index(line[start:], "'")
					if end > 0 {
						imports = append(imports, line[start:start+end])
					}
				} else if strings.Contains(line, "from \"") {
					start := strings.Index(line, "from \"") + 6
					end := strings.Index(line[start:], "\"")
					if end > 0 {
						imports = append(imports, line[start:start+end])
					}
				}
			}
		}
	}

	return imports
}

var identRe = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\b`)
var pyDefRe = regexp.MustCompile(`^\s*(?:async\s+def|def|class)\s+([A-Za-z_][A-Za-z0-9_]*)`)
var tsDeclRe = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?(?:function|class|interface|type|const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)`)
var tsArrowRe = regexp.MustCompile(`^\s*(?:export\s+)?const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s*)?\(`)

func extractSymbols(path, lang string) ([]string, []string) {
	if lang == "go" {
		return extractGoSymbols(path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	symbols := map[string]struct{}{}
	refs := map[string]struct{}{}

	switch lang {
	case "python":
		for _, line := range lines {
			if match := pyDefRe.FindStringSubmatch(line); len(match) == 2 {
				symbols[match[1]] = struct{}{}
			}
		}
	case "javascript", "typescript":
		for _, line := range lines {
			if match := tsDeclRe.FindStringSubmatch(line); len(match) == 2 {
				symbols[match[1]] = struct{}{}
				continue
			}
			if match := tsArrowRe.FindStringSubmatch(line); len(match) == 2 {
				symbols[match[1]] = struct{}{}
			}
		}
	}

	for _, ident := range identRe.FindAllString(content, -1) {
		if len(ident) < 3 || isKeyword(ident) {
			continue
		}
		refs[ident] = struct{}{}
	}
	return mapKeys(symbols), mapKeys(refs)
}

func extractGoSymbols(path string) ([]string, []string) {
	fset := gotoken.NewFileSet()
	file, err := goparser.ParseFile(fset, path, nil, goparser.SkipObjectResolution)
	if err != nil {
		return extractGenericSymbols(path)
	}

	symbols := map[string]struct{}{}
	refs := map[string]struct{}{}

	goast.Inspect(file, func(n goast.Node) bool {
		switch node := n.(type) {
		case *goast.FuncDecl:
			symbols[node.Name.Name] = struct{}{}
			if node.Recv != nil && len(node.Recv.List) > 0 {
				if ident, ok := recvTypeName(node.Recv.List[0].Type); ok {
					symbols[ident] = struct{}{}
				}
			}
		case *goast.TypeSpec:
			symbols[node.Name.Name] = struct{}{}
		case *goast.ValueSpec:
			for _, name := range node.Names {
				symbols[name.Name] = struct{}{}
			}
		case *goast.SelectorExpr:
			refs[node.Sel.Name] = struct{}{}
		case *goast.Ident:
			if node.Name == "" || isKeyword(node.Name) || node.Obj != nil {
				return true
			}
			refs[node.Name] = struct{}{}
		}
		return true
	})

	for sym := range symbols {
		delete(refs, sym)
	}
	return mapKeys(symbols), mapKeys(refs)
}

func recvTypeName(expr goast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *goast.Ident:
		return t.Name, true
	case *goast.StarExpr:
		return recvTypeName(t.X)
	default:
		return "", false
	}
}

func extractGenericSymbols(path string) ([]string, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	symbols := map[string]struct{}{}
	refs := map[string]struct{}{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") {
			name := symbolAfter(trimmed, "func ")
			if name != "" {
				symbols[name] = struct{}{}
			}
		}
		if strings.HasPrefix(trimmed, "type ") {
			name := symbolAfter(trimmed, "type ")
			if name != "" {
				symbols[name] = struct{}{}
			}
		}
		if strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "const ") {
			parts := strings.Fields(trimmed)
			if len(parts) > 1 {
				symbols[trimPunctuation(parts[1])] = struct{}{}
			}
		}
	}
	for _, ident := range identRe.FindAllString(content, -1) {
		if len(ident) < 3 || isKeyword(ident) {
			continue
		}
		refs[ident] = struct{}{}
	}
	for sym := range symbols {
		delete(refs, sym)
	}
	return mapKeys(symbols), mapKeys(refs)
}

func symbolAfter(line, prefix string) string {
	value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "(") {
		if idx := strings.Index(value, ")"); idx >= 0 && idx+1 < len(value) {
			value = strings.TrimSpace(value[idx+1:])
		}
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return trimPunctuation(fields[0])
}

func trimPunctuation(value string) string {
	return strings.Trim(value, "({[,:;*")
}

func makeStringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func overlapScore(refs []string, symbols []string) int {
	if len(refs) == 0 || len(symbols) == 0 {
		return 0
	}
	refSet := makeStringSet(refs)
	score := 0
	for _, symbol := range symbols {
		if _, ok := refSet[symbol]; ok {
			score++
		}
	}
	return score
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isKeyword(ident string) bool {
	switch ident {
	case "func", "type", "var", "const", "package", "import", "return", "class", "def", "from", "for", "if", "else", "switch", "case", "interface", "async", "export", "pass":
		return true
	default:
		return false
	}
}

func (g *ProjectGraph) resolveImport(fromFile, imp string) (string, bool) {
	imp = strings.TrimSpace(imp)
	if imp == "" {
		return "", false
	}

	if strings.HasPrefix(imp, "./") || strings.HasPrefix(imp, "../") {
		candidate := filepath.Clean(filepath.Join(filepath.Dir(fromFile), imp))
		if resolved, ok := g.matchCandidate(candidate); ok {
			return resolved, true
		}
	}

	if g.module != "" && strings.HasPrefix(imp, g.module+"/") {
		candidate := strings.TrimPrefix(imp, g.module+"/")
		if resolved, ok := g.matchCandidate(candidate); ok {
			return resolved, true
		}
	}

	if resolved, ok := g.matchCandidate(imp); ok {
		return resolved, true
	}

	return "", false
}

func (g *ProjectGraph) matchCandidate(candidate string) (string, bool) {
	candidate = filepath.Clean(filepath.ToSlash(candidate))
	if _, ok := g.nodes[candidate]; ok {
		return candidate, true
	}

	for node := range g.nodes {
		if filepath.ToSlash(filepath.Dir(node)) == candidate {
			return node, true
		}
	}
	return "", false
}

func readModuleName(rootDir string) string {
	file, err := os.Open(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// FormatGraphStats returns a human-readable stats string.
func FormatGraphStats(stats map[string]any) string {
	return fmt.Sprintf("Graph: %d files, %d edges, %d languages",
		stats["total_files"], stats["total_edges"], len(stats["by_language"].(map[string]int)))
}
