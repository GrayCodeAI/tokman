package filter

import (
	"bufio"
	"container/heap"
	"strings"
)

// H2OFilter implements Heavy-Hitter Oracle compression.
// Research basis: "H2O: Heavy-Hitter Oracle for Efficient Generative Inference"
// (Zhang et al., NeurIPS 2023) - achieves 30x+ compression via intelligent eviction.
//
// Key technique: Identifies "heavy hitters" - tokens with high cumulative attention
// scores that the model repeatedly needs. Combines with:
// 1. Recent token window for local context
// 2. Attention sinks (initial tokens) for computational stability
//
// This is Layer 13 in the pipeline, implementing KV cache-style compression
// for text without requiring actual model attention scores.
type H2OFilter struct {
	config H2OConfig
}

// H2OConfig holds configuration for H2O compression
type H2OConfig struct {
	// Enable H2O filtering
	Enabled bool

	// Number of attention sink tokens to always preserve (first N tokens)
	SinkSize int

	// Number of recent tokens to always preserve
	RecentSize int

	// Number of heavy hitter tokens to preserve based on importance
	HeavyHitterSize int

	// Minimum content length to apply compression
	MinContentLength int

	// Window size for chunk processing
	ChunkWindow int
}

// DefaultH2OConfig returns default configuration
func DefaultH2OConfig() H2OConfig {
	return H2OConfig{
		Enabled:          true,
		SinkSize:         4,  // First 4 tokens are attention sinks
		RecentSize:       20, // Keep last 20 tokens for local context
		HeavyHitterSize:  40, // Keep top 40 heavy hitters
		MinContentLength: 100,
		ChunkWindow:      100, // Process in 100-token windows
	}
}

// NewH2OFilter creates a new H2O filter
func NewH2OFilter() *H2OFilter {
	return &H2OFilter{
		config: DefaultH2OConfig(),
	}
}

// Name returns the filter name
func (h *H2OFilter) Name() string {
	return "h2o"
}

// Apply applies H2O compression to the input
func (h *H2OFilter) Apply(input string, mode Mode) (string, int) {
	if !h.config.Enabled {
		return input, 0
	}

	// Skip short content
	if len(input) < h.config.MinContentLength {
		return input, 0
	}

	originalTokens := EstimateTokens(input)

	// For large content, use line-based processing (memory efficient)
	// Line-based approach reduces allocations by 10-20x
	if originalTokens > 50000 {
		return h.applyLineBased(input, mode, originalTokens)
	}

	// Tokenize
	tokens := h.tokenize(input)
	if len(tokens) < h.config.SinkSize+h.config.RecentSize+h.config.HeavyHitterSize {
		return input, 0
	}

	// Calculate importance scores (simulated attention)
	scores := h.calculateImportance(tokens, input)

	// Build heavy hitter priority queue
	heavyHitters := h.identifyHeavyHitters(tokens, scores)

	// Build output: sinks + heavy hitters + recent
	output := h.buildOutput(tokens, heavyHitters, scores)

	finalTokens := EstimateTokens(output)
	saved := originalTokens - finalTokens

	// Return original if we didn't save much
	if saved < 5 {
		return input, 0
	}

	return output, saved
}

// applyLineBased processes content line-by-line for memory efficiency
// Used for large contexts (>50K tokens) to reduce memory overhead
// Uses streaming to avoid allocating full string slice
func (h *H2OFilter) applyLineBased(input string, mode Mode, originalTokens int) (string, int) {
	// First pass: count lines and collect line indices (streaming)
	lineCount := 0
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		lineCount++
	}
	if scanner.Err() != nil {
		return input, 0
	}

	n := lineCount
	if n < h.config.SinkSize+h.config.RecentSize+10 {
		return input, 0
	}

	// Second pass: score lines and store minimal data
	// Only store scores for middle section (not sinks or recent)
	recentStart := n - h.config.RecentSize
	if recentStart < h.config.SinkSize {
		recentStart = h.config.SinkSize
	}

	// Collect scores for middle section only
	type lineScore struct {
		index int
		score float64
		text  string
	}
	middleLines := make([]lineScore, 0, recentStart-h.config.SinkSize)

	lineIdx := 0
	scanner = bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if lineIdx >= h.config.SinkSize && lineIdx < recentStart {
			if len(strings.TrimSpace(line)) > 0 {
				middleLines = append(middleLines, lineScore{
					index: lineIdx,
					score: h.scoreLine(line, lineIdx, n),
					text:  line,
				})
			}
		}
		lineIdx++
	}
	if scanner.Err() != nil {
		return input, 0
	}

	// Build keep set for middle lines using heap
	hh := &tokenHeap{}
	heap.Init(hh)

	for _, ls := range middleLines {
		heap.Push(hh, &scoredToken{
			index: ls.index,
			score: ls.score,
		})
	}

	// Extract heavy hitter indices
	keepMiddle := make(map[int]bool)
	heavyHitterCount := h.config.HeavyHitterSize
	for hh.Len() > 0 && heavyHitterCount > 0 {
		st := heap.Pop(hh).(*scoredToken)
		keepMiddle[st.index] = true
		heavyHitterCount--
	}

	// Third pass: build output (streaming)
	var result strings.Builder
	result.Grow(originalTokens * 4 / 10) // Pre-allocate ~40% of original

	lineIdx = 0
	scanner = bufio.NewScanner(strings.NewReader(input))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		keep := false

		if lineIdx < h.config.SinkSize {
			keep = true // Sink
		} else if lineIdx >= recentStart {
			keep = true // Recent
		} else if keepMiddle[lineIdx] {
			keep = true // Heavy hitter
		}

		if keep {
			if !first {
				result.WriteString("\n")
			}
			result.WriteString(line)
			first = false
		}
		lineIdx++
	}
	if scanner.Err() != nil {
		return input, 0
	}

	output := result.String()
	finalTokens := EstimateTokens(output)
	saved := originalTokens - finalTokens

	if saved < 5 {
		return input, 0
	}

	return output, saved
}

