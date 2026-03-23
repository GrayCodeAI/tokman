package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/llm"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

const maxCompactionCacheSize = 100

// Pre-compiled regexes for compaction hot paths (avoid per-call compilation)
var (
	// extractCritical patterns
	reCriticalError = regexp.MustCompile(`(?i)(error|failed|exception)[:：]\s*(.+)`)
	reCriticalFile  = regexp.MustCompile(`(?i)(file|path)[:：]\s*(.+)`)
	reCriticalTodo  = regexp.MustCompile(`(?i)(todo|fixme|important|note)[:：]\s*(.+)`)

	// extractKeyValuePairs patterns
	reKVGeneral = regexp.MustCompile(`(?i)(\w+)\s*[:：=]\s*([^\n]+)`)
	reKVDQuoted = regexp.MustCompile(`(?i)"(\w+)":\s*"([^"]+)"`)
	reKVSQuoted = regexp.MustCompile(`(?i)'(\w+)':\s*'([^']+)'`)

	// inferNextAction patterns
	reNextColon  = regexp.MustCompile(`next[:：]\s*(.+)`)
	reThenAction = regexp.MustCompile(`then\s+(.+)`)
	reProceedTo  = regexp.MustCompile(`proceed\s+to\s+(.+)`)
)

// CompactionLayer provides semantic compression for chat/conversation content.
// It creates state snapshots with 4 sections:
//
// 1. session_history: user queries + activity log (what was done)
// 2. current_state: focus + next_action (what's active now)
// 3. context: critical + working knowledge (what to remember)
// 4. pending_plan: future milestones (what's next)
//
// Research basis: "MemGPT" (UC Berkeley, 2023) semantic compression
// achieves 98%+ compression ratios while preserving semantic meaning.
//
// This layer is designed for:
// - Chat history compression
// - Conversation-style content
// - Session state preservation
// - Multi-turn context management
type CompactionLayer struct {
	config         CompactionConfig
	summarizer     *llm.Summarizer
	sessionTracker *ConversationTracker
	cache          map[string]*CompactionResult
	cacheMu        sync.RWMutex
	fallback       Filter
}

// CompactionConfig holds configuration for the compaction layer
type CompactionConfig struct {
	// Enable LLM-based compaction
	Enabled bool

	// Minimum content size to trigger compaction (in lines)
	ThresholdLines int

	// Minimum content size to trigger compaction (in tokens)
	ThresholdTokens int

	// Number of recent turns to preserve verbatim
	PreserveRecentTurns int

	// Maximum summary length in tokens
	MaxSummaryTokens int

	// Content types to compact (chat, conversation, session)
	ContentTypes []string

	// Enable caching of compaction results
	CacheEnabled bool

	// Custom prompt template for compaction
	PromptTemplate string

	// Detect content type automatically
	AutoDetect bool

	// Create state snapshot format (4-section XML)
	StateSnapshotFormat bool

	// Extract key-value pairs from content
	ExtractKeyValuePairs bool

	// Maximum context entries to preserve
	MaxContextEntries int
}

// DefaultCompactionConfig returns default configuration
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		Enabled:              false, // Opt-in via flag
		ThresholdLines:       100,
		ThresholdTokens:      2000,
		PreserveRecentTurns:  5,
		MaxSummaryTokens:     500,
		ContentTypes:         []string{"chat", "conversation", "session", "transcript"},
		CacheEnabled:         true,
		AutoDetect:           true,
		StateSnapshotFormat:  true,
		ExtractKeyValuePairs: true,
		MaxContextEntries:    20,
	}
}

// CompactionResult represents a compaction result
type CompactionResult struct {
	Snapshot         *StateSnapshot
	OriginalTokens   int
	FinalTokens      int
	SavedTokens      int
	CompressionRatio float64
	Cached           bool
	Timestamp        time.Time
}

// StateSnapshot represents semantic compaction output
type StateSnapshot struct {
	SessionHistory SessionHistory  `json:"session_history"`
	CurrentState   CurrentState    `json:"current_state"`
	Context        SnapshotContext `json:"context"`
	PendingPlan    []Milestone     `json:"pending_plan"`
}

