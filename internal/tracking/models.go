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
	// AI Agent attribution fields
	AgentName   string `json:"agent_name,omitempty"`   // e.g., "Claude Code", "OpenCode", "Cursor"
	ModelName   string `json:"model_name,omitempty"`   // e.g., "claude-3-opus", "gpt-4", "gemini-pro"
	Provider    string `json:"provider,omitempty"`     // e.g., "Anthropic", "OpenAI", "Google"
	ModelFamily string `json:"model_family,omitempty"` // e.g., "claude", "gpt", "gemini"
	// Smart context read metadata
	ContextKind         string `json:"context_kind,omitempty"`          // e.g., "read", "delta", "mcp"
	ContextMode         string `json:"context_mode,omitempty"`          // requested mode: auto, graph, delta, ...
	ContextResolvedMode string `json:"context_resolved_mode,omitempty"` // effective mode after auto-resolution
	ContextTarget       string `json:"context_target,omitempty"`        // file path or target identifier
	ContextRelatedFiles int    `json:"context_related_files,omitempty"` // number of related files included
	ContextBundle       bool   `json:"context_bundle,omitempty"`        // whether multiple files were delivered together
}

// SavingsSummary represents aggregated token savings.
type SavingsSummary struct {
	TotalCommands int     `json:"total_commands"`
	TotalSaved    int     `json:"total_saved"`
	TotalOriginal int     `json:"total_original"`
	TotalFiltered int     `json:"total_filtered"`
	ReductionPct  float64 `json:"reduction_percent"`
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
