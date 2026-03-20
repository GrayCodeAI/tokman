package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CommandContext provides metadata about the command being executed.
// Used for intelligent filtering decisions.
type CommandContext struct {
	Command    string // "git", "npm", "cargo", etc.
	Subcommand string // "status", "test", "build"
	ExitCode   int    // Non-zero = likely has errors
	Intent     string // "debug", "review", "deploy", "search"
	IsTest     bool   // Test output detection
	IsBuild    bool   // Build output detection
	IsError    bool   // Error output detection
}

// PipelineManager handles resilient large-context processing.
// Supports streaming for inputs up to 2M tokens with automatic
// chunking, validation, and failure recovery.
type PipelineManager struct {
	config      ManagerConfig
	coordinator *PipelineCoordinator
	cache       *CompressionCache
	lruCache    *LRUCache // T101: LRU cache for better eviction
	teeDir      string
	mu          sync.RWMutex
}

// ManagerConfig configures the pipeline manager
type ManagerConfig struct {
	// Context limits
	MaxContextTokens int
	ChunkSize        int
	StreamThreshold  int

	// Resilience
	TeeOnFailure       bool
	FailSafeMode       bool
	ValidateOutput     bool
	ShortCircuitBudget bool

	// Performance
	CacheEnabled bool
	CacheMaxSize int

	// Layer config
	PipelineCfg PipelineConfig
}

// NewPipelineManager creates a new pipeline manager
func NewPipelineManager(cfg ManagerConfig) *PipelineManager {
	m := &PipelineManager{
		config: cfg,
	}

	// Create coordinator with pipeline config
	m.coordinator = NewPipelineCoordinator(cfg.PipelineCfg)

	// Initialize cache
	if cfg.CacheEnabled {
		m.cache = NewCompressionCache(cfg.CacheMaxSize)
		m.lruCache = NewLRUCache(cfg.CacheMaxSize, 5*time.Minute) // T101
	}

	// Set tee directory
	if cfg.TeeOnFailure {
		m.teeDir = os.TempDir()
	}

	return m
}

// ProcessResult contains the result of processing
type ProcessResult struct {
	Output           string
	OriginalTokens   int
	FinalTokens      int
	SavedTokens      int
	ReductionPercent float64
	LayerStats       map[string]LayerStat
	CacheHit         bool
	Chunks           int
	Validated        bool
	TeeFile          string // If failure occurred
	Warning          string
}

// Process processes input with full resilience and large context support.
// For inputs > StreamThreshold, uses streaming chunk processing.
func (m *PipelineManager) Process(input string, mode Mode, ctx CommandContext) (*ProcessResult, error) {
	result := &ProcessResult{
		LayerStats: make(map[string]LayerStat),
	}

	// Validate context size
	tokens := EstimateTokens(input)
	result.OriginalTokens = tokens

	if tokens > m.config.MaxContextTokens {
		return nil, fmt.Errorf("input exceeds max context tokens (%d > %d)", tokens, m.config.MaxContextTokens)
	}

	// Check cache (T101: LRU cache with TTL)
	cacheKey := m.cacheKey(input, mode, ctx)

	// Try LRU cache first (faster, TTL-aware)
	if m.lruCache != nil {
		if cached := m.lruCache.Get(cacheKey); cached != nil {
			result.Output = cached.Output
			result.FinalTokens = EstimateTokens(result.Output)
			result.SavedTokens = result.OriginalTokens - result.FinalTokens
			result.CacheHit = true
			return result, nil
		}
	}

	// Fall back to legacy cache
	if m.cache != nil {
		if cached, ok := m.cache.Get(cacheKey); ok {
			result.Output = cached.Output
			result.FinalTokens = EstimateTokens(result.Output)
			result.SavedTokens = result.OriginalTokens - result.FinalTokens
			result.CacheHit = true
			return result, nil
		}
	}

	// Choose processing strategy based on size
	if tokens > m.config.StreamThreshold {
		return m.processStreaming(input, mode, ctx, result)
	}

	return m.processSingle(input, mode, ctx, result)
}

