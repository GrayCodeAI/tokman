package filter

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// CrossRunDedupFilter deduplicates content across multiple pipeline runs within
// a session. When the same file/block appears again (by SHA-256 fingerprint),
// it emits a compact reference token instead of the full content.
//
// This is especially powerful in multi-turn LLM conversations where the same
// source files are sent repeatedly. Savings are 100% for repeated blocks after
// the first occurrence.
//
// Thread-safe. Designed to be shared across pipeline invocations.
type CrossRunDedupFilter struct {
	mu    sync.RWMutex
	seen  map[string]crossRunSeenEntry // fingerprint → metadata
	ttl   time.Duration        // evict entries older than TTL
	maxSz int                  // max number of entries
}

type crossRunSeenEntry struct {
	firstSeen time.Time
	label     string // short label for the reference token
	tokens    int    // token count of the original content
	runCount  int    // how many times this has been seen
}

const (
	// Minimum block size (tokens) to consider for cross-run dedup.
	crossRunMinTokens = 30
	// Short fingerprint prefix used in reference tokens.
	fpPrefixLen = 8
)

// NewCrossRunDedupFilter creates a cross-run deduplication filter.
// TTL defaults to 30 minutes; max 1000 entries.
func NewCrossRunDedupFilter() *CrossRunDedupFilter {
	return &CrossRunDedupFilter{
		seen:  make(map[string]crossRunSeenEntry),
		ttl:   30 * time.Minute,
		maxSz: 1000,
	}
}

// NewCrossRunDedupFilterWithTTL creates a filter with a custom TTL.
func NewCrossRunDedupFilterWithTTL(ttl time.Duration, maxEntries int) *CrossRunDedupFilter {
	return &CrossRunDedupFilter{
		seen:  make(map[string]crossRunSeenEntry),
		ttl:   ttl,
		maxSz: maxEntries,
	}
}

// Name returns the filter name.
func (f *CrossRunDedupFilter) Name() string {
	return "crossrun_dedup"
}

// Reset clears all cached fingerprints (call at session end).
func (f *CrossRunDedupFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = make(map[string]crossRunSeenEntry)
}

// Apply replaces content seen in previous pipeline runs with reference tokens.
func (f *CrossRunDedupFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Split into logical blocks (separated by blank lines or file markers)
	blocks := splitBlocks(input)
	if len(blocks) == 0 {
		return input, 0
	}

	original := core.EstimateTokens(input)
	f.evict()

	var result []string
	totalSaved := 0

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			result = append(result, block)
			continue
		}

		toks := core.EstimateTokens(block)
		if toks < crossRunMinTokens {
			result = append(result, block)
			continue
		}

		fp := fingerprint(block)
		ref := f.lookup(fp)
		if ref != "" {
			// Already seen — emit reference token
			saved := toks - core.EstimateTokens(ref)
			if saved > 0 {
				result = append(result, ref)
				totalSaved += saved
				continue
			}
		}

		// Not yet seen — register and pass through
		f.register(fp, block, toks)
		result = append(result, block)
	}

	if totalSaved <= 0 {
		return input, 0
	}
	return strings.Join(result, "\n"), original - core.EstimateTokens(strings.Join(result, "\n"))
}

// lookup checks if a fingerprint has been seen and returns a reference token.
func (f *CrossRunDedupFilter) lookup(fp string) string {
	f.mu.RLock()
	entry, ok := f.seen[fp]
	f.mu.RUnlock()
	if !ok {
		return ""
	}
	f.mu.Lock()
	entry.runCount++
	f.seen[fp] = entry
	f.mu.Unlock()
	return fmt.Sprintf("[⟳ %s — %d tokens, seen %d times — content omitted]",
		entry.label, entry.tokens, entry.runCount)
}

// register adds a fingerprint to the cache.
func (f *CrossRunDedupFilter) register(fp, content string, tokens int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.seen) >= f.maxSz {
		return // at capacity; don't evict here (evict() handles that)
	}
	label := extractLabel(content, fp)
	f.seen[fp] = crossRunSeenEntry{
		firstSeen: time.Now(),
		label:     label,
		tokens:    tokens,
		runCount:  1,
	}
}

// evict removes entries older than TTL.
func (f *CrossRunDedupFilter) evict() {
	f.mu.Lock()
	defer f.mu.Unlock()
	cutoff := time.Now().Add(-f.ttl)
	for k, v := range f.seen {
		if v.firstSeen.Before(cutoff) {
			delete(f.seen, k)
		}
	}
}

// fingerprint returns a SHA-256 hex fingerprint for a string.
func fingerprint(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

// extractLabel extracts a short human-readable label from a content block.
func extractLabel(content, fp string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && len(trimmed) < 60 {
			// Use first non-blank, reasonably-short line as label
			return trimmed
		}
	}
	// Fallback: use fingerprint prefix
	if len(fp) >= fpPrefixLen {
		return "block:" + fp[:fpPrefixLen]
	}
	return "block"
}

// splitBlocks splits a document into logical blocks separated by blank lines
// or common file markers (--- separator, === separator, ``` fences).
func splitBlocks(input string) []string {
	var blocks []string
	var current strings.Builder
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isSeparator := trimmed == "" ||
			strings.HasPrefix(trimmed, "---") ||
			strings.HasPrefix(trimmed, "===") ||
			strings.HasPrefix(trimmed, "```")

		if isSeparator {
			if current.Len() > 0 {
				blocks = append(blocks, current.String())
				current.Reset()
			}
			current.WriteString(line + "\n")
		} else {
			current.WriteString(line + "\n")
		}
	}
	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}
	return blocks
}
