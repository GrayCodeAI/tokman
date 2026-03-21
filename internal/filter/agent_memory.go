package filter

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// AgentMemoryFilter implements Layer 20: Agent Memory Mode (Focus-inspired).
//
// Research Source: "Active Context Compression / Focus" (arXiv, January 2026)
// Key Innovation: Agent-centric autonomous memory management inspired by slime mold.
// Results: 22.7% token reduction (14.9M → 11.5M), 57% savings on individual instances.
//
// Methodology (Physarum polycephalum inspired):
// 1. Knowledge Consolidation - Extract learnings into "Knowledge" block
// 2. Active Withdrawal - Prune raw interaction history after consolidation
// 3. Self-Regulation - Agent decides when to consolidate vs. keep raw
//
// This filter maintains session state and autonomously manages context bloating
// in long-horizon agent tasks by distinguishing between:
// - Knowledge: Consolidated insights (high value, permanent)
// - History: Raw interaction logs (transient, prunable)
type AgentMemoryFilter struct {
	config         AgentMemoryConfig
	knowledgeBlock strings.Builder // Consolidated learnings
	historyBlock   strings.Builder // Raw interaction history
	mu             sync.RWMutex

	// Session tracking for autonomous decisions
	lastConsolidation time.Time
	consolidationRate int // Tokens consolidated per session
	totalPruned       int
}

// AgentMemoryConfig holds configuration for agent memory management
type AgentMemoryConfig struct {
	// KnowledgeRetentionRatio is the ratio of knowledge to keep (0.0-1.0)
	KnowledgeRetentionRatio float64

	// HistoryPruneRatio is the ratio of history to prune after consolidation
	HistoryPruneRatio float64

	// ConsolidationThreshold triggers consolidation when history exceeds this
	ConsolidationThreshold int

	// EnableAutoConsolidation allows autonomous memory management
	EnableAutoConsolidation bool

	// KnowledgeMaxSize limits the knowledge block size
	KnowledgeMaxSize int

	// PreservePatterns are regex patterns for important content to always keep
	PreservePatterns []*regexp.Regexp
}

// AgentMemoryStats tracks memory management statistics
type AgentMemoryStats struct {
	TotalConsolidated int
	TotalPruned       int
	KnowledgeTokens   int
	HistoryTokens     int
	TokensSaved       int
}

// Memory patterns for knowledge extraction
var (
	// Insight patterns - indicate learnings/decisions
	insightPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(found|discovered|learned|concluded|determined|identified)`),
		regexp.MustCompile(`(?i)(the (solution|fix|issue|problem|root cause) is)`),
		regexp.MustCompile(`(?i)(this means|therefore|thus|so we)`),
		regexp.MustCompile(`(?i)(key (insight|finding|observation))`),
		regexp.MustCompile(`(?i)(successfully (completed|fixed|implemented))`),
	}

	// Action patterns - indicate operations performed
	actionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(created|updated|deleted|modified|ran|executed|called)`),
		regexp.MustCompile(`(?i)(reading|writing|checking|verifying)`),
	}

	// Noise patterns - low-value content to prune
	noisePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(okay|ok|sure|yes|no|right|correct)\.?`),
		regexp.MustCompile(`(?i)^(\s*[\*\-\_]{3,}\s*)$`),
		regexp.MustCompile(`(?i)^(waiting|loading|processing)\.{3}`),
	}
)

// DefaultAgentMemoryConfig returns default configuration
func DefaultAgentMemoryConfig() AgentMemoryConfig {
	return AgentMemoryConfig{
		KnowledgeRetentionRatio: 0.8,
		HistoryPruneRatio:       0.6,
		ConsolidationThreshold:  500,
		EnableAutoConsolidation: true,
		KnowledgeMaxSize:        2000,
		PreservePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(error|fail|critical|important|warning)`),
			regexp.MustCompile(`(?i)(TODO|FIXME|BUG|HACK)`),
		},
	}
}

// NewAgentMemoryFilter creates a new agent memory filter
func NewAgentMemoryFilter() *AgentMemoryFilter {
	return NewAgentMemoryFilterWithConfig(DefaultAgentMemoryConfig())
}

// NewAgentMemoryFilterWithConfig creates a filter with custom config
func NewAgentMemoryFilterWithConfig(cfg AgentMemoryConfig) *AgentMemoryFilter {
	return &AgentMemoryFilter{
		config:             cfg,
		lastConsolidation:  time.Now(),
		consolidationRate:  0,
		totalPruned:        0,
	}
}

// Name returns the filter name
func (f *AgentMemoryFilter) Name() string {
	return "agent_memory"
}

