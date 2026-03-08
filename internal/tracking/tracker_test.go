package tracking

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "short string",
			input:    "hello",
			expected: 2, // ceil(5/4) = 2
		},
		{
			name:     "medium string",
			input:    "hello world",
			expected: 3, // ceil(11/4) = 3
		},
		{
			name:     "exact multiple",
			input:    "four",
			expected: 1, // ceil(4/4) = 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result != tt.expected {
				t.Errorf("EstimateTokens() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestNewTracker(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	// Check database was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestRecord(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	record := &CommandRecord{
		Command:        "git status",
		OriginalOutput: "long output here",
		FilteredOutput: "short output",
		OriginalTokens: 100,
		FilteredTokens: 20,
		SavedTokens:    80,
		ProjectPath:    "/test/project",
		ExecTimeMs:     50,
		ParseSuccess:   true,
	}

	err = tracker.Record(record)
	if err != nil {
		t.Errorf("Record() error = %v", err)
	}
}

func TestGetSavings(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	// Record some commands
	records := []*CommandRecord{
		{Command: "git status", SavedTokens: 100, ProjectPath: "/test/project", ParseSuccess: true},
		{Command: "git diff", SavedTokens: 200, ProjectPath: "/test/project", ParseSuccess: true},
		{Command: "ls", SavedTokens: 50, ProjectPath: "/test/project", ParseSuccess: true},
	}

	for _, r := range records {
		tracker.Record(r)
	}

	summary, err := tracker.GetSavings("/test/project")
	if err != nil {
		t.Errorf("GetSavings() error = %v", err)
	}

	if summary.TotalCommands != 3 {
		t.Errorf("TotalCommands = %d, want 3", summary.TotalCommands)
	}

	if summary.TotalSaved != 350 {
		t.Errorf("TotalSaved = %d, want 350", summary.TotalSaved)
	}
}

func TestGetRecentCommands(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	// Record some commands
	for i := 0; i < 10; i++ {
		tracker.Record(&CommandRecord{
			Command:      "test command",
			ProjectPath:  "/test",
			ParseSuccess: true,
		})
	}

	commands, err := tracker.GetRecentCommands("/test", 5)
	if err != nil {
		t.Errorf("GetRecentCommands() error = %v", err)
	}

	if len(commands) != 5 {
		t.Errorf("GetRecentCommands() returned %d commands, want 5", len(commands))
	}
}
