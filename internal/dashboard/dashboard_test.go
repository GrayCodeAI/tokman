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

	var response map[string]interface{}
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

	var response []map[string]interface{}
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

	var response []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response) == 0 {
		t.Error("Expected at least one command")
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

	var response map[string]interface{}
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

	var response map[string]interface{}
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

	var response map[string]interface{}
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
	tracker.Record(&tracking.CommandRecord{
		Command:        "test",
		OriginalTokens: 200,
		FilteredTokens: 50,
		SavedTokens:    150,
		ExecTimeMs:     10,
	})

	req := httptest.NewRequest("GET", "/api/cache-metrics", nil)
	w := httptest.NewRecorder()

	handler := cacheMetricsHandler(tracker)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["efficiency_pct"] == nil {
		t.Error("Expected efficiency_pct field")
	}
}