// SessionHistory tracks what happened in the session
type SessionHistory struct {
	UserQueries []string `json:"user_queries"`
	ActivityLog []string `json:"activity_log"`
	FilesRead   []string `json:"files_read,omitempty"`
	FilesEdited []string `json:"files_edited,omitempty"`
	CommandsRun []string `json:"commands_run,omitempty"`
	Decisions   []string `json:"decisions,omitempty"`
}

// CurrentState tracks what's currently active
type CurrentState struct {
	Focus      string `json:"focus"`
	NextAction string `json:"next_action"`
	ActiveFile string `json:"active_file,omitempty"`
	Mode       string `json:"mode,omitempty"`
}

// SnapshotContext preserves important knowledge
type SnapshotContext struct {
	Critical    []string          `json:"critical"`  // Must preserve (can't rediscover)
	Working     []string          `json:"working"`   // Summarized knowledge
	KeyValue    map[string]string `json:"key_value"` // Extracted facts
	CodeContext []CodeContext     `json:"code_context,omitempty"`
}

// CodeContext preserves code-specific context
type CodeContext struct {
	File    string   `json:"file"`
	Symbols []string `json:"symbols"`
	Lines   string   `json:"lines,omitempty"`
}

// Milestone represents a pending task
type Milestone struct {
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Status      string `json:"status"`
}

// ConversationTracker tracks conversation turns
type ConversationTracker struct {
	turns    []Turn
	maxTurns int
	mu       sync.RWMutex
}

// Turn represents a single conversation turn
type Turn struct {
	Role      string // "user" or "assistant"
	Content   string
	Timestamp time.Time
	Hash      string
	Tokens    int
}

// NewCompactionLayer creates a new compaction layer
func NewCompactionLayer(cfg CompactionConfig) *CompactionLayer {
	c := &CompactionLayer{
		config:         cfg,
		summarizer:     llm.NewSummarizerFromEnv(),
		sessionTracker: NewConversationTracker(cfg.PreserveRecentTurns),
		cache:          make(map[string]*CompactionResult),
		fallback:       NewSemanticFilter(),
	}

	// Set defaults
	if c.config.ThresholdLines == 0 {
		c.config.ThresholdLines = 100
	}
	if c.config.ThresholdTokens == 0 {
		c.config.ThresholdTokens = 2000
	}
	if c.config.PreserveRecentTurns == 0 {
		c.config.PreserveRecentTurns = 5
	}
	if c.config.MaxSummaryTokens == 0 {
		c.config.MaxSummaryTokens = 500
	}

	return c
}

// Name returns the filter name
func (c *CompactionLayer) Name() string {
	return "compaction"
}

// Apply applies compaction to the input
func (c *CompactionLayer) Apply(input string, mode Mode) (string, int) {
	// Check if enabled
	if !c.config.Enabled {
		return input, 0
	}

	// Check cache
	if c.config.CacheEnabled {
		cacheKey := c.hashContent(input)
		c.cacheMu.RLock()
		cached, ok := c.cache[cacheKey]
		c.cacheMu.RUnlock()
		if ok {
			return c.snapshotToString(cached.Snapshot), cached.SavedTokens
		}
	}

	// Detect content type
	if c.config.AutoDetect && !c.isCompactable(input) {
		return input, 0
	}

	// Check threshold
	originalTokens := EstimateTokens(input)
	if originalTokens < c.config.ThresholdTokens {
		return input, 0
	}

	// Parse turns from input
	turns := c.parseTurns(input)

	// If no turns detected, this isn't conversation content - return unchanged
	if len(turns) == 0 {
		return input, 0
	}

	// Create state snapshot
	snapshot := c.createSnapshot(turns, input)

	// Use LLM for semantic compression
	if c.summarizer.IsAvailable() {
		c.enrichWithLLM(snapshot, turns, input)
	}

	// Calculate savings
	output := c.snapshotToString(snapshot)
	finalTokens := EstimateTokens(output)
	savedTokens := originalTokens - finalTokens

	// Return original if compaction produced empty or invalid output
	if len(output) == 0 || finalTokens == 0 {
		return input, 0
	}

	// Return original if compaction didn't save tokens
	if savedTokens <= 0 {
		return input, 0
	}

	// Cache result
	if c.config.CacheEnabled && savedTokens > 0 {
		cacheKey := c.hashContent(input)
		c.cacheMu.Lock()
		// Evict oldest if at capacity
		if len(c.cache) >= maxCompactionCacheSize {
			oldest := ""
			oldestTime := time.Now()
			for k, v := range c.cache {
				if v.Timestamp.Before(oldestTime) {
					oldestTime = v.Timestamp
					oldest = k
				}
			}
			if oldest != "" {
				delete(c.cache, oldest)
			}
		}
		c.cache[cacheKey] = &CompactionResult{
			Snapshot:         snapshot,
			OriginalTokens:   originalTokens,
			FinalTokens:      finalTokens,
			SavedTokens:      savedTokens,
			CompressionRatio: float64(originalTokens) / float64(finalTokens),
			Timestamp:        time.Now(),
		}
		c.cacheMu.Unlock()
	}

	return output, savedTokens
}

