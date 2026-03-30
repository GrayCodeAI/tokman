package filter

import (
	"strings"
	"testing"
)

func TestSessionTracker_Name(t *testing.T) {
	tracker := NewSessionTracker()
	if tracker.Name() != "session" {
		t.Errorf("Name() = %q, want %q", tracker.Name(), "session")
	}
}

func TestSessionTracker_Apply_ShortInput(t *testing.T) {
	tracker := NewSessionTracker()
	
	// Inputs shorter than 50 chars should be unchanged
	input := "short input"
	output, tokens := tracker.Apply(input, ModeNone)
	
	if output != input {
		t.Errorf("Apply(short input) = %q, want %q", output, input)
	}
	if tokens != 0 {
		t.Errorf("Apply(short input) tokens = %d, want 0", tokens)
	}
}

func TestSessionTracker_Apply_FirstOccurrence(t *testing.T) {
	tracker := NewSessionTracker()
	
	// First occurrence should return content unchanged
	input := strings.Repeat("line of content\n", 10) // 160 chars
	output, tokens := tracker.Apply(input, ModeNone)
	
	if output != input {
		t.Errorf("Apply(first occurrence) should return unchanged content")
	}
	if tokens != 0 {
		t.Errorf("Apply(first occurrence) tokens = %d, want 0", tokens)
	}
}

func TestSessionTracker_Apply_SecondOccurrence(t *testing.T) {
	tracker := NewSessionTracker()
	
	input := strings.Repeat("repeated content line\n", 10)
	
	// First occurrence - unchanged
	output1, _ := tracker.Apply(input, ModeNone)
	if output1 != input {
		t.Errorf("First occurrence should be unchanged")
	}
	
	// Second occurrence - should have [seen] marker
	output2, _ := tracker.Apply(input, ModeNone)
	if !strings.Contains(output2, "[seen]") {
		t.Errorf("Second occurrence should contain [seen] marker, got: %q", output2)
	}
}

func TestSessionTracker_Apply_MultipleOccurrences(t *testing.T) {
	tracker := NewSessionTracker()
	
	// Long content that will be compressed after 3+ occurrences
	input := strings.Repeat("this is a longer line of repeated content\n", 5) // > 100 chars
	
	// Apply 3 times
	tracker.Apply(input, ModeNone)
	tracker.Apply(input, ModeNone)
	output3, _ := tracker.Apply(input, ModeNone)
	
	// After 3 occurrences, content should be compressed to a summary marker
	if !strings.Contains(output3, "[seen x3:") {
		t.Errorf("Third+ occurrence should be compressed to [seen xN: summary], got: %q", output3)
	}
}

func TestSessionTracker_Stats(t *testing.T) {
	tracker := NewSessionTracker()
	
	// Initial stats should be zero
	stats := tracker.Stats()
	if stats.UniqueEntries != 0 {
		t.Errorf("Initial UniqueEntries = %d, want 0", stats.UniqueEntries)
	}
	
	// Add content - use a single segment (< 10 lines) to get predictable entries
	input := "test content for stats line one\ntest content for stats line two"
	tracker.Apply(input, ModeNone)
	tracker.Apply(input, ModeNone)
	
	stats = tracker.Stats()
	// Note: processSegments splits into segments, so we may get multiple entries
	if stats.UniqueEntries < 1 {
		t.Errorf("After processing, UniqueEntries = %d, want >= 1", stats.UniqueEntries)
	}
	if stats.TotalOccurrences < 2 {
		t.Errorf("TotalOccurrences = %d, want >= 2", stats.TotalOccurrences)
	}
	// With identical content applied twice, we should have multi-occurrences
	if stats.MultiOccurrences < 1 {
		t.Errorf("MultiOccurrences = %d, want >= 1", stats.MultiOccurrences)
	}
}

func TestSessionTracker_Clear(t *testing.T) {
	tracker := NewSessionTracker()
	
	// Add content - single segment
	input := "content to clear line one\ncontent to clear line two"
	tracker.Apply(input, ModeNone)
	
	// Verify entry exists (may be multiple due to segmentation)
	stats := tracker.Stats()
	if stats.UniqueEntries < 1 {
		t.Fatalf("Expected >= 1 entry before clear, got %d", stats.UniqueEntries)
	}
	
	// Clear (ignore error if file doesn't exist)
	_ = tracker.Clear()
	
	// Verify cleared
	stats = tracker.Stats()
	if stats.UniqueEntries != 0 {
		t.Errorf("After Clear(), UniqueEntries = %d, want 0", stats.UniqueEntries)
	}
}

func TestSessionTracker_HashContent(t *testing.T) {
	tracker := NewSessionTracker()
	
	// Same content should produce same hash
	hash1 := tracker.hashContent("Test Content")
	hash2 := tracker.hashContent("test content") // lowercase
	
	if hash1 != hash2 {
		t.Errorf("hashContent should be case-insensitive: %q vs %q", hash1, hash2)
	}
	
	// Different content should produce different hash
	hash3 := tracker.hashContent("different content")
	if hash1 == hash3 {
		t.Errorf("Different content should produce different hashes")
	}
}

func TestSessionTracker_SummarizeSegment(t *testing.T) {
	tracker := NewSessionTracker()
	
	tests := []struct {
		name    string
		input   string
		wantLen int // max length
	}{
		{
			name:    "single line",
			input:   "This is a test line",
			wantLen: 50,
		},
		{
			name:    "long line truncated",
			input:   "This is a very long line that should be truncated to fit within the limit",
			wantLen: 50,
		},
		{
			name:    "empty",
			input:   "",
			wantLen: 10, // "X lines"
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tracker.summarizeSegment(tt.input)
			if len(summary) > tt.wantLen {
				t.Errorf("summarizeSegment() length = %d, want <= %d", len(summary), tt.wantLen)
			}
		})
	}
}

func TestSessionTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewSessionTracker()
	
	// Run concurrent Apply operations
	done := make(chan bool)
	
	for i := 0; i < 10; i++ {
		go func(idx int) {
			input := strings.Repeat("concurrent test content\n", 10)
			tracker.Apply(input, ModeNone)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should have tracked the content
	stats := tracker.Stats()
	if stats.UniqueEntries == 0 {
		t.Errorf("Concurrent access should have tracked entries")
	}
}

func TestRemoveTimestamps(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // substring check
	}{
		{
			name:  "date only",
			input: "Log from 2024-01-15 shows error",
			want:  "Log from            shows error",
		},
		{
			name:  "no date",
			input: "No timestamp here",
			want:  "No timestamp here",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTimestamps(tt.input)
			if got != tt.want {
				t.Errorf("removeTimestamps() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsOnlyNumbers(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"12345", true},
		{"12 34 56", true},
		{"2024-01-15", true},
		{"12:34:56", true},
		{"abc123", false},
		{"", true}, // empty is considered "only numbers"
	}
	
	for _, tt := range tests {
		got := isOnlyNumbers(tt.input)
		if got != tt.want {
			t.Errorf("isOnlyNumbers(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsDigit(t *testing.T) {
	for i := byte('0'); i <= byte('9'); i++ {
		if !isDigit(i) {
			t.Errorf("isDigit(%q) = false, want true", i)
		}
	}
	
	if isDigit('a') {
		t.Errorf("isDigit('a') = true, want false")
	}
}
