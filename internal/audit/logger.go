// Package audit provides audit logging capabilities
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// AuditLogger provides audit logging
type AuditLogger struct {
	mu      sync.Mutex
	entries []AuditEntry
	output  *os.File
	maxSize int
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	User      string                 `json:"user"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Status    string                 `json:"status"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(outputPath string, maxSize int) (*AuditLogger, error) {
	var output *os.File
	var err error

	if outputPath == "" {
		output = os.Stdout
	} else {
		output, err = os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log: %w", err)
		}
	}

	if maxSize <= 0 {
		maxSize = 10000
	}

	return &AuditLogger{
		entries: make([]AuditEntry, 0, maxSize),
		output:  output,
		maxSize: maxSize,
	}, nil
}

// Log logs an audit entry
func (al *AuditLogger) Log(entry AuditEntry) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if entry.ID == "" {
		entry.ID = generateID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	al.entries = append(al.entries, entry)

	// Write to output
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = al.output.Write(append(data, '\n'))
	return err
}

// GetEntries returns audit entries
func (al *AuditLogger) GetEntries(filter AuditFilter) []AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	result := make([]AuditEntry, 0)

	for _, entry := range al.entries {
		if filter.Matches(entry) {
			result = append(result, entry)
		}
	}

	return result
}

// AuditFilter filters audit entries
type AuditFilter struct {
	User     string
	Action   string
	Resource string
	FromDate time.Time
	ToDate   time.Time
	Status   string
}

// Matches checks if an entry matches the filter
func (f AuditFilter) Matches(entry AuditEntry) bool {
	if f.User != "" && entry.User != f.User {
		return false
	}
	if f.Action != "" && entry.Action != f.Action {
		return false
	}
	if f.Resource != "" && entry.Resource != f.Resource {
		return false
	}
	if !f.FromDate.IsZero() && entry.Timestamp.Before(f.FromDate) {
		return false
	}
	if !f.ToDate.IsZero() && entry.Timestamp.After(f.ToDate) {
		return false
	}
	if f.Status != "" && entry.Status != f.Status {
		return false
	}
	return true
}

// Export exports audit entries
func (al *AuditLogger) Export(format string) ([]byte, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	switch format {
	case "json":
		return json.MarshalIndent(al.entries, "", "  ")
	case "csv":
		return al.exportCSV()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (al *AuditLogger) exportCSV() ([]byte, error) {
	output := "Timestamp,User,Action,Resource,Status,IP Address\n"
	for _, entry := range al.entries {
		output += fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.User,
			entry.Action,
			entry.Resource,
			entry.Status,
			entry.IPAddress,
		)
	}
	return []byte(output), nil
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	if al.output != os.Stdout {
		return al.output.Close()
	}
	return nil
}

func generateID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}
