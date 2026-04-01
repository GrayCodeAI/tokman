package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func setupTestDB(t *testing.T) *tracking.Tracker {
	tmpDir, err := os.MkdirTemp("", "tokman-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	dbPath := filepath.Join(tmpDir, "test.db")
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tracker.Close() })

	return tracker
}

func TestStatsHandler(t *testing.T) {
	tracker := setupTestDB(t)

	// Record some test data
	tracker.Record(&tracking.CommandRecord{
		Command:        "ls -la",
		OriginalTokens: 100,
		FilteredTokens: 50,
		SavedTokens:    50,
		ExecTimeMs:     10,
	})

	req := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()

	handler := statsHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["commands_count"].(float64) != 1 {
		t.Errorf("Expected 1 command, got %v", response["commands_count"])
	}
}

func TestDailyHandler(t *testing.T) {
	tracker := setupTestDB(t)

	// Record test data
	for i := 0; i < 5; i++ {
		tracker.Record(&tracking.CommandRecord{
			Command:        "test command",
			OriginalTokens: 100,
			FilteredTokens: 80,
			SavedTokens:    20,
			ExecTimeMs:     5,
		})
	}

	req := httptest.NewRequest("GET", "/api/daily?days=7", nil)
	w := httptest.NewRecorder()

	handler := dailyHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have at least one day with data
	if len(response) == 0 {
		t.Error("Expected at least one day of data")
	}
}

func TestTopCommandsHandler(t *testing.T) {
	tracker := setupTestDB(t)

	// Record various commands
	commands := []struct {
		cmd   string
		saved int
		count int
	}{
		{"ls -la", 100, 3},
		{"cat file.txt", 200, 2},
		{"grep pattern", 50, 5},
	}

	for _, c := range commands {
		for i := 0; i < c.count; i++ {
			tracker.Record(&tracking.CommandRecord{
				Command:        c.cmd,
				OriginalTokens: c.saved + 50,
				FilteredTokens: 50,
				SavedTokens:    c.saved,
				ExecTimeMs:     10,
			})
		}
	}

	req := httptest.NewRequest("GET", "/api/top-commands", nil)
	w := httptest.NewRecorder()

	handler := topCommandsHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response) == 0 {
		t.Error("Expected at least one command")
	}
}