// isCompactable checks if content is suitable for compaction
func (c *CompactionLayer) isCompactable(content string) bool {
	lower := strings.ToLower(content)

	// Detect chat/conversation patterns
	chatPatterns := []string{
		"turn ", "turn:", "user:", "assistant:", "query:",
		"session", "conversation", "chat",
		"q:", "a:", "question:", "answer:",
		">>>", "<<<", "```", // Code blocks often in conversations
	}

	matches := 0
	for _, pattern := range chatPatterns {
		if strings.Contains(lower, pattern) {
			matches++
		}
	}

	// If multiple chat patterns found, likely compactable
	return matches >= 2
}

// parseTurns parses conversation turns from content
// Pre-compiled regexes for parseTurns to avoid per-call compilation
var parseTurnPatterns = []struct {
	regex   *regexp.Regexp
	roleMap map[string]string
}{
	{
		regex: regexp.MustCompile(`(?i)^(user|assistant|system):\s*(.*)$`),
		roleMap: map[string]string{
			"user":      "user",
			"assistant": "assistant",
			"system":    "system",
		},
	},
	{
		regex: regexp.MustCompile(`(?i)^(turn\s*\d+):\s*(.*)$`),
		roleMap: map[string]string{
			"turn": "user",
		},
	},
	{
		regex: regexp.MustCompile(`(?i)^(q|question):\s*(.*)$`),
		roleMap: map[string]string{
			"q":        "user",
			"question": "user",
		},
	},
}

func (c *CompactionLayer) parseTurns(content string) []Turn {
	var turns []Turn

	lines := strings.Split(content, "\n")
	var currentTurn *Turn

	for _, line := range lines {
		matched := false

		for _, p := range parseTurnPatterns {
			if matches := p.regex.FindStringSubmatch(line); matches != nil {
				// Save previous turn
				if currentTurn != nil {
					turns = append(turns, *currentTurn)
				}

				// Start new turn
				role := matches[1]
				if mapped, ok := p.roleMap[strings.ToLower(role)]; ok {
					role = mapped
				}

				currentTurn = &Turn{
					Role:      role,
					Content:   matches[2],
					Timestamp: time.Now(),
					Hash:      c.hashContent(matches[2]),
					Tokens:    EstimateTokens(matches[2]),
				}
				matched = true
				break
			}
		}

		// Append to current turn if not a new turn marker
		if !matched && currentTurn != nil {
			currentTurn.Content += "\n" + line
			currentTurn.Tokens = EstimateTokens(currentTurn.Content)
		}
	}

	// Save last turn
	if currentTurn != nil {
		turns = append(turns, *currentTurn)
	}

	// If no turns detected, treat entire content as one user turn
	if len(turns) == 0 {
		turns = []Turn{{
			Role:      "user",
			Content:   content,
			Timestamp: time.Now(),
			Hash:      c.hashContent(content),
			Tokens:    EstimateTokens(content),
		}}
	}

	return turns
}

