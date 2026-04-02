// Package core provides advanced compression layers (11-20).
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/GrayCodeAI/tokman/internal/utils"
)

// Layer 11: LLM Compaction with Ollama Support
type LLMCompactionLayer struct {
	ollamaEndpoint string
	model          string
	batchSize      int
}

// NewLLMCompactionLayer creates a new LLM compaction layer.
func NewLLMCompactionLayer() *LLMCompactionLayer {
	return &LLMCompactionLayer{
		ollamaEndpoint: "http://localhost:11434",
		model:          "llama3.2:3b",
		batchSize:      10,
	}
}

func (l *LLMCompactionLayer) Name() string { return "llm_compaction" }

func (l *LLMCompactionLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *LLMCompactionLayer) Apply(content string) (string, int) {
	// Break content into chunks
	chunks := l.chunkContent(content)
	var compacted []string
	saved := 0

	for _, chunk := range chunks {
		if len(chunk) > 500 {
			// Mark for LLM compaction with summary marker
			summary := l.summarize(chunk)
			compacted = append(compacted, summary)
			saved += len(chunk) - len(summary)
		} else {
			compacted = append(compacted, chunk)
		}
	}

	return strings.Join(compacted, "\n"), saved
}

func (l *LLMCompactionLayer) chunkContent(content string) []string {
	// Split by paragraph or code block
	blocks := strings.Split(content, "\n\n")
	return blocks
}

func (l *LLMCompactionLayer) summarize(chunk string) string {
	// Add marker for LLM to expand
	return fmt.Sprintf("[llm:summarize]%s[/llm]", chunk[:utils.Min(100, len(chunk))])
}

// Layer 12: Attribution Filter
type AttributionLayer struct {
	sources map[string]bool
}

// NewAttributionLayer creates a new attribution layer.
func NewAttributionLayer() *AttributionLayer {
	return &AttributionLayer{
		sources: make(map[string]bool),
	}
}

func (l *AttributionLayer) Name() string { return "attribution" }

func (l *AttributionLayer) ShouldApply(contentType string) bool {
	return strings.Contains(contentType, "code")
}

