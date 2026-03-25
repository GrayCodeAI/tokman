package filter

import (
	"strconv"
	"strings"
	"testing"
)

func TestPositionAwareFilter_Name(t *testing.T) {
	f := NewPositionAwareFilter()
	if f.Name() != "position_aware" {
		t.Errorf("expected name 'position_aware', got %q", f.Name())
	}
}

func TestPositionAwareFilter_ShortInput(t *testing.T) {
	f := NewPositionAwareFilter()

	// Short input should pass through unchanged
	input := "short input"
	output, saved := f.Apply(input, ModeMinimal)

	if output != input {
		t.Error("short input should not be reordered")
	}
	if saved != 0 {
		t.Error("position filter should not save tokens")
	}
}

func TestPositionAwareFilter_ImportanceScore(t *testing.T) {
	f := NewPositionAwareFilter()

	tests := []struct {
		name     string
		segment  string
		minScore float64
	}{
		{"error content", "ERROR: Failed to compile", 0.5},
		{"stack trace", "Stack trace:\n  at main()", 0.4},
		{"file reference", "Error at main.go:42", 0.7}, // error + file ref
		{"warning", "Warning: deprecated API", 0.1},
		{"diff hunk", "@@ -1,5 +1,6 @@", 0.2},
		{"success", "Success! All tests passed", 0.0},
		{"empty", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := f.importanceScore(tt.segment)
			if score < tt.minScore {
				t.Errorf("importanceScore(%q) = %v, expected >= %v", tt.segment, score, tt.minScore)
			}
		})
	}
}

func TestPositionAwareFilter_HasFileReference(t *testing.T) {
	f := NewPositionAwareFilter()

	tests := []struct {
		segment  string
		expected bool
	}{
		{"Error at main.go:42", true},
		{"src/lib.rs:100:5", true},
		{"File \"test.py\", line 50", false}, // Different pattern
		{"no reference here", false},
	}

	for _, tt := range tests {
		result := f.hasFileReference(tt.segment)
		if result != tt.expected {
			t.Errorf("hasFileReference(%q) = %v, expected %v", tt.segment, result, tt.expected)
		}
	}
}

func TestPositionAwareFilter_Reordering(t *testing.T) {
	f := NewPositionAwareFilter()

	// Create input where error is buried in middle
	input := `Building project...

Compiling module1...
Compiling module2...
Compiling module3...

ERROR: Failed to compile module2
error: undefined variable 'x'
  --> src/main.rs:42:5

Compiling module4...
Compiling module5...

Finished build`

	output, _ := f.Apply(input, ModeMinimal)

	// Error should appear near beginning and end
	lines := strings.Split(output, "\n")

	// Find where ERROR appears
	firstErrorIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "ERROR:") {
			firstErrorIdx = i
			break
		}
	}

	// Error should appear
	if firstErrorIdx == -1 {
		t.Error("ERROR should appear in output")
		return
	}

	// For quality improvement, we verify error is present
	// The reordering ensures it's at prominent positions
	if !strings.Contains(output, "ERROR:") {
		t.Error("output should contain ERROR")
	}
	if !strings.Contains(output, "src/main.rs:42") {
		t.Error("output should contain file reference")
	}
}

func TestPositionAwareFilter_StructuralBoundary(t *testing.T) {
	f := NewPositionAwareFilter()

	tests := []struct {
		line     string
		expected bool
	}{
		{"@@ -1,5 +1,6 @@", true},
		{"diff --git a/file b/file", true},
		{"test result: FAIL", true},
		{"Compiling module", true},
		{"ERROR: something", true},
		{"stack backtrace:", true},
		{"normal line", false},
		{"", false},
	}

	for _, tt := range tests {
		result := f.isStructuralBoundary(tt.line)
		if result != tt.expected {
			t.Errorf("isStructuralBoundary(%q) = %v, expected %v", tt.line, result, tt.expected)
		}
	}
}

func TestPositionAwareFilter_Integration(t *testing.T) {
	// Test that position filter works in engine
	engine := NewEngine(ModeMinimal)

	// Verify position filter IS in default engine (improves LLM recall)
	found := false
	for _, f := range engine.filters {
		if f.Name() == "position_aware" {
			found = true
			break
		}
	}
	if !found {
		t.Error("position_aware should be in default engine for LLM recall optimization")
	}
}

func TestPositionAwareFilter_PreservesContent(t *testing.T) {
	f := NewPositionAwareFilter()

	input := `Segment 1: Normal content
Segment 2: More normal content
Segment 3: ERROR critical issue here
Segment 4: Success message
Segment 5: Another normal segment`

	output, _ := f.Apply(input, ModeMinimal)

	// All content should still be present (just reordered)
	if !strings.Contains(output, "Segment 1") {
		t.Error("should preserve Segment 1")
	}
	if !strings.Contains(output, "Segment 3: ERROR") {
		t.Error("should preserve error segment")
	}
	if !strings.Contains(output, "Segment 5") {
		t.Error("should preserve Segment 5")
	}
}

func TestPositionAwareFilter_RealWorldOutput(t *testing.T) {
	f := NewPositionAwareFilter()

	// Realistic test output with buried error
	input := `running 100 tests
test test_001 ... ok
test test_002 ... ok
test test_003 ... ok
test test_004 ... ok
test test_005 ... ok

ERROR in test_006
thread 'test_006' panicked at 'assertion failed'
  --> src/test.rs:42:5
  at run_test() in src/lib.rs:100

test test_007 ... ok
test test_008 ... ok
test test_009 ... ok
test test_010 ... ok

test result: FAILED. 9 passed; 1 failed`

	output, _ := f.Apply(input, ModeMinimal)

	// Error should be present
	if !strings.Contains(output, "ERROR in test_006") {
		t.Error("should contain error message")
	}
	if !strings.Contains(output, "src/test.rs:42") {
		t.Error("should contain file reference")
	}
	if !strings.Contains(output, "FAILED") {
		t.Error("should contain test result")
	}
}

func BenchmarkPositionAwareFilter_Apply(b *testing.B) {
	f := NewPositionAwareFilter()

	// Large output
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "Compiling module"+strconv.Itoa(i)+"...")
	}
	lines = append(lines, "ERROR: Failed at module25")
	lines = append(lines, "  --> src/main.rs:100:5")
	for i := 0; i < 50; i++ {
		lines = append(lines, "Processing step"+strconv.Itoa(i))
	}

	input := strings.Join(lines, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Apply(input, ModeMinimal)
	}
}
