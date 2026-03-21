package filter

import (
	"strings"
	"testing"
)

func TestHierarchicalFilter_Name(t *testing.T) {
	f := NewHierarchicalFilter()
	if f.Name() != "hierarchical" {
		t.Errorf("expected name 'hierarchical', got %q", f.Name())
	}
}

func TestHierarchicalFilter_SmallInput(t *testing.T) {
	f := NewHierarchicalFilter()
	input := "line 1\nline 2\nline 3"

	output, saved := f.Apply(input, ModeMinimal)

	// Should not process small inputs
	if output != input {
		t.Error("expected small input to pass through unchanged")
	}
	if saved != 0 {
		t.Error("expected 0 tokens saved for small input")
	}
}

func TestHierarchicalFilter_LargeInput(t *testing.T) {
	f := NewHierarchicalFilter()
	f.SetLineThreshold(10) // Lower threshold for testing

	// Create a large input with sections
	var lines []string
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			lines = append(lines, "--- Section "+string(rune('A'+i/10)))
		}
		lines = append(lines, "line "+string(rune('0'+i%10)))
	}
	input := strings.Join(lines, "\n")

	output, saved := f.Apply(input, ModeMinimal)

	// Should process and compress
	if output == input {
		t.Error("expected large input to be compressed")
	}
	if saved <= 0 {
		t.Error("expected positive token savings")
	}

	// Should include hierarchical header
	if !strings.Contains(output, "Hierarchical Summary") {
		t.Error("expected hierarchical summary header")
	}
}

func TestHierarchicalFilter_SectionBoundary(t *testing.T) {
	f := NewHierarchicalFilter()

	tests := []struct {
		line     string
		expected bool
	}{
		{"--- separator", true},
		{"=== header ===", true},
		{"+++ file", true},
		{"# Markdown header", true},
		{"test result: PASS", true},
		{"Compiling crate", true},
		{"diff --git a/file b/file", true},
		{"error: something failed", true},
		{"regular line", false},
		{"", false}, // Empty line needs context
	}

	for _, tt := range tests {
		result := f.isSectionBoundary(tt.line, 0, []string{tt.line})
		if result != tt.expected {
			t.Errorf("isSectionBoundary(%q) = %v, expected %v", tt.line, result, tt.expected)
		}
	}
}

func TestHierarchicalFilter_ScoreSection(t *testing.T) {
	f := NewHierarchicalFilter()
	sf := NewSemanticFilter()

	tests := []struct {
		content  string
		minScore float64
		maxScore float64
	}{
		{"error: failed to compile", 0.2, 1.0},       // High importance keywords boost score
		{"warning: deprecated", 0.1, 0.8},            // Medium importance keywords
		{"success: build complete", 0.0, 0.3},        // Low importance keywords
		{"file.go:42: undefined variable", 0.3, 1.0}, // File reference + error keyword
	}

	for _, tt := range tests {
		s := section{content: tt.content}
		score := f.calculateSectionScore(s, sf)
		if score < tt.minScore || score > tt.maxScore {
			t.Errorf("scoreSection(%q) = %v, expected [%v, %v]", tt.content, score, tt.minScore, tt.maxScore)
		}
	}
}

func TestHierarchicalFilter_GenerateSummary(t *testing.T) {
	f := NewHierarchicalFilter()

	tests := []struct {
		content  string
		contains string
		maxLen   int
	}{
		{
			"line 1\nerror: something failed\nline 3",
			"error",
			80,
		},
		{
			"short content",
			"short",
			80,
		},
	}

	for _, tt := range tests {
		s := section{content: tt.content}
		summary := f.generateSectionSummary(s)

		if len(summary) > tt.maxLen {
			t.Errorf("summary too long: %q (max %d)", summary, tt.maxLen)
		}
	}
}

func TestHierarchicalFilter_Thresholds(t *testing.T) {
	f := NewHierarchicalFilter()

	tests := []struct {
		mode    Mode
		highMin float64
		midMin  float64
	}{
		{ModeAggressive, 0.6, 0.3},
		{ModeMinimal, 0.4, 0.2},
		{ModeNone, 0.5, 0.25},
	}

	for _, tt := range tests {
		high, mid := f.getThresholds(tt.mode)
		if high < tt.highMin {
			t.Errorf("high threshold for %v too low: %v", tt.mode, high)
		}
		if mid < tt.midMin {
			t.Errorf("mid threshold for %v too low: %v", tt.mode, mid)
		}
	}
}

func TestHierarchicalFilter_SetLineThreshold(t *testing.T) {
	f := NewHierarchicalFilter()
	f.SetLineThreshold(100)

	if f.lineThreshold != 100 {
		t.Errorf("expected threshold 100, got %v", f.lineThreshold)
	}
}

func TestHierarchicalFilter_SetMaxDepth(t *testing.T) {
	f := NewHierarchicalFilter()
	f.SetMaxDepth(5)

	if f.maxDepth != 5 {
		t.Errorf("expected max depth 5, got %v", f.maxDepth)
	}
}
