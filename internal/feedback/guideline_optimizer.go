package feedback

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// GuidelineOptimizer learns compression rules from agent failure analysis.
// Based on "ACON: Agent-Optimized Context" (Zhang et al., 2024).
//
// When an agent fails a task, analyze what context was missing or excessive,
// and learn guidelines to prevent similar failures in the future.
type GuidelineOptimizer struct {
	guidelines    []CompressionGuideline
	failures      []AgentFailure
	mu            sync.RWMutex
	filePath      string
	maxGuidelines int
}

// CompressionGuideline represents a learned compression rule
type CompressionGuideline struct {
	Pattern    string  `json:"pattern"`     // e.g., "Always keep test names in failing test output"
	Confidence float64 `json:"confidence"`  // 0.0-1.0, increases with successful applications
	Source     string  `json:"source"`      // Which failure taught this
	ApplyCount int     `json:"apply_count"` // How many times successfully applied
}

// AgentFailure represents a failure event to learn from
type AgentFailure struct {
	Task       string `json:"task"`       // What the agent was trying to do
	Compressed string `json:"compressed"` // What the agent received
	Issue      string `json:"issue"`      // Why it failed
	Missing    string `json:"missing"`    // What context was needed
	Timestamp  string `json:"timestamp"`  // When it happened
}

// NewGuidelineOptimizer creates a new guideline optimizer
func NewGuidelineOptimizer(dataDir string) *GuidelineOptimizer {
	filePath := filepath.Join(dataDir, "guidelines.json")

	opt := &GuidelineOptimizer{
		guidelines:    make([]CompressionGuideline, 0),
		failures:      make([]AgentFailure, 0),
		filePath:      filePath,
		maxGuidelines: 100, // Keep top 100 guidelines
	}

	// Load existing guidelines
	opt.load()

	return opt
}

// AnalyzeFailure learns from an agent failure event
func (o *GuidelineOptimizer) AnalyzeFailure(failure AgentFailure) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Store failure
	o.failures = append(o.failures, failure)

	// Keep only last 100 failures for analysis
	if len(o.failures) > 100 {
		o.failures = o.failures[1:]
	}

	// Extract what was missing
	pattern := o.extractPattern(failure.Missing)
	if pattern == "" {
		return
	}

	// Check if we already have this guideline
	for i, g := range o.guidelines {
		if g.Pattern == pattern {
			// Increase confidence
			o.guidelines[i].Confidence = minFloat64(g.Confidence+0.1, 1.0)
			o.save()
			return
		}
	}

	// Add new guideline
	guideline := CompressionGuideline{
		Pattern:    pattern,
		Confidence: 0.5, // Start with moderate confidence
		Source:     failure.Task,
		ApplyCount: 0,
	}

	o.guidelines = append(o.guidelines, guideline)

	// Trim to max guidelines (keep highest confidence)
	if len(o.guidelines) > o.maxGuidelines {
		o.trimGuidelines()
	}

	o.save()
}

// extractPattern extracts a actionable pattern from missing context
func (o *GuidelineOptimizer) extractPattern(missing string) string {
	if missing == "" {
		return ""
	}

	lower := strings.ToLower(missing)

	// Common patterns to look for
	patterns := map[string]string{
		"test name":     "keep test names in output",
		"error message": "keep error messages visible",
		"stack trace":   "preserve stack traces for debugging",
		"file path":     "keep file paths for navigation",
		"line number":   "keep line numbers with file references",
		"diff":          "preserve diff hunks in review context",
		"assertion":     "keep assertion failures visible",
		"version":       "show version info for deployments",
	}

	for keyword, pattern := range patterns {
		if strings.Contains(lower, keyword) {
			return pattern
		}
	}

	// If no pattern matches, return a generic one based on the first 50 chars
	if len(missing) > 50 {
		return "keep context about: " + missing[:50]
	}
	return "keep context about: " + missing
}

// EnhanceOutput applies learned guidelines to improve filtered output
func (o *GuidelineOptimizer) EnhanceOutput(original, filtered string) string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if len(o.guidelines) == 0 {
		return filtered
	}

	output := filtered

	for _, g := range o.guidelines {
		if g.Confidence < 0.7 {
			continue // Only apply high-confidence guidelines
		}

		// Check if original contains relevant pattern
		origMatches := o.countKeywordMatches(original, g.Pattern)
		if origMatches > 0 {
			// Check if filtered has fewer matches (lost some keywords)
			filteredMatches := o.countKeywordMatches(output, g.Pattern)
			if filteredMatches < origMatches {
				// Try to restore the missing content
				output = o.restorePattern(original, output, g.Pattern)
			}
		}
	}

	return output
}

