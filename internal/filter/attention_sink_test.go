package filter

import (
	"strings"
	"testing"
)

func TestAttentionSinkFilter_Basic(t *testing.T) {
	filter := NewAttentionSinkFilter()

	// Create content with enough lines to actually filter
	// Need more than 4 sinks + anchors + 8 recent lines
	content := `CRITICAL ERROR: System failure detected
File: /home/user/app/main.go
Line: 42

Fourth sink line that should be preserved.

Filler line 1 that should be filtered out completely.
Filler line 2 that should be filtered out completely.
Filler line 3 that should be filtered out completely.
Filler line 4 that should be filtered out completely.
Filler line 5 that should be filtered out completely.
Filler line 6 that should be filtered out completely.
Filler line 7 that should be filtered out completely.
Filler line 8 that should be filtered out completely.
Filler line 9 that should be filtered out completely.
Filler line 10 that should be filtered out completely.
Filler line 11 that should be filtered out completely.
Filler line 12 that should be filtered out completely.

FINAL RESULT: Operation completed successfully after retry.`

	output, saved := filter.Apply(content, ModeMinimal)

	t.Logf("Original tokens: %d", EstimateTokens(content))
	t.Logf("Saved tokens: %d", saved)
	t.Logf("Output length: %d chars", len(output))

	// Should have saved some tokens
	if saved <= 0 {
		t.Errorf("Expected tokens saved > 0, got %d", saved)
	}

	// Should preserve sink tokens (first lines)
	if !strings.Contains(output, "CRITICAL") {
		t.Error("Expected output to contain 'CRITICAL' (attention sink)")
	}

	// Should preserve recent tokens (last lines)
	if !strings.Contains(output, "FINAL RESULT") {
		t.Error("Expected output to contain 'FINAL RESULT' (recent token)")
	}
}

func TestAttentionSinkFilter_Anchors(t *testing.T) {
	filter := NewAttentionSinkFilter()

	content := `Starting process...
Line 1 content here
Line 2 content here
Line 3 content here
Error: Something went wrong in the middle
Line 5 content here
Line 6 content here
Line 7 content here
Line 8 content here
Line 9 content here
Warning: This is a warning message
Line 11 content here
Process complete.`

	output, _ := filter.Apply(content, ModeMinimal)

	// Anchor patterns should be preserved
	if !strings.Contains(output, "Error:") {
		t.Error("Expected output to contain 'Error:' (anchor pattern)")
	}
	if !strings.Contains(output, "Warning:") {
		t.Error("Expected output to contain 'Warning:' (anchor pattern)")
	}
}

func TestAttentionSinkFilter_SinkPreservation(t *testing.T) {
	filter := NewAttentionSinkFilter()

	// First meaningful lines should always be preserved
	content := `IMPORTANT: This is the first line
This is the second line of content
This is the third line
This is the fourth line
This is the fifth line with more content
This is the sixth line
This is the seventh line
This is the eighth line
This is the ninth line
This is the tenth line and final.`

	output, _ := filter.Apply(content, ModeMinimal)

	// First 4 meaningful lines should be preserved
	if !strings.Contains(output, "IMPORTANT") {
		t.Error("Expected output to contain first line (attention sink)")
	}
	if !strings.Contains(output, "second line") {
		t.Error("Expected output to contain second line (attention sink)")
	}
}

func TestAttentionSinkFilter_Disabled(t *testing.T) {
	filter := NewAttentionSinkFilter()
	filter.SetEnabled(false)

	content := "This is some content that should not be processed because the filter is disabled."

	output, saved := filter.Apply(content, ModeMinimal)

	if output != content {
		t.Error("Expected unchanged output when disabled")
	}
	if saved != 0 {
		t.Errorf("Expected 0 tokens saved when disabled, got %d", saved)
	}
}

func TestAttentionSinkFilter_ShortContent(t *testing.T) {
	filter := NewAttentionSinkFilter()

	content := "Short"

	output, saved := filter.Apply(content, ModeMinimal)

	// Short content should be returned unchanged
	if output != content {
		t.Error("Expected unchanged output for short content")
	}
	if saved != 0 {
		t.Errorf("Expected 0 tokens saved for short content, got %d", saved)
	}
}

func TestAttentionSinkFilter_Stats(t *testing.T) {
	filter := NewAttentionSinkFilter()

	stats := filter.GetStats()

	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	if _, ok := stats["enabled"]; !ok {
		t.Error("Expected 'enabled' in stats")
	}
	if _, ok := stats["sink_token_count"]; !ok {
		t.Error("Expected 'sink_token_count' in stats")
	}
	if _, ok := stats["recent_token_count"]; !ok {
		t.Error("Expected 'recent_token_count' in stats")
	}
}

func TestAttentionSinkFilter_RecentPreservation(t *testing.T) {
	filter := NewAttentionSinkFilter()

	// Build content with many lines
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "Line content number "+string(rune('0'+i%10)))
	}
	lines = append(lines, "FINAL LINE: This must be preserved")

	content := strings.Join(lines, "\n")

	output, _ := filter.Apply(content, ModeMinimal)

	// Last lines should be preserved
	if !strings.Contains(output, "FINAL LINE") {
		t.Error("Expected output to contain last line (recent token)")
	}
}
