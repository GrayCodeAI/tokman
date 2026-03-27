package filter

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// HistoryEntry records a single compression operation.
type HistoryEntry struct {
	ID              int
	Timestamp       time.Time
	FilterName      string
	Mode            Mode
	OriginalTokens  int
	CompressedTokens int
	Saved           int
	ReductionPct    float64
	// Fingerprint is a short hash of the input for dedup/replay.
	Fingerprint     string
}

// CompressionHistory stores a bounded circular history of compression operations.
// Task #137: Compression history and undo.
type CompressionHistory struct {
	mu      sync.Mutex
	entries []HistoryEntry
	maxSize int
	nextID  int

	// undoStack holds the last few original inputs for undo.
	undoStack []string
	undoMax   int
}

// NewCompressionHistory creates a history with the given max size.
func NewCompressionHistory(maxSize int) *CompressionHistory {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &CompressionHistory{
		maxSize: maxSize,
		undoMax: 10,
	}
}

// Record adds a compression operation to the history.
func (h *CompressionHistory) Record(original, compressed string, filterName string, mode Mode) {
	h.mu.Lock()
	defer h.mu.Unlock()

	origToks := core.EstimateTokens(original)
	compToks := core.EstimateTokens(compressed)
	saved := origToks - compToks

	var reduction float64
	if origToks > 0 {
		reduction = float64(saved) / float64(origToks) * 100
	}

	fp := ""
	if len(original) > 0 {
		// Short fingerprint from first 32 + last 8 chars
		end := original
		if len(end) > 8 {
			end = end[len(end)-8:]
		}
		start := original
		if len(start) > 32 {
			start = start[:32]
		}
		fp = fmt.Sprintf("%x", len(original))+":" + start[:min8(len(start), 8)] + end[:min8(len(end), 4)]
	}

	h.nextID++
	entry := HistoryEntry{
		ID:               h.nextID,
		Timestamp:        time.Now(),
		FilterName:       filterName,
		Mode:             mode,
		OriginalTokens:   origToks,
		CompressedTokens: compToks,
		Saved:            saved,
		ReductionPct:     reduction,
		Fingerprint:      fp,
	}

	// Circular eviction
	if len(h.entries) >= h.maxSize {
		h.entries = h.entries[1:]
	}
	h.entries = append(h.entries, entry)

	// Push to undo stack
	if len(h.undoStack) >= h.undoMax {
		h.undoStack = h.undoStack[1:]
	}
	h.undoStack = append(h.undoStack, original)
}

// Undo returns the most recently recorded original input, removing it from the stack.
// Returns "", false if the undo stack is empty.
func (h *CompressionHistory) Undo() (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.undoStack) == 0 {
		return "", false
	}
	last := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	return last, true
}

// Recent returns the most recent N entries (newest first).
func (h *CompressionHistory) Recent(n int) []HistoryEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n > len(h.entries) {
		n = len(h.entries)
	}
	result := make([]HistoryEntry, n)
	for i := 0; i < n; i++ {
		result[i] = h.entries[len(h.entries)-1-i]
	}
	return result
}

// Stats returns aggregate statistics across all history entries.
func (h *CompressionHistory) Stats() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) == 0 {
		return "No compression history."
	}

	totalSaved := 0
	totalOrig := 0
	filterCounts := make(map[string]int)

	for _, e := range h.entries {
		totalSaved += e.Saved
		totalOrig += e.OriginalTokens
		filterCounts[e.FilterName]++
	}

	var reduction float64
	if totalOrig > 0 {
		reduction = float64(totalSaved) / float64(totalOrig) * 100
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Compression History: %d operations\n", len(h.entries)))
	sb.WriteString(fmt.Sprintf("  Total saved: %d tokens (%.1f%% avg reduction)\n", totalSaved, reduction))
	sb.WriteString("  Top filters:\n")

	// Find top 3 filters
	type kv struct{ k string; v int }
	var sorted []kv
	for k, v := range filterCounts {
		sorted = append(sorted, kv{k, v})
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].v > sorted[i].v {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for i := 0; i < len(sorted) && i < 3; i++ {
		sb.WriteString(fmt.Sprintf("    %s: %d uses\n", sorted[i].k, sorted[i].v))
	}
	return sb.String()
}

// min8 is a small helper to avoid importing math for this file.
func min8(a, b int) int {
	if a < b {
		return a
	}
	return b
}
