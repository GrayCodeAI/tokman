package filter

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// minURLLength is the minimum byte length a URL must exceed before shortening.
const minURLLength = 60

// urlRe matches http/https URLs.
var urlRe = regexp.MustCompile(`https?://[^\s"'<>)}\]]+`)

// absPathRe matches Unix absolute file paths (must start with / and contain at
// least one path separator after the root).
var absPathRe = regexp.MustCompile(`(?:^|[\s"'({\[])(/(?:[^/\s"'<>)}\]]+/)+[^/\s"'<>)}\]]+)`)

// URLCompressFilter shortens long URLs, replaces home dirs with ~, and
// deduplicates identical URLs/paths within the same Apply call. Task #73.
type URLCompressFilter struct {
	homeDir string
}

// NewURLCompressFilter creates a new URLCompressFilter.
// It captures the current user's home directory at construction time so the
// filter remains pure (no os.Getenv calls) during Apply.
func NewURLCompressFilter() *URLCompressFilter {
	home, _ := os.UserHomeDir()
	return &URLCompressFilter{homeDir: home}
}

// Name returns the filter name.
func (f *URLCompressFilter) Name() string {
	return "url_compress"
}

// Apply compresses URLs and file paths in input and returns token savings.
// Runs only in ModeMinimal and ModeAggressive.
func (f *URLCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	// seen maps an original token to a stable numeric index (1-based).
	seen := make(map[string]int)

	// --- Pass 1: shorten URLs ---
	output := urlRe.ReplaceAllStringFunc(input, func(raw string) string {
		shortened := f.shortenURL(raw)
		return f.dedupToken(shortened, seen)
	})

	// --- Pass 2: compress absolute file paths ---
	output = absPathRe.ReplaceAllStringFunc(output, func(match string) string {
		// The regex may capture a leading whitespace/punctuation character in
		// group 0 to avoid false positives inside words. Separate it out.
		prefix := ""
		pathPart := match
		if len(match) > 0 && !strings.HasPrefix(match, "/") {
			prefix = string(match[0])
			pathPart = match[1:]
		}
		compressed := f.compressPath(pathPart)
		return prefix + f.dedupToken(compressed, seen)
	})

	saved := originalTokens - core.EstimateTokens(output)
	if saved < 0 {
		saved = 0
	}
	return output, saved
}

// shortenURL reduces a long URL to scheme + host + .../lastSegment.
// URLs shorter than minURLLength are returned unchanged.
func (f *URLCompressFilter) shortenURL(raw string) string {
	// Strip trailing punctuation that was likely captured by the greedy regex
	// but is not part of the URL itself (e.g. trailing comma/period).
	raw = strings.TrimRight(raw, ".,;:!?")

	if len(raw) <= minURLLength {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	lastSeg := path.Base(parsed.Path)
	if lastSeg == "." || lastSeg == "/" || lastSeg == "" {
		// Nothing useful to keep; return as-is.
		return raw
	}

	// Rebuild as scheme://host/.../lastSeg (query/fragment dropped).
	short := fmt.Sprintf("%s://%s/.../%s", parsed.Scheme, parsed.Host, lastSeg)
	if len(short) >= len(raw) {
		return raw
	}
	return short
}

// compressPath replaces the home directory prefix with ~ and returns the path.
func (f *URLCompressFilter) compressPath(p string) string {
	if f.homeDir != "" && strings.HasPrefix(p, f.homeDir) {
		return "~" + p[len(f.homeDir):]
	}
	return p
}

// dedupToken checks whether token has been seen before. On the first
// occurrence it records it and returns it unchanged. On subsequent
// occurrences it returns a short placeholder like [URL:1].
func (f *URLCompressFilter) dedupToken(token string, seen map[string]int) string {
	if idx, ok := seen[token]; ok {
		return fmt.Sprintf("[URL:%d]", idx)
	}
	seen[token] = len(seen) + 1
	return token
}