// processSingle processes input in a single pass
func (m *PipelineManager) processSingle(input string, mode Mode, ctx CommandContext, result *ProcessResult) (*ProcessResult, error) {
	// Set query intent if provided
	if ctx.Intent != "" {
		m.coordinator.config.QueryIntent = ctx.Intent
	}

	// Process through pipeline
	output, stats := m.coordinator.Process(input)

	// Validate output
	if m.config.ValidateOutput {
		if !m.validateOutput(output, input, ctx) {
			// Output validation failed
			if m.config.FailSafeMode {
				result.Output = input
				result.Warning = "output validation failed, returning original"
			} else {
				result.Output = output
				result.Warning = "output may be corrupted"
			}
		} else {
			result.Output = output
			result.Validated = true
		}
	} else {
		result.Output = output
	}

	// Check for empty output (failure)
	if result.Output == "" && input != "" {
		if m.config.TeeOnFailure {
			teeFile := m.saveTee(input, ctx, "empty_output")
			result.TeeFile = teeFile
			result.Warning = "pipeline produced empty output, original saved to tee file"
		}

		if m.config.FailSafeMode {
			result.Output = input
		}
	}

	// Copy stats
	result.FinalTokens = stats.FinalTokens
	result.SavedTokens = stats.TotalSaved
	result.ReductionPercent = stats.ReductionPercent
	result.LayerStats = stats.LayerStats

	// Cache result - prefer LRU cache (has TTL eviction), fall back to legacy
	if !result.CacheHit {
		cacheKey := m.cacheKey(input, mode, ctx)
		cached := &CachedResult{
			Output:   result.Output,
			Tokens:   result.FinalTokens,
			CachedAt: time.Now(),
		}
		if m.lruCache != nil {
			m.lruCache.Set(cacheKey, cached)
		} else if m.cache != nil {
			m.cache.Set(cacheKey, cached)
		}
	}

	return result, nil
}

// processStreaming processes large input in chunks
func (m *PipelineManager) processStreaming(input string, mode Mode, ctx CommandContext, result *ProcessResult) (*ProcessResult, error) {
	// Split into processable chunks
	chunks := m.chunkInput(input, m.config.ChunkSize)
	result.Chunks = len(chunks)

	var processedChunks []string
	totalSaved := 0

	for i, chunk := range chunks {
		chunkResult, err := m.processSingle(chunk, mode, ctx, &ProcessResult{
			LayerStats: make(map[string]LayerStat),
		})
		if err != nil {
			// Handle chunk failure
			if m.config.FailSafeMode {
				processedChunks = append(processedChunks, chunk)
				continue
			}
			return nil, fmt.Errorf("chunk %d failed: %w", i, err)
		}

		processedChunks = append(processedChunks, chunkResult.Output)
		totalSaved += chunkResult.SavedTokens

		// Short-circuit if budget met
		if m.config.ShortCircuitBudget && m.coordinator.config.Budget > 0 {
			currentTokens := EstimateTokens(strings.Join(processedChunks, "\n"))
			if currentTokens <= m.coordinator.config.Budget {
				break
			}
		}
	}

	// Combine chunks
	result.Output = strings.Join(processedChunks, "\n\n--- Chunk Boundary ---\n\n")
	result.FinalTokens = EstimateTokens(result.Output)
	result.SavedTokens = result.OriginalTokens - result.FinalTokens
	if result.OriginalTokens > 0 {
		result.ReductionPercent = float64(result.SavedTokens) / float64(result.OriginalTokens) * 100
	}

	return result, nil
}

