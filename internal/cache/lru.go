package cache

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

// LRUCache provides a thread-safe LRU cache with TTL support.
// Uses interface{} for values to support different result types.
type LRUCache struct {
	maxSize int
	ttl     time.Duration
	entries map[string]*list.Element
	order   *list.List
	mu      sync.RWMutex
	hits    atomic.Int64
	misses  atomic.Int64
}

// LRUStats holds basic LRU cache statistics.
type LRUStats struct {
	Entries int
	MaxSize int
	Hits    int64
	Misses  int64
	HitRate float64
}

type lruEntry struct {
	key       string
	value     interface{}
	createdAt time.Time
}

// NewLRUCache creates an LRU cache with given max size and TTL.
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

// Get retrieves a cached value. Returns nil if not found or expired.
func (c *LRUCache) Get(key string) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		c.misses.Add(1)
		return nil
	}

	entry := elem.Value.(*lruEntry)

	// Check TTL
	if time.Since(entry.createdAt) > c.ttl {
		c.removeElement(elem)
		c.misses.Add(1)
		return nil
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	c.hits.Add(1)
	return entry.value
}

// Set stores a value in the cache.
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if elem, ok := c.entries[key]; ok {
		elem.Value.(*lruEntry).value = value
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
		value:     value,
		createdAt: time.Now(),
	}
	elem := c.order.PushFront(entry)
	c.entries[key] = elem
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

// Stats returns cache statistics.
func (c *LRUCache) Stats() LRUStats {
	c.mu.RLock()
	entries := len(c.entries)
	c.mu.RUnlock()

	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return LRUStats{
		Entries: entries,
		MaxSize: c.maxSize,
		Hits:    hits,
		Misses:  misses,
		HitRate: hitRate,
	}
}
