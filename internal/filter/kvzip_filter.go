package filter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// KVzipFilter implements KVzip-style query-agnostic compression with context reconstruction.
// Research Source: "KVzip: Query-Agnostic KV Cache Compression with Context Reconstruction" (2025)
// Key Innovation: Build a compressed representation that can reconstruct context
// for any query, not just the current one. Designed for KV reuse across sessions.
//
// This creates a "zip" of the content that preserves enough information to
// reconstruct any relevant subset, while being much smaller than the original.
type KVzipFilter struct {
	config KVzipConfig
}

// KVzipConfig holds configuration for KVzip compression
type KVzipConfig struct {
	// Enabled controls whether the filter is active
	Enabled bool

	// CompressionRatio target compression (0-1, lower = more aggressive)
	CompressionRatio float64

	// PreserveStructure keeps code structure markers
	PreserveStructure bool

	// ReconstructableTags marks sections for query-agnostic reconstruction
	ReconstructableTags bool

	// MinContentLength minimum chars to apply
	MinContentLength int
}

// DefaultKVzipConfig returns default configuration
func DefaultKVzipConfig() KVzipConfig {
	return KVzipConfig{
		Enabled:             true,
		CompressionRatio:    0.3,
		PreserveStructure:   true,
		ReconstructableTags: true,
		MinContentLength:    500,
	}
}

// NewKVzipFilter creates a new KVzip filter
func NewKVzipFilter() *KVzipFilter {
	return &KVzipFilter{
		config: DefaultKVzipConfig(),
	}
}

// Name returns the filter name
func (f *KVzipFilter) Name() string {
	return "kvzip"
}

// Apply applies KVzip-style compression
func (f *KVzipFilter) Apply(input string, mode Mode) (string, int) {
	if !f.config.Enabled || mode == ModeNone {
		return input, 0
	}

	if len(input) < f.config.MinContentLength {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	// Split into context blocks
	blocks := f.extractContextBlocks(input)

	// Score blocks by reconstructability importance
	f.scoreBlocks(blocks, mode)

	// Compress: keep important blocks, summarize others
	output := f.compressBlocks(blocks, mode)

	finalTokens := core.EstimateTokens(output)
	saved := originalTokens - finalTokens
	if saved < 5 {
		return input, 0
	}

	return output, saved
}

// contextBlock represents a reconstructable unit of context
type contextBlock struct {
	content     string
	blockType   string
	score       float64
	isSignature bool // Function/class signatures
	isImport    bool // Import statements
	isError     bool // Error messages
	isResult    bool // Results/output
	lineStart   int
	lineEnd     int
}

// extractContextBlocks extracts semantically meaningful blocks
func (f *KVzipFilter) extractContextBlocks(input string) []contextBlock {
	lines := strings.Split(input, "\n")
	var blocks []contextBlock

	var currentBlock strings.Builder
	var blockStart int
	blockType := "text"

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect block boundaries
		newBlockType := f.detectBlockType(trimmed)

		if newBlockType != blockType && currentBlock.Len() > 0 {
			blocks = append(blocks, contextBlock{
				content:   strings.TrimSpace(currentBlock.String()),
				blockType: blockType,
				lineStart: blockStart,
				lineEnd:   i - 1,
			})
			currentBlock.Reset()
			blockStart = i
			blockType = newBlockType
		} else if currentBlock.Len() == 0 {
			blockStart = i
			blockType = newBlockType
		}

		currentBlock.WriteString(line)
		currentBlock.WriteString("\n")
	}

	// Final block
	if currentBlock.Len() > 0 {
		blocks = append(blocks, contextBlock{
			content:   strings.TrimSpace(currentBlock.String()),
			blockType: blockType,
			lineStart: blockStart,
			lineEnd:   len(lines) - 1,
		})
	}

	return blocks
}