// Apply applies agent memory management compression
func (f *AgentMemoryFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return input, 0
	}

	// Step 1: Classify content into knowledge vs history
	knowledge, history := f.classifyContent(input)

	// Step 2: Check if consolidation is needed
	shouldConsolidate := f.shouldConsolidate(len(history))
	if shouldConsolidate && f.config.EnableAutoConsolidation {
		f.consolidate(knowledge, history)
	}

	// Step 3: Apply pruning to history
	prunedHistory := f.pruneHistory(history)

	// Step 4: Reconstruct output
	output := f.reconstruct(knowledge, prunedHistory)

	// Calculate savings
	originalTokens := len(tokens)
	outputTokens := len(strings.Fields(output))
	saved := originalTokens - outputTokens

	f.totalPruned += saved

	return output, saved
}

// classifyContent separates knowledge-worthy content from transient history
func (f *AgentMemoryFilter) classifyContent(input string) (knowledge, history string) {
	lines := strings.Split(input, "\n")

	var knowledgeLines, historyLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			historyLines = append(historyLines, line)
			continue
		}

		// Check for insight patterns (knowledge-worthy)
		isKnowledge := false
		for _, pattern := range insightPatterns {
			if pattern.MatchString(trimmed) {
				isKnowledge = true
				break
			}
		}

		// Check preserve patterns (always keep)
		for _, pattern := range f.config.PreservePatterns {
			if pattern.MatchString(trimmed) {
				isKnowledge = true
				break
			}
		}

		// Check for noise (prunable)
		isNoise := false
		for _, pattern := range noisePatterns {
			if pattern.MatchString(trimmed) {
				isNoise = true
				break
			}
		}

		if isNoise {
			continue // Skip noise entirely
		}

		if isKnowledge {
			knowledgeLines = append(knowledgeLines, line)
		} else {
			historyLines = append(historyLines, line)
		}
	}

	return strings.Join(knowledgeLines, "\n"), strings.Join(historyLines, "\n")
}

// shouldConsolidate determines if autonomous consolidation is needed
func (f *AgentMemoryFilter) shouldConsolidate(historyTokens int) bool {
	// Check threshold
	if historyTokens < f.config.ConsolidationThreshold {
		return false
	}

	// Check time since last consolidation (at least 5 seconds apart)
	if time.Since(f.lastConsolidation) < 5*time.Second {
		return false
	}

	return true
}

// consolidate extracts knowledge from history and merges into knowledge block
func (f *AgentMemoryFilter) consolidate(knowledge, history string) {
	// Extract key insights from history
	insights := f.extractInsights(history)

	// Merge with existing knowledge
	existingKnowledge := f.knowledgeBlock.String()
	f.knowledgeBlock.Reset()

	// Write existing knowledge first
	if existingKnowledge != "" {
		f.knowledgeBlock.WriteString(existingKnowledge)
		if !strings.HasSuffix(existingKnowledge, "\n") {
			f.knowledgeBlock.WriteString("\n")
		}
	}

	// Add new insights
	for _, insight := range insights {
		f.knowledgeBlock.WriteString(insight)
		f.knowledgeBlock.WriteString("\n")
	}

	// Enforce knowledge size limit
	f.enforceKnowledgeLimit()

	// Update consolidation tracking
	f.lastConsolidation = time.Now()
	f.consolidationRate += len(insights)
}

// extractInsights extracts key learnings from history
func (f *AgentMemoryFilter) extractInsights(history string) []string {
	lines := strings.Split(history, "\n")
	var insights []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for action patterns that indicate important operations
		for _, pattern := range actionPatterns {
			if pattern.MatchString(trimmed) {
				// Summarize action
				summary := f.summarizeAction(trimmed)
				if summary != "" {
					insights = append(insights, summary)
				}
				break
			}
		}

		// Check for insight patterns
		for _, pattern := range insightPatterns {
			if pattern.MatchString(trimmed) {
				insights = append(insights, trimmed)
				break
			}
		}
	}

	return insights
}

// summarizeAction creates a concise summary of an action
func (f *AgentMemoryFilter) summarizeAction(action string) string {
	// Extract key components (verb + object)
	words := strings.Fields(action)
	if len(words) < 2 {
		return ""
	}

	// Keep first 5-7 words for context
	maxWords := 7
	if len(words) < maxWords {
		maxWords = len(words)
	}

	summary := strings.Join(words[:maxWords], " ")
	if len(words) > maxWords {
		summary += "..."
	}

	return "→ " + summary
}

