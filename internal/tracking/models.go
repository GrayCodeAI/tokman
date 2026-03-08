package tracking

import "time"

// CommandRecord represents a single command execution record.
type CommandRecord struct {
	ID             int64     `json:"id"`
	Command        string    `json:"command"`
	OriginalOutput string    `json:"original_output,omitempty"`
	FilteredOutput string    `json:"filtered_output,omitempty"`
	OriginalTokens int       `json:"original_tokens"`
	FilteredTokens int       `json:"filtered_tokens"`
	SavedTokens    int       `json:"saved_tokens"`
	ProjectPath    string    `json:"project_path"`
	SessionID      string    `json:"session_id,omitempty"`
	ExecTimeMs     int64     `json:"exec_time_ms"`
	Timestamp      time.Time `json:"timestamp"`
	ParseSuccess   bool      `json:"parse_success"`
}

// SavingsSummary represents aggregated token savings.
type SavingsSummary struct {
	TotalCommands  int   `json:"total_commands"`
	TotalSaved     int   `json:"total_saved"`
	TotalOriginal  int   `json:"total_original"`
	TotalFiltered  int   `json:"total_filtered"`
	ReductionPct   float64 `json:"reduction_percent"`
}

// CommandStats represents statistics for a specific command type.
type CommandStats struct {
	Command        string  `json:"command"`
	ExecutionCount int     `json:"execution_count"`
	TotalSaved     int     `json:"total_saved"`
	TotalOriginal  int     `json:"total_original"`
	ReductionPct   float64 `json:"reduction_percent"`
}

// SessionInfo represents information about a shell session.
type SessionInfo struct {
	SessionID   string    `json:"session_id"`
	StartedAt   time.Time `json:"started_at"`
	ProjectPath string    `json:"project_path"`
}

// ReportFilter represents filters for generating reports.
type ReportFilter struct {
	ProjectPath string     `json:"project_path,omitempty"`
	SessionID   string     `json:"session_id,omitempty"`
	Command     string     `json:"command,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
}
