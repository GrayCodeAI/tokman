package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEvent records a single compression pipeline event.
type AuditEvent struct {
	Timestamp      time.Time `json:"ts"`
	EventType      string    `json:"type"`   // "compress", "skip", "error", "budget_exceeded"
	FilterName     string    `json:"filter,omitempty"`
	Mode           string    `json:"mode,omitempty"`
	OriginalTokens int       `json:"orig_tokens,omitempty"`
	FinalTokens    int       `json:"final_tokens,omitempty"`
	Saved          int       `json:"saved,omitempty"`
	ReductionPct   float64   `json:"reduction_pct,omitempty"`
	Note           string    `json:"note,omitempty"`
}

// AuditLog writes compression events to a JSON-lines audit file.
// Task #99: Compression audit log.
type AuditLog struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

// NewAuditLog creates (or appends to) an audit log at the given path.
func NewAuditLog(path string) (*AuditLog, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("audit log: mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("audit log: open: %w", err)
	}
	return &AuditLog{path: path, f: f}, nil
}

// Close flushes and closes the audit log file.
func (a *AuditLog) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.f != nil {
		err := a.f.Close()
		a.f = nil
		return err
	}
	return nil
}

// Record appends an event to the audit log.
func (a *AuditLog) Record(event AuditEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.f == nil {
		return fmt.Errorf("audit log is closed")
	}

	_, err = fmt.Fprintf(a.f, "%s\n", data)
	return err
}

// RecordCompression is a convenience helper for recording a compression event.
func (a *AuditLog) RecordCompression(filterName string, mode Mode, origTokens, finalTokens int) error {
	saved := origTokens - finalTokens
	var reduction float64
	if origTokens > 0 {
		reduction = float64(saved) / float64(origTokens) * 100
	}
	return a.Record(AuditEvent{
		EventType:      "compress",
		FilterName:     filterName,
		Mode:           string(mode),
		OriginalTokens: origTokens,
		FinalTokens:    finalTokens,
		Saved:          saved,
		ReductionPct:   reduction,
	})
}

// RecordSkip records that a filter was skipped for a given reason.
func (a *AuditLog) RecordSkip(filterName, reason string) error {
	return a.Record(AuditEvent{
		EventType:  "skip",
		FilterName: filterName,
		Note:       reason,
	})
}

// RecordError records a compression error.
func (a *AuditLog) RecordError(filterName string, err error) error {
	return a.Record(AuditEvent{
		EventType:  "error",
		FilterName: filterName,
		Note:       err.Error(),
	})
}

// NullAuditLog is a no-op audit log for use when auditing is disabled.
type NullAuditLog struct{}

func (n *NullAuditLog) Record(AuditEvent) error        { return nil }
func (n *NullAuditLog) RecordCompression(string, Mode, int, int) error { return nil }
func (n *NullAuditLog) RecordSkip(string, string) error { return nil }
func (n *NullAuditLog) RecordError(string, error) error { return nil }
func (n *NullAuditLog) Close() error                    { return nil }