// scoreLine calculates importance score for a single line
func (h *H2OFilter) scoreLine(line string, index, totalLines int) float64 {
	var score float64

	// Positional weight
	pos := float64(index) / float64(totalLines)

	// Sinks (already handled, but score anyway)
	if index < h.config.SinkSize {
		score += 1.0
	}

	// Recent lines get boost
	if pos > 0.8 {
		score += 0.5 * (pos - 0.8) / 0.2
	}

	lineLower := strings.ToLower(line)

	// Important keywords
	keywords := []string{
		"error", "fail", "warning", "success", "done", "complete",
		"file:", "line:", "path:", "function:", "class:", "method:",
		"http://", "https://", "import", "export", "return",
		"---", "===", "***", "```",
	}
	for _, kw := range keywords {
		if strings.Contains(lineLower, kw) {
			score += 0.4
			break
		}
	}

	// Structural markers
	if strings.HasSuffix(line, ":") || strings.HasPrefix(line, "#") {
		score += 0.3
	}

	// Non-empty lines score higher
	if len(strings.TrimSpace(line)) > 0 {
		score += 0.2
	}

	// Very long lines might be important (code, paths)
	if len(line) > 100 {
		score += 0.2
	}

	return score
}

// token represents a token with position info
type h2oToken struct {
	text  string
	start int
	end   int
	index int
}

// tokenize splits content into tokens
func (h *H2OFilter) tokenize(content string) []h2oToken {
	var tokens []h2oToken

	// Split by words and whitespace
	wordStart := -1
	for i, c := range content {
		if isWordChar(c) {
			if wordStart == -1 {
				wordStart = i
			}
		} else {
			if wordStart != -1 {
				tokens = append(tokens, h2oToken{
					text:  content[wordStart:i],
					start: wordStart,
					end:   i,
					index: len(tokens),
				})
				wordStart = -1
			}
			// Add punctuation/whitespace as separate tokens
			tokens = append(tokens, h2oToken{
				text:  string(c),
				start: i,
				end:   i + 1,
				index: len(tokens),
			})
		}
	}
	if wordStart != -1 {
		tokens = append(tokens, h2oToken{
			text:  content[wordStart:],
			start: wordStart,
			end:   len(content),
			index: len(tokens),
		})
	}

	return tokens
}

// isWordChar is defined in ngram.go

