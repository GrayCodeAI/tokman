package tracking

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		minExpected int // BPE may differ from heuristic, use minimum
	}{
		{
			name:        "empty string",
			input:       "",
			minExpected: 0,
		},
		{
			name:        "short string",
			input:       "test",
			minExpected: 1,
		},
		{
			name:        "medium string",
			input:       "hello world",
			minExpected: 2,
		},
		{
			name:        "exact multiple",
			input:       "four",
			minExpected: 1,
		},
		{
			name:        "long string",
			input:       "The quick brown fox jumps over the lazy dog and runs away into the forest.",
			minExpected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result < tt.minExpected {
				t.Errorf("EstimateTokens() = %d, want >= %d", result, tt.minExpected)
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

func TestNewTrackerCreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "tracker", "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("expected parent directory to exist: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected database file to exist: %v", err)
	}
}

func TestDatabasePathUsesConfiguredPath(t *testing.T) {
	tmpDir := t.TempDir()
	xdgConfigHome := filepath.Join(tmpDir, "xdg-config")
	configDir := filepath.Join(xdgConfigHome, "tokman")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	wantPath := filepath.Join(tmpDir, "custom", "tracking.sqlite")
	content := []byte("[tracking]\ndatabase_path = \"" + wantPath + "\"\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	if got := DatabasePath(); got != wantPath {
		t.Fatalf("DatabasePath() = %q, want %q", got, wantPath)
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

func TestGetSavingsForCommands(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	records := []*CommandRecord{
		{
			Command:        "tokman ctx read main.go",
			OriginalTokens: 200,
			FilteredTokens: 80,
			SavedTokens:    120,
			ProjectPath:    "/test/project",
			ParseSuccess:   true,
		},
		{
			Command:        "tokman read main.go",
			OriginalTokens: 100,
			FilteredTokens: 40,
			SavedTokens:    60,
			ProjectPath:    "/test/project",
			ParseSuccess:   true,
		},
		{
			Command:        "git status",
			OriginalTokens: 80,
			FilteredTokens: 20,
			SavedTokens:    60,
			ProjectPath:    "/test/project",
			ParseSuccess:   true,
		},
	}
	for _, record := range records {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	summary, err := tracker.GetSavingsForCommands("/test/project", []string{"tokman ctx read *", "tokman read *"})
	if err != nil {
		t.Fatalf("GetSavingsForCommands() error = %v", err)
	}

	if summary.TotalCommands != 2 {
		t.Fatalf("TotalCommands = %d, want 2", summary.TotalCommands)
	}
	if summary.TotalSaved != 180 {
		t.Fatalf("TotalSaved = %d, want 180", summary.TotalSaved)
	}
	if summary.TotalOriginal != 300 {
		t.Fatalf("TotalOriginal = %d, want 300", summary.TotalOriginal)
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

func TestProjectPathsAreNormalizedForRecordAndQuery(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink normalization test is Unix-focused")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	realProject := filepath.Join(tmpDir, "real-project")
	if err := os.MkdirAll(realProject, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	wantProjectPath, err := filepath.EvalSymlinks(realProject)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	linkProject := filepath.Join(tmpDir, "project-link")
	if err := os.Symlink(realProject, linkProject); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if err := tracker.Record(&CommandRecord{
		Command:      "echo hi",
		ProjectPath:  linkProject,
		SavedTokens:  5,
		ParseSuccess: true,
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	summary, err := tracker.GetSavings(realProject)
	if err != nil {
		t.Fatalf("GetSavings() error = %v", err)
	}
	if summary.TotalCommands != 1 {
		t.Fatalf("TotalCommands = %d, want 1", summary.TotalCommands)
	}

	records, err := tracker.GetRecentCommands(linkProject, 5)
	if err != nil {
		t.Fatalf("GetRecentCommands() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].ProjectPath != wantProjectPath {
		t.Fatalf("stored ProjectPath = %q, want %q", records[0].ProjectPath, wantProjectPath)
	}
}

func TestRecordAndQueryContextMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	record := &CommandRecord{
		Command:             "tokman ctx read main.go",
		OriginalTokens:      200,
		FilteredTokens:      60,
		SavedTokens:         140,
		ProjectPath:         "/test/project",
		ParseSuccess:        true,
		ContextKind:         "read",
		ContextMode:         "auto",
		ContextResolvedMode: "signatures",
		ContextTarget:       "/test/project/main.go",
		ContextRelatedFiles: 4,
		ContextBundle:       true,
	}
	if err := tracker.Record(record); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	records, err := tracker.GetRecentContextReads("/test/project", "read", "", 10)
	if err != nil {
		t.Fatalf("GetRecentContextReads() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].ContextResolvedMode != "signatures" {
		t.Fatalf("expected resolved mode signatures, got %q", records[0].ContextResolvedMode)
	}
	if records[0].ContextTarget != "/test/project/main.go" {
		t.Fatalf("expected target to persist, got %q", records[0].ContextTarget)
	}
	if !records[0].ContextBundle {
		t.Fatal("expected bundle flag to persist")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	if err := tracker.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := tracker.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestGetSavingsForContextReadsFallbackAndMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	for _, record := range []*CommandRecord{
		{
			Command:             "tokman ctx read alpha.go",
			OriginalTokens:      120,
			FilteredTokens:      30,
			SavedTokens:         90,
			ProjectPath:         "/test/project",
			ParseSuccess:        true,
			ContextKind:         "read",
			ContextMode:         "auto",
			ContextResolvedMode: "map",
		},
		{
			Command:        "tokman mcp bundle /tmp/alpha.go",
			OriginalTokens: 200,
			FilteredTokens: 80,
			SavedTokens:    120,
			ProjectPath:    "/test/project",
			ParseSuccess:   true,
		},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	modeSummary, err := tracker.GetSavingsForContextReads("/test/project", "read", "map")
	if err != nil {
		t.Fatalf("GetSavingsForContextReads() mode error = %v", err)
	}
	if modeSummary.TotalCommands != 1 || modeSummary.TotalSaved != 90 {
		t.Fatalf("unexpected mode summary: %+v", modeSummary)
	}

	fallbackSummary, err := tracker.GetSavingsForContextReads("/test/project", "mcp", "")
	if err != nil {
		t.Fatalf("GetSavingsForContextReads() fallback error = %v", err)
	}
	if fallbackSummary.TotalCommands != 1 || fallbackSummary.TotalSaved != 120 {
		t.Fatalf("unexpected fallback summary: %+v", fallbackSummary)
	}
}
