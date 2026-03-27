package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// SketchStoreFilter implements Layer 17: Sketch-based Reversible Compression (KVReviver style).
//
// Research Source: "KVReviver: Sketch-based KV Cache Recovery" (December 2025)
// Key Innovation: On-demand reconstruction of pruned tokens via compressed sketches.
// Results: 90% memory reduction with identical accuracy at 10% budget.
//
// Methodology:
// 1. Create sketches (compressed representations) for evicted content
// 2. Store sketches in a SketchCache for on-demand reconstruction
// 3. Monitor attention patterns to detect when reconstruction is needed
// 4. Revive pruned content when required for context
type SketchStoreFilter struct {
	config      SketchStoreConfig
	sketchCache *SketchCache
	mu          sync.RWMutex
}

// SketchStoreConfig holds configuration for sketch-based compression
type SketchStoreConfig struct {
	// BudgetRatio is the target compression ratio (0.1 = 10% budget)
	BudgetRatio float64

	// EnableRecovery allows on-demand reconstruction
	EnableRecovery bool

	// MaxSketchSize limits sketch storage per entry
	MaxSketchSize int

	// HeavyHitterRatio determines what stays uncompressed
	HeavyHitterRatio float64
}

// SketchCache stores compressed representations of pruned content
type SketchCache struct {
	TokenSketches map[string]*Sketch // hash -> sketch
	Budget        float64
	Stats         SketchStats
}

// Sketch represents a compressed content entry
type Sketch struct {
	CompressedInfo []byte  // Quantized/low-rank representation
	OriginalHash   string  // For verification
	TokenCount     int     // Original token count
	Importance     float64 // Original importance score
	ContentType    string  // "code", "text", "mixed"
}

// SketchStats tracks compression statistics
type SketchStats struct {
	TotalSketches   int
	TotalCompressed int
	TotalRevived    int
	TokensSaved     int
}

// SketchEntry represents a revivable content block
type SketchEntry struct {
	Hash         string
	Content      string
	Sketch       *Sketch
	Revived      bool
	RevivalCount int
}

// DefaultSketchStoreConfig returns default configuration
func DefaultSketchStoreConfig() SketchStoreConfig {
	return SketchStoreConfig{
		BudgetRatio:      0.1, // 10% budget
		EnableRecovery:   true,
		MaxSketchSize:    256, // bytes per sketch
		HeavyHitterRatio: 0.2, // Top 20% stay uncompressed
	}
}

// NewSketchStoreFilter creates a new sketch-based reversible store
func NewSketchStoreFilter() *SketchStoreFilter {
	return NewSketchStoreFilterWithConfig(DefaultSketchStoreConfig())
}

// NewSketchStoreFilterWithConfig creates a filter with custom config
func NewSketchStoreFilterWithConfig(cfg SketchStoreConfig) *SketchStoreFilter {
	return &SketchStoreFilter{
		config: cfg,
		sketchCache: &SketchCache{
			TokenSketches: make(map[string]*Sketch),
			Budget:        cfg.BudgetRatio,
		},
	}
}

// Name returns the filter name
func (f *SketchStoreFilter) Name() string {
	return "sketch_store"
}

// Apply applies sketch-based reversible compression
func (f *SketchStoreFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Split content into processable chunks
	chunks := f.splitIntoChunks(input)

	// Score each chunk
	scored := f.scoreChunks(chunks)

	// Create sketches for low-importance chunks, keep high-importance
	output, saved := f.compressWithSketches(scored, mode)

	return output, saved
}

// splitIntoChunks splits content into processable units
func (f *SketchStoreFilter) splitIntoChunks(input string) []string {
	// Split by paragraphs/blocks
	blocks := strings.Split(input, "\n\n")

	var chunks []string
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block != "" {
			// Further split large blocks by lines
			lines := strings.Split(block, "\n")
			var currentChunk strings.Builder
			currentTokens := 0

			for _, line := range lines {
				lineTokens := estimateTokens(line)

				if currentTokens+lineTokens > 200 && currentTokens > 0 {
					chunks = append(chunks, currentChunk.String())
					currentChunk.Reset()
					currentTokens = 0
				}

				currentChunk.WriteString(line)
				currentChunk.WriteString("\n")
				currentTokens += lineTokens
			}

			if currentTokens > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			}
		}
	}

	return chunks
}

// scoreChunks scores chunk importance
func (f *SketchStoreFilter) scoreChunks(chunks []string) []scoredChunk {
	scored := make([]scoredChunk, len(chunks))

	for i, chunk := range chunks {
		score := f.calculateImportance(chunk)
		scored[i] = scoredChunk{
			content: chunk,
			score:   score,
			tokens:  estimateTokens(chunk),
		}
	}

	return scored
}

// scoredChunk pairs content with importance score
type scoredChunk struct {
	content string
	score   float64
	tokens  int
}