// createSnapshot creates a state snapshot from turns
func (c *CompactionLayer) createSnapshot(turns []Turn, originalContent string) *StateSnapshot {
	snapshot := &StateSnapshot{
		SessionHistory: SessionHistory{
			UserQueries: []string{},
			ActivityLog: []string{},
		},
		CurrentState: CurrentState{
			Focus:      "Processing",
			NextAction: "Continue",
		},
		Context: SnapshotContext{
			Critical: []string{},
			Working:  []string{},
			KeyValue: make(map[string]string),
		},
		PendingPlan: []Milestone{},
	}

	// Preserve recent turns verbatim
	recentStart := 0
	if len(turns) > c.config.PreserveRecentTurns {
		recentStart = len(turns) - c.config.PreserveRecentTurns
	}

	// Extract user queries and activity from older turns
	for i, turn := range turns {
		if i < recentStart {
			// Compress older turns
			if turn.Role == "user" {
				query := c.summarizeTurn(turn.Content)
				snapshot.SessionHistory.UserQueries = append(snapshot.SessionHistory.UserQueries, query)
			} else if turn.Role == "assistant" {
				activity := c.summarizeTurn(turn.Content)
				snapshot.SessionHistory.ActivityLog = append(snapshot.SessionHistory.ActivityLog, activity)
			}
		}
	}

	// Extract critical context from recent turns
	for i := recentStart; i < len(turns); i++ {
		turn := turns[i]
		critical := c.extractCritical(turn.Content)
		snapshot.Context.Critical = append(snapshot.Context.Critical, critical...)

		// Extract key-value pairs
		kv := c.extractKeyValuePairs(turn.Content)
		for k, v := range kv {
			snapshot.Context.KeyValue[k] = v
		}
	}

	// Limit context entries
	if len(snapshot.Context.Critical) > c.config.MaxContextEntries {
		snapshot.Context.Critical = snapshot.Context.Critical[:c.config.MaxContextEntries]
	}

	// Set current state from last turn
	if len(turns) > 0 {
		lastTurn := turns[len(turns)-1]
		snapshot.CurrentState.Focus = c.extractFocus(lastTurn.Content)
		snapshot.CurrentState.NextAction = c.inferNextAction(lastTurn.Content)
	}

	return snapshot
}

// enrichWithLLM uses LLM to enrich the snapshot
func (c *CompactionLayer) enrichWithLLM(snapshot *StateSnapshot, turns []Turn, originalContent string) {
	// Build prompt for LLM
	prompt := c.buildCompactionPrompt(snapshot, turns)

	req := llm.SummaryRequest{
		Content:   prompt,
		MaxTokens: c.config.MaxSummaryTokens,
		Intent:    "general",
	}

	resp, err := c.summarizer.Summarize(req)
	if err != nil {
		utils.Warn("compaction: LLM summarization failed", "error", err)
		return
	}

	// Parse LLM response to enrich snapshot
	c.parseLLMResponse(resp.Summary, snapshot)
}

// buildCompactionPrompt creates a prompt for LLM compaction
func (c *CompactionLayer) buildCompactionPrompt(snapshot *StateSnapshot, turns []Turn) string {
	var sb strings.Builder

	sb.WriteString("You are a context compactor. Create a concise state snapshot from this conversation.\n\n")
	sb.WriteString("Extract:\n")
	sb.WriteString("1. User queries (what the user asked)\n")
	sb.WriteString("2. Key activities (what was done)\n")
	sb.WriteString("3. Critical facts (must preserve)\n")
	sb.WriteString("4. Next action (what to do next)\n\n")

	sb.WriteString("Format as JSON:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"session_history\": {\"user_queries\": [...], \"activity_log\": [...]},\n")
	sb.WriteString("  \"current_state\": {\"focus\": \"...\", \"next_action\": \"...\"},\n")
	sb.WriteString("  \"context\": {\"critical\": [...], \"working\": [...]}\n")
	sb.WriteString("}\n\n")

	// Add turn summaries (limit context)
	maxTurns := 20
	start := 0
	if len(turns) > maxTurns {
		start = len(turns) - maxTurns
	}

	sb.WriteString("Conversation turns:\n")
	for i := start; i < len(turns); i++ {
		turn := turns[i]
		content := turn.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", turn.Role, content))
	}

	return sb.String()
}

