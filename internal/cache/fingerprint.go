package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"
)

// FingerprintCache implements content-based caching using SHA-256 fingerprints.
// T35: Cache by content hash, not full content - enables:
// 1. Deduplication of identical outputs
// 2. Fast cache lookups for repeated content
// 3. Memory-efficient caching (hash keys vs full content)
type FingerprintCache struct {
	mu sync.RWMutex

	// cache maps fingerprint -> cached result
	cache map[string]*CachedResult

	// maxEntries limits cache size
	maxEntries int

	// ttl for cache entries
	ttl time.Duration

	// stats — accessed atomically, kept separate from mutex-protected fields
	hits   atomic.Int64
	misses atomic.Int64
}

// CachedResult holds a cached compression result
type CachedResult struct {
	Fingerprint    string
	OriginalSize   int
	Compressed     string
	CompressedSize int
	TokensSaved    int
	CreatedAt      time.Time
	ExpiresAt      time.Time
	AccessCount    int64 // accessed atomically
}

// FingerprintResult holds the result of a fingerprint operation
type FingerprintResult struct {
	Hash   string
	Hit    bool
	Cached *CachedResult
}

// NewFingerprintCache creates a new fingerprint-based cache
func NewFingerprintCache(maxEntries int, ttl time.Duration) *FingerprintCache {
	return &FingerprintCache{
		cache:      make(map[string]*CachedResult),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

// ComputeFingerprint generates a SHA-256 hash of the content
func ComputeFingerprint(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))[:16] // Use first 16 chars for efficiency
}

// ComputeFingerprintBytes generates a SHA-256 hash of byte content
func ComputeFingerprintBytes(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Get retrieves a cached result by content fingerprint
func (fc *FingerprintCache) Get(content string) *FingerprintResult {
	fp := ComputeFingerprint(content)
	return fc.GetByFingerprint(fp)
}

// GetByFingerprint retrieves a cached result by fingerprint
func (fc *FingerprintCache) GetByFingerprint(fp string) *FingerprintResult {
	fc.mu.RLock()
	cached, exists := fc.cache[fp]
	fc.mu.RUnlock()

	result := &FingerprintResult{Hash: fp}
	if exists {
		if time.Now().Before(cached.ExpiresAt) {
			result.Hit = true
			result.Cached = cached
			atomic.AddInt64(&cached.AccessCount, 1)
			fc.hits.Add(1)
		}
	} else {
		fc.misses.Add(1)
	}

	return result
}

// Set stores a result in the cache
func (fc *FingerprintCache) Set(content string, compressed string, tokensSaved int) {
	fp := ComputeFingerprint(content)
	fc.SetByFingerprint(fp, content, compressed, tokensSaved)
}

// SetByFingerprint stores a result with a known fingerprint
func (fc *FingerprintCache) SetByFingerprint(fp string, original string, compressed string, tokensSaved int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Evict oldest if at capacity
	if len(fc.cache) >= fc.maxEntries {
		fc.evictOldest()
	}

	now := time.Now()
	fc.cache[fp] = &CachedResult{
		Fingerprint:    fp,
		OriginalSize:   len(original),
		Compressed:     compressed,
		CompressedSize: len(compressed),
		TokensSaved:    tokensSaved,
		CreatedAt:      now,
		ExpiresAt:      now.Add(fc.ttl),
		AccessCount:    0,
	}
}

// evictOldest removes the oldest entry (called under lock)
func (fc *FingerprintCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range fc.cache {
		if oldestKey == "" || v.CreatedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(fc.cache, oldestKey)
	}
}

// Stats returns cache statistics
func (fc *FingerprintCache) Stats() CacheStats {
	fc.mu.RLock()
	entries := len(fc.cache)
	fc.mu.RUnlock()

	return CacheStats{
		Entries:    entries,
		MaxEntries: fc.maxEntries,
		Hits:       fc.hits.Load(),
		Misses:     fc.misses.Load(),
		HitRate:    fc.computeHitRate(),
	}
}

// CacheStats holds cache statistics
type CacheStats struct {
	Entries    int
	MaxEntries int
	Hits       int64
	Misses     int64
	HitRate    float64
}

func (fc *FingerprintCache) computeHitRate() float64 {
	hits := fc.hits.Load()
	misses := fc.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

// Clear clears the cache
func (fc *FingerprintCache) Clear() {
	fc.mu.Lock()
	fc.cache = make(map[string]*CachedResult)
	fc.mu.Unlock()
	fc.hits.Store(0)
	fc.misses.Store(0)
}

// Prune removes expired entries
func (fc *FingerprintCache) Prune() int {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	now := time.Now()
	pruned := 0

	for k, v := range fc.cache {
		if now.After(v.ExpiresAt) {
			delete(fc.cache, k)
			pruned++
		}
	}

	return pruned
}

// Size returns the number of entries in the cache
func (fc *FingerprintCache) Size() int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return len(fc.cache)
}

// Global cache instance
var globalCache *FingerprintCache
var globalCacheOnce sync.Once

// GetGlobalCache returns the global fingerprint cache
func GetGlobalCache() *FingerprintCache {
	globalCacheOnce.Do(func() {
		globalCache = NewFingerprintCache(10000, 24*time.Hour)
	})
	return globalCache
}