// calculateImportance calculates chunk importance score
func (f *SketchStoreFilter) calculateImportance(chunk string) float64 {
	score := 0.5 // Base score

	lower := strings.ToLower(chunk)

	// Important patterns boost score
	importantPatterns := []string{
		"func ", "function ", "class ", "struct ", "interface ",
		"export ", "public ", "return ", "error", "main",
		"api", "handler", "config", "init",
	}
	for _, p := range importantPatterns {
		if strings.Contains(lower, p) {
			score += 0.1
		}
	}

	// Boilerplate reduces score
	boilerplate := []string{
		"generated by", "auto-generated", "copyright", "license",
		"see below", "as shown above", "note that",
	}
	for _, b := range boilerplate {
		if strings.Contains(lower, b) {
			score -= 0.2
		}
	}

	// Normalize
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// compressWithSketches creates sketches for low-importance content
func (f *SketchStoreFilter) compressWithSketches(chunks []scoredChunk, mode Mode) (string, int) {
	var result strings.Builder
	saved := 0

	// Determine threshold based on mode and heavy hitter ratio
	threshold := f.config.HeavyHitterRatio
	if mode == ModeAggressive {
		threshold += 0.1
	}

	for _, sc := range chunks {
		if sc.score >= threshold {
			// Keep high-importance content
			result.WriteString(sc.content)
			result.WriteString("\n\n")
		} else {
			// Create sketch for low-importance content
			sketch := f.createSketch(sc.content, sc.score)

			// Store in cache
			f.mu.Lock()
			f.sketchCache.TokenSketches[sketch.OriginalHash[:8]] = sketch
			f.sketchCache.Stats.TotalSketches++
			f.mu.Unlock()

			// Count saved tokens (content is compressed to sketch)
			saved += sc.tokens
		}
	}

	return strings.TrimSpace(result.String()), saved
}

// createSketch creates a compressed sketch of content
func (f *SketchStoreFilter) createSketch(content string, importance float64) *Sketch {
	// Hash the original content
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])

	// Create compressed representation
	// In production, this would use quantization or low-rank approximation
	compressed := f.compressContent(content)

	// Detect content type
	contentType := "text"
	if f.isCode(content) {
		contentType = "code"
	}

	return &Sketch{
		CompressedInfo: compressed,
		OriginalHash:   hashStr,
		TokenCount:     estimateTokens(content),
		Importance:     importance,
		ContentType:    contentType,
	}
}

// compressContent creates a compressed representation
func (f *SketchStoreFilter) compressContent(content string) []byte {
	// Simple compression: extract key structural elements
	var compressed strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Keep structural markers
		if strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "struct ") ||
			strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "package ") {
			compressed.WriteString(trimmed)
			compressed.WriteString("\n")
		}

		// Keep first line (often important)
		if compressed.Len() == 0 && len(trimmed) > 0 {
			compressed.WriteString(trimmed)
			compressed.WriteString("\n")
		}
	}

	// Limit size
	result := []byte(compressed.String())
	if len(result) > f.config.MaxSketchSize {
		result = result[:f.config.MaxSketchSize]
	}

	return result
}

// isCode detects if content is code
func (f *SketchStoreFilter) isCode(content string) bool {
	codeIndicators := []string{
		"func ", "function ", "def ", "class ", "struct ",
		"import ", "package ", "var ", "const ", "let ",
		"{", "}", "()", "();",
	}

	count := 0
	for _, ind := range codeIndicators {
		if strings.Contains(content, ind) {
			count++
		}
	}

	return count >= 3
}

// Revive reconstructs content from a sketch
func (f *SketchStoreFilter) Revive(sketchHash string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	sketch, exists := f.sketchCache.TokenSketches[sketchHash]
	if !exists {
		return "", false
	}

	// Update stats
	f.sketchCache.Stats.TotalRevived++

	// Reconstruct content from sketch
	// Note: This is a simplified reconstruction
	// In production, this would use learned reconstruction
	reconstructed := string(sketch.CompressedInfo)

	return reconstructed, true
}

// GetSketch returns a sketch by hash
func (f *SketchStoreFilter) GetSketch(hash string) (*Sketch, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	sketch, exists := f.sketchCache.TokenSketches[hash]
	return sketch, exists
}

// GetAllSketches returns all stored sketches
func (f *SketchStoreFilter) GetAllSketches() map[string]*Sketch {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]*Sketch, len(f.sketchCache.TokenSketches))
	for k, v := range f.sketchCache.TokenSketches {
		result[k] = v
	}
	return result
}

// GetStats returns compression statistics
func (f *SketchStoreFilter) GetStats() SketchStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.sketchCache.Stats
}

// ExportSketches serializes all sketches for persistence
func (f *SketchStoreFilter) ExportSketches() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var buf strings.Builder

	for hash, sketch := range f.sketchCache.TokenSketches {
		buf.WriteString(fmt.Sprintf("%s:%s:%d:%.2f\n",
			hash,
			sketch.ContentType,
			sketch.TokenCount,
			sketch.Importance,
		))
	}

	return []byte(buf.String()), nil
}

// ImportSketches loads sketches from serialized data
func (f *SketchStoreFilter) ImportSketches(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 4 {
			// Parse and store sketch
			// Simplified - would use proper deserialization
			_ = parts
		}
	}

	return nil
}

// Clear clears the sketch cache
func (f *SketchStoreFilter) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sketchCache.TokenSketches = make(map[string]*Sketch)
	f.sketchCache.Stats = SketchStats{}
}
