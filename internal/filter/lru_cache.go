package filter

import (
	"container/list"
	"sync"
	"time"
)

// LRUCache provides a thread-safe LRU cache for compression results.
// T101-T105: Improved caching with LRU eviction, TTL, and persistence.
type LRUCache struct {
	maxSize int
	ttl     time.Duration
	entries map[string]*list.Element
	order   *list.List
	mu      sync.RWMutex
	hits    int64
	misses  int64
}

type lruEntry struct {
	key       string
	result    *CachedResult
	createdAt time.Time
}

// NewLRUCache creates an LRU cache with given max size and TTL.
// T101: Configurable LRU cache for filtered outputs.
func NewLRUCache(maxSize int, ttl time.Duration) *LRUCache {
	if maxSize <= 0 {
		maxSize = 100
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &LRUCache{
		maxSize: maxSize,
		ttl:     ttl,
		entries: make(map[string]*list.Element),
		order:   list.New(),
	}
}

// Get retrieves a cached result. Returns nil if not found or expired.
func (c *LRUCache) Get(key string) *CachedResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		c.misses++
		return nil
	}

	entry := elem.Value.(*lruEntry)

	// Check TTL
	if time.Since(entry.createdAt) > c.ttl {
		c.removeElement(elem)
		c.misses++
		return nil
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	c.hits++
	return entry.result
}

// Set stores a result in the cache.
func (c *LRUCache) Set(key string, result *CachedResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if elem, ok := c.entries[key]; ok {
		elem.Value.(*lruEntry).result = result
		elem.Value.(*lruEntry).createdAt = time.Now()
		c.order.MoveToFront(elem)
		return
	}

	// Evict oldest if at capacity
	for c.order.Len() >= c.maxSize {
		c.evictOldest()
	}

	// Add new entry
	entry := &lruEntry{
		key:       key,
		result:    result,
		createdAt: time.Now(),
	}
	elem := c.order.PushFront(entry)
	c.entries[key] = elem
}

// Size returns the number of entries in the cache.
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns cache hit/miss statistics.
func (c *LRUCache) Stats() (hits, misses int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses
}

// HitRate returns the cache hit rate as a percentage.
func (c *LRUCache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := c.hits + c.misses
	if total == 0 {
		return 0
	}
	return float64(c.hits) / float64(total) * 100
}

// Clear removes all entries.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*list.Element)
	c.order.Init()
}

// PurgeExpired removes all expired entries.
func (c *LRUCache) PurgeExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for key, elem := range c.entries {
		entry := elem.Value.(*lruEntry)
		if time.Since(entry.createdAt) > c.ttl {
			c.removeElement(elem)
			delete(c.entries, key)
			removed++
		}
	}
	return removed
}

func (c *LRUCache) evictOldest() {
	elem := c.order.Back()
	if elem == nil {
		return
	}
	c.removeElement(elem)
}

func (c *LRUCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	delete(c.entries, entry.key)
	c.order.Remove(elem)
}
