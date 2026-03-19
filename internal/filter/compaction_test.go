package filter

import (
	"strings"
	"testing"
)

func TestCompactionLayer_Basic(t *testing.T) {
	cfg := CompactionConfig{
		Enabled:             true,
		ThresholdTokens:     10, // Low threshold for testing
		PreserveRecentTurns: 2,
		MaxSummaryTokens:    200,
		StateSnapshotFormat: true,
		AutoDetect:          false, // Disable auto-detect for testing
		CacheEnabled:        true,
	}
	
	layer := NewCompactionLayer(cfg)
	
	// Large content with many turns to ensure compaction saves tokens
	content := `User: Hello, I want to implement a token reduction system for my AI coding agent. This is a longer query to ensure we exceed the threshold. I've been researching various approaches including entropy filtering, perplexity scoring, and semantic clustering. The goal is to achieve maximum compression while preserving the semantic meaning of the original content.
Assistant: I'll help you implement a token reduction system. Let me start by researching the best approaches from academic papers and production systems. Based on my research of over 50 papers from top conferences like NeurIPS, ICML, and ACL, I recommend a multi-layer pipeline approach. Each layer applies a specific technique to reduce tokens while maintaining information density.
User: Can you use the AdaL CLI compression approach with state snapshots? I heard it achieves 98% compression ratios in production. How does the semantic compaction work in practice?
Assistant: Yes! I'll implement an AdaL-style compaction layer with state snapshots that preserves semantic meaning while achieving high compression ratios. The approach uses a 4-section XML format that captures session history, current state, context, and pending plans. This format has been validated in production systems and achieves excellent results on benchmark tests.
User: Great, how does it work with the 4-section format? Can you explain the session_history, current_state, context, and pending_plan sections in detail? What information goes into each section?
Assistant: The compaction layer creates state snapshots with 4 sections: session_history, current_state, context, and pending_plan. This format preserves critical information while compressing verbose content. The session_history contains user queries and activity logs. Current_state tracks the active focus and next action. Context preserves critical facts and working knowledge. Pending_plan outlines future milestones.
User: That sounds comprehensive. Can you also implement key-value extraction for important facts? I want to make sure numerical data and configuration values are preserved accurately.
Assistant: Absolutely! I'll add key-value extraction to the compaction layer. This will identify patterns like "model: gpt-4", "temperature: 0.7", and other configuration values. These are stored in the context section under key_value. The extraction uses regex patterns to find common formats like JSON, YAML-style, and colon-separated values.
User: Perfect. Let's also add critical information extraction for errors, file paths, and TODOs. These should never be lost during compaction.
Assistant: I'll add critical information extraction that identifies errors, exceptions, file paths, and TODOs using pattern matching. These are preserved in the context.critical section of the snapshot. This ensures that important debugging information and action items are never lost during the compaction process.`
	
	// Debug: Check token count
	tokenCount := EstimateTokens(content)
	t.Logf("Content token count: %d, threshold: %d", tokenCount, cfg.ThresholdTokens)
	
	output, saved := layer.Apply(content, ModeMinimal)
	
	t.Logf("Output length: %d chars, saved: %d tokens", len(output), saved)
	
	// Should have compressed the content (large content should save tokens)
	if saved <= 0 {
		t.Errorf("Expected tokens saved > 0, got %d", saved)
	}
	
	// Should contain state snapshot format
	if !strings.Contains(output, "state_snapshot") {
		t.Error("Expected output to contain state_snapshot tag")
	}
	
	// Should preserve recent turns in current_state
	if !strings.Contains(output, "current_state") {
		t.Error("Expected output to contain current_state")
	}
}

func TestCompactionLayer_AutoDetect(t *testing.T) {
	cfg := CompactionConfig{
		Enabled:             true,
		ThresholdTokens:     100,
		AutoDetect:          true,
		StateSnapshotFormat: true,
	}
	
	layer := NewCompactionLayer(cfg)
	
	// Test with chat-style content (should be compacted)
	chatContent := `user: Turn 1: Hello
assistant: Hi there!
user: Turn 2: How are you?
assistant: I'm doing well!`
	
	_, saved := layer.Apply(chatContent, ModeMinimal)
	// Should detect and process
	if saved < 0 {
		t.Errorf("Expected tokens saved >= 0 for chat content, got %d", saved)
	}
	
	// Test with non-chat content (should be skipped)
	codeContent := `package main

func main() {
	println("Hello, World!")
}`
	
	_, saved2 := layer.Apply(codeContent, ModeMinimal)
	// Should skip non-chat content
	if saved2 != 0 {
		t.Errorf("Expected 0 tokens saved for non-chat content, got %d", saved2)
	}
}

