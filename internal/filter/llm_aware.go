package filter

import (
	"strings"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/llm"
)

// LLMAwareFilter uses local LLM for high-quality summarization.
// This filter is optional and only activates when:
// 1. An LLM provider is available (Ollama, LM Studio, etc.)
// 2. The content exceeds the threshold for LLM-based processing
// 3. The user has enabled LLM mode via flag or config
//
// Research basis: "LLM-based Context Compression" shows 40-60% better
// semantic preservation compared to heuristic-only approaches.
type LLMAwareFilter struct {
	summarizer   *llm.Summarizer
	threshold    int  // Minimum content size to trigger LLM processing
	enabled      bool // Whether LLM mode is enabled
	cacheEnabled bool // Whether to cache summaries
	cache        map[string]string
	cacheMutex   sync.RWMutex
	fallback     Filter // Fallback filter when LLM unavailable
}

// LLMAwareConfig holds configuration for the LLM-aware filter
type LLMAwareConfig struct {
	Threshold      int
	Enabled        bool
	CacheEnabled   bool
	PromptTemplate string // Template name for intent-specific summarization
}

// NewLLMAwareFilter creates a new LLM-aware filter
func NewLLMAwareFilter(cfg LLMAwareConfig) *LLMAwareFilter {
	f := &LLMAwareFilter{
		summarizer:   llm.NewSummarizerFromEnv(),
		threshold:    cfg.Threshold,
		enabled:      cfg.Enabled,
		cacheEnabled: cfg.CacheEnabled,
		cache:        make(map[string]string),
		fallback:     NewSemanticFilter(), // Use semantic filter as fallback
	}

	if f.threshold == 0 {
		f.threshold = 2000 // Default: 2000 lines (~8K tokens)
	}

	return f
}

// Name returns the filter name
func (f *LLMAwareFilter) Name() string {
	return "llm_aware"
}

// Apply applies LLM-based summarization if available, otherwise falls back to heuristic
func (f *LLMAwareFilter) Apply(input string, mode Mode) (string, int) {
	// Check if LLM is enabled and available
	if !f.enabled || !f.summarizer.IsAvailable() {
		// Fall back to semantic filter
		return f.fallback.Apply(input, mode)
	}

	// Check threshold
	lines := strings.Count(input, "\n") + 1
	if lines < f.threshold {
		return f.fallback.Apply(input, mode)
	}

	// Check cache
	if f.cacheEnabled {
		cached := f.getFromCache(input)
		if cached != "" {
			return cached, EstimateTokens(input) - EstimateTokens(cached)
		}
	}

	// Use LLM for summarization
	req := llm.SummaryRequest{
		Content:   input,
		MaxTokens: 500, // Target summary length
		Intent:    "general",
	}

	// Detect intent from content
	req.Intent = f.detectIntent(input)

	resp, err := f.summarizer.Summarize(req)
	if err != nil {
		// Fall back to semantic filter on error
		return f.fallback.Apply(input, mode)
	}

	output := resp.Summary

	// Cache the result
	if f.cacheEnabled {
		f.addToCache(input, output)
	}

	tokensSaved := EstimateTokens(input) - EstimateTokens(output)
	if tokensSaved < 0 {
		tokensSaved = 0
	}

	return output, tokensSaved
}

// detectIntent infers the intent from content
func (f *LLMAwareFilter) detectIntent(content string) string {
	lower := strings.ToLower(content)

	// Debug intent
	if strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "exception") ||
		strings.Contains(lower, "traceback") ||
		strings.Contains(lower, "stack trace") {
		return "debug"
	}

	// Review intent
	if strings.Contains(lower, "diff --git") ||
		strings.Contains(lower, "modified") ||
		strings.Contains(lower, "deleted") ||
		strings.Contains(lower, "added") ||
		strings.Contains(lower, "@@ ") {
		return "review"
	}

	// Test intent
	if strings.Contains(lower, "test") ||
		strings.Contains(lower, "pass") ||
		strings.Contains(lower, "fail") ||
		strings.Contains(lower, "assert") {
		return "test"
	}

	// Build intent
	if strings.Contains(lower, "compiling") ||
		strings.Contains(lower, "building") ||
		strings.Contains(lower, "finished") ||
		strings.Contains(lower, "error[e") {
		return "build"
	}

	return "general"
}

// getFromCache retrieves a cached summary
func (f *LLMAwareFilter) getFromCache(content string) string {
	if !f.cacheEnabled {
		return ""
	}

	f.cacheMutex.RLock()
	defer f.cacheMutex.RUnlock()

	// Use first 100 chars as key (simple caching)
	key := content
	if len(key) > 100 {
		key = key[:100]
	}

	return f.cache[key]
}

// addToCache stores a summary in cache
func (f *LLMAwareFilter) addToCache(content, summary string) {
	if !f.cacheEnabled {
		return
	}

	f.cacheMutex.Lock()
	defer f.cacheMutex.Unlock()

	// Use first 100 chars as key
	key := content
	if len(key) > 100 {
		key = key[:100]
	}

	f.cache[key] = summary
}

// SetEnabled enables or disables LLM mode
func (f *LLMAwareFilter) SetEnabled(enabled bool) {
	f.enabled = enabled
}

// IsAvailable returns true if LLM is available
func (f *LLMAwareFilter) IsAvailable() bool {
	return f.summarizer.IsAvailable()
}

// GetProvider returns the current LLM provider name
func (f *LLMAwareFilter) GetProvider() string {
	return string(f.summarizer.GetProvider())
}

// GetModel returns the current model name
func (f *LLMAwareFilter) GetModel() string {
	return f.summarizer.GetModel()
}

// SummarizeWithIntent provides intent-aware summarization
func (f *LLMAwareFilter) SummarizeWithIntent(content string, intent string) (string, int) {
	if !f.enabled || !f.summarizer.IsAvailable() {
		return f.fallback.Apply(content, ModeMinimal)
	}

	req := llm.SummaryRequest{
		Content:   content,
		MaxTokens: 500,
		Intent:    intent,
	}

	resp, err := f.summarizer.Summarize(req)
	if err != nil {
		return f.fallback.Apply(content, ModeMinimal)
	}

	return resp.Summary, EstimateTokens(content) - EstimateTokens(resp.Summary)
}
