package tracking

// SchemaVersion is the current database schema version.
const SchemaVersion = 1

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

// Migrations contains all migration statements in order.
var Migrations = []string{
	CreateCommandsTable,
	CreateSummaryView,
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
