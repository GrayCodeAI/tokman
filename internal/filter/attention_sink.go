package filter

import (
	"strings"
)

// AttentionSinkFilter implements StreamingLLM-style attention sink preservation.
// Research basis: "Efficient Streaming Language Models with Attention Sinks"
// (Xiao et al., 2023) - enables infinite-length generation with bounded memory.
//
// Key insight: The first few tokens in a sequence act as "attention sinks" -
// they absorb excess attention weight due to softmax normalization. Removing
// these tokens breaks the attention distribution the model learned during training.
//
// This layer:
// 1. Always preserves initial tokens (attention sinks)
// 2. Preserves structural anchors (headers, prefixes, markers)
// 3. Applies rolling cache to remaining content
//
// This is Layer 14 in the pipeline, ensuring stable compression for long content.
type AttentionSinkFilter struct {
	config SinkConfig
}

// SinkConfig holds configuration for attention sink preservation
type SinkConfig struct {
	// Enable attention sink filtering
	Enabled bool

	// Number of initial tokens to always preserve as sinks
	SinkTokenCount int

	// Number of recent tokens to preserve in rolling cache
	RecentTokenCount int

	// Preserve structural markers (headers, prefixes)
	PreserveStructural bool

	// Minimum content length to apply
	MinContentLength int

	// Anchor patterns to always preserve
	AnchorPatterns []string
}

// DefaultSinkConfig returns default configuration
func DefaultSinkConfig() SinkConfig {
	return SinkConfig{
		Enabled:            true,
		SinkTokenCount:     4, // First 4 tokens are attention sinks
		RecentTokenCount:   8, // Keep last 8 lines (rolling cache)
		PreserveStructural: true,
		MinContentLength:   100,
		AnchorPatterns: []string{
			"Error:", "Warning:", "Success:", "Failed:", "CRITICAL:",
			"INFO:", "DEBUG:", "TRACE:", "FATAL:",
			"File:", "Line:", "Function:", "Class:",
			"---", "===", "###", "***",
			"http://", "https://",
		},
	}
}

// NewAttentionSinkFilter creates a new attention sink filter
func NewAttentionSinkFilter() *AttentionSinkFilter {
	return &AttentionSinkFilter{
		config: DefaultSinkConfig(),
	}
}

// NewAdaptiveAttentionSinkFilter creates a filter with adaptive sink count.
// R5: StreamingLLM insight — sink count scales with output length.
func NewAdaptiveAttentionSinkFilter(outputLines int) *AttentionSinkFilter {
	cfg := DefaultSinkConfig()

	// Adaptive: more sinks for longer content
	if outputLines > 500 {
		cfg.SinkTokenCount = 8
		cfg.RecentTokenCount = 16
	} else if outputLines > 100 {
		cfg.SinkTokenCount = 6
		cfg.RecentTokenCount = 12
	} else if outputLines > 20 {
		cfg.SinkTokenCount = 4
		cfg.RecentTokenCount = 8
	} else {
		cfg.SinkTokenCount = 2
		cfg.RecentTokenCount = 4
	}

	return &AttentionSinkFilter{config: cfg}
}

// Name returns the filter name
func (a *AttentionSinkFilter) Name() string {
	return "attention_sink"
}

// Apply applies attention sink preservation to the input
func (a *AttentionSinkFilter) Apply(input string, mode Mode) (string, int) {
	if !a.config.Enabled {
		return input, 0
	}

	// Skip short content
	if len(input) < a.config.MinContentLength {
		return input, 0
	}

	originalTokens := EstimateTokens(input)

	// Tokenize into lines for better structure preservation
	lines := strings.Split(input, "\n")
	if len(lines) < 3 {
		return input, 0
	}

	// Identify sink lines (first N meaningful lines)
	sinkLines := a.identifySinkLines(lines)

	// Identify anchor lines (structural markers)
	anchorLines := a.identifyAnchorLines(lines)

	// Identify recent lines (last N lines)
	recentStart := len(lines) - a.config.RecentTokenCount
	if recentStart < 0 {
		recentStart = 0
	}

	// Build the keep set
	keep := make(map[int]bool)

	// Always keep sinks
	for _, idx := range sinkLines {
		keep[idx] = true
	}

	// Always keep anchors
	for _, idx := range anchorLines {
		keep[idx] = true
	}

	// Always keep recent
	for i := recentStart; i < len(lines); i++ {
		keep[i] = true
	}

	// Build output
	var result strings.Builder
	for i, line := range lines {
		if keep[i] {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(line)
		}
	}

	output := result.String()
	finalTokens := EstimateTokens(output)
	saved := originalTokens - finalTokens

	// Return the compressed output with savings
	// Note: even small savings accumulate across pipeline layers
	return output, saved
}

// identifySinkLines identifies the first N meaningful lines as sinks
func (a *AttentionSinkFilter) identifySinkLines(lines []string) []int {
	var sinks []int
	count := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue // Skip empty lines
		}

		sinks = append(sinks, i)
		count++

		if count >= a.config.SinkTokenCount {
			break
		}
	}

	return sinks
}

// identifyAnchorLines identifies lines with structural markers
func (a *AttentionSinkFilter) identifyAnchorLines(lines []string) []int {
	if !a.config.PreserveStructural {
		return nil
	}

	var anchors []int

	for i, line := range lines {
		for _, pattern := range a.config.AnchorPatterns {
			if strings.Contains(line, pattern) {
				anchors = append(anchors, i)
				break
			}
		}
	}

	return anchors
}

// SetEnabled enables or disables the filter
func (a *AttentionSinkFilter) SetEnabled(enabled bool) {
	a.config.Enabled = enabled
}

// GetStats returns filter statistics
func (a *AttentionSinkFilter) GetStats() map[string]any {
	return map[string]any{
		"enabled":             a.config.Enabled,
		"sink_token_count":    a.config.SinkTokenCount,
		"recent_token_count":  a.config.RecentTokenCount,
		"preserve_structural": a.config.PreserveStructural,
	}
}
