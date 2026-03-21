package filter

import (
	"testing"
)

func TestLLMAwareFilter_Name(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{})
	if f.Name() != "llm_aware" {
		t.Errorf("expected name 'llm_aware', got %q", f.Name())
	}
}

func TestLLMAwareFilter_DisabledByDefault(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{
		Enabled: false,
	})

	input := "test content with error: something failed"
	output, _ := f.Apply(input, ModeMinimal)

	// Should fall back to semantic filter
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestLLMAwareFilter_DetectIntent(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{})

	tests := []struct {
		content  string
		expected string
	}{
		{"error: compilation failed", "debug"},
		{"diff --git a/file b/file", "review"},
		{"test result: PASS", "test"},
		{"Compiling myapp v1.0", "build"},
		{"some random content", "general"},
	}

	for _, tt := range tests {
		intent := f.detectIntent(tt.content)
		if intent != tt.expected {
			t.Errorf("detectIntent(%q) = %q, expected %q", tt.content, intent, tt.expected)
		}
	}
}

func TestLLMAwareFilter_Threshold(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{
		Threshold: 100,
		Enabled:   true,
	})

	// Small content should fall back to semantic filter
	smallInput := "short content"
	output, _ := f.Apply(smallInput, ModeMinimal)

	if output != smallInput {
		t.Error("expected small content to pass through semantic filter")
	}
}

func TestLLMAwareFilter_SetEnabled(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{Enabled: false})

	f.SetEnabled(true)
	if !f.enabled {
		t.Error("expected enabled to be true")
	}

	f.SetEnabled(false)
	if f.enabled {
		t.Error("expected enabled to be false")
	}
}

func TestLLMAwareFilter_Cache(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{
		CacheEnabled: true,
	})

	content := "test content for caching"
	summary := "cached summary"

	// Add to cache
	f.addToCache(content, summary)

	// Retrieve from cache
	cached := f.getFromCache(content)
	if cached != summary {
		t.Errorf("expected cached summary %q, got %q", summary, cached)
	}
}

func TestLLMAwareFilter_CacheDisabled(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{
		CacheEnabled: false,
	})

	content := "test content"
	summary := "summary"

	// Try to add to cache (should be no-op)
	f.addToCache(content, summary)

	// Should return empty
	cached := f.getFromCache(content)
	if cached != "" {
		t.Error("expected empty cache when disabled")
	}
}

func TestLLMAwareFilter_DefaultThreshold(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{})

	if f.threshold != 2000 {
		t.Errorf("expected default threshold 2000, got %d", f.threshold)
	}
}

func TestLLMAwareFilter_SummarizeWithIntent(t *testing.T) {
	f := NewLLMAwareFilter(LLMAwareConfig{
		Enabled: false, // Will fall back to semantic
	})

	content := "error: something went wrong"
	summary, saved := f.SummarizeWithIntent(content, "debug")

	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if saved < 0 {
		t.Error("expected non-negative tokens saved")
	}
}