func TestContextReadsHandler(t *testing.T) {
	tracker := setupTestDB(t)

	if err := tracker.Record(&tracking.CommandRecord{
		Command:        "tokman ctx read main.go",
		OriginalTokens: 100,
		FilteredTokens: 40,
		SavedTokens:    60,
		ExecTimeMs:     12,
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if err := tracker.Record(&tracking.CommandRecord{
		Command:        "git status",
		OriginalTokens: 50,
		FilteredTokens: 10,
		SavedTokens:    40,
		ExecTimeMs:     7,
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/context-reads", nil)
	w := httptest.NewRecorder()

	handler := contextReadsHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 context-read record, got %d", len(response))
	}
	if response[0]["command"].(string) != "tokman ctx read main.go" {
		t.Fatalf("unexpected command %v", response[0]["command"])
	}
}

func TestContextReadsHandler_FilterByKind(t *testing.T) {
	tracker := setupTestDB(t)

	for _, record := range []*tracking.CommandRecord{
		{Command: "tokman ctx read main.go", OriginalTokens: 100, FilteredTokens: 40, SavedTokens: 60, ExecTimeMs: 12, ContextKind: "read", ContextMode: "auto", ContextResolvedMode: "signatures", ContextTarget: "main.go"},
		{Command: "tokman ctx delta main.go", OriginalTokens: 50, FilteredTokens: 10, SavedTokens: 40, ExecTimeMs: 8, ContextKind: "delta", ContextMode: "delta", ContextResolvedMode: "delta", ContextTarget: "main.go"},
		{Command: "tokman mcp read /tmp/main.go", OriginalTokens: 70, FilteredTokens: 20, SavedTokens: 50, ExecTimeMs: 5, ContextKind: "mcp", ContextMode: "graph", ContextResolvedMode: "graph", ContextTarget: "/tmp/main.go", ContextBundle: true, ContextRelatedFiles: 4},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/context-reads?kind=delta", nil)
	w := httptest.NewRecorder()
	contextReadsHandler(tracker)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 delta record, got %d", len(response))
	}
	if response[0]["command"].(string) != "tokman ctx delta main.go" {
		t.Fatalf("unexpected command %v", response[0]["command"])
	}
	if response[0]["kind"].(string) != "delta" {
		t.Fatalf("expected delta kind, got %v", response[0]["kind"])
	}
}

func TestContextReadsHandler_FilterByMode(t *testing.T) {
	tracker := setupTestDB(t)

	for _, record := range []*tracking.CommandRecord{
		{Command: "tokman ctx read alpha.go", OriginalTokens: 100, FilteredTokens: 25, SavedTokens: 75, ContextKind: "read", ContextMode: "auto", ContextResolvedMode: "map", ContextTarget: "alpha.go"},
		{Command: "tokman ctx read beta.go", OriginalTokens: 80, FilteredTokens: 20, SavedTokens: 60, ContextKind: "read", ContextMode: "auto", ContextResolvedMode: "signatures", ContextTarget: "beta.go"},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/context-reads?mode=map", nil)
	w := httptest.NewRecorder()
	contextReadsHandler(tracker)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 map record, got %d", len(response))
	}
	if response[0]["mode"].(string) != "map" {
		t.Fatalf("expected map mode, got %v", response[0]["mode"])
	}
}

func TestContextReadSummaryHandler(t *testing.T) {
	tracker := setupTestDB(t)

	for _, record := range []*tracking.CommandRecord{
		{Command: "tokman ctx read main.go", OriginalTokens: 100, FilteredTokens: 40, SavedTokens: 60},
		{Command: "tokman ctx delta main.go", OriginalTokens: 50, FilteredTokens: 10, SavedTokens: 40},
		{Command: "tokman mcp read /tmp/main.go", OriginalTokens: 70, FilteredTokens: 20, SavedTokens: 50},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/context-read-summary", nil)
	w := httptest.NewRecorder()
	contextReadSummaryHandler(tracker)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if response["read"]["commands"].(float64) != 1 {
		t.Fatalf("expected 1 read command, got %v", response["read"]["commands"])
	}
	if response["delta"]["saved"].(float64) != 40 {
		t.Fatalf("expected 40 delta tokens saved, got %v", response["delta"]["saved"])
	}
	if response["mcp"]["saved"].(float64) != 50 {
		t.Fatalf("expected 50 mcp tokens saved, got %v", response["mcp"]["saved"])
	}
}

func TestContextReadTrendHandler(t *testing.T) {
	tracker := setupTestDB(t)
	if err := tracker.Record(&tracking.CommandRecord{
		Command:        "tokman ctx read main.go",
		OriginalTokens: 100,
		FilteredTokens: 40,
		SavedTokens:    60,
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/context-read-trend", nil)
	w := httptest.NewRecorder()
	contextReadTrendHandler(tracker)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(response) == 0 {
		t.Fatal("expected at least one trend point")
	}
}

func TestContextReadTopFilesHandler(t *testing.T) {
	tracker := setupTestDB(t)
	for _, record := range []*tracking.CommandRecord{
		{Command: "tokman ctx read alpha.go", OriginalTokens: 100, FilteredTokens: 30, SavedTokens: 70},
		{Command: "tokman ctx read alpha.go", OriginalTokens: 90, FilteredTokens: 20, SavedTokens: 70},
		{Command: "tokman ctx delta beta.go", OriginalTokens: 50, FilteredTokens: 10, SavedTokens: 40},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/context-read-top-files", nil)
	w := httptest.NewRecorder()
	contextReadTopFilesHandler(tracker)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(response) == 0 || response[0]["file"].(string) != "alpha.go" {
		t.Fatalf("expected alpha.go to rank first, got %v", response)
	}
}

func TestContextReadComparisonHandler(t *testing.T) {
	tracker := setupTestDB(t)
	for _, record := range []*tracking.CommandRecord{
		{Command: "tokman ctx read alpha.go", OriginalTokens: 100, FilteredTokens: 30, SavedTokens: 70},
		{Command: "tokman mcp bundle /tmp/alpha.go", OriginalTokens: 200, FilteredTokens: 60, SavedTokens: 140},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/context-read-comparison", nil)
	w := httptest.NewRecorder()
	contextReadComparisonHandler(tracker)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if response["single"]["tokens_saved"].(float64) != 70 {
		t.Fatalf("expected 70 single-file tokens saved, got %v", response["single"]["tokens_saved"])
	}
	if response["bundle"]["tokens_saved"].(float64) != 140 {
		t.Fatalf("expected 140 bundle tokens saved, got %v", response["bundle"]["tokens_saved"])
	}
}

func TestContextReadQualityHandler(t *testing.T) {
	tracker := setupTestDB(t)
	for _, record := range []*tracking.CommandRecord{
		{Command: "tokman ctx read alpha.go", OriginalTokens: 100, FilteredTokens: 30, SavedTokens: 70, ContextKind: "read", ContextMode: "auto", ContextResolvedMode: "map", ContextTarget: "alpha.go"},
		{Command: "tokman mcp bundle /tmp/alpha.go", OriginalTokens: 200, FilteredTokens: 60, SavedTokens: 140, ContextKind: "mcp", ContextMode: "graph", ContextResolvedMode: "graph", ContextTarget: "/tmp/alpha.go", ContextBundle: true, ContextRelatedFiles: 4},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/context-read-quality", nil)
	w := httptest.NewRecorder()
	contextReadQualityHandler(tracker)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string][]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(response["modes"]) == 0 {
		t.Fatal("expected at least one mode entry")
	}
	if response["modes"][0]["mode"] == nil {
		t.Fatalf("expected mode field in response: %v", response["modes"][0])
	}
}

func TestExportCSVHandler(t *testing.T) {
	tracker := setupTestDB(t)

	// Record test data
	tracker.Record(&tracking.CommandRecord{
		Command:        "test command",
		OriginalTokens: 100,
		FilteredTokens: 50,
		SavedTokens:    50,
		ExecTimeMs:     10,
		ProjectPath:    "/test/path",
	})

	req := httptest.NewRequest("GET", "/api/export/csv", nil)
	w := httptest.NewRecorder()

	handler := exportCSVHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	if w.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("Expected text/csv content type, got %s", w.Header().Get("Content-Type"))
	}

	// Check disposition header
	if w.Header().Get("Content-Disposition") == "" {
		t.Error("Expected Content-Disposition header")
	}
}

func TestExportJSONHandler(t *testing.T) {
	tracker := setupTestDB(t)

	// Record test data
	tracker.Record(&tracking.CommandRecord{
		Command:        "test command",
		OriginalTokens: 100,
		FilteredTokens: 50,
		SavedTokens:    50,
		ExecTimeMs:     10,
	})

	req := httptest.NewRequest("GET", "/api/export/json", nil)
	w := httptest.NewRecorder()

	handler := exportJSONHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["exported_at"] == nil {
		t.Error("Expected exported_at field")
	}
}

func TestAlertsHandler(t *testing.T) {
	tracker := setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/alerts", nil)
	w := httptest.NewRecorder()

	handler := alertsHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["enabled"] == nil {
		t.Error("Expected enabled field")
	}
}

func TestReportHandler(t *testing.T) {
	tracker := setupTestDB(t)

	// Record test data
	for i := 0; i < 10; i++ {
		tracker.Record(&tracking.CommandRecord{
			Command:        "test command",
			OriginalTokens: 100,
			FilteredTokens: 50,
			SavedTokens:    50,
			ExecTimeMs:     10,
			ProjectPath:    "/test/project",
		})
	}

	req := httptest.NewRequest("GET", "/api/report?type=weekly", nil)
	w := httptest.NewRecorder()

	handler := reportHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["report_type"] != "weekly" {
		t.Errorf("Expected report_type 'weekly', got %v", response["report_type"])
	}
}

func TestCacheMetricsHandler(t *testing.T) {
	tracker := setupTestDB(t)

	// Record test data
	for _, record := range []*tracking.CommandRecord{
		{
			Command:        "tokman ctx read alpha.go",
			OriginalTokens: 200,
			FilteredTokens: 50,
			SavedTokens:    150,
			ExecTimeMs:     10,
			ProjectPath:    "/test/project-a",
		},
		{
			Command:        "tokman ctx delta beta.go",
			OriginalTokens: 100,
			FilteredTokens: 20,
			SavedTokens:    80,
			ExecTimeMs:     8,
			ProjectPath:    "/test/project-b",
		},
	} {
		if err := tracker.Record(record); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/cache-metrics", nil)
	w := httptest.NewRecorder()

	handler := cacheMetricsHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["efficiency_pct"] == nil {
		t.Error("Expected efficiency_pct field")
	}
	if response["context_effectiveness_by_kind"] == nil {
		t.Error("Expected context_effectiveness_by_kind field")
	}
	if response["context_effectiveness_by_project"] == nil {
		t.Error("Expected context_effectiveness_by_project field")
	}
}
