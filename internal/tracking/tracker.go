package tracking

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Tracker manages token tracking persistence.
type Tracker struct {
	db *sql.DB
}

// TimedExecution tracks execution time and token savings.
type TimedExecution struct {
	startTime time.Time
	command   string
	rtkCmd    string
	once      sync.Once
}

var (
	globalTracker *Tracker
	trackerMu     sync.Mutex
)

// Start begins a timed execution for tracking.
func Start() *TimedExecution {
	return &TimedExecution{
		startTime: time.Now(),
	}
}

// Track records the execution with token savings.
func (t *TimedExecution) Track(command, rtkCmd string, originalTokens, filteredTokens int) {
	t.once.Do(func() {
		execTime := time.Since(t.startTime)
		saved := originalTokens - filteredTokens
		if saved < 0 {
			saved = 0
		}

		// Get or create global tracker
		tracker := getGlobalTracker()
		if tracker == nil {
			return
		}

		cwd, _ := os.Getwd()
		tracker.Record(&CommandRecord{
			Command:        command,
			OriginalTokens: originalTokens,
			FilteredTokens: filteredTokens,
			SavedTokens:    saved,
			ProjectPath:    cwd,
			ExecTimeMs:     execTime.Milliseconds(),
			Timestamp:      time.Now(),
			ParseSuccess:   true,
		})
	})
}

// TrackPassthrough records a passthrough command (no filtering).
func (t *TimedExecution) TrackPassthrough(command, rtkCmd string) {
	t.once.Do(func() {
		execTime := time.Since(t.startTime)

		tracker := getGlobalTracker()
		if tracker == nil {
			return
		}

		cwd, _ := os.Getwd()
		tracker.RecordFallback(command, cwd, "", execTime.Milliseconds())
	})
}

// getGlobalTracker returns the global tracker instance.
func getGlobalTracker() *Tracker {
	trackerMu.Lock()
	defer trackerMu.Unlock()

	if globalTracker != nil {
		return globalTracker
	}

	// Initialize tracker
	dbPath := DatabasePath()
	if dbPath == "" {
		return nil
	}

	tracker, err := NewTracker(dbPath)
	if err != nil {
		return nil
	}

	globalTracker = tracker
	return globalTracker
}

// DatabasePath returns the default database path.
func DatabasePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "tokman", "tracking.db")
}

