package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// ResultFingerprinter provides content-hash based caching.
// R13: Cache by content hash, not full content — faster lookups.
type ResultFingerprinter struct {
	cache   map[string]*FingerPrintEntry
	keys    []string // insertion order for FIFO eviction
	mu      sync.RWMutex
	maxSize int
}

// FingerPrintEntry holds a fingerprinted result.
type FingerPrintEntry struct {
	Fingerprint string
	Output      string
	Tokens      int
	Command     string
}

// NewResultFingerprinter creates a fingerprinter with given capacity.
func NewResultFingerprinter(maxSize int) *ResultFingerprinter {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &ResultFingerprinter{
		cache:   make(map[string]*FingerPrintEntry),
		maxSize: maxSize,
	}
}

// Fingerprint computes a short hash of content.
func Fingerprint(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:8]) // 16 chars
}

// FingerprintWithCommand computes a fingerprint including command context.
func FingerprintWithCommand(command, content string) string {
	h := sha256.Sum256([]byte(command + ":" + content))
	return hex.EncodeToString(h[:8])
}

// Get retrieves a cached result by fingerprint.
func (f *ResultFingerprinter) Get(fp string) (*FingerPrintEntry, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	entry, ok := f.cache[fp]
	return entry, ok
}

// Set stores a result with its fingerprint.
func (f *ResultFingerprinter) Set(fp string, entry *FingerPrintEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.cache[fp]; !exists {
		// Evict oldest entry (FIFO) if at capacity.
		if len(f.cache) >= f.maxSize && len(f.keys) > 0 {
			oldest := f.keys[0]
			f.keys = f.keys[1:]
			delete(f.cache, oldest)
		}
		f.keys = append(f.keys, fp)
	}

	f.cache[fp] = entry
}

// Size returns cache size.
func (f *ResultFingerprinter) Size() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.cache)
}

// Clear removes all entries.
func (f *ResultFingerprinter) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cache = make(map[string]*FingerPrintEntry)
	f.keys = f.keys[:0]
}
