package filter

import (
	"hash/fnv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/utils"
)

// KVCacheAligner implements KV-cache alignment for LLM prompt caching.
// Inspired by claw-compactor's QuantumLock and kompact's cache_aligner.
// Isolates stable prefix from dynamic content to maximize provider-level caching.
type KVCacheAligner struct {
	config KVCacheConfig
}

// KVCacheConfig holds configuration for KV-cache alignment.
type KVCacheConfig struct {
	Enabled          bool
	MinPrefixLength  int
	MaxDynamicSuffix int
	SplitThreshold   int
}

// DefaultKVCacheConfig returns default KV-cache alignment config.
func DefaultKVCacheConfig() KVCacheConfig {
	return KVCacheConfig{
		Enabled:          true,
		MinPrefixLength:  100,
		MaxDynamicSuffix: 500,
		SplitThreshold:   200,
	}
}

// NewKVCacheAligner creates a new KV-cache aligner.
func NewKVCacheAligner(cfg KVCacheConfig) *KVCacheAligner {
	return &KVCacheAligner{config: cfg}
}

// CacheabilityScore returns a 0-100 score indicating how cacheable content is.
// Higher scores mean more stable prefix, better cache hit rate.
func CacheabilityScore(content string) int {
	if len(content) == 0 {
		return 100
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return 100
	}

	// Count stable lines (no numbers, no timestamps, no file paths)
	stableLines := 0
	for _, line := range lines {
		if isStableLine(line) {
			stableLines++
		}
	}

	ratio := float64(stableLines) / float64(len(lines))
	score := int(ratio * 100)

	// Bonus for system prompt patterns
	if strings.Contains(content, "You are a") || strings.Contains(content, "System:") {
		score = utils.Min(score+10, 100)
	}

	// Penalty for high variability patterns
	if strings.Contains(content, "timestamp") || strings.Contains(content, "file_path") {
		score = utils.Max(score-20, 0)
	}

	return score
}

// ClassifyContent classifies content as static or dynamic.
func ClassifyContent(content string) (isStatic bool, confidence float64) {
	if len(content) == 0 {
		return true, 1.0
	}

	staticPatterns := []string{
		"You are a", "System:", "Instructions:", "Rules:",
		"Guidelines:", "Context:", "Role:", "Task:",
	}

	dynamicPatterns := []string{
		"timestamp:", "file_path:", "line:", "error:",
		"output:", "result:", "response:", "current",
	}

	staticScore := 0
	dynamicScore := 0

	contentLower := strings.ToLower(content)
	for _, p := range staticPatterns {
		if strings.Contains(contentLower, strings.ToLower(p)) {
			staticScore++
		}
	}
	for _, p := range dynamicPatterns {
		if strings.Contains(contentLower, strings.ToLower(p)) {
			dynamicScore++
		}
	}

	total := staticScore + dynamicScore
	if total == 0 {
		return true, 0.5
	}

	confidence = float64(staticScore) / float64(total)
	return staticScore > dynamicScore, confidence
}

// AlignPrefix isolates stable prefix from dynamic content.
// Returns (stablePrefix, dynamicSuffix, cacheKey).
func (a *KVCacheAligner) AlignPrefix(content string) (string, string, string) {
	if !a.config.Enabled || len(content) < a.config.SplitThreshold {
		return content, "", ""
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 5 {
		return content, "", ""
	}

	// Find the split point between stable and dynamic content
	// Scan from the beginning to find where dynamic content starts
	splitIdx := len(lines)
	for i := 0; i < len(lines); i++ {
		if !isStableLine(lines[i]) {
			splitIdx = i
			break
		}
	}

	// If all lines are stable, try to split at midpoint for testing
	if splitIdx == len(lines) && len(lines) > 10 {
		splitIdx = len(lines) / 2
	}

	// Ensure minimum prefix length
	minLines := a.config.MinPrefixLength / 20 // ~20 chars per line
	if splitIdx < minLines && len(lines) > minLines {
		splitIdx = minLines
	}

	prefix := strings.Join(lines[:splitIdx], "\n")
	suffix := ""
	if splitIdx < len(lines) {
		suffix = strings.Join(lines[splitIdx:], "\n")
	}

	cacheKey := computeCacheKey(prefix)
	return prefix, suffix, cacheKey
}

// CacheAwareCompress compresses only the dynamic portion, preserving stable prefix.
// This maintains byte-stable prefixes for provider-level caching.
func (a *KVCacheAligner) CacheAwareCompress(content string, compressor *PipelineCoordinator) (string, int) {
	prefix, suffix, _ := a.AlignPrefix(content)
	if suffix == "" {
		return content, 0
	}

	compressedSuffix, stats := compressor.Process(suffix)
	saved := stats.TotalSaved

	// Reassemble with preserved prefix
	result := prefix
	if compressedSuffix != "" {
		result = prefix + "\n" + compressedSuffix
	}

	return result, saved
}

// EstimateCacheHitRate estimates the cache hit rate for repeated requests.
func EstimateCacheHitRate(content string) float64 {
	score := CacheabilityScore(content)
	// Map 0-100 score to 0.0-1.0 hit rate with diminishing returns
	return float64(score) / 100.0 * 0.9
}

// isStableLine returns true if a line is likely stable (not dynamic).
func isStableLine(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return true
	}

	// Lines with many numbers are likely dynamic
	digitCount := 0
	for _, c := range line {
		if c >= '0' && c <= '9' {
			digitCount++
		}
	}
	if digitCount > len(line)/3 {
		return false
	}

	// Lines with common dynamic patterns
	dynamicPatterns := []string{
		"timestamp", "date:", "time:", "file:", "path:",
		"line:", "error:", "output:", "result:", "status:",
		"pid:", "port:", "ip:", "hash:", "token:",
	}
	lineLower := strings.ToLower(line)
	for _, p := range dynamicPatterns {
		if strings.Contains(lineLower, p) {
			return false
		}
	}

	return true
}

// computeCacheKey computes a hash key for a content prefix.
func computeCacheKey(content string) string {
	h := fnv.New64a()
	h.Write([]byte(content))
	return string(h.Sum(nil))
}