// countKeywordMatches returns how many significant keywords from pattern are in content
func (o *GuidelineOptimizer) countKeywordMatches(content, pattern string) int {
	lower := strings.ToLower(content)
	patternLower := strings.ToLower(pattern)

	fillers := map[string]bool{
		"keep": true, "in": true, "the": true, "a": true, "an": true,
		"for": true, "to": true, "and": true, "or": true, "with": true,
		"from": true, "about": true, "output": true, "visible": true,
		"context": true,
	}

	keywords := strings.Fields(patternLower)
	matches := 0

	for _, kw := range keywords {
		if len(kw) < 3 || fillers[kw] {
			continue
		}
		if strings.Contains(lower, kw) {
			matches++
		} else if strings.HasSuffix(kw, "s") && strings.Contains(lower, kw[:len(kw)-1]) {
			matches++
		} else if strings.Contains(lower, kw+"s") {
			matches++
		}
	}

	return matches
}

// matchesGuideline checks if content matches a guideline pattern
func (o *GuidelineOptimizer) matchesGuideline(content, pattern string) bool {
	lower := strings.ToLower(content)
	patternLower := strings.ToLower(pattern)

	// Extract key terms from pattern (skip common filler words)
	fillers := map[string]bool{
		"keep": true, "in": true, "the": true, "a": true, "an": true,
		"for": true, "to": true, "and": true, "or": true, "with": true,
		"from": true, "about": true, "output": true, "visible": true,
		"context": true,
	}

	keywords := strings.Fields(patternLower)
	significantKeywords := 0
	matches := 0

	for _, kw := range keywords {
		if len(kw) < 3 || fillers[kw] {
			continue
		}
		significantKeywords++
		// Check for substring match (e.g., "names" matches "TestName")
		// Also check if the keyword itself is a substring of content
		if strings.Contains(lower, kw) {
			matches++
		} else {
			// Try singular/plural variations
			if strings.HasSuffix(kw, "s") && strings.Contains(lower, kw[:len(kw)-1]) {
				matches++
			} else if strings.Contains(lower, kw+"s") {
				matches++
			}
		}
	}

	// If no significant keywords, fall back to any match
	if significantKeywords == 0 {
		for _, kw := range keywords {
			if len(kw) >= 3 && strings.Contains(lower, kw) {
				return true
			}
		}
		return false
	}

	// Require majority of significant keywords to match
	// This ensures we detect when content is truly missing key parts
	minMatches := (significantKeywords + 1) / 2 // Round up
	return matches >= minMatches
}

// restorePattern tries to restore content matching a pattern from original to filtered
func (o *GuidelineOptimizer) restorePattern(original, filtered, pattern string) string {
	lines := strings.Split(original, "\n")
	filteredLines := strings.Split(filtered, "\n")

	// Find lines in original that match the pattern
	var matchingLines []string
	for _, line := range lines {
		if o.matchesGuideline(line, pattern) {
			matchingLines = append(matchingLines, line)
		}
	}

	// If no matching lines, return as-is
	if len(matchingLines) == 0 {
		return filtered
	}

	// Check if any matching lines are already in filtered
	filteredSet := make(map[string]bool)
	for _, line := range filteredLines {
		filteredSet[strings.TrimSpace(line)] = true
	}

	var toAdd []string
	for _, line := range matchingLines {
		if !filteredSet[strings.TrimSpace(line)] {
			toAdd = append(toAdd, line)
		}
	}

	// Add missing lines at the end
	if len(toAdd) > 0 {
		return filtered + "\n\n[Restored from guidelines:]\n" + strings.Join(toAdd, "\n")
	}

	return filtered
}

// GetGuidelines returns all current guidelines
func (o *GuidelineOptimizer) GetGuidelines() []CompressionGuideline {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]CompressionGuideline, len(o.guidelines))
	copy(result, o.guidelines)
	return result
}

// RecordSuccess records a successful guideline application
func (o *GuidelineOptimizer) RecordSuccess(pattern string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for i, g := range o.guidelines {
		if g.Pattern == pattern {
			o.guidelines[i].ApplyCount++
			o.guidelines[i].Confidence = minFloat64(g.Confidence+0.05, 1.0)
			break
		}
	}

	o.save()
}

// load loads guidelines from disk
func (o *GuidelineOptimizer) load() {
	data, err := os.ReadFile(o.filePath)
	if err != nil {
		return // No existing file
	}

	var guidelines []CompressionGuideline
	if err := json.Unmarshal(data, &guidelines); err != nil {
		return
	}

	o.guidelines = guidelines
}

// save saves guidelines to disk
func (o *GuidelineOptimizer) save() {
	data, err := json.MarshalIndent(o.guidelines, "", "  ")
	if err != nil {
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(o.filePath)
	os.MkdirAll(dir, 0755)

	os.WriteFile(o.filePath, data, 0644)
}

// trimGuidelines keeps only the top guidelines by confidence
func (o *GuidelineOptimizer) trimGuidelines() {
	// Simple selection sort for top guidelines
	target := o.maxGuidelines
	for i := 0; i < target && i < len(o.guidelines); i++ {
		maxIdx := i
		for j := i + 1; j < len(o.guidelines); j++ {
			if o.guidelines[j].Confidence > o.guidelines[maxIdx].Confidence {
				maxIdx = j
			}
		}
		o.guidelines[i], o.guidelines[maxIdx] = o.guidelines[maxIdx], o.guidelines[i]
	}

	o.guidelines = o.guidelines[:minInt(target, len(o.guidelines))]
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minFloat64 returns the minimum of two floats
func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