// detectBlockType detects the type of a content block
func (f *KVzipFilter) detectBlockType(line string) string {
	trimmed := strings.TrimSpace(line)

	// Signatures
	if regexp.MustCompile(`^(func|function|def|class|struct|interface|type|impl)\s+`).MatchString(trimmed) {
		return "signature"
	}

	// Imports
	if regexp.MustCompile(`^(import|use|require|include|from|package)\s+`).MatchString(trimmed) {
		return "import"
	}

	// Errors
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "error") || strings.Contains(lower, "fail") ||
		strings.Contains(lower, "exception") || strings.Contains(lower, "panic") {
		return "error"
	}

	// Results
	if strings.Contains(lower, "success") || strings.Contains(lower, "pass") ||
		strings.Contains(lower, "ok") || strings.Contains(lower, "done") ||
		strings.Contains(lower, "complete") {
		return "result"
	}

	// Code
	if strings.Contains(trimmed, "{") || strings.Contains(trimmed, "}") ||
		strings.Contains(trimmed, "=>") || strings.Contains(trimmed, ":=") {
		return "code"
	}

	return "text"
}

// scoreBlocks scores blocks by importance for query-agnostic reconstruction
func (f *KVzipFilter) scoreBlocks(blocks []contextBlock, mode Mode) {
	for i := range blocks {
		score := 0.5

		switch blocks[i].blockType {
		case "signature":
			score = 0.9
			blocks[i].isSignature = true
		case "import":
			score = 0.3
			blocks[i].isImport = true
		case "error":
			score = 0.95
			blocks[i].isError = true
		case "result":
			score = 0.8
			blocks[i].isResult = true
		case "code":
			score = 0.6
		case "text":
			score = 0.4
		}

		// Mode adjustment
		if mode == ModeAggressive {
			score -= 0.1
		}

		blocks[i].score = score
	}
}

// compressBlocks compresses blocks based on scores
func (f *KVzipFilter) compressBlocks(blocks []contextBlock, mode Mode) string {
	threshold := f.config.CompressionRatio
	if mode == ModeAggressive {
		threshold *= 0.7
	}

	var result strings.Builder

	for _, block := range blocks {
		if block.score >= 0.7 || block.isSignature || block.isError {
			// Keep important blocks as-is
			if f.config.ReconstructableTags {
				result.WriteString("<!-- " + block.blockType + " -->\n")
			}
			result.WriteString(block.content)
			result.WriteString("\n\n")
		} else if block.score >= 0.4 {
			// Summarize medium blocks
			summary := f.summarizeBlock(block)
			if f.config.ReconstructableTags {
				result.WriteString("<!-- " + block.blockType + " (summary) -->\n")
			}
			result.WriteString(summary)
			result.WriteString("\n\n")
		}
		// Low-score blocks are dropped entirely
	}

	return strings.TrimSpace(result.String())
}

// summarizeBlock creates a concise summary of a block
func (f *KVzipFilter) summarizeBlock(block contextBlock) string {
	lines := strings.Split(block.content, "\n")

	if len(lines) <= 2 {
		return block.content
	}

	// Keep first and last line, summarize middle
	var summary strings.Builder
	summary.WriteString(lines[0])
	if len(lines) > 2 {
		summary.WriteString("\n[... ")
		summary.WriteString(strconv.Itoa(len(lines) - 2))
		summary.WriteString(" lines ...]")
	}
	summary.WriteString("\n")
	summary.WriteString(lines[len(lines)-1])

	return summary.String()
}

// Reconstruct attempts to reconstruct a specific subset of the original content
// based on a query. This is the query-agnostic reconstruction capability.
func (f *KVzipFilter) Reconstruct(compressed, query string) string {
	lines := strings.Split(compressed, "\n")
	queryLower := strings.ToLower(query)

	var result strings.Builder
	inRelevantBlock := false

	for _, line := range lines {
		// Check for block markers
		if strings.HasPrefix(line, "<!-- ") {
			blockType := strings.TrimPrefix(line, "<!-- ")
			blockType = strings.TrimSuffix(blockType, " -->")
			blockType = strings.TrimSuffix(blockType, " (summary)")

			// Determine if this block type is relevant to the query
			inRelevantBlock = f.isBlockRelevant(blockType, queryLower)
		}

		if inRelevantBlock {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return strings.TrimSpace(result.String())
}

// isBlockRelevant determines if a block type is relevant to a query
func (f *KVzipFilter) isBlockRelevant(blockType, query string) bool {
	switch blockType {
	case "signature":
		return true // Always relevant
	case "error":
		return true // Always relevant
	case "result":
		return strings.Contains(query, "result") || strings.Contains(query, "output")
	case "import":
		return strings.Contains(query, "import") || strings.Contains(query, "depend")
	case "code":
		return strings.Contains(query, "code") || strings.Contains(query, "implement")
	default:
		return false
	}
}