// NewTracker creates a new Tracker with the given database path.
func NewTracker(dbPath string) (*Tracker, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Run migrations
	for _, migration := range InitSchema() {
		if _, err := db.Exec(migration); err != nil {
			return nil, fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return &Tracker{db: db}, nil
}

// Close closes the database connection.
func (t *Tracker) Close() error {
	return t.db.Close()
}

// EstimateTokens provides a heuristic token count.
// Uses the formula: ceil(text.length / 4.0)
func EstimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// Record saves a command execution to the database.
func (t *Tracker) Record(record *CommandRecord) error {
	query := `
		INSERT INTO commands (
			command, original_output, filtered_output,
			original_tokens, filtered_tokens, saved_tokens,
			project_path, session_id, exec_time_ms, parse_success
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := t.db.Exec(query,
		record.Command,
		record.OriginalOutput,
		record.FilteredOutput,
		record.OriginalTokens,
		record.FilteredTokens,
		record.SavedTokens,
		record.ProjectPath,
		record.SessionID,
		record.ExecTimeMs,
		record.ParseSuccess,
	)
	if err != nil {
		return fmt.Errorf("failed to record command: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		record.ID = id
	}

	return nil
}

// GetSavings returns the total token savings for a project path.
// Uses GLOB matching to include child directories.
func (t *Tracker) GetSavings(projectPath string) (*SavingsSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_commands,
			COALESCE(SUM(saved_tokens), 0) as total_saved,
			COALESCE(SUM(original_tokens), 0) as total_original,
			COALESCE(SUM(filtered_tokens), 0) as total_filtered
		FROM commands
		WHERE project_path GLOB ? OR project_path = ?
	`

	pattern := projectPath + "/*"
	summary := &SavingsSummary{}

	err := t.db.QueryRow(query, pattern, projectPath).Scan(
		&summary.TotalCommands,
		&summary.TotalSaved,
		&summary.TotalOriginal,
		&summary.TotalFiltered,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get savings: %w", err)
	}

	if summary.TotalOriginal > 0 {
		summary.ReductionPct = float64(summary.TotalSaved) / float64(summary.TotalOriginal) * 100
	}

	return summary, nil
}

// GetCommandStats returns statistics grouped by command.
func (t *Tracker) GetCommandStats(projectPath string) ([]CommandStats, error) {
	query := `
		SELECT 
			command,
			COUNT(*) as execution_count,
			COALESCE(SUM(saved_tokens), 0) as total_saved,
			COALESCE(SUM(original_tokens), 0) as total_original
		FROM commands
		WHERE project_path GLOB ? OR project_path = ?
		GROUP BY command
		ORDER BY total_saved DESC
	`

	pattern := projectPath + "/*"
	rows, err := t.db.Query(query, pattern, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get command stats: %w", err)
	}
	defer rows.Close()

	var stats []CommandStats
	for rows.Next() {
		var s CommandStats
		if err := rows.Scan(&s.Command, &s.ExecutionCount, &s.TotalSaved, &s.TotalOriginal); err != nil {
			return nil, err
		}
		if s.TotalOriginal > 0 {
			s.ReductionPct = float64(s.TotalSaved) / float64(s.TotalOriginal) * 100
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetRecentCommands returns the most recent command executions.
func (t *Tracker) GetRecentCommands(projectPath string, limit int) ([]CommandRecord, error) {
	query := `
		SELECT id, command, original_tokens, filtered_tokens, saved_tokens,
		       project_path, session_id, exec_time_ms, timestamp, parse_success
		FROM commands
		WHERE project_path GLOB ? OR project_path = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	pattern := projectPath + "/*"
	rows, err := t.db.Query(query, pattern, projectPath, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commands: %w", err)
	}
	defer rows.Close()

	var records []CommandRecord
	for rows.Next() {
		var r CommandRecord
		var parseSuccess int
		if err := rows.Scan(
			&r.ID, &r.Command, &r.OriginalTokens, &r.FilteredTokens, &r.SavedTokens,
			&r.ProjectPath, &r.SessionID, &r.ExecTimeMs, &r.Timestamp, &parseSuccess,
		); err != nil {
			return nil, err
		}
		r.ParseSuccess = parseSuccess == 1
		records = append(records, r)
	}

	return records, nil
}

// GetDailySavings returns token savings grouped by day.
func (t *Tracker) GetDailySavings(projectPath string, days int) ([]struct {
	Date     string
	Saved    int
	Original int
	Commands int
}, error) {
	query := `
		SELECT 
			DATE(timestamp) as date,
			COALESCE(SUM(saved_tokens), 0) as saved,
			COALESCE(SUM(original_tokens), 0) as original,
			COUNT(*) as commands
		FROM commands
		WHERE (project_path GLOB ? OR project_path = ?)
		  AND timestamp >= DATE('now', ?)
		GROUP BY DATE(timestamp)
		ORDER BY date DESC
	`

	pattern := projectPath + "/*"
	daysStr := fmt.Sprintf("-%d days", days)
	rows, err := t.db.Query(query, pattern, projectPath, daysStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily savings: %w", err)
	}
	defer rows.Close()

	var results []struct {
		Date     string
		Saved    int
		Original int
		Commands int
	}
	for rows.Next() {
		var r struct {
			Date     string
			Saved    int
			Original int
			Commands int
		}
		if err := rows.Scan(&r.Date, &r.Saved, &r.Original, &r.Commands); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// RecordFallback records a command that wasn't recognized (parse failure).
func (t *Tracker) RecordFallback(command string, projectPath string, output string, execTimeMs int64) error {
	originalTokens := EstimateTokens(output)
	return t.Record(&CommandRecord{
		Command:        command,
		OriginalOutput: output,
		FilteredOutput: output,
		OriginalTokens: originalTokens,
		FilteredTokens: originalTokens,
		SavedTokens:    0,
		ProjectPath:    projectPath,
		ExecTimeMs:     execTimeMs,
		Timestamp:      time.Now(),
		ParseSuccess:   false,
	})
}
