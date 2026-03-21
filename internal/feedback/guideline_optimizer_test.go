package feedback

import (
	"testing"
)

func TestNewGuidelineOptimizer(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	opt := NewGuidelineOptimizer(tempDir)

	if opt == nil {
		t.Fatal("Expected non-nil optimizer")
	}

	if len(opt.guidelines) != 0 {
		t.Errorf("Expected empty guidelines, got %d", len(opt.guidelines))
	}

	if opt.maxGuidelines != 100 {
		t.Errorf("Expected maxGuidelines=100, got %d", opt.maxGuidelines)
	}
}

func TestAnalyzeFailure_ExtractsPattern(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	tests := []struct {
		name        string
		missing     string
		wantPattern string
	}{
		{
			name:        "test name pattern",
			missing:     "test name was removed",
			wantPattern: "keep test names in output",
		},
		{
			name:        "error message pattern",
			missing:     "error message not visible",
			wantPattern: "keep error messages visible",
		},
		{
			name:        "stack trace pattern",
			missing:     "stack trace was filtered out",
			wantPattern: "preserve stack traces for debugging",
		},
		{
			name:        "file path pattern",
			missing:     "file path missing",
			wantPattern: "keep file paths for navigation",
		},
		{
			name:        "line number pattern",
			missing:     "line number was removed",
			wantPattern: "keep line numbers with file references",
		},
		{
			name:        "diff pattern",
			missing:     "diff was compressed",
			wantPattern: "preserve diff hunks in review context",
		},
		{
			name:        "assertion pattern",
			missing:     "assertion failure hidden",
			wantPattern: "keep assertion failures visible",
		},
		{
			name:        "version pattern",
			missing:     "version info missing",
			wantPattern: "show version info for deployments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt.guidelines = nil // Reset

			failure := AgentFailure{
				Task:    "test task",
				Missing: tt.missing,
			}

			opt.AnalyzeFailure(failure)

			if len(opt.guidelines) != 1 {
				t.Fatalf("Expected 1 guideline, got %d", len(opt.guidelines))
			}

			if opt.guidelines[0].Pattern != tt.wantPattern {
				t.Errorf("Expected pattern %q, got %q", tt.wantPattern, opt.guidelines[0].Pattern)
			}

			if opt.guidelines[0].Confidence != 0.5 {
				t.Errorf("Expected initial confidence 0.5, got %f", opt.guidelines[0].Confidence)
			}
		})
	}
}

func TestAnalyzeFailure_IncreasesConfidence(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	failure := AgentFailure{
		Task:    "test task",
		Missing: "test name was removed",
	}

	// First failure creates guideline with confidence 0.5
	opt.AnalyzeFailure(failure)

	if opt.guidelines[0].Confidence != 0.5 {
		t.Errorf("Expected confidence 0.5, got %f", opt.guidelines[0].Confidence)
	}

	// Second similar failure increases confidence
	opt.AnalyzeFailure(failure)

	if opt.guidelines[0].Confidence != 0.6 {
		t.Errorf("Expected confidence 0.6, got %f", opt.guidelines[0].Confidence)
	}

	// Third failure increases confidence again
	opt.AnalyzeFailure(failure)

	if opt.guidelines[0].Confidence != 0.7 {
		t.Errorf("Expected confidence 0.7, got %f", opt.guidelines[0].Confidence)
	}
}

func TestAnalyzeFailure_StoresFailure(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	failure := AgentFailure{
		Task:       "test task",
		Compressed: "filtered output",
		Issue:      "missing context",
		Missing:    "test name was removed",
		Timestamp:  "2024-01-01",
	}

	opt.AnalyzeFailure(failure)

	if len(opt.failures) != 1 {
		t.Errorf("Expected 1 failure stored, got %d", len(opt.failures))
	}

	if opt.failures[0].Task != "test task" {
		t.Errorf("Expected task 'test task', got %q", opt.failures[0].Task)
	}
}

func TestAnalyzeFailure_TrimFailures(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	// Add 101 failures
	for i := 0; i < 101; i++ {
		failure := AgentFailure{
			Task:    "test task",
			Missing: "test name was removed",
		}
		opt.AnalyzeFailure(failure)
	}

	// Should keep only 100
	if len(opt.failures) > 100 {
		t.Errorf("Expected at most 100 failures, got %d", len(opt.failures))
	}
}

func TestEnhanceOutput_NoGuidelines(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	original := "test output with test name"
	filtered := "test output"

	result := opt.EnhanceOutput(original, filtered)

	if result != filtered {
		t.Errorf("Expected filtered output unchanged, got %q", result)
	}
}

func TestEnhanceOutput_LowConfidenceGuideline(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	// Add guideline with low confidence
	opt.guidelines = []CompressionGuideline{
		{Pattern: "keep test names in output", Confidence: 0.5},
	}

	original := "TestName: test_something PASSED"
	filtered := "test PASSED"

	result := opt.EnhanceOutput(original, filtered)

	// Low confidence guidelines should not be applied
	if result != filtered {
		t.Errorf("Expected filtered unchanged for low confidence, got %q", result)
	}
}

func TestEnhanceOutput_HighConfidenceGuideline(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	// Add guideline with high confidence
	opt.guidelines = []CompressionGuideline{
		{Pattern: "keep test names in output", Confidence: 0.8},
	}

	original := "TestName: test_something PASSED"
	filtered := "test PASSED"

	result := opt.EnhanceOutput(original, filtered)

	// Should restore the missing test name line
	if result == filtered {
		t.Error("Expected output to be enhanced with restored content")
	}

	if !contains(result, "test_something") {
		t.Errorf("Expected result to contain 'test_something', got %q", result)
	}
}