// calculateImportance calculates importance scores for each token
// Simulates attention patterns without actual model
func (h *H2OFilter) calculateImportance(tokens []h2oToken, content string) []float64 {
	n := len(tokens)
	scores := make([]float64, n)

	// Track word frequency
	freq := make(map[string]int)
	for _, t := range tokens {
		word := strings.ToLower(strings.TrimSpace(t.text))
		if len(word) > 0 && !isWhitespace(t.text) {
			freq[word]++
		}
	}

	// Track position weights
	for i, t := range tokens {
		// Skip whitespace in scoring
		if isWhitespace(t.text) {
			scores[i] = 0.1 // Low but non-zero for structure
			continue
		}

		var score float64

		// 1. Positional importance (sinks and end)
		pos := float64(i) / float64(n)

		// Sinks: first few tokens get high scores
		if i < h.config.SinkSize {
			score += 1.0 - float64(i)/float64(h.config.SinkSize)*0.5
		}

		// Recent tokens: last portion gets boost
		if pos > 0.8 {
			score += 0.5 * (pos - 0.8) / 0.2
		}

		// 2. Semantic importance
		word := strings.ToLower(t.text)

		// Important keywords
		keywords := []string{
			"error", "fail", "warning", "success", "done", "complete",
			"file", "path", "line", "function", "class", "method",
			"import", "export", "return", "def", "var", "const",
			"http", "api", "url", "id", "key", "token", "auth",
			"config", "env", "debug", "info", "trace",
		}
		for _, kw := range keywords {
			if strings.Contains(word, kw) {
				score += 0.4
				break
			}
		}

		// Numbers are important
		if isNumeric(t.text) {
			score += 0.5
		}

		// File paths and URLs (check if looks like part of a path)
		if isFilePath(t.text) || isURL(t.text) {
			score += 0.8
		}
		// Path-like tokens (contain / or \)
		if strings.Contains(t.text, "/") || strings.Contains(t.text, "\\") {
			score += 0.7
		}
		// File extensions
		for _, ext := range []string{".go", ".py", ".js", ".ts", ".json", ".yaml", ".yml", ".toml", ".md"} {
			if strings.HasSuffix(t.text, ext) {
				score += 0.7
				break
			}
		}

		// Code symbols
		if isCodeSymbol(t.text) {
			score += 0.3
		}

		// 3. Frequency-based (unique words are often important)
		if freq[word] == 1 && len(word) > 3 {
			score += 0.2
		} else if freq[word] > 5 {
			// Very common words get slight penalty
			score -= 0.1
		}

		// 4. Structural markers
		if strings.HasSuffix(t.text, ":") || strings.HasSuffix(t.text, "=") {
			score += 0.3
		}

		// 5. Length-based (very short tokens often less important)
		if len(t.text) <= 2 && !isCodeSymbol(t.text) && !isNumeric(t.text) {
			score -= 0.15
		}

		// Ensure non-negative
		if score < 0 {
			score = 0
		}
		scores[i] = score
	}

	return scores
}

// identifyHeavyHitters finds tokens with highest importance scores
func (h *H2OFilter) identifyHeavyHitters(tokens []h2oToken, scores []float64) map[int]bool {
	n := len(tokens)
	keep := make(map[int]bool)

	// Always keep sinks
	for i := 0; i < h.config.SinkSize && i < n; i++ {
		keep[i] = true
	}

	// Always keep recent
	recentStart := n - h.config.RecentSize
	if recentStart < h.config.SinkSize {
		recentStart = h.config.SinkSize
	}
	for i := recentStart; i < n; i++ {
		keep[i] = true
	}

	// Use a max-heap to find heavy hitters
	hh := &tokenHeap{}
	heap.Init(hh)

	for i := h.config.SinkSize; i < recentStart; i++ {
		if !isWhitespace(tokens[i].text) {
			heap.Push(hh, &scoredToken{
				index: i,
				score: scores[i],
			})
		}
	}

	// Extract top heavy hitters
	for hh.Len() > 0 && len(keep) < h.config.SinkSize+h.config.RecentSize+h.config.HeavyHitterSize {
		st := heap.Pop(hh).(*scoredToken)
		keep[st.index] = true
	}

	return keep
}

// scoredToken is a token with importance score
type scoredToken struct {
	index int
	score float64
}

// tokenHeap implements heap.Interface for max-heap
type tokenHeap []*scoredToken

func (h tokenHeap) Len() int           { return len(h) }
func (h tokenHeap) Less(i, j int) bool { return h[i].score > h[j].score } // Max heap
func (h tokenHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *tokenHeap) Push(x interface{}) {
	*h = append(*h, x.(*scoredToken))
}

func (h *tokenHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// buildOutput constructs the compressed output
func (h *H2OFilter) buildOutput(tokens []h2oToken, keep map[int]bool, scores []float64) string {
	var result strings.Builder
	lastKept := -1

	for i, t := range tokens {
		if keep[i] {
			// Add space if we skipped tokens
			if lastKept >= 0 && i > lastKept+1 && !isWhitespace(t.text) {
				// Check if previous kept token ended with space
				if lastKept >= 0 && !endsWithSpace(tokens[lastKept].text) {
					result.WriteString(" ")
				}
			}
			result.WriteString(t.text)
			lastKept = i
		}
	}

	return result.String()
}

// isWhitespace checks if string is only whitespace
func isWhitespace(s string) bool {
	for _, c := range s {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return len(s) > 0
}

// isNumeric is defined in perplexity.go

// endsWithSpace checks if string ends with whitespace
func endsWithSpace(s string) bool {
	if len(s) == 0 {
		return false
	}
	c := s[len(s)-1]
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// SetEnabled enables or disables the filter
func (h *H2OFilter) SetEnabled(enabled bool) {
	h.config.Enabled = enabled
}

// GetStats returns filter statistics
func (h *H2OFilter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":           h.config.Enabled,
		"sink_size":         h.config.SinkSize,
		"recent_size":       h.config.RecentSize,
		"heavy_hitter_size": h.config.HeavyHitterSize,
	}
}
