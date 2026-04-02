// Package commandhistory provides command history management
package commandhistory

import (
	"fmt"
	"sync"
	"time"
)

// HistoryEntry represents a command history entry
type HistoryEntry struct {
	ID        string
	Command   string
	Args      []string
	Output    string
	ExitCode  int
	Duration  time.Duration
	Timestamp time.Time
	User      string
	Tags      []string
}

// HistoryManager manages command history
type HistoryManager struct {
	entries []HistoryEntry
	maxSize int
	mu      sync.RWMutex
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(maxSize int) *HistoryManager {
	if maxSize <= 0 {
		maxSize = 1000
	}

	return &HistoryManager{
		entries: make([]HistoryEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// AddEntry adds a command to history
func (hm *HistoryManager) AddEntry(entry HistoryEntry) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("hist-%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	hm.entries = append(hm.entries, entry)

	// Trim if needed
	if len(hm.entries) > hm.maxSize {
		hm.entries = hm.entries[len(hm.entries)-hm.maxSize:]
	}
}

// GetRecent returns recent history entries
func (hm *HistoryManager) GetRecent(count int) []HistoryEntry {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if count > len(hm.entries) {
		count = len(hm.entries)
	}

	start := len(hm.entries) - count
	return hm.entries[start:]
}

// Search searches history by command
func (hm *HistoryManager) Search(query string) []HistoryEntry {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make([]HistoryEntry, 0)

	for _, entry := range hm.entries {
		if contains(entry.Command, query) {
			results = append(results, entry)
		}
	}

	return results
}

// GetByTag returns entries with a specific tag
func (hm *HistoryManager) GetByTag(tag string) []HistoryEntry {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make([]HistoryEntry, 0)

	for _, entry := range hm.entries {
		for _, t := range entry.Tags {
			if t == tag {
				results = append(results, entry)
				break
			}
		}
	}

	return results
}

// GetStats returns history statistics
func (hm *HistoryManager) GetStats() HistoryStats {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	stats := HistoryStats{
		TotalCommands: len(hm.entries),
	}

	if len(hm.entries) == 0 {
		return stats
	}

	stats.OldestCommand = hm.entries[0].Timestamp
	stats.NewestCommand = hm.entries[len(hm.entries)-1].Timestamp

	var totalDuration time.Duration
	successCount := 0

	for _, entry := range hm.entries {
		totalDuration += entry.Duration
		if entry.ExitCode == 0 {
			successCount++
		}
	}

	stats.AvgDuration = totalDuration / time.Duration(len(hm.entries))
	stats.SuccessRate = float64(successCount) / float64(len(hm.entries)) * 100

	return stats
}

// HistoryStats represents history statistics
type HistoryStats struct {
	TotalCommands int
	SuccessRate   float64
	AvgDuration   time.Duration
	OldestCommand time.Time
	NewestCommand time.Time
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