func TestEnhanceOutput_AlreadyInFiltered(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	// Add guideline with high confidence
	opt.guidelines = []CompressionGuideline{
		{Pattern: "keep test names in output", Confidence: 0.8},
	}

	original := "TestName: test_something PASSED"
	filtered := "TestName: test_something PASSED" // Already has the content

	result := opt.EnhanceOutput(original, filtered)

	// Should not add duplicate content
	if result != filtered {
		t.Errorf("Expected no change when content already present, got %q", result)
	}
}

func TestRecordSuccess(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	opt.guidelines = []CompressionGuideline{
		{Pattern: "keep test names in output", Confidence: 0.7, ApplyCount: 0},
	}

	opt.RecordSuccess("keep test names in output")

	if opt.guidelines[0].ApplyCount != 1 {
		t.Errorf("Expected ApplyCount=1, got %d", opt.guidelines[0].ApplyCount)
	}

	if opt.guidelines[0].Confidence < 0.75 {
		t.Errorf("Expected confidence increase, got %f", opt.guidelines[0].Confidence)
	}
}

func TestGetGuidelines(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	opt.guidelines = []CompressionGuideline{
		{Pattern: "pattern1", Confidence: 0.8},
		{Pattern: "pattern2", Confidence: 0.6},
	}

	guidelines := opt.GetGuidelines()

	if len(guidelines) != 2 {
		t.Errorf("Expected 2 guidelines, got %d", len(guidelines))
	}

	// Verify it's a copy
	guidelines[0].Pattern = "modified"
	if opt.guidelines[0].Pattern == "modified" {
		t.Error("GetGuidelines should return a copy")
	}
}

func TestPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create optimizer and add a guideline
	opt1 := NewGuidelineOptimizer(tempDir)
	opt1.AnalyzeFailure(AgentFailure{
		Task:    "test",
		Missing: "test name was removed",
	})

	// Create new optimizer - should load existing guidelines
	opt2 := NewGuidelineOptimizer(tempDir)

	if len(opt2.guidelines) != 1 {
		t.Errorf("Expected 1 guideline loaded, got %d", len(opt2.guidelines))
	}

	if opt2.guidelines[0].Pattern != "keep test names in output" {
		t.Errorf("Expected loaded pattern, got %q", opt2.guidelines[0].Pattern)
	}
}

func TestTrimGuidelines(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)
	opt.maxGuidelines = 5

	// Add 10 guidelines with different confidence
	for i := 0; i < 10; i++ {
		confidence := float64(i) / 10.0 // 0.0 to 0.9
		opt.guidelines = append(opt.guidelines, CompressionGuideline{
			Pattern:    "pattern",
			Confidence: confidence,
		})
	}

	opt.trimGuidelines()

	if len(opt.guidelines) != 5 {
		t.Errorf("Expected 5 guidelines after trim, got %d", len(opt.guidelines))
	}

	// Verify top 5 confidence are kept (0.5, 0.6, 0.7, 0.8, 0.9)
	for _, g := range opt.guidelines {
		if g.Confidence < 0.5 {
			t.Errorf("Expected only high confidence guidelines, got %f", g.Confidence)
		}
	}
}

func TestExtractPattern_Generic(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	// Test with unknown pattern
	failure := AgentFailure{
		Task:    "test",
		Missing: "some unknown context that was missing",
	}

	opt.AnalyzeFailure(failure)

	if len(opt.guidelines) != 1 {
		t.Fatalf("Expected 1 guideline, got %d", len(opt.guidelines))
	}

	if opt.guidelines[0].Pattern != "keep context about: some unknown context that was missing" {
		t.Errorf("Expected generic pattern, got %q", opt.guidelines[0].Pattern)
	}
}

func TestMatchesGuideline(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	tests := []struct {
		content  string
		pattern  string
		expected bool
	}{
		{"TestName: test_something PASSED", "keep test names in output", true},
		{"Error: something failed", "keep error messages visible", true},
		{"random output", "keep test names in output", false},
		{"", "keep test names in output", false},
	}

	for _, tt := range tests {
		result := opt.matchesGuideline(tt.content, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchesGuideline(%q, %q) = %v, want %v", tt.content, tt.pattern, result, tt.expected)
		}
	}
}

func TestRestorePattern(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	original := `TestName: test_one PASSED
TestName: test_two FAILED
Some other output
Error: something went wrong`

	filtered := `test PASSED
test FAILED
Some other output`

	result := opt.restorePattern(original, filtered, "keep test names in output")

	// Should have restored the test name lines
	if !contains(result, "test_one") {
		t.Error("Expected 'test_one' to be restored")
	}
	if !contains(result, "test_two") {
		t.Error("Expected 'test_two' to be restored")
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	opt := NewGuidelineOptimizer(tempDir)

	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			opt.AnalyzeFailure(AgentFailure{
				Task:    "concurrent test",
				Missing: "test name was removed",
			})
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			opt.GetGuidelines()
		}
		done <- true
	}()

	// Concurrent enhance
	go func() {
		for i := 0; i < 100; i++ {
			opt.EnhanceOutput("original", "filtered")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not panic or race
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