// chunkInput splits large input into processable chunks
func (m *PipelineManager) chunkInput(input string, maxTokens int) []string {
	tokens := EstimateTokens(input)
	if tokens <= maxTokens {
		return []string{input}
	}

	// Split by logical boundaries
	lines := strings.Split(input, "\n")
	var chunks []string
	var currentChunk []string
	currentTokens := 0

	for _, line := range lines {
		lineTokens := EstimateTokens(line)

		// Check if adding this line exceeds chunk size
		if currentTokens+lineTokens > maxTokens && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, "\n"))
			currentChunk = nil
			currentTokens = 0
		}

		currentChunk = append(currentChunk, line)
		currentTokens += lineTokens
	}

	// Add remaining chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, "\n"))
	}

	return chunks
}

// validateOutput checks if output is valid
func (m *PipelineManager) validateOutput(output, original string, ctx CommandContext) bool {
	// Check for empty output when original was not empty
	if output == "" && original != "" {
		return false
	}

	// Check for reasonable compression (not more than 99% unless aggressive)
	compressionRatio := float64(len(output)) / float64(len(original))
	if compressionRatio < 0.01 {
		// Suspicious - output is less than 1% of original
		// This might indicate corruption
		return false
	}

	// Check that important content is preserved
	if ctx.IsError || ctx.ExitCode != 0 {
		// For error output, ensure error markers are preserved
		if strings.Contains(original, "error") && !strings.Contains(strings.ToLower(output), "error") {
			return false
		}
		if strings.Contains(original, "Error:") && !strings.Contains(output, "Error:") {
			return false
		}
	}

	// Check for structural integrity (balanced brackets, etc.)
	if !m.checkStructure(output) {
		return false
	}

	return true
}

// checkStructure performs basic structural validation
func (m *PipelineManager) checkStructure(s string) bool {
	// Check balanced brackets
	parens := 0
	brackets := 0
	braces := 0

	for _, c := range s {
		switch c {
		case '(':
			parens++
		case ')':
			parens--
		case '[':
			brackets++
		case ']':
			brackets--
		case '{':
			braces++
		case '}':
			braces--
		}

		// Allow some imbalance (code snippets, etc.)
		if parens < -10 || brackets < -10 || braces < -10 {
			return false
		}
	}

	// Allow moderate imbalance
	if parens > 50 || brackets > 50 || braces > 50 {
		return false
	}

	return true
}

// saveTee saves raw output to a file for recovery
func (m *PipelineManager) saveTee(input string, ctx CommandContext, reason string) string {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("tokman-tee-%s-%s-%s.txt", timestamp, ctx.Command, reason)
	path := filepath.Join(m.teeDir, filename)

	data := struct {
		Timestamp  time.Time
		Command    string
		Subcommand string
		Reason     string
		Input      string
		Context    CommandContext
	}{
		Timestamp:  time.Now(),
		Command:    ctx.Command,
		Subcommand: ctx.Subcommand,
		Reason:     reason,
		Input:      input,
		Context:    ctx,
	}

	content, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(path, content, 0644)

	return path
}

// cacheKey generates a cache key for the input
func (m *PipelineManager) cacheKey(input string, mode Mode, ctx CommandContext) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%s-%s-%s-%s",
		hex.EncodeToString(hash[:8]),
		mode,
		ctx.Command,
		ctx.Intent,
	)
}

// GetStats returns pipeline statistics
func (m *PipelineManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"max_context_tokens": m.config.MaxContextTokens,
		"chunk_size":         m.config.ChunkSize,
		"stream_threshold":   m.config.StreamThreshold,
		"cache_enabled":      m.config.CacheEnabled,
	}

	if m.cache != nil {
		stats["cache_size"] = m.cache.Size()
	}

	return stats
}

// CompressionCache provides caching for compression results
type CompressionCache struct {
	maxSize int
	entries map[string]*CachedResult
	mu      sync.RWMutex
}

// CachedResult represents a cached compression result
type CachedResult struct {
	Output   string
	Tokens   int
	CachedAt time.Time
}

// NewCompressionCache creates a new compression cache
func NewCompressionCache(maxSize int) *CompressionCache {
	return &CompressionCache{
		maxSize: maxSize,
		entries: make(map[string]*CachedResult),
	}
}

