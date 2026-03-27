package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UserFeedback is the user's subjective assessment of a compression result.
// Task #129: Compression effectiveness feedback collection.
type UserFeedback int

const (
	FeedbackPositive UserFeedback = 1   // compression was helpful
	FeedbackNeutral  UserFeedback = 0   // no strong opinion
	FeedbackNegative UserFeedback = -1  // compression was harmful
)

// FeedbackRecord pairs a compression result with user feedback.
type FeedbackRecord struct {
	Timestamp      time.Time      `json:"ts"`
	FilterChain    []string       `json:"filter_chain"`
	Mode           string         `json:"mode"`
	OriginalTokens int            `json:"orig_tokens"`
	FinalTokens    int            `json:"final_tokens"`
	ReductionPct   float64        `json:"reduction_pct"`
	Signal         UserFeedback `json:"signal"`
	Note           string         `json:"note,omitempty"`
	// ContentHash is a short fingerprint of the original (for dedup tracking).
	ContentHash string `json:"content_hash,omitempty"`
}

// FeedbackCollector records user feedback on compression quality.
type FeedbackCollector struct {
	mu      sync.Mutex
	records []FeedbackRecord
	path    string // optional file path for persistence
}

// NewFeedbackCollector creates an in-memory feedback collector.
// If path is non-empty, feedback is appended to the file as JSONL on Save().
func NewFeedbackCollector(path string) *FeedbackCollector {
	return &FeedbackCollector{path: path}
}

// DefaultFeedbackCollector returns a collector backed by the XDG-standard path.
func DefaultFeedbackCollector() *FeedbackCollector {
	home, _ := os.UserHomeDir()
	return NewFeedbackCollector(
		filepath.Join(home, ".local", "share", "tokman", "feedback.jsonl"),
	)
}

// Record adds a feedback record.
func (c *FeedbackCollector) Record(rec FeedbackRecord) {
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	c.mu.Lock()
	c.records = append(c.records, rec)
	c.mu.Unlock()
}

// RecordResult is a convenience helper to record feedback from a compression result.
func (c *FeedbackCollector) RecordResult(filters []string, mode Mode, origTokens, finalTokens int, signal UserFeedback, note string) {
	var pct float64
	if origTokens > 0 {
		pct = float64(origTokens-finalTokens) / float64(origTokens) * 100
	}
	names := make([]string, len(filters))
	copy(names, filters)
	c.Record(FeedbackRecord{
		FilterChain:    names,
		Mode:           string(mode),
		OriginalTokens: origTokens,
		FinalTokens:    finalTokens,
		ReductionPct:   pct,
		Signal:         signal,
		Note:           note,
	})
}

// Save appends all in-memory records to the JSONL file and clears the buffer.
func (c *FeedbackCollector) Save() error {
	c.mu.Lock()
	records := c.records
	c.records = nil
	c.mu.Unlock()

	if len(records) == 0 || c.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0700); err != nil {
		return fmt.Errorf("feedback: mkdir: %w", err)
	}
	f, err := os.OpenFile(c.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("feedback: open: %w", err)
	}
	defer f.Close()

	for _, rec := range records {
		data, err := json.Marshal(rec)
		if err != nil {
			continue
		}
		if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
			return fmt.Errorf("feedback: write: %w", err)
		}
	}
	return nil
}

// Summary returns aggregate statistics across all recorded feedback.
func (c *FeedbackCollector) Summary() FeedbackSummary {
	c.mu.Lock()
	records := make([]FeedbackRecord, len(c.records))
	copy(records, c.records)
	c.mu.Unlock()

	var s FeedbackSummary
	s.Total = len(records)
	for _, r := range records {
		switch r.Signal {
		case FeedbackPositive:
			s.Positive++
		case FeedbackNegative:
			s.Negative++
		default:
			s.Neutral++
		}
		s.AvgReductionPct += r.ReductionPct
	}
	if s.Total > 0 {
		s.AvgReductionPct /= float64(s.Total)
	}
	return s
}

// FeedbackSummary aggregates feedback statistics.
type FeedbackSummary struct {
	Total           int
	Positive        int
	Neutral         int
	Negative        int
	AvgReductionPct float64
}

func (s FeedbackSummary) String() string {
	return fmt.Sprintf("Feedback: %d total (+%d/~%d/-%d), avg reduction %.1f%%",
		s.Total, s.Positive, s.Neutral, s.Negative, s.AvgReductionPct)
}
