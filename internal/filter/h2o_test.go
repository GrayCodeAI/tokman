package filter

import (
	"strings"
	"testing"
)

func TestH2OFilter_Basic(t *testing.T) {
	filter := NewH2OFilter()

	content := `Error: Failed to connect to database
File: /home/user/config/database.go
Line: 42

This is a long error message that contains a lot of verbose output which may not be entirely necessary for understanding the core issue. The error occurred while trying to establish a connection to the PostgreSQL database server.

Additional context: The connection timeout was set to 30 seconds but the server did not respond within that time frame. This could be due to network issues, firewall rules, or the server being overloaded.

Success: Connection established after retry`

	output, saved := filter.Apply(content, ModeMinimal)

	t.Logf("Original tokens: %d", EstimateTokens(content))
	t.Logf("Saved tokens: %d", saved)
	t.Logf("Output length: %d chars", len(output))

	// Should have saved some tokens
	if saved <= 0 {
		t.Errorf("Expected tokens saved > 0, got %d", saved)
	}

	// Should preserve important keywords
	if !strings.Contains(output, "Error") {
		t.Error("Expected output to contain 'Error'")
	}
	// H2O filter saves tokens by keeping sinks, heavy hitters, and recent tokens
	// Key metric is that we're saving tokens while preserving core meaning
	if saved < 50 {
		t.Errorf("Expected significant token savings, got %d", saved)
	}
}

func TestH2OFilter_AttentionSinks(t *testing.T) {
	filter := NewH2OFilter()

	// First tokens are critical - they should always be preserved
	content := `CRITICAL ERROR: System failure detected
This is the second line of content.
This is the third line.
More content here that fills out the message to reach minimum length requirements.
Additional filler content to ensure we have enough tokens to process.
Even more content to make sure we exceed the minimum threshold for processing.`

	output, _ := filter.Apply(content, ModeMinimal)

	// First tokens (attention sinks) should be preserved
	if !strings.Contains(output, "CRITICAL") {
		t.Error("Expected output to contain 'CRITICAL' (attention sink)")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("Expected output to contain 'ERROR' (attention sink)")
	}
}

func TestH2OFilter_RecentTokens(t *testing.T) {
	filter := NewH2OFilter()

	// Last tokens should be preserved
	content := `Starting process...
Line 1 content here
Line 2 content here
Line 3 content here
Line 4 content here
Line 5 content here
Line 6 content here
Line 7 content here
Line 8 content here
Line 9 content here
FINAL RESULT: Operation completed successfully`

	output, _ := filter.Apply(content, ModeMinimal)

	// Recent tokens should be preserved
	if !strings.Contains(output, "FINAL") {
		t.Error("Expected output to contain 'FINAL' (recent token)")
	}
	if !strings.Contains(output, "successfully") {
		t.Error("Expected output to contain 'successfully' (recent token)")
	}
}

func TestH2OFilter_Numbers(t *testing.T) {
	filter := NewH2OFilter()

	content := `Temperature: 0.75
Max tokens: 4096
Threshold: 0.3
Count: 100
Ratio: 15.5
More filler content to reach minimum length requirement for the filter to process.
Additional text to ensure we have enough tokens to trigger compression.`

	output, saved := filter.Apply(content, ModeMinimal)

	t.Logf("Saved tokens: %d", saved)

	// Numbers should be preserved as heavy hitters
	numbers := []string{"0.75", "4096", "0.3", "100", "15.5"}
	for _, num := range numbers {
		if !strings.Contains(output, num) {
			t.Errorf("Expected output to contain number %s", num)
		}
	}
}

func TestH2OFilter_FilePaths(t *testing.T) {
	filter := NewH2OFilter()

	content := `Reading file: /home/user/project/main.go
Writing to: ./output/result.json
Config: C:\Users\config.yaml
More content here to ensure we have enough length to trigger compression.
Additional filler text to meet the minimum content length requirement.
Even more text to make sure we exceed the threshold for processing.`

	output, saved := filter.Apply(content, ModeMinimal)

	t.Logf("Saved tokens: %d", saved)

	// File paths should be partially preserved (components may be split)
	hasPath := strings.Contains(output, "main") || strings.Contains(output, "result") ||
		strings.Contains(output, "config") || strings.Contains(output, ".go") ||
		strings.Contains(output, ".json") || strings.Contains(output, ".yaml")
	if !hasPath {
		t.Error("Expected output to contain some path info")
	}
}

func TestH2OFilter_Disabled(t *testing.T) {
	filter := NewH2OFilter()
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

func TestH2OFilter_ShortContent(t *testing.T) {
	filter := NewH2OFilter()

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

func TestH2OFilter_Stats(t *testing.T) {
	filter := NewH2OFilter()

	stats := filter.GetStats()

	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	if _, ok := stats["enabled"]; !ok {
		t.Error("Expected 'enabled' in stats")
	}
	if _, ok := stats["sink_size"]; !ok {
		t.Error("Expected 'sink_size' in stats")
	}
	if _, ok := stats["recent_size"]; !ok {
		t.Error("Expected 'recent_size' in stats")
	}
	if _, ok := stats["heavy_hitter_size"]; !ok {
		t.Error("Expected 'heavy_hitter_size' in stats")
	}
}

func TestH2OFilter_AggressiveMode(t *testing.T) {
	filter := NewH2OFilter()

	content := `Error: Failed to process request
This is a very long error message with lots of verbose details that could be compressed.
File: error.log
Line: 100
More filler content to reach the minimum threshold for compression to be applied.
Additional text to ensure we have enough tokens for the filter to process.`

	_, savedMinimal := filter.Apply(content, ModeMinimal)
	_, savedAggressive := filter.Apply(content, ModeAggressive)

	t.Logf("Minimal saved: %d, Aggressive saved: %d", savedMinimal, savedAggressive)
}
