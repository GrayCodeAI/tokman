package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReversibleStore stores original outputs indexed by content hash.
// Claw-compactor style reversible compression.
// Users can restore any compressed output to its original form.
type ReversibleStore struct {
	baseDir string
	mu      sync.RWMutex
}

// StoredEntry holds a reversible compression entry.
type StoredEntry struct {
	Hash         string         `json:"hash"`
	Command      string         `json:"command"`
	Original     string         `json:"original"`
	Compressed   string         `json:"compressed"`
	OriginalHash string         `json:"original_hash"`
	Mode         string         `json:"mode"`
	Budget       int            `json:"budget"`
	Timestamp    time.Time      `json:"timestamp"`
	LayerStats   map[string]int `json:"layer_stats,omitempty"`
}

// NewReversibleStore creates a store in the tokman data directory.
func NewReversibleStore() *ReversibleStore {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	baseDir := filepath.Join(home, ".local", "share", "tokman", "reversible")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}
	return &ReversibleStore{baseDir: baseDir}
}

// Store saves an original-compressed pair for later restoration.
func (s *ReversibleStore) Store(command, original, compressed string, mode string, budget int, layerStats map[string]int) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := s.computeHash(original)
	entry := StoredEntry{
		Hash:         hash[:12],
		Command:      command,
		Original:     original,
		Compressed:   compressed,
		OriginalHash: hash,
		Mode:         mode,
		Budget:       budget,
		Timestamp:    time.Now(),
		LayerStats:   layerStats,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to marshal entry: %v\n", err)
		return hash[:12]
	}
	filename := fmt.Sprintf("%s_%s.json", time.Now().Format("20060102_150405"), hash[:8])
	path := filepath.Join(s.baseDir, filename)
	if err := os.WriteFile(path, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", path, err)
	}

	return hash[:12]
}

// Restore retrieves the original output by hash prefix.
func (s *ReversibleStore) Restore(hashPrefix string) (*StoredEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("no reversible entries found")
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.baseDir, e.Name()))
		if err != nil {
			continue
		}
		var entry StoredEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if entry.Hash == hashPrefix || (len(entry.OriginalHash) >= 12 && entry.OriginalHash[:12] == hashPrefix) {
			return &entry, nil
		}
	}

	return nil, fmt.Errorf("no entry found for hash: %s", hashPrefix)
}

// ListRecent returns the N most recent reversible entries.
func (s *ReversibleStore) ListRecent(n int) ([]StoredEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var results []StoredEntry
	for _, e := range entries {
		if e.IsDir() || len(results) >= n {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.baseDir, e.Name()))
		if err != nil {
			continue
		}
		var entry StoredEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		results = append(results, entry)
	}

	return results, nil
}

func (s *ReversibleStore) computeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
