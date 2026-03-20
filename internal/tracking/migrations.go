package tracking

// SchemaVersion is the current database schema version.
const SchemaVersion = 2

// CreateCommandsTable creates the main commands table.
const CreateCommandsTable = `
CREATE TABLE IF NOT EXISTS commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command TEXT NOT NULL,
    original_output TEXT,
    filtered_output TEXT,
    original_tokens INTEGER NOT NULL,
    filtered_tokens INTEGER NOT NULL,
    saved_tokens INTEGER NOT NULL,
    project_path TEXT NOT NULL,
    session_id TEXT,
    exec_time_ms INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    parse_success BOOLEAN DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_commands_timestamp ON commands(timestamp);
CREATE INDEX IF NOT EXISTS idx_commands_project ON commands(project_path);
CREATE INDEX IF NOT EXISTS idx_commands_session ON commands(session_id);
CREATE INDEX IF NOT EXISTS idx_commands_command ON commands(command);
`

// AddCompositeIndexes adds composite indexes for common query patterns.
// T181: Composite indexes on (project_path, timestamp) and (command, timestamp).
const AddCompositeIndexes = `
CREATE INDEX IF NOT EXISTS idx_commands_project_ts ON commands(project_path, timestamp);
CREATE INDEX IF NOT EXISTS idx_commands_command_ts ON commands(command, timestamp);
CREATE INDEX IF NOT EXISTS idx_commands_saved ON commands(saved_tokens DESC);
`

// CreateSummaryView creates a view for aggregated statistics.
const CreateSummaryView = `
CREATE VIEW IF NOT EXISTS command_summary AS
SELECT 
    project_path,
    command,
    COUNT(*) as execution_count,
    SUM(saved_tokens) as total_saved,
    SUM(original_tokens) as total_original,
    ROUND(100.0 * SUM(saved_tokens) / NULLIF(SUM(original_tokens), 0), 2) as reduction_percent
FROM commands
GROUP BY project_path, command;
`

// CreateParseFailuresTable creates a table for tracking parse failures.
const CreateParseFailuresTable = `
CREATE TABLE IF NOT EXISTS parse_failures (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    raw_command TEXT NOT NULL,
    error_message TEXT NOT NULL,
    fallback_succeeded BOOLEAN DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_parse_failures_timestamp ON parse_failures(timestamp);
`

// CreateLayerStatsTable tracks per-layer savings for detailed analysis.
// T184: Per-layer savings tracking.
const CreateLayerStatsTable = `
CREATE TABLE IF NOT EXISTS layer_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command_id INTEGER NOT NULL,
    layer_name TEXT NOT NULL,
    tokens_saved INTEGER NOT NULL DEFAULT 0,
    duration_us INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (command_id) REFERENCES commands(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_layer_stats_command ON layer_stats(command_id);
CREATE INDEX IF NOT EXISTS idx_layer_stats_name ON layer_stats(layer_name);
`

// Migrations contains all migration statements in order.
var Migrations = []string{
	CreateCommandsTable,
	CreateSummaryView,
	CreateParseFailuresTable,
	AddCompositeIndexes,
	CreateLayerStatsTable,
}

// MigrationHistory tracks applied migrations.
const CreateMigrationTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// InitSchema initializes all database tables and migrations.
func InitSchema() []string {
	return append([]string{CreateMigrationTable}, Migrations...)
}