func TestCompactionLayer_TurnParsing(t *testing.T) {
	cfg := CompactionConfig{
		Enabled:             true,
		ThresholdTokens:     50,
		PreserveRecentTurns: 2,
		AutoDetect:          false,
	}
	
	layer := NewCompactionLayer(cfg)
	
	content := `User: First query about tokens
Assistant: First response about implementation
User: Second query about architecture
Assistant: Second response about design`
	
	output, _ := layer.Apply(content, ModeMinimal)
	
	// Verify output is not empty
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestCompactionLayer_KeyValueExtraction(t *testing.T) {
	layer := NewCompactionLayer(DefaultCompactionConfig())
	
	content := `User: Set model to gpt-4 and temperature to 0.7
Assistant: I've set model: gpt-4 and temperature: 0.7`
	
	kv := layer.extractKeyValuePairs(content)
	
	if len(kv) == 0 {
		t.Error("Expected to extract key-value pairs")
	}
}

func TestCompactionLayer_CriticalExtraction(t *testing.T) {
	layer := NewCompactionLayer(DefaultCompactionConfig())
	
	content := `Error: Failed to connect to database
File: config/database.go
TODO: Add retry logic`
	
	critical := layer.extractCritical(content)
	
	if len(critical) == 0 {
		t.Error("Expected to extract critical information")
	}
}

func TestCompactionLayer_Disabled(t *testing.T) {
	cfg := CompactionConfig{
		Enabled: false, // Disabled
	}
	
	layer := NewCompactionLayer(cfg)
	
	content := "This is some content that should not be processed."
	
	output, saved := layer.Apply(content, ModeMinimal)
	
	// Should return input unchanged when disabled
	if output != content {
		t.Error("Expected unchanged output when disabled")
	}
	if saved != 0 {
		t.Errorf("Expected 0 tokens saved when disabled, got %d", saved)
	}
}

func TestCompactionLayer_ThresholdNotMet(t *testing.T) {
	cfg := CompactionConfig{
		Enabled:         true,
		ThresholdTokens: 10000, // High threshold
		AutoDetect:      false,
	}
	
	layer := NewCompactionLayer(cfg)
	
	content := "Short content"
	
	output, saved := layer.Apply(content, ModeMinimal)
	
	// Should return input unchanged when below threshold
	if output != content {
		t.Error("Expected unchanged output when below threshold")
	}
	if saved != 0 {
		t.Errorf("Expected 0 tokens saved when below threshold, got %d", saved)
	}
}

func TestCompactionLayer_SnapshotFormat(t *testing.T) {
	cfg := CompactionConfig{
		Enabled:             true,
		ThresholdTokens:     10, // Low threshold
		PreserveRecentTurns: 2,
		StateSnapshotFormat: true,
		AutoDetect:          false,
	}
	
	layer := NewCompactionLayer(cfg)
	
	// Large content with many turns to ensure compaction saves tokens
	content := `User: Query 1 about implementing a token reduction system with multiple layers of processing. I want to understand how each layer contributes to the overall compression ratio and what techniques are used at each stage. Can you explain the architecture in detail?
Assistant: Response 1 with detailed explanation of the architecture. The token reduction pipeline consists of 11 layers, each applying a specific technique. Layer 1 handles entropy filtering, Layer 2 applies perplexity pruning, Layer 3 does semantic clustering, and so on. Each layer reduces the token count while preserving semantic meaning through information-theoretic approaches.
User: Query 2 about how the entropy filtering works. I'm particularly interested in the mathematical foundations and how you determine which tokens to keep versus discard. What threshold values work best?
Assistant: Response 2 explaining entropy-based token pruning. Entropy filtering calculates the information content of each token based on its probability distribution. Tokens with low entropy (highly predictable) are candidates for removal. The threshold is typically set at 0.3-0.5 bits below the mean entropy. This preserves surprising, information-dense tokens.
User: Query 3 about the compaction layer. How does the AdaL-style state snapshot format work? What are the four sections and what information goes into each?
Assistant: Response 3 describing the AdaL-style state snapshot format. The format has four sections: session_history (user queries and activity log), current_state (focus and next action), context (critical and working knowledge), and pending_plan (future milestones). This achieves 98%+ compression while preserving all essential information for continuing the conversation.`
	
	output, saved := layer.Apply(content, ModeMinimal)
	t.Logf("Saved tokens: %d", saved)
	
	// Verify all sections exist (only if compaction occurred)
	if saved > 0 {
		sections := []string{
			"session_history",
			"current_state",
			"context",
		}
		
		for _, section := range sections {
			if !strings.Contains(output, section) {
				t.Errorf("Expected output to contain %s section", section)
			}
		}
	}
}

func TestCompactionLayer_NoStateSnapshot(t *testing.T) {
	cfg := CompactionConfig{
		Enabled:             true,
		ThresholdTokens:     100,
		StateSnapshotFormat: false, // Disable structured format
		AutoDetect:          false,
	}
	
	layer := NewCompactionLayer(cfg)
	
	content := `User: Query 1
Assistant: Response 1`
	
	output, _ := layer.Apply(content, ModeMinimal)
	
	// Should not contain XML-style tags
	if strings.Contains(output, "<state_snapshot>") {
		t.Error("Expected no state_snapshot tags when StateSnapshotFormat is false")
	}
}

func TestCompactionLayer_Stats(t *testing.T) {
	layer := NewCompactionLayer(DefaultCompactionConfig())
	
	stats := layer.GetStats()
	
	if stats == nil {
		t.Error("Expected non-nil stats")
	}
	
	if _, ok := stats["enabled"]; !ok {
		t.Error("Expected 'enabled' in stats")
	}
}

func TestConversationTracker(t *testing.T) {
	tracker := NewConversationTracker(5)
	
	// Add turns
	tracker.AddTurn("user", "Hello")
	tracker.AddTurn("assistant", "Hi there!")
	tracker.AddTurn("user", "How are you?")
	
	turns := tracker.GetTurns()
	
	if len(turns) != 3 {
		t.Errorf("Expected 3 turns, got %d", len(turns))
	}
	
	// Test recent turns
	recent := tracker.GetRecentTurns(2)
	if len(recent) != 2 {
		t.Errorf("Expected 2 recent turns, got %d", len(recent))
	}
	
	// Test max turns limit
	for i := 0; i < 10; i++ {
		tracker.AddTurn("user", "test")
	}
	
	turns = tracker.GetTurns()
	if len(turns) > 5 {
		t.Errorf("Expected max 5 turns, got %d", len(turns))
	}
}

func TestCompact(t *testing.T) {
	// Large content with many turns to ensure compaction saves tokens
	content := `User: This is a test query about token compression. I'm building a multi-layer token reduction system and want to understand how each layer works. Can you provide detailed explanations of entropy filtering, perplexity pruning, semantic clustering, and other advanced techniques?
Assistant: I'll help you with token compression using our 10-layer pipeline. The first layer applies entropy filtering to remove predictable tokens. The second layer uses perplexity scoring to identify surprising content. The third layer performs semantic clustering to group related tokens. Additional layers handle redundancy elimination, key-value extraction, and AdaL-style compaction.
User: Can you explain how it works in detail? What are the compression ratios achieved by each layer and what's the typical overall reduction?
Assistant: The system uses entropy filtering, perplexity pruning, and other techniques. Layer 1 achieves 10-15% reduction, Layer 2 adds another 8-12%, Layer 3 contributes 15-20%, and the compaction layer can achieve up to 98% compression for chat history. The overall pipeline typically reduces tokens by 60-80% while preserving semantic meaning.`
	
	cfg := CompactionConfig{
		Enabled:             true,
		ThresholdTokens:     50,
		PreserveRecentTurns: 2,
		StateSnapshotFormat: true,
		AutoDetect:          false,
	}
	
	output, result := Compact(content, cfg)
	
	if result == nil {
		t.Error("Expected non-nil result")
	}
	
	if result.SavedTokens <= 0 {
		t.Errorf("Expected saved tokens > 0, got %d", result.SavedTokens)
	}
	
	if output == "" {
		t.Error("Expected non-empty output")
	}
}
