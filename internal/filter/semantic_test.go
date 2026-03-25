package filter

import (
	"strconv"
	"strings"
	"testing"
)

func TestSemanticFilter_Name(t *testing.T) {
	f := NewSemanticFilter()
	if f.Name() != "semantic" {
		t.Errorf("expected name 'semantic', got %q", f.Name())
	}
}

func TestSemanticFilter_ShortInput(t *testing.T) {
	f := NewSemanticFilter()

	// Very short input should pass through unchanged
	input := "short"
	output, saved := f.Apply(input, ModeMinimal)

	if output != input {
		t.Errorf("short input should not be filtered: got %q", output)
	}
	if saved != 0 {
		t.Errorf("expected 0 tokens saved for short input, got %d", saved)
	}
}

func TestSemanticFilter_HighImportanceContent(t *testing.T) {
	f := NewSemanticFilter()

	// Content with errors should be kept
	input := `Running tests...
test_auth.py:42: ERROR: Authentication failed
  File "test_auth.py", line 42, in test_login
    assert response.status_code == 200
AssertionError: Expected 200, got 401

test_api.py:15: FAILED - Connection refused
Stack trace:
  at connect() in api.py:15
  at main() in app.py:100

test result: FAILED. 2 passed; 2 failed; 0 ignored`

	output, _ := f.Apply(input, ModeMinimal)

	// Should keep error-related content
	if !strings.Contains(output, "ERROR") {
		t.Error("expected output to contain 'ERROR'")
	}
	if !strings.Contains(output, "FAILED") {
		t.Error("expected output to contain 'FAILED'")
	}
	if !strings.Contains(output, "test_auth.py:42") {
		t.Error("expected output to contain file:line reference")
	}
}

func TestSemanticFilter_LowImportanceContent(t *testing.T) {
	f := NewSemanticFilter()

	// Repetitive low-importance content should be compressed
	input := strings.Repeat("Success: operation completed successfully\n", 50)
	input += strings.Repeat("OK: task done\n", 50)

	output, saved := f.Apply(input, ModeAggressive)

	// Should compress significantly
	if saved == 0 {
		t.Error("expected some token savings for low-importance repetitive content")
	}
	if len(output) >= len(input) {
		t.Error("expected output to be smaller than input")
	}
}

func TestSemanticFilter_ModeThresholds(t *testing.T) {
	f := NewSemanticFilter()

	// Mixed content with some important, some noise
	input := `Building project...
Compiling module1...
Compiling module2...
Compiling module3...
Compiling module4...
Compiling module5...
Compiling module6...
Compiling module7...
Compiling module8...
Compiling module9...
Compiling module10...
ERROR: Failed to compile module5
error: cannot find type 'User' in scope
  --> src/auth.rs:42:5
   |
42 |     user: User,
   |     ^^^^^^^^^^ not found in this scope

Finished compilation with errors`

	// Test different modes
	outputMinimal, _ := f.Apply(input, ModeMinimal)
	outputAggressive, _ := f.Apply(input, ModeAggressive)

	// Aggressive should be shortest
	if len(outputAggressive) > len(outputMinimal) {
		t.Error("aggressive mode should produce shorter output than minimal")
	}

	// All modes should keep the error
	if !strings.Contains(outputAggressive, "ERROR") {
		t.Error("aggressive mode should keep ERROR content")
	}
	if !strings.Contains(outputAggressive, "src/auth.rs:42") {
		t.Error("aggressive mode should keep file:line reference")
	}
}

func TestSemanticFilter_SegmentBoundary(t *testing.T) {
	f := NewSemanticFilter()

	// Test boundary detection
	tests := []struct {
		line     string
		lines    []string
		idx      int
		expected bool
	}{
		{"", []string{"previous", "", "next"}, 1, true},
		{"---", []string{"prev", "---", "next"}, 1, true},
		{"test result: PASS", []string{"prev", "test result: PASS", "next"}, 1, true},
		{"Compiling module", []string{"prev", "Compiling module", "next"}, 1, true},
		{"normal line", []string{"prev", "normal line", "next"}, 1, false},
	}

	for _, tt := range tests {
		result := f.isSegmentBoundary(tt.line, tt.lines, tt.idx)
		if result != tt.expected {
			t.Errorf("isSegmentBoundary(%q) = %v, expected %v", tt.line, result, tt.expected)
		}
	}
}

