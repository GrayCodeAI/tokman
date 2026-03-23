package filter

import (
	"encoding/json"
	"strings"
)

// ContentRouter selects optimal compression strategy based on content type.
// R4: Claw-compactor style content routing — JSON→JSON parser, code→AST, logs→dedup.
type ContentRouter struct {
	detector *AdaptiveLayerSelector
}

// NewContentRouter creates a content router.
func NewContentRouter() *ContentRouter {
	return &ContentRouter{
		detector: NewAdaptiveLayerSelector(),
	}
}

// Route detects content type and returns optimal pipeline config.
func (r *ContentRouter) Route(input string) (ContentType, PipelineConfig) {
	ct := r.detector.AnalyzeContent(input)
	cfg := r.detector.RecommendedConfig(ct, ModeMinimal)
	return ct, cfg
}

// RouteWithMode detects content type and returns config for given mode.
func (r *ContentRouter) RouteWithMode(input string, mode Mode) (ContentType, PipelineConfig) {
	ct := r.detector.AnalyzeContent(input)
	cfg := r.detector.RecommendedConfig(ct, mode)
	return ct, cfg
}

// IsJSON checks if input is valid JSON.
func IsJSON(input string) bool {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) == 0 {
		return false
	}
	first := trimmed[0]
	if first != '{' && first != '[' {
		return false
	}
	var js json.RawMessage
	return json.Unmarshal([]byte(trimmed), &js) == nil
}

// IsDiff checks if input looks like a diff.
func IsDiff(input string) bool {
	lines := strings.Split(input, "\n")
	if len(lines) < 2 {
		return false
	}
	diffLines := 0
	checkCount := len(lines)
	if checkCount > 20 {
		checkCount = 20
	}
	for _, line := range lines[:checkCount] {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "diff --git") {
			diffLines++
		}
	}
	return diffLines >= 2
}

// CompressJSON applies JSON-specific compression.
// Removes whitespace, truncates large values, deduplicates keys.
func CompressJSON(input string) string {
	trimmed := strings.TrimSpace(input)
	if !IsJSON(trimmed) {
		return input
	}

	// Parse and re-serialize compactly
	var data any
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		return input
	}

	// Truncate large string values
	data = truncateJSONValues(data, 200)

	compact, err := json.Marshal(data)
	if err != nil {
		return input
	}

	return string(compact)
}

// truncateJSONValues recursively truncates long string values in JSON.
func truncateJSONValues(v any, maxLen int) any {
	switch val := v.(type) {
	case string:
		if len(val) > maxLen {
			return val[:maxLen] + "... (truncated)"
		}
		return val
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = truncateJSONValues(v, maxLen)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = truncateJSONValues(v, maxLen)
		}
		return result
	default:
		return v
	}
}
