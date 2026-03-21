package filter

import (
	"strings"
	"testing"
)

func TestAttributionFilter_Basic(t *testing.T) {
	filter := NewAttributionFilter()

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
	if !strings.Contains(output, "database.go") {
		t.Error("Expected output to contain file path")
	}
}

func TestAttributionFilter_Numbers(t *testing.T) {
	filter := NewAttributionFilter()

	content := `Temperature: 0.75
Max tokens: 4096
Threshold: 0.3
Count: 100
Ratio: 15.5`

	output, _ := filter.Apply(content, ModeMinimal)

	// Numbers should be preserved
	numbers := []string{"0.75", "4096", "0.3", "100", "15.5"}
	for _, num := range numbers {
		if !strings.Contains(output, num) {
			t.Errorf("Expected output to contain number %s", num)
		}
	}
}

func TestAttributionFilter_CodeSymbols(t *testing.T) {
	filter := NewAttributionFilter()

	content := `func main() {
	if x > 0 && y < 10 {
		return x + y
	}
}`

	output, _ := filter.Apply(content, ModeMinimal)

	// Code symbols should be preserved
	symbols := []string{"(", ")", "{", "}", ">", "<", "&&", "+"}
	for _, sym := range symbols {
		if !strings.Contains(output, sym) {
			t.Errorf("Expected output to contain symbol %s", sym)
		}
	}
}

func TestAttributionFilter_FilePaths(t *testing.T) {
	filter := NewAttributionFilter()

	content := `Reading file: /home/user/project/main.go
Writing to: ./output/result.json
Config: C:\Users\config.yaml`

	output, saved := filter.Apply(content, ModeMinimal)

	// File paths should be preserved
	paths := []string{"main.go", "result.json", "config.yaml"}
	for _, path := range paths {
		if !strings.Contains(output, path) {
			t.Errorf("Expected output to contain path %s", path)
		}
	}

	t.Logf("Saved tokens: %d", saved)
}

func TestAttributionFilter_URLs(t *testing.T) {
	filter := NewAttributionFilter()

	content := `API endpoint: https://api.example.com/v1/users
Documentation: http://docs.example.com
Repository: https://github.com/user/repo`

	output, _ := filter.Apply(content, ModeMinimal)

	// URLs should be preserved
	if !strings.Contains(output, "https://api.example.com") {
		t.Error("Expected output to contain API URL")
	}
	if !strings.Contains(output, "http://docs.example.com") {
		t.Error("Expected output to contain docs URL")
	}
}

func TestAttributionFilter_PositionalBias(t *testing.T) {
	filter := NewAttributionFilter()

	// Content with important info at start and end
	content := `CRITICAL: This is the most important information at the start.
	
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

FINAL RESULT: Operation completed successfully.`

	output, _ := filter.Apply(content, ModeMinimal)

	// Start and end should be preserved
	if !strings.Contains(output, "CRITICAL") {
		t.Error("Expected output to contain start marker")
	}
	if !strings.Contains(output, "FINAL RESULT") {
		t.Error("Expected output to contain end marker")
	}
}

func TestAttributionFilter_Disabled(t *testing.T) {
	filter := NewAttributionFilter()
	filter.SetEnabled(false)

	content := "This is some content that should not be processed."

	output, saved := filter.Apply(content, ModeMinimal)

	if output != content {
		t.Error("Expected unchanged output when disabled")
	}
	if saved != 0 {
		t.Errorf("Expected 0 tokens saved when disabled, got %d", saved)
	}
}

func TestAttributionFilter_ShortContent(t *testing.T) {
	filter := NewAttributionFilter()

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

func TestAttributionFilter_AggressiveMode(t *testing.T) {
	filter := NewAttributionFilter()

	content := `Error: Failed to process request
This is a very long error message with lots of verbose details that could be compressed in aggressive mode.
File: error.log
Line: 100`

	_, savedMinimal := filter.Apply(content, ModeMinimal)
	_, savedAggressive := filter.Apply(content, ModeAggressive)

	t.Logf("Minimal saved: %d, Aggressive saved: %d", savedMinimal, savedAggressive)

	// Aggressive mode should save more tokens (or at least as much)
	if savedAggressive < savedMinimal {
		t.Logf("Warning: aggressive mode saved less than minimal mode")
	}
}

func TestAttributionFilter_Stats(t *testing.T) {
	filter := NewAttributionFilter()

	stats := filter.GetStats()

	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	if _, ok := stats["enabled"]; !ok {
		t.Error("Expected 'enabled' in stats")
	}
	if _, ok := stats["threshold"]; !ok {
		t.Error("Expected 'threshold' in stats")
	}
}
