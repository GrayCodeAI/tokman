package tracking

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/core"
)

// HistoryRetentionDays is the number of days to retain tracking data.
// Records older than this are automatically cleaned up on each write.
const HistoryRetentionDays = 90

// TrackerInterface defines the contract for command tracking.
// Implementations can use SQLite, in-memory stores, or mocks for testing.
type TrackerInterface interface {
	Record(record *CommandRecord) error
	GetSavings(projectPath string) (*SavingsSummary, error)
	GetRecentCommands(projectPath string, limit int) ([]CommandRecord, error)
	Query(query string, args ...any) (*sql.Rows, error)
	Close() error
}

// Tracker manages token tracking persistence.
type Tracker struct {
	db            *sql.DB
	lastCleanupMs int64         // atomic: unix timestamp of last cleanup
	cleanupCh     chan struct{} // non-blocking cleanup trigger
	cleanupWg     sync.WaitGroup // waits for cleanup goroutine to finish
}

// TimedExecution tracks execution time and token savings.
type TimedExecution struct {
	startTime time.Time
	command   string
	tokmanCmd string
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
func (t *TimedExecution) Track(command, tokmanCmd string, originalTokens, filteredTokens int) {
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
func (t *TimedExecution) TrackPassthrough(command, tokmanCmd string) {
	t.once.Do(func() {
		execTime := time.Since(t.startTime)

		tracker := getGlobalTracker()
		if tracker == nil {
			return
		}

		cwd, _ := os.Getwd()
		tracker.Record(&CommandRecord{
			Command:      command,
			ProjectPath:  cwd,
			ExecTimeMs:   execTime.Milliseconds(),
			ParseSuccess: false,
		})
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

// GetGlobalTracker returns the global tracker instance (exported for external use).
func GetGlobalTracker() *Tracker {
	return getGlobalTracker()
}

// DatabasePath returns the default database path.
// Delegates to config.DatabasePath for consistent XDG compliance.
func DatabasePath() string {
	return config.DatabasePath()
}

// NewTracker creates a new Tracker with the given database path.
func NewTracker(dbPath string) (*Tracker, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to retry on locked database
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	for _, migration := range InitSchema() {
		if _, err := db.Exec(migration); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run migration: %w", err)
		}
	}

	t := &Tracker{
		db:        db,
		cleanupCh: make(chan struct{}, 1),
	}
	t.cleanupWg.Add(1)
	go t.cleanupWorker()
	return t, nil
}

// Close closes the database connection.
func (t *Tracker) Close() error {
	close(t.cleanupCh)
	t.cleanupWg.Wait() // Wait for cleanup goroutine to finish before closing DB
	return t.db.Close()
}

// cleanupWorker processes cleanup triggers from the channel.
func (t *Tracker) cleanupWorker() {
	defer t.cleanupWg.Done()
	for range t.cleanupCh {
		t.cleanupOld()
	}
}

// Query executes a raw SQL query and returns the rows.
// This is exposed for custom aggregations in the economics package.
func (t *Tracker) Query(query string, args ...any) (*sql.Rows, error) {
	return t.db.Query(query, args...)
}

// EstimateTokens provides a heuristic token count.
// Delegates to core.EstimateTokens for single source of truth (T22).
func EstimateTokens(text string) int {
	return core.EstimateTokens(text)
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

	// Run cleanup after recording (throttled - at most once per minute)
	select {
	case t.cleanupCh <- struct{}{}:
	default:
	}

	return nil
}

// LayerStatRecord holds per-layer statistics for database recording.
type LayerStatRecord struct {
	LayerName   string
	TokensSaved int
	DurationUs  int64
}

// RecordLayerStats saves per-layer statistics for a command.
// T184: Per-layer savings tracking.
func (t *Tracker) RecordLayerStats(commandID int64, stats []LayerStatRecord) error {
	if len(stats) == 0 {
		return nil
	}

	tx, err := t.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		"INSERT INTO layer_stats (command_id, layer_name, tokens_saved, duration_us) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, s := range stats {
		if _, err := stmt.Exec(commandID, s.LayerName, s.TokensSaved, s.DurationUs); err != nil {
			return fmt.Errorf("failed to insert layer stat: %w", err)
		}
	}

	return tx.Commit()
}

// GetLayerStats returns per-layer statistics for a command.
func (t *Tracker) GetLayerStats(commandID int64) ([]LayerStatRecord, error) {
	rows, err := t.db.Query(
		"SELECT layer_name, tokens_saved, duration_us FROM layer_stats WHERE command_id = ? ORDER BY id",
		commandID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query layer stats: %w", err)
	}
	defer rows.Close()

	var stats []LayerStatRecord
	for rows.Next() {
		var s LayerStatRecord
		if err := rows.Scan(&s.LayerName, &s.TokensSaved, &s.DurationUs); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

// GetTopLayers returns the most effective compression layers.
func (t *Tracker) GetTopLayers(limit int) ([]struct {
	LayerName  string
	TotalSaved int64
	AvgSaved   float64
	CallCount  int64
}, error) {
	query := `
		SELECT layer_name, SUM(tokens_saved) as total_saved,
		       AVG(tokens_saved) as avg_saved, COUNT(*) as call_count
		FROM layer_stats
		GROUP BY layer_name
		ORDER BY total_saved DESC
		LIMIT ?
	`
	rows, err := t.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		LayerName  string
		TotalSaved int64
		AvgSaved   float64
		CallCount  int64
	}
	for rows.Next() {
		var r struct {
			LayerName  string
			TotalSaved int64
			AvgSaved   float64
			CallCount  int64
		}
		if err := rows.Scan(&r.LayerName, &r.TotalSaved, &r.AvgSaved, &r.CallCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// cleanupOld removes records older than HistoryRetentionDays.
// This is called automatically after each Record operation.
func (t *Tracker) cleanupOld() {
	// Throttle: at most one cleanup per 60 seconds
	now := time.Now().UnixMilli()
	last := atomic.LoadInt64(&t.lastCleanupMs)
	if now-last < 60000 {
		return
	}
	if !atomic.CompareAndSwapInt64(&t.lastCleanupMs, last, now) {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -HistoryRetentionDays)
	if _, err := t.db.Exec(
		"DELETE FROM commands WHERE timestamp < ?",
		cutoff.Format(time.RFC3339),
	); err != nil {
		log.Printf("[tokman] tracking cleanup failed: %v", err)
	}
}

// CleanupOld manually triggers cleanup of old records.
// Returns the number of records deleted.
func (t *Tracker) CleanupOld() (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -HistoryRetentionDays)
	result, err := t.db.Exec(
		"DELETE FROM commands WHERE timestamp < ?",
		cutoff.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old records: %w", err)
	}
	return result.RowsAffected()
}

// CleanupWithRetention removes records older than specified days.
// T183: Configurable data retention policy.
func (t *Tracker) CleanupWithRetention(days int) (int64, error) {
	if days <= 0 {
		days = HistoryRetentionDays
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	tx, err := t.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete layer stats for old commands
	if _, err := tx.Exec(
		"DELETE FROM layer_stats WHERE command_id IN (SELECT id FROM commands WHERE timestamp < ?)",
		cutoff.Format(time.RFC3339),
	); err != nil {
		log.Printf("[tokman] layer_stats cleanup failed: %v", err)
	}

	// Delete old commands
	result, err := tx.Exec(
		"DELETE FROM commands WHERE timestamp < ?",
		cutoff.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup: %w", err)
	}

	// Delete old parse failures
	if _, err := tx.Exec(
		"DELETE FROM parse_failures WHERE timestamp < ?",
		cutoff.Format(time.RFC3339),
	); err != nil {
		log.Printf("[tokman] parse_failures cleanup failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit cleanup: %w", err)
	}

	return result.RowsAffected()
}

// DatabaseSize returns the size of the tracking database in bytes.
func (t *Tracker) DatabaseSize() (int64, error) {
	var pageCount, pageSize int64
	err := t.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return 0, err
	}
	err = t.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return 0, err
	}
	return pageCount * pageSize, nil
}

// Vacuum reclaims unused space in the database.
func (t *Tracker) Vacuum() error {
	_, err := t.db.Exec("VACUUM")
	return err
}

// GetSavings returns the total token savings for a project path.
// Uses GLOB matching for case-sensitive path comparison.
// When projectPath is empty, returns all records without filtering.
func (t *Tracker) GetSavings(projectPath string) (*SavingsSummary, error) {
	var query string
	var args []any

	if projectPath == "" {
		query = `
			SELECT 
				COUNT(*) as total_commands,
				COALESCE(SUM(saved_tokens), 0) as total_saved,
				COALESCE(SUM(original_tokens), 0) as total_original,
				COALESCE(SUM(filtered_tokens), 0) as total_filtered
			FROM commands
		`
		args = nil
	} else {
		query = `
			SELECT 
				COUNT(*) as total_commands,
				COALESCE(SUM(saved_tokens), 0) as total_saved,
				COALESCE(SUM(original_tokens), 0) as total_original,
				COALESCE(SUM(filtered_tokens), 0) as total_filtered
			FROM commands
			WHERE project_path GLOB ? OR project_path = ?
		`
		pattern := projectPath + "/%"
		args = []any{pattern, projectPath}
	}

	summary := &SavingsSummary{}

	err := t.db.QueryRow(query, args...).Scan(
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

// CountCommandsSince returns the count of commands executed since the given time.
func (t *Tracker) CountCommandsSince(since time.Time) (int64, error) {
	var count int64
	err := t.db.QueryRow(
		"SELECT COUNT(*) FROM commands WHERE timestamp >= ?",
		since.Format(time.RFC3339),
	).Scan(&count)
	return count, err
}

// ParseFailureRecord represents a single parse failure event.
type ParseFailureRecord struct {
	ID                int64     `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	RawCommand        string    `json:"raw_command"`
	ErrorMessage      string    `json:"error_message"`
	FallbackSucceeded bool      `json:"fallback_succeeded"`
}

// ParseFailureSummary represents aggregated parse failure analytics.
type ParseFailureSummary struct {
	Total          int64                 `json:"total"`
	RecoveryRate   float64               `json:"recovery_rate"`
	TopCommands    []CommandFailureCount `json:"top_commands"`
	RecentFailures []ParseFailureRecord  `json:"recent_failures"`
}

// CommandFailureCount represents a command and its failure count.
type CommandFailureCount struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
}

// TopCommands returns the top N commands by execution count.
func (t *Tracker) TopCommands(limit int) ([]string, error) {
	rows, err := t.db.Query(
		`SELECT command FROM commands
		 GROUP BY command
		 ORDER BY COUNT(*) DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []string
	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err != nil {
			continue
		}
		commands = append(commands, cmd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return commands, nil
}

// OverallSavingsPct returns the overall savings percentage across all commands.
func (t *Tracker) OverallSavingsPct() (float64, error) {
	var saved, original int64
	err := t.db.QueryRow(
		"SELECT COALESCE(SUM(saved_tokens), 0), COALESCE(SUM(original_tokens), 0) FROM commands",
	).Scan(&saved, &original)
	if err != nil {
		return 0, err
	}
	if original == 0 {
		return 0, nil
	}
	return float64(saved) / float64(original) * 100, nil
}

// TokensSaved24h returns tokens saved in the last 24 hours.
func (t *Tracker) TokensSaved24h() (int64, error) {
	var saved int64
	err := t.db.QueryRow(
		"SELECT COALESCE(SUM(saved_tokens), 0) FROM commands WHERE timestamp >= ?",
		time.Now().Add(-24*time.Hour).Format(time.RFC3339),
	).Scan(&saved)
	return saved, err
}

// TokensSavedTotal returns total tokens saved across all time.
func (t *Tracker) TokensSavedTotal() (int64, error) {
	var saved int64
	err := t.db.QueryRow(
		"SELECT COALESCE(SUM(saved_tokens), 0) FROM commands",
	).Scan(&saved)
	return saved, err
}

// GetCommandStats returns statistics grouped by command.
// When projectPath is empty, returns all commands without filtering.
func (t *Tracker) GetCommandStats(projectPath string) ([]CommandStats, error) {
	var query string
	var rows *sql.Rows
	var err error

	if projectPath == "" {
		query = `
			SELECT 
				command,
				COUNT(*) as execution_count,
				COALESCE(SUM(saved_tokens), 0) as total_saved,
				COALESCE(SUM(original_tokens), 0) as total_original
			FROM commands
			GROUP BY command
			ORDER BY total_saved DESC
		`
		rows, err = t.db.Query(query)
	} else {
		query = `
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
		pattern := projectPath + "/%"
		rows, err = t.db.Query(query, pattern, projectPath)
	}
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetRecentCommands returns the most recent command executions.
// When projectPath is empty, returns all recent commands without filtering.
func (t *Tracker) GetRecentCommands(projectPath string, limit int) ([]CommandRecord, error) {
	var query string
	var rows *sql.Rows
	var err error

	if projectPath == "" {
		query = `
			SELECT id, command, original_tokens, filtered_tokens, saved_tokens,
			       project_path, session_id, exec_time_ms, timestamp, parse_success
			FROM commands
			ORDER BY timestamp DESC
			LIMIT ?
		`
		rows, err = t.db.Query(query, limit)
	} else {
		query = `
			SELECT id, command, original_tokens, filtered_tokens, saved_tokens,
			       project_path, session_id, exec_time_ms, timestamp, parse_success
			FROM commands
			WHERE project_path GLOB ? OR project_path = ?
			ORDER BY timestamp DESC
			LIMIT ?
		`
		pattern := projectPath + "/%"
		rows, err = t.db.Query(query, pattern, projectPath, limit)
	}
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
	if err := rows.Err(); err != nil {
		return nil, err
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
		WHERE (project_path LIKE ? OR project_path = ?)
		  AND timestamp >= DATE('now', ?)
		GROUP BY DATE(timestamp)
		ORDER BY date DESC
	`

	pattern := projectPath + "/%"
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// RecordParseFailure records a parse failure event.
func (t *Tracker) RecordParseFailure(rawCommand string, errorMessage string, fallbackSucceeded bool) error {
	_, err := t.db.Exec(
		`INSERT INTO parse_failures (timestamp, raw_command, error_message, fallback_succeeded)
		 VALUES (?, ?, ?, ?)`,
		time.Now().Format(time.RFC3339),
		rawCommand,
		errorMessage,
		fallbackSucceeded,
	)
	if err != nil {
		return fmt.Errorf("failed to record parse failure: %w", err)
	}

	// Cleanup old records (throttled)
	select {
	case t.cleanupCh <- struct{}{}:
	default:
	}

	return nil
}

// GetParseFailureSummary returns aggregated parse failure analytics.
func (t *Tracker) GetParseFailureSummary() (*ParseFailureSummary, error) {
	summary := &ParseFailureSummary{}

	// Get total count
	err := t.db.QueryRow("SELECT COUNT(*) FROM parse_failures").Scan(&summary.Total)
	if err != nil {
		return nil, fmt.Errorf("failed to get parse failure count: %w", err)
	}

	if summary.Total == 0 {
		return summary, nil
	}

	// Get recovery rate
	var succeeded int64
	err = t.db.QueryRow(
		"SELECT COUNT(*) FROM parse_failures WHERE fallback_succeeded = 1",
	).Scan(&succeeded)
	if err == nil {
		summary.RecoveryRate = float64(succeeded) / float64(summary.Total) * 100
	}

	// Get top 10 failing commands
	topRows, err := t.db.Query(
		`SELECT raw_command, COUNT(*) as cnt
		 FROM parse_failures
		 GROUP BY raw_command
		 ORDER BY cnt DESC
		 LIMIT 10`,
	)
	if err != nil {
		return summary, nil // return partial results on query failure
	}
	defer topRows.Close()
	for topRows.Next() {
		var cfc CommandFailureCount
		if err := topRows.Scan(&cfc.Command, &cfc.Count); err == nil {
			summary.TopCommands = append(summary.TopCommands, cfc)
		}
	}

	// Get recent 10 failures
	recentRows, err := t.db.Query(
		`SELECT id, timestamp, raw_command, error_message, fallback_succeeded
		 FROM parse_failures
		 ORDER BY timestamp DESC
		 LIMIT 10`,
	)
	if err != nil {
		return summary, nil // return partial results on query failure
	}
	defer recentRows.Close()
	for recentRows.Next() {
		var pfr ParseFailureRecord
		var ts string
		var fb int
		if err := recentRows.Scan(&pfr.ID, &ts, &pfr.RawCommand, &pfr.ErrorMessage, &fb); err == nil {
			parsed, parseErr := time.Parse(time.RFC3339, ts)
			if parseErr != nil {
				log.Printf("[tokman] failed to parse timestamp %q: %v", ts, parseErr)
			}
			pfr.Timestamp = parsed
			pfr.FallbackSucceeded = fb == 1
			summary.RecentFailures = append(summary.RecentFailures, pfr)
		}
	}

	return summary, nil
}
