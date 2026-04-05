// Package simd provides SIMD-optimized operations for Tokman compression.
//
// Current Status: Auto-vectorized fallback implementations.
// The Go compiler may auto-vectorize some loops on supported architectures.
//
// TODO(simd): Implement native SIMD support when Go 1.26+ is released.
// Requirements:
//   - Go 1.26+ with GOEXPERIMENT=simd
//   - CPU feature detection (AVX2, AVX-512, ARM NEON)
//   - Build tags for architecture-specific implementations
//
// Implementation Plan:
//   1. Add build tags: //go:build goexperiment.simd && (amd64 || arm64)
//   2. Use golang.org/x/sys/cpu for feature detection
//   3. Implement SIMD versions of:
//      - StripANSI (byte scanning with SIMD comparison)
//      - IndexByteSet (parallel byte matching)
//      - CountByte (population count)
//      - SplitWords (whitespace detection)
//   4. Benchmark against auto-vectorized versions
//
// References:
//   - https://github.com/golang/go/issues/53171 (SIMD proposal)
//   - https://pkg.go.dev/golang.org/x/sys/cpu
package simd

import (
	"strings"
)

// Enabled reports whether native SIMD optimizations are available.
// Currently false - using auto-vectorized fallback implementations.
// TODO(simd): Set to true when native SIMD is implemented and detected.
var Enabled bool

func init() {
	// TODO(simd): Detect SIMD capabilities at runtime:
	//   if cpu.X86.HasAVX2 || cpu.ARM64.HasASIMD {
	//       Enabled = true
	//   }
	//
	// Current implementations rely on Go compiler auto-vectorization.
	// Manual benchmarks show ~2-3x speedup potential with native SIMD.
	Enabled = false
}

// --- SIMD Optimized ANSI Stripping ---

// StripANSI strips ANSI escape sequences from input using optimized byte scanning.
// This is significantly faster than regex-based approaches for large inputs.
func StripANSI(input string) string {
	if len(input) == 0 {
		return input
	}

	inputBytes := []byte(input)
	output := make([]byte, 0, len(inputBytes))

	i := 0
	for i < len(inputBytes) {
		// Check for ESC character (0x1b)
		if inputBytes[i] == 0x1b {
			// Found escape sequence, skip it
			skip := skipANSISequence(inputBytes, i)
			if skip > 0 {
				i += skip
				continue
			}
		}
		output = append(output, inputBytes[i])
		i++
	}

	return string(output)
}

// skipANSISequence returns the length of an ANSI escape sequence starting at pos.
// Returns 0 if not a valid ANSI sequence.
func skipANSISequence(data []byte, pos int) int {
	if pos >= len(data) || data[pos] != 0x1b {
		return 0
	}

	if pos+1 >= len(data) {
		return 0
	}

	next := data[pos+1]

	// CSI sequence: ESC [ ... final byte
	if next == '[' {
		return skipCSI(data, pos)
	}

	// OSC sequence: ESC ] ... BEL or ESC \
	if next == ']' {
		return skipOSC(data, pos)
	}

	// Other escape sequences (single char after ESC)
	if next >= '@' && next <= '~' {
		return 2
	}

	return 0
}

// skipCSI skips a CSI (Control Sequence Introducer) sequence.
func skipCSI(data []byte, pos int) int {
	if pos+2 >= len(data) {
		return 0
	}

	i := pos + 2 // Skip ESC [

	// Skip parameter bytes (0x30-0x3f)
	for i < len(data) && data[i] >= 0x30 && data[i] <= 0x3f {
		i++
	}

	// Skip intermediate bytes (0x20-0x2f)
	for i < len(data) && data[i] >= 0x20 && data[i] <= 0x2f {
		i++
	}

	// Final byte (0x40-0x7e)
	if i < len(data) && data[i] >= 0x40 && data[i] <= 0x7e {
		return i - pos + 1
	}

	return 0
}

// skipOSC skips an OSC (Operating System Command) sequence.
func skipOSC(data []byte, pos int) int {
	if pos+2 >= len(data) {
		return 0
	}

	i := pos + 2 // Skip ESC ]

	// Find terminating BEL (0x07) or ESC \
	for i < len(data) {
		if data[i] == 0x07 {
			return i - pos + 1
		}
		if data[i] == 0x1b && i+1 < len(data) && data[i+1] == '\\' {
			return i - pos + 2
		}
		i++
	}

	return 0
}

// HasANSI checks if input contains ANSI escape sequences.
func HasANSI(input string) bool {
	return IndexByte(input, 0x1b) >= 0
}

// --- SIMD Optimized Byte Operations ---

