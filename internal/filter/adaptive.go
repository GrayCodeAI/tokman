package filter

import (
	"strings"
)

// AdaptiveLayerSelector dynamically enables/disables layers based on content type.
// Uses heuristic analysis to optimize compression for different input patterns.
type AdaptiveLayerSelector struct {
	// Thresholds for content type detection
	codeThreshold      float64
	logThreshold       float64
	conversationThreshold float64
}

// NewAdaptiveLayerSelector creates a new adaptive selector
func NewAdaptiveLayerSelector() *AdaptiveLayerSelector {
	return &AdaptiveLayerSelector{
		codeThreshold:          0.15,
		logThreshold:           0.3,
		conversationThreshold:  0.2,
	}
}

// ContentType represents the detected content type
type ContentType int

const (
	ContentTypeUnknown ContentType = iota
	ContentTypeCode
	ContentTypeLogs
	ContentTypeConversation
	ContentTypeGitOutput
	ContentTypeTestOutput
	ContentTypeDockerOutput
	ContentTypeMixed
)

// AnalyzeContent detects the primary content type
func (a *AdaptiveLayerSelector) AnalyzeContent(input string) ContentType {
	if len(input) == 0 {
		return ContentTypeUnknown
	}

	scores := make(map[ContentType]float64)
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return ContentTypeUnknown
	}

	// Code indicators
	codeIndicators := []string{"func ", "func(", "class ", "def ", "import ", "package ", "const ", "var ", "type ", "interface ", "struct ", "return ", "if ", "for ", "while "}
	codeCount := 0
	for _, line := range lines {
		for _, ind := range codeIndicators {
			if strings.Contains(line, ind) {
				codeCount++
				break
			}
		}
	}
	scores[ContentTypeCode] = float64(codeCount) / float64(len(lines))

	// Log indicators
	logIndicators := []string{"INFO", "WARN", "ERROR", "DEBUG", "TRACE", "FATAL", "[", "]", "timestamp", "level"}
	logCount := 0
	for _, line := range lines {
		for _, ind := range logIndicators {
			if strings.Contains(line, ind) {
				logCount++
				break
			}
		}
	}
	scores[ContentTypeLogs] = float64(logCount) / float64(len(lines))

	// Conversation indicators
	convIndicators := []string{"User:", "Assistant:", "AI:", "Human:", "System:", "Q:", "A:", "Question:", "Answer:"}
	convCount := 0
	for _, line := range lines {
		for _, ind := range convIndicators {
			if strings.Contains(line, ind) {
				convCount++
				break
			}
		}
	}
	scores[ContentTypeConversation] = float64(convCount) / float64(len(lines))

	// Git output indicators
	if strings.Contains(input, "commit ") || strings.Contains(input, "branch ") || strings.Contains(input, "modified:") || strings.Contains(input, "On branch") {
		scores[ContentTypeGitOutput] = 0.8
	}

	// Test output indicators
	if strings.Contains(input, "=== RUN") || strings.Contains(input, "--- PASS") || strings.Contains(input, "--- FAIL") || strings.Contains(input, "PASS") || strings.Contains(input, "FAIL") {
		scores[ContentTypeTestOutput] = 0.8
	}

	// Docker output indicators
	if strings.Contains(input, "CONTAINER ID") || strings.Contains(input, "IMAGE") || strings.Contains(input, "docker") || strings.Contains(input, "kubectl") {
		scores[ContentTypeDockerOutput] = 0.8
	}

	// Find dominant content type
	maxScore := 0.0
	dominant := ContentTypeUnknown
	for ct, score := range scores {
		if score > maxScore {
			maxScore = score
			dominant = ct
		}
	}

	// If multiple types have similar scores, mark as mixed
	count := 0
	for _, score := range scores {
		if score > 0.3 {
			count++
		}
	}
	if count > 1 {
		return ContentTypeMixed
	}

	return dominant
}

// RecommendedConfig returns optimal layer configuration for the content type
func (a *AdaptiveLayerSelector) RecommendedConfig(ct ContentType, mode Mode) PipelineConfig {
	config := PipelineConfig{
		Mode:            mode,
		Budget:          4000, // Default budget
		SessionTracking: false,
		NgramEnabled:    true,
	}

	switch ct {
	case ContentTypeCode:
		// Code benefits from semantic dedup, budget enforcement
		config.EnableCompaction = false // Don't summarize code
		config.EnableAttribution = true
		config.EnableH2O = true
		config.EnableAttentionSink = true

	case ContentTypeLogs:
		// Logs benefit from entropy filtering, n-gram merger
		config.EnableCompaction = false
		config.EnableAttribution = false
		config.EnableH2O = true
		config.EnableAttentionSink = true

	case ContentTypeConversation:
		// Conversations benefit from session tracking, compaction
		config.SessionTracking = true
		config.EnableCompaction = true
		config.EnableAttribution = true
		config.EnableH2O = true
		config.EnableAttentionSink = true

	case ContentTypeGitOutput:
		// Git output is already structured, minimal filtering
		config.EnableCompaction = false
		config.EnableAttribution = false
		config.EnableH2O = false
		config.EnableAttentionSink = false

	case ContentTypeTestOutput:
		// Test output benefits from aggregation
		config.EnableCompaction = false
		config.EnableAttribution = false
		config.EnableH2O = true
		config.EnableAttentionSink = false

	case ContentTypeDockerOutput:
		// Docker/infra output is structured
		config.EnableCompaction = false
		config.EnableAttribution = false
		config.EnableH2O = true
		config.EnableAttentionSink = false

	case ContentTypeMixed:
		// Mixed content: enable most layers
		config.SessionTracking = true
		config.EnableCompaction = true
		config.EnableAttribution = true
		config.EnableH2O = true
		config.EnableAttentionSink = true

	default:
		// Unknown: use safe defaults
		config.EnableCompaction = false
		config.EnableAttribution = true
		config.EnableH2O = true
		config.EnableAttentionSink = true
	}

	return config
}

// OptimizePipeline returns an optimized coordinator for the given input
func (a *AdaptiveLayerSelector) OptimizePipeline(input string, mode Mode) *PipelineCoordinator {
	ct := a.AnalyzeContent(input)
	config := a.RecommendedConfig(ct, mode)
	return NewPipelineCoordinator(config)
}

// ContentTypeString returns a human-readable content type name
func (ct ContentType) String() string {
	switch ct {
	case ContentTypeCode:
		return "code"
	case ContentTypeLogs:
		return "logs"
	case ContentTypeConversation:
		return "conversation"
	case ContentTypeGitOutput:
		return "git"
	case ContentTypeTestOutput:
		return "test"
	case ContentTypeDockerOutput:
		return "docker/infra"
	case ContentTypeMixed:
		return "mixed"
	default:
		return "unknown"
	}
}
