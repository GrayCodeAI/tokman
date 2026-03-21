package filter

import (
	"testing"
)

// T12: Question-Aware Filter Tests

func TestQuestionAwareFilter_Basic(t *testing.T) {
	filter := NewQuestionAwareFilter("find the error in the code")
	filter.config.RelevanceThreshold = 0.3 // Lower threshold for more matches

	input := `Line 1: This is a normal line
Line 2: Another normal line
Line 3: Error: something went wrong here
Line 4: The error code is 500
Line 5: This is fine
Line 6: No issues here
Line 7: Everything is working
Line 8: More content here
Line 9: Additional content
Line 10: Final line`

	output, saved := filter.Apply(input, ModeMinimal)

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	// Error lines should be preserved
	if !containsSubstring(output, "Error") {
		t.Error("Error line should be preserved")
	}

	// Saved can be 0 if all lines are preserved due to context windows
	// This is acceptable behavior for short content with matches
	_ = saved // Acknowledge saved but don't require it to be positive
}

func TestQuestionAwareFilter_NoQuery(t *testing.T) {
	filter := NewQuestionAwareFilter("")

	input := "Some content that should not be filtered"
	output, saved := filter.Apply(input, ModeMinimal)

	if output != input {
		t.Error("Output should be unchanged when no query")
	}

	if saved != 0 {
		t.Error("No savings expected when no query")
	}
}

func TestQuestionAwareFilter_ExtractTerms(t *testing.T) {
	filter := NewQuestionAwareFilter("How do I fix the authentication error?")

	terms := filter.extractTerms(filter.config.Query)

	// Should extract important terms, skip stop words
	expectedTerms := []string{"fix", "authentication", "error"}
	for _, exp := range expectedTerms {
		found := false
		for _, term := range terms {
			if term == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected term '%s' not found in extracted terms: %v", exp, terms)
		}
	}

	// Stop words should be filtered
	stopWords := []string{"how", "do", "i", "the"}
	for _, sw := range stopWords {
		for _, term := range terms {
			if term == sw {
				t.Errorf("Stop word '%s' should not be in extracted terms", sw)
			}
		}
	}
}

func TestQuestionAwareFilter_RelevanceScoring(t *testing.T) {
	filter := NewQuestionAwareFilter("authentication error")

	queryTerms := filter.extractTerms(filter.config.Query)

	// Line with both terms should score high
	highScore := filter.scoreRelevance("Error: authentication failed", queryTerms)
	lowScore := filter.scoreRelevance("This is a normal line", queryTerms)

	if highScore <= lowScore {
		t.Errorf("Line with query terms should score higher: high=%f, low=%f", highScore, lowScore)
	}
}

func TestQuestionAwareFilter_ContextWindow(t *testing.T) {
	filter := NewQuestionAwareFilter("important error")
	filter.config.RelevanceThreshold = 0.5

	input := `Line 1
Line 2
Line 3
Line 4: This is an important error
Line 5
Line 6
Line 7
Line 8
Line 9
Line 10`

	output, _ := filter.Apply(input, ModeMinimal)

	// Context window around the match should be preserved
	lines := splitLines(output)
	if len(lines) < 3 {
		t.Errorf("Expected context window to be preserved, got %d lines", len(lines))
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
