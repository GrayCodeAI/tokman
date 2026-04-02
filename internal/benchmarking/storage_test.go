package benchmarking

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	if storage.db == nil {
		t.Error("expected db to be initialized")
	}
}

func TestStorageSaveAndLoadSuite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create a report
	report := &SuiteReport{
		Name:      "test-suite",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(10 * time.Second),
		Duration:  10 * time.Second,
		Results: []BenchmarkResult{
			{
				Name:       "test-1",
				Type:       TypeCompression,
				Duration:   1 * time.Second,
				TokensIn:   1000,
				TokensOut:  500,
				Throughput: 1000,
				Timestamp:  time.Now(),
			},
		},
	}

	// Save
	if err := storage.SaveSuite(report); err != nil {
		t.Fatalf("failed to save suite: %v", err)
	}

	// List suites
	suites, err := storage.ListSuites(10)
	if err != nil {
		t.Fatalf("failed to list suites: %v", err)
	}

	if len(suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites))
	}

	// Load
	loaded, err := storage.LoadSuite(suites[0].ID)
	if err != nil {
		t.Fatalf("failed to load suite: %v", err)
	}

	if loaded.Name != "test-suite" {
		t.Errorf("expected name 'test-suite', got %s", loaded.Name)
	}

	if len(loaded.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(loaded.Results))
	}
}

func TestStorageListSuitesLimit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	// Save multiple suites
	for i := 0; i < 5; i++ {
		report := &SuiteReport{
			Name:      fmt.Sprintf("suite-%d", i),
			StartTime: time.Now(),
			Results:   []BenchmarkResult{},
		}
		if err := storage.SaveSuite(report); err != nil {
			t.Fatalf("failed to save suite: %v", err)
		}
	}

	// List with limit
	suites, err := storage.ListSuites(3)
	if err != nil {
		t.Fatalf("failed to list suites: %v", err)
	}

	if len(suites) != 3 {
		t.Errorf("expected 3 suites, got %d", len(suites))
	}
}

func TestStorageSaveAndLoadTrend(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	trend := &Trend{
		BenchmarkID: "test-bench",
		Metric:      "throughput",
		Direction:   TrendUp,
		Slope:       10.5,
		Correlation: 0.95,
		ChangePct:   15.0,
		History: []HistoricalPoint{
			{Timestamp: time.Now(), Value: 100},
			{Timestamp: time.Now(), Value: 115},
		},
	}

	// Save
	if err := storage.SaveTrend(trend); err != nil {
		t.Fatalf("failed to save trend: %v", err)
	}

	// Load
	trends, err := storage.LoadTrends("test-bench")
	if err != nil {
		t.Fatalf("failed to load trends: %v", err)
	}

	if len(trends) != 1 {
		t.Fatalf("expected 1 trend, got %d", len(trends))
	}

	if trends[0].Metric != "throughput" {
		t.Errorf("expected metric 'throughput', got %s", trends[0].Metric)
	}
}

func TestStorageSaveAndLoadRegression(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	regression := &Regression{
		BenchmarkID: "test",
		Metric:      "latency",
		Baseline:    100,
		Current:     120,
		ChangePct:   20,
		Severity:    SeverityHigh,
		Description: "Latency increased",
	}

	// Save
	if err := storage.SaveRegression(regression); err != nil {
		t.Fatalf("failed to save regression: %v", err)
	}

	// Load
	regressions, err := storage.LoadRegressions(time.Time{})
	if err != nil {
		t.Fatalf("failed to load regressions: %v", err)
	}

	if len(regressions) != 1 {
		t.Fatalf("expected 1 regression, got %d", len(regressions))
	}

	if regressions[0].BenchmarkID != "test" {
		t.Errorf("expected benchmark 'test', got %s", regressions[0].BenchmarkID)
	}
}

func TestStorageGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	// Save some data
	report := &SuiteReport{
		Name:    "test",
		Results: []BenchmarkResult{{Name: "bench"}},
	}
	storage.SaveSuite(report)

	// Get stats
	stats, err := storage.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.SuiteCount != 1 {
		t.Errorf("expected 1 suite, got %d", stats.SuiteCount)
	}

	if stats.ResultCount != 1 {
		t.Errorf("expected 1 result, got %d", stats.ResultCount)
	}
}

func TestStorageCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	// Save old data
	oldReport := &SuiteReport{
		Name:      "old",
		StartTime: time.Now().Add(-30 * 24 * time.Hour),
		Results:   []BenchmarkResult{{Name: "bench", Timestamp: time.Now().Add(-30 * 24 * time.Hour)}},
	}
	storage.SaveSuite(oldReport)

	// Save new data
	newReport := &SuiteReport{
		Name:      "new",
		StartTime: time.Now(),
		Results:   []BenchmarkResult{{Name: "bench", Timestamp: time.Now()}},
	}
	storage.SaveSuite(newReport)

	// Cleanup old data (older than 7 days)
	if err := storage.Cleanup(7 * 24 * time.Hour); err != nil {
		t.Fatalf("failed to cleanup: %v", err)
	}

	// Check stats
	stats, _ := storage.GetStats()
	// Should have 1 suite (the new one)
	if stats.SuiteCount != 1 {
		t.Errorf("expected 1 suite after cleanup, got %d", stats.SuiteCount)
	}
}

func BenchmarkStorageSaveSuite(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	storage, _ := NewStorage(dbPath)
	defer storage.Close()

	report := &SuiteReport{
		Name:    "bench",
		Results: make([]BenchmarkResult, 10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.SaveSuite(report)
	}
}
