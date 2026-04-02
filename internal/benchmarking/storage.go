// Package benchmarking provides storage capabilities for benchmark results
package benchmarking

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Storage provides persistent storage for benchmark results
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new benchmark storage
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &Storage{db: db}
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return storage, nil
}

// Close closes the storage
func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS benchmark_suites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT
);

CREATE TABLE IF NOT EXISTS benchmark_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    suite_id INTEGER,
    name TEXT NOT NULL,
    type TEXT,
    duration_ms INTEGER,
    tokens_in INTEGER,
    tokens_out INTEGER,
    throughput REAL,
    memory_used_mb REAL,
    allocations INTEGER,
    latency_p50_ms INTEGER,
    latency_p95_ms INTEGER,
    latency_p99_ms INTEGER,
    errors INTEGER,
    success_rate REAL,
    timestamp TIMESTAMP,
    metadata TEXT,
    FOREIGN KEY (suite_id) REFERENCES benchmark_suites(id)
);

CREATE TABLE IF NOT EXISTS benchmark_trends (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    benchmark_name TEXT NOT NULL,
    metric TEXT NOT NULL,
    direction TEXT,
    slope REAL,
    correlation REAL,
    change_pct REAL,
    detected_at TIMESTAMP,
    history TEXT
);

CREATE TABLE IF NOT EXISTS benchmark_regressions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    benchmark_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    baseline REAL,
    current REAL,
    change_pct REAL,
    severity TEXT,
    description TEXT,
    detected_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_results_suite ON benchmark_results(suite_id);