func TestSemanticFilter_ScoreSegment(t *testing.T) {
	f := NewSemanticFilter()

	tests := []struct {
		name     string
		segment  string
		minScore float64
		maxScore float64
	}{
		{
			name:     "error content",
			segment:  "ERROR: Failed to compile\nerror: undefined variable",
			minScore: 0.3,
			maxScore: 1.0,
		},
		{
			name:     "empty content",
			segment:  "",
			minScore: 0.0,
			maxScore: 0.0,
		},
		{
			name:     "success content",
			segment:  "Success! OK! Done! Complete!",
			minScore: 0.0,
			maxScore: 0.5, // Low score due to success keywords, but entropy/unique ratio add to it
		},
		{
			name:     "mixed content",
			segment:  "Building project...\nCompiling module1...\nWarning: deprecated API",
			minScore: 0.2,
			maxScore: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := f.scoreSegment(tt.segment)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("scoreSegment(%q) = %v, expected between %v and %v",
					tt.segment, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestSemanticFilter_UniqueTokenRatio(t *testing.T) {
	f := NewSemanticFilter()

	tests := []struct {
		name     string
		segment  string
		expected float64
	}{
		{
			name:     "all unique",
			segment:  "one two three four five",
			expected: 1.0, // All words unique
		},
		{
			name:     "all same",
			segment:  "test test test test test",
			expected: 0.2, // Only one unique out of 5
		},
		{
			name:     "mixed",
			segment:  "error in test error found",
			expected: 0.8, // 4 unique out of 5: error, in, test, found (error repeated once)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := f.uniqueTokenRatio(tt.segment)
			// Allow some tolerance
			if ratio < tt.expected-0.1 || ratio > tt.expected+0.1 {
				t.Errorf("uniqueTokenRatio(%q) = %v, expected ~%v",
					tt.segment, ratio, tt.expected)
			}
		})
	}
}

func TestSemanticFilter_KeywordDensity(t *testing.T) {
	f := NewSemanticFilter()

	// High importance keywords should increase score
	highScore := f.keywordDensity("ERROR: Failed to compile - critical failure")

	// Low importance keywords should decrease score
	lowScore := f.keywordDensity("Success! OK! Done! Complete! success ok done")

	// Medium importance should be in between
	mediumScore := f.keywordDensity("Warning: deprecated function in build")

	if highScore <= lowScore {
		t.Errorf("high importance keywords should score higher: high=%v, low=%v", highScore, lowScore)
	}

	if mediumScore <= lowScore {
		t.Errorf("medium importance should score higher than low: medium=%v, low=%v", mediumScore, lowScore)
	}
}

func TestSemanticFilter_StructuralMarkers(t *testing.T) {
	f := NewSemanticFilter()

	tests := []struct {
		name     string
		segment  string
		minScore float64
	}{
		{
			name:     "file path reference",
			segment:  "Error at main.go:42: undefined variable",
			minScore: 0.3, // Should detect .go:
		},
		{
			name:     "stack trace",
			segment:  "at connect() in api.js:15\nStack trace:",
			minScore: 0.4, // Should detect "at " and "Stack trace"
		},
		{
			name:     "diff hunk",
			segment:  "@@ -42,5 +42,6 @@ function test() {",
			minScore: 0.3, // Should detect @@
		},
		{
			name:     "no markers",
			segment:  "just some regular text without markers",
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := f.structuralMarkers(tt.segment)
			if score < tt.minScore {
				t.Errorf("structuralMarkers(%q) = %v, expected >= %v",
					tt.segment, score, tt.minScore)
			}
		})
	}
}

func TestSemanticFilter_CharEntropy(t *testing.T) {
	f := NewSemanticFilter()

	// High entropy (varied characters)
	highEntropy := f.charEntropy("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%")

	// Low entropy (repetitive)
	lowEntropy := f.charEntropy("aaaaaaaaaaaaaaaaaaaaaaaa")

	if highEntropy <= lowEntropy {
		t.Errorf("varied text should have higher entropy: high=%v, low=%v", highEntropy, lowEntropy)
	}
}

func TestSemanticFilter_CompressSegment(t *testing.T) {
	f := NewSemanticFilter()

	// Long segment should be compressed
	longSegment := strings.Repeat("line content\n", 10)
	compressed := f.compressSegment(longSegment)

	if !strings.Contains(compressed, "...") {
		t.Error("compressed segment should contain ellipsis")
	}
	if !strings.Contains(compressed, "lines omitted") {
		t.Error("compressed segment should indicate lines omitted")
	}

	// Short segment should not be compressed
	shortSegment := "line1\nline2\nline3"
	notCompressed := f.compressSegment(shortSegment)

	if notCompressed != shortSegment {
		t.Error("short segment should not be compressed")
	}
}

func TestSemanticFilter_Integration(t *testing.T) {
	// Test the semantic filter in the full engine
	engine := NewEngine(ModeMinimal)

	// Verify semantic filter is included
	found := false
	for _, f := range engine.filters {
		if f.Name() == "semantic" {
			found = true
			break
		}
	}

	if !found {
		t.Error("semantic filter should be included in engine")
	}
}

func TestSemanticFilter_RealWorldTestOutput(t *testing.T) {
	f := NewSemanticFilter()

	// Realistic test output
	input := `running 150 tests
test test_auth_login ... ok
test test_auth_logout ... ok
test test_auth_refresh ... ok
test test_user_create ... ok
test test_user_delete ... ok
test test_user_update ... ok
test test_api_get_users ... ok
test test_api_create_user ... ok
test test_api_delete_user ... ok
test test_database_connect ... FAILED

---- test_database_connect stdout ----
thread 'test_database_connect' panicked at 'assertion failed: conn.is_ok()',
src/database.rs:42:5
stack backtrace:
   0: std::panicking::begin_panic
   1: test_database_connect
   2: test_main
note: Some details are omitted, run with RUST_BACKTRACE=full for details

test test_cache_get ... ok
test test_cache_set ... ok

test result: FAILED. 149 passed; 1 failed; 0 ignored; finished in 0.45s`

	output, _ := f.Apply(input, ModeMinimal)

	// Should keep the failed test details
	if !strings.Contains(output, "FAILED") {
		t.Error("should keep FAILED markers")
	}
	if !strings.Contains(output, "test_database_connect") {
		t.Error("should keep failed test name")
	}
	if !strings.Contains(output, "src/database.rs:42") {
		t.Error("should keep file:line reference")
	}
	if !strings.Contains(output, "stack backtrace") {
		t.Error("should keep stack trace indicator")
	}
}

func BenchmarkSemanticFilter_Apply(b *testing.B) {
	f := NewSemanticFilter()

	// Create a realistic large output
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "Compiling module"+strconv.Itoa(i)+"...")
	}
	lines = append(lines, "ERROR: Failed to compile module50")
	lines = append(lines, "error: undefined reference")
	lines = append(lines, "  --> src/main.rs:100:5")

	input := strings.Join(lines, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Apply(input, ModeMinimal)
	}
}