// pruneHistory removes redundant or low-value history entries
func (f *AgentMemoryFilter) pruneHistory(history string) string {
	lines := strings.Split(history, "\n")
	var prunedLines []string

	pruneRatio := f.config.HistoryPruneRatio
	linesToKeep := int(float64(len(lines)) * (1 - pruneRatio))

	// Use sliding window - keep most recent and most relevant
	if len(lines) <= linesToKeep {
		return history
	}

	// Score each line by relevance
	type scoredLine struct {
		line  string
		score float64
		index int
	}

	scored := make([]scoredLine, len(lines))
	for i, line := range lines {
		score := f.scoreLineRelevance(line)
		scored[i] = scoredLine{line: line, score: score, index: i}
	}

	// Keep lines with highest scores and most recent
	keepIndices := make(map[int]bool)

	// Always keep last 20% of lines (most recent)
	recentCount := linesToKeep / 5
	for i := len(lines) - recentCount; i < len(lines); i++ {
		if i >= 0 {
			keepIndices[i] = true
		}
	}

	// Fill remaining with highest-scored lines
	remainingCount := linesToKeep - recentCount
	for _, s := range scored {
		if remainingCount <= 0 {
			break
		}
		if !keepIndices[s.index] && s.score > 0.3 {
			keepIndices[s.index] = true
			remainingCount--
		}
	}

	// Build pruned output maintaining original order
	for i, line := range lines {
		if keepIndices[i] {
			prunedLines = append(prunedLines, line)
		}
	}

	return strings.Join(prunedLines, "\n")
}

// scoreLineRelevance scores a line's importance (0.0-1.0)
func (f *AgentMemoryFilter) scoreLineRelevance(line string) float64 {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return 0.0
	}

	score := 0.5 // Base score

	// Boost for preserve patterns
	for _, pattern := range f.config.PreservePatterns {
		if pattern.MatchString(trimmed) {
			score += 0.3
		}
	}

	// Boost for insight patterns
	for _, pattern := range insightPatterns {
		if pattern.MatchString(trimmed) {
			score += 0.2
		}
	}

	// Boost for code-like content
	if strings.Contains(trimmed, "{") || strings.Contains(trimmed, "}") ||
		strings.Contains(trimmed, "()") || strings.Contains(trimmed, "=>") {
		score += 0.1
	}

	// Reduce for noise
	for _, pattern := range noisePatterns {
		if pattern.MatchString(trimmed) {
			score -= 0.4
		}
	}

	// Normalize to 0.0-1.0
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// enforceKnowledgeLimit ensures knowledge block stays within size limits
func (f *AgentMemoryFilter) enforceKnowledgeLimit() {
	knowledge := f.knowledgeBlock.String()
	tokens := strings.Fields(knowledge)

	if len(tokens) <= f.config.KnowledgeMaxSize {
		return
	}

	// Keep most important knowledge (last 80% typically has most recent insights)
	keepCount := int(float64(f.config.KnowledgeMaxSize) * f.config.KnowledgeRetentionRatio)
	startIdx := len(tokens) - keepCount
	if startIdx < 0 {
		startIdx = 0
	}

	prunedTokens := tokens[startIdx:]
	f.knowledgeBlock.Reset()
	f.knowledgeBlock.WriteString(strings.Join(prunedTokens, " "))
}

// reconstruct builds the final output from knowledge and history
func (f *AgentMemoryFilter) reconstruct(knowledge, history string) string {
	var parts []string

	// Knowledge block first (if exists)
	existingKnowledge := f.knowledgeBlock.String()
	if existingKnowledge != "" {
		parts = append(parts, "╔════════════════════════════════════════╗")
		parts = append(parts, "║           Knowledge Block              ║")
		parts = append(parts, "╠════════════════════════════════════════╣")
		parts = append(parts, existingKnowledge)
		parts = append(parts, "╚════════════════════════════════════════╝")
	}

	// Current knowledge (if any new)
	if knowledge != "" {
		parts = append(parts, knowledge)
	}

	// History block (pruned)
	if history != "" {
		parts = append(parts, history)
	}

	return strings.Join(parts, "\n")
}

// GetStats returns current memory management statistics
func (f *AgentMemoryFilter) GetStats() AgentMemoryStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	knowledge := f.knowledgeBlock.String()
	history := f.historyBlock.String()

	return AgentMemoryStats{
		TotalConsolidated: f.consolidationRate,
		TotalPruned:       f.totalPruned,
		KnowledgeTokens:   len(strings.Fields(knowledge)),
		HistoryTokens:     len(strings.Fields(history)),
	}
}

// Reset clears the memory state (for new sessions)
func (f *AgentMemoryFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.knowledgeBlock.Reset()
	f.historyBlock.Reset()
	f.lastConsolidation = time.Now()
	f.consolidationRate = 0
	f.totalPruned = 0
}