CREATE INDEX IF NOT EXISTS idx_results_name ON benchmark_results(name);
CREATE INDEX IF NOT EXISTS idx_results_timestamp ON benchmark_results(timestamp);
CREATE INDEX IF NOT EXISTS idx_trends_name ON benchmark_trends(benchmark_name);
CREATE INDEX IF NOT EXISTS idx_regressions_name ON benchmark_regressions(benchmark_id);
`

	_, err := s.db.Exec(schema)
	return err
}

// SaveSuite saves a benchmark suite and its results
func (s *Storage) SaveSuite(report *SuiteReport) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert suite
	metadata, _ := json.Marshal(map[string]interface{}{
		"duration_ms": report.Duration.Milliseconds(),
	})

	res, err := tx.Exec(
		"INSERT INTO benchmark_suites (name, metadata) VALUES (?, ?)",
		report.Name, string(metadata),
	)
	if err != nil {
		return err
	}

	suiteID, _ := res.LastInsertId()

	// Insert results
	for _, r := range report.Results {
		metadata, _ := json.Marshal(r.Metadata)

		_, err := tx.Exec(`
			INSERT INTO benchmark_results (
				suite_id, name, type, duration_ms, tokens_in, tokens_out,
				throughput, memory_used_mb, allocations, latency_p50_ms,
				latency_p95_ms, latency_p99_ms, errors, success_rate,
				timestamp, metadata
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			suiteID, r.Name, string(r.Type), r.Duration.Milliseconds(),
			r.TokensIn, r.TokensOut, r.Throughput, r.MemoryUsedMB,
			r.Allocations, r.LatencyP50.Milliseconds(), r.LatencyP95.Milliseconds(),
			r.LatencyP99.Milliseconds(), r.Errors, r.SuccessRate,
			r.Timestamp, string(metadata),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadSuite loads a benchmark suite by ID
func (s *Storage) LoadSuite(suiteID int64) (*SuiteReport, error) {
	// Load suite info
	var name string
	var createdAt time.Time
	err := s.db.QueryRow(
		"SELECT name, created_at FROM benchmark_suites WHERE id = ?",
		suiteID,
	).Scan(&name, &createdAt)
	if err != nil {
		return nil, err
	}

	// Load results
	rows, err := s.db.Query(`
		SELECT name, type, duration_ms, tokens_in, tokens_out,
			throughput, memory_used_mb, allocations, latency_p50_ms,
			latency_p95_ms, latency_p99_ms, errors, success_rate,
			timestamp, metadata
		FROM benchmark_results WHERE suite_id = ?
	`, suiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := &SuiteReport{
		Name:      name,
		StartTime: createdAt,
		Results:   make([]BenchmarkResult, 0),
	}

	for rows.Next() {
		var r BenchmarkResult
		var durationMs, latencyP50Ms, latencyP95Ms, latencyP99Ms int64
		var timestamp time.Time
		var metadataStr string

		var typeStr string
		err := rows.Scan(
			&r.Name, &typeStr, &durationMs, &r.TokensIn, &r.TokensOut,
			&r.Throughput, &r.MemoryUsedMB, &r.Allocations, &latencyP50Ms,
			&latencyP95Ms, &latencyP99Ms, &r.Errors, &r.SuccessRate,
			&timestamp, &metadataStr,
		)
		if err != nil {
			return nil, err
		}

		r.Type = BenchmarkType(typeStr)
		r.Duration = time.Duration(durationMs) * time.Millisecond
		r.LatencyP50 = time.Duration(latencyP50Ms) * time.Millisecond
		r.LatencyP95 = time.Duration(latencyP95Ms) * time.Millisecond
		r.LatencyP99 = time.Duration(latencyP99Ms) * time.Millisecond
		r.Timestamp = timestamp

		if metadataStr != "" {
			json.Unmarshal([]byte(metadataStr), &r.Metadata)
		}

		report.Results = append(report.Results, r)
	}

	return report, nil
}

// ListSuites lists all benchmark suites
func (s *Storage) ListSuites(limit int) ([]SuiteInfo, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.Query(`
		SELECT id, name, created_at, metadata
		FROM benchmark_suites
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suites []SuiteInfo
	for rows.Next() {
		var info SuiteInfo
		var metadataStr string
		err := rows.Scan(&info.ID, &info.Name, &info.CreatedAt, &metadataStr)
		if err != nil {
			return nil, err
		}
		suites = append(suites, info)
	}

	return suites, nil
}

// SuiteInfo provides info about a stored suite
type SuiteInfo struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// SaveTrend saves a trend analysis
func (s *Storage) SaveTrend(trend *Trend) error {
	historyJSON, _ := json.Marshal(trend.History)

	_, err := s.db.Exec(`
		INSERT INTO benchmark_trends (
			benchmark_name, metric, direction, slope, correlation,
			change_pct, detected_at, history
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		trend.BenchmarkID, trend.Metric, trend.Direction,
		trend.Slope, trend.Correlation, trend.ChangePct,
		time.Now(), string(historyJSON),
	)

	return err
}

// LoadTrends loads trends for a benchmark
func (s *Storage) LoadTrends(benchmarkName string) ([]Trend, error) {
	rows, err := s.db.Query(`
		SELECT metric, direction, slope, correlation, change_pct, history
		FROM benchmark_trends
		WHERE benchmark_name = ?
		ORDER BY detected_at DESC
	`, benchmarkName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []Trend
	for rows.Next() {
		var t Trend
		var historyJSON string

		err := rows.Scan(
			&t.Metric, &t.Direction, &t.Slope,
			&t.Correlation, &t.ChangePct, &historyJSON,
		)
		if err != nil {
			return nil, err
		}

		t.BenchmarkID = benchmarkName
		if historyJSON != "" {
			json.Unmarshal([]byte(historyJSON), &t.History)
		}

		trends = append(trends, t)
	}

	return trends, nil
}

// SaveRegression saves a regression detection
func (s *Storage) SaveRegression(regression *Regression) error {
	_, err := s.db.Exec(`
		INSERT INTO benchmark_regressions (
			benchmark_id, metric, baseline, current, change_pct,
			severity, description, detected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		regression.BenchmarkID, regression.Metric,
		regression.Baseline, regression.Current,
		regression.ChangePct, regression.Severity,
		regression.Description, time.Now(),
	)

	return err
}

// LoadRegressions loads recent regressions
func (s *Storage) LoadRegressions(since time.Time) ([]Regression, error) {
	rows, err := s.db.Query(`
		SELECT benchmark_id, metric, baseline, current, change_pct,
			severity, description, detected_at
		FROM benchmark_regressions
		WHERE detected_at > ?
		ORDER BY detected_at DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regressions []Regression
	for rows.Next() {
		var r Regression
		err := rows.Scan(
			&r.BenchmarkID, &r.Metric, &r.Baseline, &r.Current,
			&r.ChangePct, &r.Severity, &r.Description, &r.DetectedAt,
		)
		if err != nil {
			return nil, err
		}
		regressions = append(regressions, r)
	}

	return regressions, nil
}

// GetStats returns statistics about stored data
func (s *Storage) GetStats() (StorageStats, error) {
	var stats StorageStats

	// Count suites
	err := s.db.QueryRow("SELECT COUNT(*) FROM benchmark_suites").Scan(&stats.SuiteCount)
	if err != nil {
		return stats, err
	}

	// Count results
	err = s.db.QueryRow("SELECT COUNT(*) FROM benchmark_results").Scan(&stats.ResultCount)
	if err != nil {
		return stats, err
	}

	// Count regressions
	err = s.db.QueryRow("SELECT COUNT(*) FROM benchmark_regressions").Scan(&stats.RegressionCount)
	if err != nil {
		return stats, err
	}

	// Get date range
	var firstDate, lastDate sql.NullTime
	s.db.QueryRow("SELECT MIN(created_at), MAX(created_at) FROM benchmark_suites").Scan(&firstDate, &lastDate)
	if firstDate.Valid {
		stats.FirstRun = firstDate.Time
	}
	if lastDate.Valid {
		stats.LastRun = lastDate.Time
	}

	return stats, nil
}

// StorageStats provides storage statistics
type StorageStats struct {
	SuiteCount      int
	ResultCount     int
	RegressionCount int
	FirstRun        time.Time
	LastRun         time.Time
}

// Export exports all data to JSON
func (s *Storage) Export() (map[string]interface{}, error) {
	export := make(map[string]interface{})

	// Export suites
	suites, err := s.ListSuites(0)
	if err != nil {
		return nil, err
	}
	export["suites"] = suites

	// Export regressions
	regressions, err := s.LoadRegressions(time.Time{})
	if err != nil {
		return nil, err
	}
	export["regressions"] = regressions

	// Add stats
	stats, err := s.GetStats()
	if err != nil {
		return nil, err
	}
	export["stats"] = stats

	return export, nil
}

// Cleanup removes old data
func (s *Storage) Cleanup(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Delete old results
	_, err := s.db.Exec(
		"DELETE FROM benchmark_results WHERE timestamp < ?",
		cutoff,
	)
	if err != nil {
		return err
	}

	// Delete old regressions
	_, err = s.db.Exec(
		"DELETE FROM benchmark_regressions WHERE detected_at < ?",
		cutoff,
	)
	if err != nil {
		return err
	}

	// Delete old trends
	_, err = s.db.Exec(
		"DELETE FROM benchmark_trends WHERE detected_at < ?",
		cutoff,
	)
	if err != nil {
		return err
	}

	// Delete orphaned suites
	_, err = s.db.Exec(`
		DELETE FROM benchmark_suites
		WHERE id NOT IN (SELECT DISTINCT suite_id FROM benchmark_results)
	`)

	return err
}