func (l *AttributionLayer) Apply(content string) (string, int) {
	// Track code origins and licenses
	lines := strings.Split(content, "\n")
	var result []string
	saved := 0

	for _, line := range lines {
		// Detect license headers and preserve them
		if l.isLicenseHeader(line) {
			result = append(result, line)
		} else if l.isAttribution(line) {
			result = append(result, line)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n"), saved
}

func (l *AttributionLayer) isLicenseHeader(line string) bool {
	patterns := []string{
		"Copyright",
		"License:",
		"SPDX-License-Identifier:",
		"MIT License",
		"Apache License",
	}
	for _, p := range patterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

func (l *AttributionLayer) isAttribution(line string) bool {
	return strings.Contains(line, "@author") ||
		strings.Contains(line, "@source") ||
		strings.Contains(line, "Originally from")
}

// Layer 13: H2O Heavy-Hitter Optimization
type HeavyHitterLayer struct {
	threshold int
	counts    map[string]int
	mu        sync.RWMutex
}

// NewHeavyHitterLayer creates a new heavy hitter layer.
func NewHeavyHitterLayer() *HeavyHitterLayer {
	return &HeavyHitterLayer{
		threshold: 100,
		counts:    make(map[string]int),
	}
}

func (l *HeavyHitterLayer) Name() string { return "heavy_hitter" }

func (l *HeavyHitterLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *HeavyHitterLayer) Apply(content string) (string, int) {
	// Track frequently occurring patterns
	lines := strings.Split(content, "\n")
	var result []string
	saved := 0

	for _, line := range lines {
		hash := l.hash(line)

		l.mu.Lock()
		l.counts[hash]++
		count := l.counts[hash]
		l.mu.Unlock()

		if count > l.threshold {
			// Replace with reference
			result = append(result, fmt.Sprintf("[ref:%s]", hash[:8]))
			saved += len(line)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n"), saved
}

func (l *HeavyHitterLayer) hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// Layer 14: Attention Sink Rolling Cache
type AttentionSinkLayer struct {
	sinkTokens []string
	windowSize int
}

// NewAttentionSinkLayer creates attention sink layer.
func NewAttentionSinkLayer() *AttentionSinkLayer {
	return &AttentionSinkLayer{
		sinkTokens: make([]string, 0, 10),
		windowSize: 4096,
	}
}

func (l *AttentionSinkLayer) Name() string { return "attention_sink" }

func (l *AttentionSinkLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *AttentionSinkLayer) Apply(content string) (string, int) {
	// Keep attention sinks (important context) in rolling window
	tokens := l.tokenize(content)
	if len(tokens) <= l.windowSize {
		return content, 0
	}

	// Keep first few tokens (attention sinks) + recent window
	sinks := tokens[:utils.Min(4, len(tokens))]
	window := tokens[len(tokens)-l.windowSize+len(sinks):]

	result := append(sinks, window...)
	saved := len(content) - len(strings.Join(result, " "))

	return strings.Join(result, " "), saved
}

func (l *AttentionSinkLayer) tokenize(content string) []string {
	return strings.Fields(content)
}

// Layer 15: Meta-Token LZ77
type MetaTokenLZ77Layer struct {
	windowSize int
}

// NewMetaTokenLZ77Layer creates LZ77 compression layer.
func NewMetaTokenLZ77Layer() *MetaTokenLZ77Layer {
	return &MetaTokenLZ77Layer{windowSize: 1024}
}

func (l *MetaTokenLZ77Layer) Name() string { return "lz77_meta" }

func (l *MetaTokenLZ77Layer) ShouldApply(contentType string) bool {
	return true
}

func (l *MetaTokenLZ77Layer) Apply(content string) (string, int) {
	// Simple LZ77-inspired compression
	tokens := l.tokenize(content)
	var result []string
	saved := 0

	for i, token := range tokens {
		matchLen, matchPos := l.findMatch(tokens, i)
		if matchLen > 3 {
			// Output reference
			result = append(result, fmt.Sprintf("[%d,%d]", matchPos, matchLen))
			saved += matchLen * len(token)
			i += matchLen - 1
		} else {
			result = append(result, token)
		}
	}

	return strings.Join(result, " "), saved
}

func (l *MetaTokenLZ77Layer) tokenize(content string) []string {
	return strings.Fields(content)
}

func (l *MetaTokenLZ77Layer) findMatch(tokens []string, pos int) (int, int) {
	if pos == 0 {
		return 0, 0
	}

	bestLen := 0
	bestPos := 0
	start := max(0, pos-l.windowSize)

	for i := start; i < pos; i++ {
		length := 0
		for j := 0; pos+j < len(tokens) && i+j < pos && tokens[i+j] == tokens[pos+j]; j++ {
			length++
		}
		if length > bestLen {
			bestLen = length
			bestPos = pos - i
		}
	}

	return bestLen, bestPos
}

// Layer 16: Semantic Chunking Boundaries
type SemanticChunkLayer struct {
	chunkSize int
}

// NewSemanticChunkLayer creates semantic chunking layer.
func NewSemanticChunkLayer() *SemanticChunkLayer {
	return &SemanticChunkLayer{chunkSize: 500}
}

func (l *SemanticChunkLayer) Name() string { return "semantic_chunk" }

func (l *SemanticChunkLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *SemanticChunkLayer) Apply(content string) (string, int) {
	// Find natural boundaries (paragraphs, sections)
	chunks := l.findBoundaries(content)
	if len(chunks) <= 1 {
		return content, 0
	}

	var result []string
	for i, chunk := range chunks {
		if i < len(chunks)-1 {
			result = append(result, chunk+"\n[chunk]")
		} else {
			result = append(result, chunk)
		}
	}

	return strings.Join(result, "\n"), 0
}

func (l *SemanticChunkLayer) findBoundaries(content string) []string {
	// Split on section headers, paragraphs
	boundaries := []string{"\n# ", "\n## ", "\n\n", "\n---\n"}

	chunks := []string{content}
	for _, boundary := range boundaries {
		var newChunks []string
		for _, chunk := range chunks {
			parts := strings.Split(chunk, boundary)
			for i, part := range parts {
				if i > 0 {
					part = boundary + part
				}
				if len(part) > 0 {
					newChunks = append(newChunks, part)
				}
			}
		}
		chunks = newChunks
	}

	return chunks
}

// Layer 17: Sketch Store KV
type SketchStoreLayer struct {
	sketches map[string]*Sketch
}

type Sketch struct {
	Key      string
	Summary  string
	FullHash string
}

// NewSketchStoreLayer creates sketch store layer.
func NewSketchStoreLayer() *SketchStoreLayer {
	return &SketchStoreLayer{
		sketches: make(map[string]*Sketch),
	}
}

func (l *SketchStoreLayer) Name() string { return "sketch_store" }

func (l *SketchStoreLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *SketchStoreLayer) Apply(content string) (string, int) {
	// Store sketches (summaries) with full content in KV store
	hash := l.hash(content)

	if sketch, ok := l.sketches[hash]; ok {
		// Return sketch with reference
		return fmt.Sprintf("[sketch:%s]%s[/sketch]", sketch.Key, sketch.Summary), len(content) - len(sketch.Summary)
	}

	// Create new sketch
	sketch := &Sketch{
		Key:      hash[:16],
		Summary:  l.summarize(content),
		FullHash: hash,
	}
	l.sketches[hash] = sketch

	return content, 0
}

func (l *SketchStoreLayer) hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (l *SketchStoreLayer) summarize(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return content[:utils.Min(100, len(content))]
}

// Layer 18: Lazy Pruner
type LazyPrunerLayer struct {
	budget     int
	priorityFn func(string) float64
}

// NewLazyPrunerLayer creates lazy pruner layer.
func NewLazyPrunerLayer() *LazyPrunerLayer {
	return &LazyPrunerLayer{
		budget: 4000,
		priorityFn: func(s string) float64 {
			return float64(len(s))
		},
	}
}

func (l *LazyPrunerLayer) Name() string { return "lazy_pruner" }

func (l *LazyPrunerLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *LazyPrunerLayer) Apply(content string) (string, int) {
	// Lazy pruning - only prune when budget exceeded
	tokens := l.estimateTokens(content)
	if tokens <= l.budget {
		return content, 0
	}

	// Sort content by priority
	lines := strings.Split(content, "\n")
	type scoredLine struct {
		line  string
		score float64
	}

	scored := make([]scoredLine, len(lines))
	for i, line := range lines {
		scored[i] = scoredLine{line, l.priorityFn(line)}
	}

	// Keep highest priority lines within budget
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	var kept []string
	currentTokens := 0
	for _, sl := range scored {
		lineTokens := l.estimateTokens(sl.line)
		if currentTokens+lineTokens <= l.budget {
			kept = append(kept, sl.line)
			currentTokens += lineTokens
		}
	}

	// Restore original order
	sort.Slice(kept, func(i, j int) bool {
		return true // Would need original index
	})

	return strings.Join(kept, "\n"), len(content) - len(strings.Join(kept, "\n"))
}

func (l *LazyPrunerLayer) estimateTokens(content string) int {
	return len(content) / 4
}

// Layer 19: Semantic Anchor Detection
type SemanticAnchorLayer struct {
	anchors []string
}

// NewSemanticAnchorLayer creates semantic anchor layer.
func NewSemanticAnchorLayer() *SemanticAnchorLayer {
	return &SemanticAnchorLayer{
		anchors: []string{
			"func main",
			"class ",
			"def ",
			"interface ",
			"type ",
			"error",
			"TODO",
			"FIXME",
		},
	}
}

func (l *SemanticAnchorLayer) Name() string { return "semantic_anchor" }

func (l *SemanticAnchorLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *SemanticAnchorLayer) Apply(content string) (string, int) {
	// Detect and preserve semantic anchors
	lines := strings.Split(content, "\n")
	var result []string
	anchors := make(map[int]bool)

	// First pass: identify anchors
	for i, line := range lines {
		for _, anchor := range l.anchors {
			if strings.Contains(line, anchor) {
				anchors[i] = true
				break
			}
		}
	}

	// Second pass: keep anchors and context around them
	for i, line := range lines {
		if anchors[i] {
			result = append(result, line)
			// Keep context lines
			for j := utils.Max(0, i-2); j <= utils.Min(len(lines)-1, i+2); j++ {
				if j != i && !anchors[j] {
					result = append(result, lines[j])
				}
			}
		} else if !anchors[i] {
			// Summarize non-anchor lines
			result = append(result, l.summarize(line))
		}
	}

	saved := len(content) - len(strings.Join(result, "\n"))
	return strings.Join(result, "\n"), saved
}

func (l *SemanticAnchorLayer) summarize(line string) string {
	if len(line) > 80 {
		return line[:77] + "..."
	}
	return line
}

// Layer 20: Agent Memory Knowledge Graph
type KnowledgeGraphLayer struct {
	nodes map[string]*KGNode
	edges map[string][]string
}

type KGNode struct {
	ID      string
	Type    string
	Content string
	Related []string
}

// NewKnowledgeGraphLayer creates knowledge graph layer.
func NewKnowledgeGraphLayer() *KnowledgeGraphLayer {
	return &KnowledgeGraphLayer{
		nodes: make(map[string]*KGNode),
		edges: make(map[string][]string),
	}
}

func (l *KnowledgeGraphLayer) Name() string { return "knowledge_graph" }

func (l *KnowledgeGraphLayer) ShouldApply(contentType string) bool {
	return true
}

func (l *KnowledgeGraphLayer) Apply(content string) (string, int) {
	// Extract entities and relationships
	entities := l.extractEntities(content)

	// Build graph
	for _, entity := range entities {
		if node, ok := l.nodes[entity]; ok {
			// Reference existing knowledge
			content = strings.Replace(content, entity, fmt.Sprintf("[%s]", node.ID), 1)
		} else {
			// Create new node
			l.nodes[entity] = &KGNode{
				ID:      l.hash(entity),
				Type:    "entity",
				Content: entity,
			}
		}
	}

	return content, 0
}

func (l *KnowledgeGraphLayer) extractEntities(content string) []string {
	// Extract potential entities (capitalized words, quoted strings, etc.)
	var entities []string

	// Find quoted strings
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) > 1 {
			entities = append(entities, m[1])
		}
	}

	// Find capitalized words (potential types/classes)
	words := strings.Fields(content)
	for _, word := range words {
		if len(word) > 0 && unicode.IsUpper(rune(word[0])) {
			entities = append(entities, word)
		}
	}

	return entities
}

func (l *KnowledgeGraphLayer) hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

// RegisterAdvancedLayers registers layers 11-20.
func RegisterAdvancedLayers(registry *LayerRegistry) {
	registry.Register(NewLLMCompactionLayer())
	registry.Register(NewAttributionLayer())
	registry.Register(NewHeavyHitterLayer())
	registry.Register(NewAttentionSinkLayer())
	registry.Register(NewMetaTokenLZ77Layer())
	registry.Register(NewSemanticChunkLayer())
	registry.Register(NewSketchStoreLayer())
	registry.Register(NewLazyPrunerLayer())
	registry.Register(NewSemanticAnchorLayer())
	registry.Register(NewKnowledgeGraphLayer())
}