// Get retrieves a cached result
func (c *CompressionCache) Get(key string) (*CachedResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	return entry, ok
}

// Set stores a result in cache
func (c *CompressionCache) Set(key string, result *CachedResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict old entries if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = result
}

// evictOldest removes the oldest cache entry
func (c *CompressionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range c.entries {
		if oldestKey == "" || v.CachedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.CachedAt
		}
	}

	delete(c.entries, oldestKey)
}

// Size returns the number of cached entries
func (c *CompressionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// ProcessWithBudget processes with a specific token budget
func (m *PipelineManager) ProcessWithBudget(input string, mode Mode, budget int, ctx CommandContext) (*ProcessResult, error) {
	// Update coordinator budget
	m.coordinator.config.Budget = budget

	// Set up budget enforcer if not present
	if m.coordinator.budgetEnforcer == nil {
		m.coordinator.budgetEnforcer = NewBudgetEnforcer(budget)
	} else {
		m.coordinator.budgetEnforcer.SetBudget(budget)
	}

	return m.Process(input, mode, ctx)
}

// ProcessWithQuery processes with query-aware compression
func (m *PipelineManager) ProcessWithQuery(input string, mode Mode, query string, ctx CommandContext) (*ProcessResult, error) {
	// Update query intent
	m.coordinator.config.QueryIntent = query
	ctx.Intent = query

	// Initialize query-aware filters if needed
	if m.coordinator.goalDrivenFilter == nil && query != "" {
		m.coordinator.goalDrivenFilter = NewGoalDrivenFilter(query)
	}
	if m.coordinator.contrastiveFilter == nil && query != "" {
		m.coordinator.contrastiveFilter = NewContrastiveFilter(query)
	}

	return m.Process(input, mode, ctx)
}

// ProcessFile processes a file with streaming for large files
func (m *PipelineManager) ProcessFile(path string, mode Mode, ctx CommandContext) (*ProcessResult, error) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Check file size
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Estimate tokens from file size (rough: 1 token ≈ 4 bytes)
	estimatedTokens := int(stat.Size() / 4)

	if estimatedTokens > m.config.MaxContextTokens {
		return nil, fmt.Errorf("file exceeds max context tokens (%d > %d)", estimatedTokens, m.config.MaxContextTokens)
	}

	// Read file
	if estimatedTokens > m.config.StreamThreshold {
		// Stream processing for large files
		return m.processFileStream(f, mode, ctx)
	}

	// Read entire file for small inputs
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return m.Process(string(content), mode, ctx)
}

// processFileStream processes a large file in streaming fashion
func (m *PipelineManager) processFileStream(r io.Reader, mode Mode, ctx CommandContext) (*ProcessResult, error) {
	result := &ProcessResult{
		LayerStats: make(map[string]LayerStat),
	}

	// Read in chunks
	buf := make([]byte, m.config.ChunkSize*4) // Convert tokens to bytes
	var chunks []string
	totalOriginal := 0
	totalFinal := 0

	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read error: %w", err)
		}

		if n == 0 {
			break
		}

		chunk := string(buf[:n])
		chunkResult, err := m.Process(chunk, mode, ctx)
		if err != nil {
			return nil, fmt.Errorf("chunk processing error: %w", err)
		}

		chunks = append(chunks, chunkResult.Output)
		totalOriginal += chunkResult.OriginalTokens
		totalFinal += chunkResult.FinalTokens
		result.Chunks++
	}

	result.Output = strings.Join(chunks, "\n\n--- Chunk Boundary ---\n\n")
	result.OriginalTokens = totalOriginal
	result.FinalTokens = totalFinal
	result.SavedTokens = totalOriginal - totalFinal
	if totalOriginal > 0 {
		result.ReductionPercent = float64(result.SavedTokens) / float64(totalOriginal) * 100
	}

	return result, nil
}
