package filter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// graphNode represents a single content node in the graph.
type graphNode struct {
	ID      string   `json:"id"`
	Content string   `json:"content"`
	Edges   []graphEdge `json:"edges"`
}

// graphEdge represents a directed relationship between two nodes.
type graphEdge struct {
	To      string `json:"to"`
	RelType string `json:"rel_type"`
}

// ContentGraph maps relationships between content segments.
type ContentGraph struct {
	mu    sync.RWMutex
	nodes map[string]*graphNode
}

// NewContentGraph creates an empty ContentGraph.
func NewContentGraph() *ContentGraph {
	return &ContentGraph{
		nodes: make(map[string]*graphNode),
	}
}

// AddNode adds a content node identified by id. If the node already exists
// the content is updated.
func (g *ContentGraph) AddNode(id, content string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if n, ok := g.nodes[id]; ok {
		n.Content = content
		return
	}
	g.nodes[id] = &graphNode{
		ID:      id,
		Content: content,
	}
}

// AddEdge adds a directed edge from→to with the given relationship type.
// Valid relType values: "imports", "calls", "similar", "references".
// The source node is created with empty content if it does not exist.
func (g *ContentGraph) AddEdge(from, to, relType string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.nodes[from]; !ok {
		g.nodes[from] = &graphNode{ID: from}
	}
	g.nodes[from].Edges = append(g.nodes[from].Edges, graphEdge{
		To:      to,
		RelType: relType,
	})
}

// FindRelated returns IDs of nodes reachable from id within maxDepth hops
// via BFS. The source node itself is not included in the result.
func (g *ContentGraph) FindRelated(id string, maxDepth int) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if maxDepth <= 0 {
		return nil
	}

	type entry struct {
		id    string
		depth int
	}

	visited := make(map[string]bool)
	visited[id] = true

	queue := []entry{{id: id, depth: 0}}
	var result []string

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.depth >= maxDepth {
			continue
		}

		node, ok := g.nodes[cur.id]
		if !ok {
			continue
		}

		for _, edge := range node.Edges {
			if !visited[edge.To] {
				visited[edge.To] = true
				result = append(result, edge.To)
				queue = append(queue, entry{id: edge.To, depth: cur.depth + 1})
			}
		}
	}

	return result
}

// ToJSON serializes the graph to a JSON string for debugging.
// Returns an empty JSON object string on error.
func (g *ContentGraph) ToJSON() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	type jsonGraph struct {
		Nodes []*graphNode `json:"nodes"`
	}

	nodes := make([]*graphNode, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}

	data, err := json.Marshal(jsonGraph{Nodes: nodes})
	if err != nil {
		return "{}"
	}
	return string(data)
}

// --- Regex patterns for Go source extraction ---

var (
	// goImportSingle matches: import "pkg/path"
	goImportSingle = regexp.MustCompile(`(?m)^\s*import\s+"([^"]+)"`)
	// goImportGroupLine matches lines inside an import block: "pkg/path" or alias "pkg/path"
	goImportGroupLine = regexp.MustCompile(`(?m)^\s*(?:\w+\s+)?"([^"]+)"`)
	// goFuncCall matches bare function/method calls: SomeName( or pkg.Method(
	goFuncCall = regexp.MustCompile(`\b([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)?)\s*\(`)
)

// extractGoImports returns the import paths found in Go source text.
func extractGoImports(src string) []string {
	var imports []string
	seen := make(map[string]bool)

	// Single-line imports
	for _, m := range goImportSingle.FindAllStringSubmatch(src, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			imports = append(imports, m[1])
		}
	}

	// Multi-line import blocks
	inBlock := false
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "import (" {
			inBlock = true
			continue
		}
		if inBlock && trimmed == ")" {
			inBlock = false
			continue
		}
		if inBlock {
			m := goImportGroupLine.FindStringSubmatch(trimmed)
			if m != nil && !seen[m[1]] {
				seen[m[1]] = true
				imports = append(imports, m[1])
			}
		}
	}

	return imports
}