// parseLLMResponse parses LLM response into snapshot
func (c *CompactionLayer) parseLLMResponse(response string, snapshot *StateSnapshot) {
	// Try to extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		// No valid JSON, use response as working context
		snapshot.Context.Working = append(snapshot.Context.Working, response)
		return
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var llmSnapshot struct {
		SessionHistory struct {
			UserQueries []string `json:"user_queries"`
			ActivityLog []string `json:"activity_log"`
		} `json:"session_history"`
		CurrentState struct {
			Focus      string `json:"focus"`
			NextAction string `json:"next_action"`
		} `json:"current_state"`
		Context struct {
			Critical []string `json:"critical"`
			Working  []string `json:"working"`
		} `json:"context"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &llmSnapshot); err != nil {
		// Fallback: use as working context
		snapshot.Context.Working = append(snapshot.Context.Working, response)
		return
	}

	// Merge LLM results into snapshot
	if len(llmSnapshot.SessionHistory.UserQueries) > 0 {
		snapshot.SessionHistory.UserQueries = append(snapshot.SessionHistory.UserQueries, llmSnapshot.SessionHistory.UserQueries...)
	}
	if len(llmSnapshot.SessionHistory.ActivityLog) > 0 {
		snapshot.SessionHistory.ActivityLog = append(snapshot.SessionHistory.ActivityLog, llmSnapshot.SessionHistory.ActivityLog...)
	}
	if llmSnapshot.CurrentState.Focus != "" {
		snapshot.CurrentState.Focus = llmSnapshot.CurrentState.Focus
	}
	if llmSnapshot.CurrentState.NextAction != "" {
		snapshot.CurrentState.NextAction = llmSnapshot.CurrentState.NextAction
	}
	if len(llmSnapshot.Context.Critical) > 0 {
		snapshot.Context.Critical = append(snapshot.Context.Critical, llmSnapshot.Context.Critical...)
	}
	if len(llmSnapshot.Context.Working) > 0 {
		snapshot.Context.Working = append(snapshot.Context.Working, llmSnapshot.Context.Working...)
	}
}

// snapshotToString converts snapshot to string format
func (c *CompactionLayer) snapshotToString(snapshot *StateSnapshot) string {
	if !c.config.StateSnapshotFormat {
		// Simple format without structure
		var parts []string
		parts = append(parts, snapshot.SessionHistory.UserQueries...)
		parts = append(parts, snapshot.SessionHistory.ActivityLog...)
		parts = append(parts, snapshot.Context.Critical...)
		parts = append(parts, snapshot.Context.Working...)
		return strings.Join(parts, "\n")
	}

	var sb strings.Builder

	sb.WriteString("[Conversation Summary]\n")
	sb.WriteString("<state_snapshot>\n")

	// Session History
	sb.WriteString("    <session_history>\n")
	sb.WriteString("        <user_queries>\n")
	for i, q := range snapshot.SessionHistory.UserQueries {
		if i < 10 { // Limit output
			sb.WriteString(fmt.Sprintf("            %d. %s\n", i+1, q))
		}
	}
	sb.WriteString("        </user_queries>\n")
	sb.WriteString("        <activity_log>\n")
	for i, a := range snapshot.SessionHistory.ActivityLog {
		if i < 10 { // Limit output
			sb.WriteString(fmt.Sprintf("            %d. %s\n", i+1, a))
		}
	}
	sb.WriteString("        </activity_log>\n")
	sb.WriteString("    </session_history>\n")

	// Current State
	sb.WriteString("    <current_state>\n")
	sb.WriteString(fmt.Sprintf("        <focus>%s</focus>\n", snapshot.CurrentState.Focus))
	sb.WriteString(fmt.Sprintf("        <next_action>%s</next_action>\n", snapshot.CurrentState.NextAction))
	sb.WriteString("    </current_state>\n")

	// Context
	sb.WriteString("    <context>\n")
	sb.WriteString("        <critical>\n")
	for i, c := range snapshot.Context.Critical {
		if i < 10 {
			sb.WriteString(fmt.Sprintf("            - %s\n", c))
		}
	}
	sb.WriteString("        </critical>\n")
	if len(snapshot.Context.Working) > 0 {
		sb.WriteString("        <working>\n")
		for i, w := range snapshot.Context.Working {
			if i < 5 {
				sb.WriteString(fmt.Sprintf("            - %s\n", w))
			}
		}
		sb.WriteString("        </working>\n")
	}
	if len(snapshot.Context.KeyValue) > 0 {
		sb.WriteString("        <key_value>\n")
		for k, v := range snapshot.Context.KeyValue {
			sb.WriteString(fmt.Sprintf("            - %s: %s\n", k, v))
		}
		sb.WriteString("        </key_value>\n")
	}
	sb.WriteString("    </context>\n")

	// Pending Plan
	if len(snapshot.PendingPlan) > 0 {
		sb.WriteString("    <pending_plan>\n")
		for _, m := range snapshot.PendingPlan {
			sb.WriteString(fmt.Sprintf("        - %s (priority: %d, status: %s)\n", m.Description, m.Priority, m.Status))
		}
		sb.WriteString("    </pending_plan>\n")
	}

	sb.WriteString("</state_snapshot>\n")

	return sb.String()
}

// summarizeTurn creates a brief summary of a turn
func (c *CompactionLayer) summarizeTurn(content string) string {
	// Truncate to reasonable length
	maxLen := 100
	if len(content) > maxLen {
		// Try to find a good break point
		breakPoints := []int{
			strings.Index(content, "\n"),
			strings.Index(content, ". "),
			strings.Index(content, "? "),
			strings.Index(content, "! "),
		}

		bestBreak := maxLen
		for _, bp := range breakPoints {
			if bp > 0 && bp < bestBreak && bp > 20 {
				bestBreak = bp
			}
		}

		if bestBreak < maxLen {
			return content[:bestBreak+1]
		}
		return content[:maxLen] + "..."
	}
	return content
}

// extractCritical extracts critical information from content
func (c *CompactionLayer) extractCritical(content string) []string {
	var critical []string

	// Use pre-compiled regexes for critical patterns
	criticalRes := []*regexp.Regexp{reCriticalError, reCriticalFile, reCriticalTodo}
	criticalExtracts := []func(m []string) string{
		func(m []string) string { return m[1] + ": " + m[2] },
		func(m []string) string { return "file: " + m[2] },
		func(m []string) string { return m[1] + ": " + m[2] },
	}

	for i, re := range criticalRes {
		if matches := re.FindStringSubmatch(content); matches != nil {
			critical = append(critical, criticalExtracts[i](matches))
		}
	}

	return critical
}

// extractKeyValuePairs extracts key-value pairs from content
func (c *CompactionLayer) extractKeyValuePairs(content string) map[string]string {
	kv := make(map[string]string)

	// Use pre-compiled regexes for KV patterns
	kvRes := []*regexp.Regexp{reKVGeneral, reKVDQuoted, reKVSQuoted}
	for _, re := range kvRes {
		matches := re.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			if len(m) >= 3 {
				key := strings.ToLower(m[1])
				value := strings.TrimSpace(m[2])
				// Filter common noise
				if key != "the" && key != "a" && key != "an" && len(value) > 0 && len(value) < 200 {
					kv[key] = value
				}
			}
		}
	}

	// Limit entries
	if len(kv) > c.config.MaxContextEntries {
		newKv := make(map[string]string)
		count := 0
		for k, v := range kv {
			if count >= c.config.MaxContextEntries {
				break
			}
			newKv[k] = v
			count++
		}
		kv = newKv
	}

	return kv
}

// extractFocus extracts the current focus from content
func (c *CompactionLayer) extractFocus(content string) string {
	// Look for action verbs
	verbs := []string{"implement", "fix", "add", "remove", "update", "create", "delete", "modify", "refactor"}
	lower := strings.ToLower(content)

	for _, verb := range verbs {
		if strings.Contains(lower, verb) {
			// Find context around verb
			idx := strings.Index(lower, verb)
			start := idx
			if start > 30 {
				start = idx - 30
			}
			end := idx + len(verb) + 50
			if end > len(content) {
				end = len(content)
			}
			focus := content[start:end]
			if start > 0 {
				focus = "..." + focus
			}
			if end < len(content) {
				focus = focus + "..."
			}
			return focus
		}
	}

	// Default: first 100 chars
	if len(content) > 100 {
		return content[:100] + "..."
	}
	return content
}

// inferNextAction infers the next action from content
func (c *CompactionLayer) inferNextAction(content string) string {
	lower := strings.ToLower(content)

	// Pattern-based inference using pre-compiled regexes
	if strings.Contains(lower, "next") || strings.Contains(lower, "then") {
		// Extract what follows
		nextRes := []*regexp.Regexp{reNextColon, reThenAction, reProceedTo}
		for _, re := range nextRes {
			if matches := re.FindStringSubmatch(lower); matches != nil {
				return strings.TrimSpace(matches[1])
			}
		}
	}

	// Question-based inference
	if strings.Contains(lower, "?") {
		return "Awaiting user's next query"
	}

	// Task completion inference
	if strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "finished") {
		return "Task completed, awaiting next query"
	}

	return "Continue processing"
}

// hashContent creates a hash of content for caching
func (c *CompactionLayer) hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// SetEnabled enables or disables the compaction layer
func (c *CompactionLayer) SetEnabled(enabled bool) {
	c.config.Enabled = enabled
}

// IsAvailable returns true if LLM is available
func (c *CompactionLayer) IsAvailable() bool {
	return c.summarizer.IsAvailable()
}

// GetStats returns compaction statistics
func (c *CompactionLayer) GetStats() map[string]any {
	return map[string]any{
		"enabled":          c.config.Enabled,
		"threshold_lines":  c.config.ThresholdLines,
		"threshold_tokens": c.config.ThresholdTokens,
		"preserve_recent":  c.config.PreserveRecentTurns,
		"llm_available":    c.summarizer.IsAvailable(),
		"cache_size":       len(c.cache),
		"state_snapshot":   c.config.StateSnapshotFormat,
	}
}

// Compact is a convenience function for one-shot compaction
func Compact(input string, cfg CompactionConfig) (string, *CompactionResult) {
	layer := NewCompactionLayer(cfg)
	output, saved := layer.Apply(input, ModeMinimal)

	result := &CompactionResult{
		OriginalTokens: EstimateTokens(input),
		FinalTokens:    EstimateTokens(output),
		SavedTokens:    saved,
		Timestamp:      time.Now(),
	}

	return output, result
}

// NewConversationTracker creates a new conversation tracker
func NewConversationTracker(maxTurns int) *ConversationTracker {
	if maxTurns <= 0 {
		maxTurns = 10
	}
	return &ConversationTracker{
		turns:    make([]Turn, 0, maxTurns),
		maxTurns: maxTurns,
	}
}

// AddTurn adds a turn to the tracker
func (t *ConversationTracker) AddTurn(role, content string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	turn := Turn{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Hash:      sha256Hash(content),
		Tokens:    EstimateTokens(content),
	}

	t.turns = append(t.turns, turn)

	// Evict oldest if over limit
	if len(t.turns) > t.maxTurns {
		t.turns = t.turns[1:]
	}
}

// GetTurns returns all turns
func (t *ConversationTracker) GetTurns() []Turn {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]Turn(nil), t.turns...)
}

// GetRecentTurns returns the most recent N turns
func (t *ConversationTracker) GetRecentTurns(n int) []Turn {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if n >= len(t.turns) {
		return append([]Turn(nil), t.turns...)
	}

	start := len(t.turns) - n
	return append([]Turn(nil), t.turns[start:]...)
}

// sha256Hash creates a SHA256 hash
func sha256Hash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
