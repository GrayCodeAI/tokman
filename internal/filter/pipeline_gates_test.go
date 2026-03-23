package filter

import (
	"strings"
	"testing"
)

// T7: Stage Gates Tests

func TestStageGates_ShortContent(t *testing.T) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:             ModeAggressive,
		EnableEntropy:    true,
		EnablePerplexity: true,
		EnableH2O:        true,
		EnableAttentionSink: true,
	})

	// Very short content should skip many layers
	shortContent := "hello world"
	output, stats := p.Process(shortContent)

	// Should return original or minimally modified
	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	// Most layers should be skipped due to stage gates
	// Only budget enforcement and session tracking might run
	skippedLayers := 0
	for _, layer := range []string{"1_entropy", "2_perplexity", "13_h2o", "14_attention_sink"} {
		if _, exists := stats.LayerStats[layer]; !exists {
			skippedLayers++
		}
	}

	if skippedLayers < 2 {
		t.Errorf("Expected at least 2 layers to be skipped for short content, got %d", skippedLayers)
	}
}

func TestStageGates_NoQueryIntent(t *testing.T) {
	// Without query intent, goal-driven and contrastive should be skipped
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:             ModeAggressive,
		QueryIntent:      "", // No query
		EnableGoalDriven: true,
		EnableContrastive: true,
	})

	content := strings.Repeat("test line with some content\n", 100)
	_, stats := p.Process(content)

	// Goal-driven and contrastive should not have run
	if _, exists := stats.LayerStats["3_goal_driven"]; exists {
		t.Error("Goal-driven layer should be skipped when no query intent")
	}
	if _, exists := stats.LayerStats["5_contrastive"]; exists {
		t.Error("Contrastive layer should be skipped when no query intent")
	}
}

func TestStageGates_WithQueryIntent(t *testing.T) {
	// With query intent, goal-driven and contrastive should run
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:             ModeAggressive,
		QueryIntent:      "find the error in the code",
		EnableGoalDriven: true,
		EnableContrastive: true,
	})

	content := strings.Repeat("test line with some content\n", 100)
	_, _ = p.Process(content)

	// Goal-driven and contrastive should have stats (may have 0 savings but still run)
	// Note: They may not save tokens if content doesn't match, but they shouldn't be skipped
	if p.shouldSkipQueryDependent() {
		t.Error("Query-dependent should not be skipped with query intent")
	}
}

func TestStageGates_ConversationContent(t *testing.T) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:             ModeAggressive,
		EnableCompaction: true,
	})

	// Conversation-like content should NOT skip compaction
	conversation := `User: Hello, how are you?
Assistant: I'm doing well, thank you for asking!

User: Can you help me with a coding problem?
Assistant: Of course! What seems to be the issue?

User: I'm getting an error in my Python script.
Assistant: Please share the error message and I'll help you debug it.`

	if p.shouldSkipCompaction(conversation) {
		t.Error("Compaction should NOT be skipped for conversation content")
	}

	// Non-conversation content should skip compaction
	plainText := strings.Repeat("This is just plain text without any conversation markers.\n", 50)
	if !p.shouldSkipCompaction(plainText) {
		t.Error("Compaction SHOULD be skipped for non-conversation content")
	}
}

func TestStageGates_H2OThreshold(t *testing.T) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:      ModeAggressive,
		EnableH2O: true,
	})

	// Short content should skip H2O
	shortContent := "short content"
	if !p.shouldSkipH2O(shortContent) {
		t.Error("H2O should be skipped for very short content")
	}

	// Long content should NOT skip H2O
	longContent := strings.Repeat("This is a longer piece of content that should trigger H2O filtering. ", 100)
	if p.shouldSkipH2O(longContent) {
		t.Error("H2O should NOT be skipped for long content")
	}
}

func TestStageGates_BudgetAware(t *testing.T) {
	// With budget, sketch store and lazy pruner should be considered
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:              ModeAggressive,
		Budget:            1000,
		EnableSketchStore: true,
		EnableLazyPruner:  true,
	})

	if p.shouldSkipBudgetDependent() {
		t.Error("Budget-dependent should NOT be skipped when budget is set")
	}

	// Without budget, they should be skipped
	p2 := NewPipelineCoordinator(PipelineConfig{
		Mode:              ModeAggressive,
		Budget:            0,
		EnableSketchStore: true,
		EnableLazyPruner:  true,
	})

	if !p2.shouldSkipBudgetDependent() {
		t.Error("Budget-dependent SHOULD be skipped when no budget")
	}
}

func TestStageGates_EntropyDensity(t *testing.T) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:          ModeAggressive,
		EnableEntropy: true,
	})

	// Dense content (many unique chars) should NOT skip entropy
	denseContent := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()"
	if p.shouldSkipEntropy(denseContent) {
		t.Error("Entropy should NOT be skipped for dense content")
	}

	// Very short content should skip entropy
	tooShort := "hi"
	if !p.shouldSkipEntropy(tooShort) {
		t.Error("Entropy SHOULD be skipped for very short content")
	}
}

func TestStageGates_PerformanceBenefit(t *testing.T) {
	// Verify that stage gates actually save processing time
	content := "short"

	p := NewPipelineCoordinator(PipelineConfig{
		Mode:                ModeAggressive,
		EnableEntropy:       true,
		EnablePerplexity:    true,
		EnableH2O:           true,
		EnableAttentionSink: true,
		EnableCompaction:    true,
	})

	_, stats := p.Process(content)

	// Count skipped layers
	skippedCount := 0
	expectedLayers := []string{"1_entropy", "2_perplexity", "11_compaction", "13_h2o", "14_attention_sink"}
	for _, layer := range expectedLayers {
		if _, exists := stats.LayerStats[layer]; !exists {
			skippedCount++
		}
	}

	// Most layers should be skipped for short content
	if skippedCount < 3 {
		t.Errorf("Expected at least 3 layers to be skipped for short content, got %d", skippedCount)
	}
}