// extractGoFuncCalls returns unique call identifiers found in Go source text.
func extractGoFuncCalls(src string) []string {
	var calls []string
	seen := make(map[string]bool)
	for _, m := range goFuncCall.FindAllStringSubmatch(src, -1) {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			calls = append(calls, name)
		}
	}
	return calls
}

// -- ContentGraphFilter --

// ContentGraphFilter builds a ContentGraph from code segments and uses
// it to decide compression aggressiveness: nodes with no relations
// (isolated nodes) are compressed harder.
type ContentGraphFilter struct {
	graph *ContentGraph
}

// NewContentGraphFilter creates a Filter that analyses Go import paths and
// function calls to populate a ContentGraph, then applies heavier compression
// to isolated nodes.
func NewContentGraphFilter() *ContentGraphFilter {
	return &ContentGraphFilter{
		graph: NewContentGraph(),
	}
}

// Name returns the filter name.
func (f *ContentGraphFilter) Name() string {
	return "content_graph"
}

// Apply builds graph relationships from code, then compresses isolated
// segments more aggressively.
func (f *ContentGraphFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Split input into logical segments (paragraphs / code blocks).
	segments := splitIntoSegments(input)
	if len(segments) == 0 {
		return input, 0
	}

	// Populate graph nodes.
	for i, seg := range segments {
		id := segmentID(i)
		f.graph.AddNode(id, seg)
	}

	// Build edges from imports and function calls.
	for i, seg := range segments {
		srcID := segmentID(i)

		imports := extractGoImports(seg)
		for _, imp := range imports {
			// Add an edge to a node representing this import path.
			f.graph.AddEdge(srcID, "import:"+imp, "imports")
		}

		calls := extractGoFuncCalls(seg)
		for _, call := range calls {
			// Find a segment that defines this symbol.
			for j, other := range segments {
				if i == j {
					continue
				}
				if definesSymbol(other, call) {
					f.graph.AddEdge(srcID, segmentID(j), "calls")
				}
			}
		}
	}

	// Apply compression: isolated segments get compressed harder.
	var out strings.Builder
	original := len(input)

	for i, seg := range segments {
		id := segmentID(i)
		related := f.graph.FindRelated(id, 2)
		isIsolated := len(related) == 0

		if isIsolated && mode == ModeAggressive {
			// Hard compression: keep first sentence / first line only.
			compressed := firstLine(seg)
			out.WriteString(compressed)
		} else if isIsolated && mode == ModeMinimal {
			// Mild compression: trim trailing blank lines.
			out.WriteString(strings.TrimRight(seg, "\n\t "))
		} else {
			out.WriteString(seg)
		}

		if i < len(segments)-1 {
			out.WriteString("\n\n")
		}
	}

	result := out.String()
	saved := (original - len(result)) / 4
	if saved < 0 {
		saved = 0
	}
	return result, saved
}

// Graph returns the underlying ContentGraph for inspection.
func (f *ContentGraphFilter) Graph() *ContentGraph {
	return f.graph
}

// --- helpers ---

func segmentID(i int) string {
	return fmt.Sprintf("seg:%d", i)
}

// splitIntoSegments splits text on double-newline boundaries.
func splitIntoSegments(input string) []string {
	raw := strings.Split(input, "\n\n")
	var segs []string
	for _, s := range raw {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			segs = append(segs, trimmed)
		}
	}
	return segs
}

// definesSymbol reports whether src contains a definition for name
// (func, type, var, const declaration).
func definesSymbol(src, name string) bool {
	patterns := []string{
		"func " + name,
		"func (" ,
		"type " + name,
		"var " + name,
		"const " + name,
	}
	for _, p := range patterns {
		if p == "func (" {
			continue
		}
		if strings.Contains(src, p) {
			return true
		}
	}
	// Method receiver: func (r Recv) Name(
	methodRe := regexp.MustCompile(`func\s+\(\s*\w+\s+\*?\w+\s*\)\s*` + regexp.QuoteMeta(name) + `\s*\(`)
	return methodRe.MatchString(src)
}

// firstLine returns only the first non-empty line of s.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
	return s
}