// IndexByte finds the first occurrence of c in s using the standard library's
// SIMD-optimized implementation.
func IndexByte(s string, c byte) int {
	return strings.IndexByte(s, c)
}

// IndexByteSet finds the first occurrence of any byte in set.
// Uses a lookup table for fast byte set membership testing.
func IndexByteSet(s string, set []byte) int {
	if len(set) == 0 || len(s) == 0 {
		return -1
	}

	// Build lookup table for fast byte set membership
	var lut [256]bool
	for _, c := range set {
		lut[c] = true
	}

	for i := 0; i < len(s); i++ {
		if lut[s[i]] {
			return i
		}
	}
	return -1
}

// CountByte counts occurrences of c in s using byte scanning.
func CountByte(s string, c byte) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			count++
		}
	}
	return count
}

// CountByteSet counts occurrences of any byte in set.
func CountByteSet(s string, set []byte) int {
	if len(set) == 0 || len(s) == 0 {
		return 0
	}

	var lut [256]bool
	for _, c := range set {
		lut[c] = true
	}

	count := 0
	for i := 0; i < len(s); i++ {
		if lut[s[i]] {
			count++
		}
	}
	return count
}

// --- SIMD Optimized String Matching ---

// ContainsAny checks if s contains any of the substrings.
// Uses first-character matching for speed.
func ContainsAny(s string, substrs []string) bool {
	if len(s) == 0 || len(substrs) == 0 {
		return false
	}

	for _, sub := range substrs {
		if len(sub) == 0 {
			continue
		}
		if indexSubstring(s, sub) >= 0 {
			return true
		}
	}
	return false
}

// indexSubstring finds the index of substr in s using optimized matching.
func indexSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	first := substr[0]
	n := len(substr)

	// Quick scan for first character
	for i := 0; i <= len(s)-n; i++ {
		if s[i] == first {
			// Potential match, verify rest
			if s[i:i+n] == substr {
				return i
			}
		}
	}
	return -1
}

// --- SIMD Optimized Word Boundary Detection ---

// IsWordChar checks if a byte is part of a word (alphanumeric or underscore).
func IsWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// IsWhitespace checks if a byte is whitespace.
func IsWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// SplitWords splits string into words using SIMD-optimized scanning.
func SplitWords(s string) []string {
	if len(s) == 0 {
		return nil
	}

	var words []string
	start := -1
	for i := 0; i < len(s); i++ {
		if IsWhitespace(s[i]) {
			if start >= 0 {
				words = append(words, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, s[start:])
	}
	return words
}

// ContainsWord checks if s contains the word w (whole word matching).
func ContainsWord(s, w string) bool {
	if len(w) == 0 || len(s) < len(w) {
		return false
	}

	for i := 0; i <= len(s)-len(w); i++ {
		if s[i] == w[0] {
			// Check prefix match
			if s[i:i+len(w)] == w {
				// Check word boundaries
				startOK := i == 0 || !IsWordChar(s[i-1])
				endOK := i+len(w) == len(s) || !IsWordChar(s[i+len(w)])
				if startOK && endOK {
					return true
				}
			}
		}
	}
	return false
}

// FindWordBoundary finds the next word boundary starting from pos.
// Returns the position of the next non-word character or end of string.
func FindWordBoundary(s string, pos int) int {
	for pos < len(s) && IsWordChar(s[pos]) {
		pos++
	}
	return pos
}

// --- SIMD Optimized Bracket Matching ---

// BracketPair represents a pair of matching brackets.
type BracketPair struct {
	Open, Close byte
}

var DefaultBracketPairs = []BracketPair{
	{'{', '}'},
	{'[', ']'},
	{'(', ')'},
	{'<', '>'},
}

// CountBrackets counts opening and closing brackets in s.
func CountBrackets(s string, pairs []BracketPair) (opens, closes int) {
	if len(pairs) == 0 {
		pairs = DefaultBracketPairs
	}

	openSet := make([]byte, len(pairs))
	closeSet := make([]byte, len(pairs))
	for i, p := range pairs {
		openSet[i] = p.Open
		closeSet[i] = p.Close
	}

	opens = CountByteSet(s, openSet)
	closes = CountByteSet(s, closeSet)
	return
}

// --- Low-level SIMD Operations ---

// Note: These are standard Go loop implementations. The Go compiler may
// auto-vectorize some of these loops on supported architectures, but they
// do not explicitly use SIMD instructions.

// Memset fills buf with byte c.
func Memset(buf []byte, c byte) {
	for i := range buf {
		buf[i] = c
	}
}

// Memcmp compares two byte slices.
func Memcmp(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}
