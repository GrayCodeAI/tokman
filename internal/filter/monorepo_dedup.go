package filter

import (
	"fmt"
	"strings"
)

// SharedSegmentStore maps a content key (first 64 chars of trimmed content)
// to the file paths that contain that segment. Task #182.
type SharedSegmentStore struct {
	index map[string][]string // contentKey → []filePaths
}

// newSharedSegmentStore creates an empty store.
func newSharedSegmentStore() *SharedSegmentStore {
	return &SharedSegmentStore{
		index: make(map[string][]string),
	}
}

// segmentKey returns a simple hash key: first 64 chars of trimmed content.
func segmentKey(segment string) string {
	trimmed := strings.TrimSpace(segment)
	if len(trimmed) > 64 {
		return trimmed[:64]
	}
	return trimmed
}

// add registers that filePath contains a segment with the given key.
func (s *SharedSegmentStore) add(key, filePath string) {
	for _, existing := range s.index[key] {
		if existing == filePath {
			return // already registered
		}
	}
	s.index[key] = append(s.index[key], filePath)
}

// sharedKeys returns all content keys that appear in ≥2 files.
func (s *SharedSegmentStore) sharedKeys() []string {
	var keys []string
	for k, paths := range s.index {
		if len(paths) >= 2 {
			keys = append(keys, k)
		}
	}
	return keys
}

// filesFor returns the file paths that share the given key.
func (s *SharedSegmentStore) filesFor(key string) []string {
	return s.index[key]
}

// MonorepoDeduplicator tracks content across multiple files in a monorepo
// and replaces shared segments with compact references during compression.
type MonorepoDeduplicator struct {
	store   *SharedSegmentStore
	files   map[string]string // path → content
	mode    Mode
}

// NewMonorepoDeduplicator creates a new deduplicator.
func NewMonorepoDeduplicator(mode Mode) *MonorepoDeduplicator {
	return &MonorepoDeduplicator{
		store: newSharedSegmentStore(),
		files: make(map[string]string),
		mode:  mode,
	}
}

// AddFile registers file content with the deduplicator. Content is split into
// paragraph-sized segments; each segment is keyed by its first 64 trimmed chars.
func (d *MonorepoDeduplicator) AddFile(path, content string) {
	d.files[path] = content
	for _, segment := range d.segmentsOf(content) {
		key := segmentKey(segment)
		if key == "" {
			continue
		}
		d.store.add(key, path)
	}
}

// segmentsOf splits content into logical segments (blank-line delimited).
func (d *MonorepoDeduplicator) segmentsOf(content string) []string {
	var segments []string
	var cur strings.Builder
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			if cur.Len() > 0 {
				segments = append(segments, cur.String())
				cur.Reset()
			}
		} else {
			cur.WriteString(line)
			cur.WriteByte('\n')
		}
	}
	if cur.Len() > 0 {
		segments = append(segments, cur.String())
	}
	return segments
}

// GetSharedContent returns the content segments (as their key strings) that
// appear in two or more registered files.
func (d *MonorepoDeduplicator) GetSharedContent() []string {
	return d.store.sharedKeys()
}

// CompressWithDedup compresses the content of a file, replacing any segment
// whose key matches a shared segment with a compact reference placeholder.
// Non-shared segments are passed through unchanged.
// Returns the compressed content and the number of characters saved.
func (d *MonorepoDeduplicator) CompressWithDedup(path, content string, mode Mode) (string, int) {
	sharedSet := make(map[string]bool)
	for _, k := range d.store.sharedKeys() {
		sharedSet[k] = true
	}

	segments := d.segmentsOf(content)
	if len(segments) == 0 {
		return content, 0
	}

	var sb strings.Builder
	saved := 0

	for _, segment := range segments {
		key := segmentKey(segment)
		if key != "" && sharedSet[key] {
			// Replace with a compact reference.
			files := d.store.filesFor(key)
			// Pick a representative "other" file for the reference.
			refFile := ""
			for _, f := range files {
				if f != path {
					refFile = f
					break
				}
			}
			placeholder := fmt.Sprintf("[shared: %d chars, see %s]\n", len(segment), refFile)
			saved += len(segment) - len(placeholder)
			sb.WriteString(placeholder)
		} else {
			sb.WriteString(segment)
			sb.WriteByte('\n')
		}
	}

	out := sb.String()
	if saved <= 0 {
		return content, 0
	}
	return out, saved
}
