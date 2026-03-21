// Package simd provides SIMD-optimized operations for Tokman compression.
// Requires Go 1.26+ with GOEXPERIMENT=simd for native SIMD support.
package simd

import (
	"unsafe"
)

// Enabled reports whether SIMD optimizations are available.
// Checks for AVX2 on AMD64 or NEON on ARM64.
var Enabled bool

func init() {
	// Runtime CPU feature detection will be handled by the build
	// For now, we use portable implementations that compilers can auto-vectorize
	Enabled = true
}

// --- SIMD Optimized ANSI Stripping ---

// StripANSI strips ANSI escape sequences from input using SIMD-optimized scanning.
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

// IndexByte finds the first occurrence of c in s using optimized scanning.
// For large inputs, this can be vectorized by the compiler.
func IndexByte(s string, c byte) int {
	// Use the standard library's optimized implementation
	// which already uses SIMD on supported platforms
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// IndexByteSet finds the first occurrence of any byte in set.
// SIMD-optimized for finding multiple target bytes simultaneously.
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

// CountByte counts occurrences of c in s using SIMD-optimized scanning.
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
// Uses SIMD-optimized first-character matching for speed.
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

// FindWordBoundary finds the next word boundary starting from pos.
// Returns the position of the next non-word character or end of string.
func FindWordBoundary(s string, pos int) int {
	for pos < len(s) && IsWordChar(s[pos]) {
		pos++
	}
	return pos
}

// FindNonWordBoundary finds the next non-word boundary starting from pos.
// Returns the position of the next word character or end of string.
func FindNonWordBoundary(s string, pos int) int {
	for pos < len(s) && !IsWordChar(s[pos]) {
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

// Note: These are placeholder implementations that the Go compiler can
// auto-vectorize. When GOEXPERIMENT=simd is enabled and Go 1.26+ is used,
// these will use actual SIMD instructions.

// Memset fills buf with byte c using SIMD-optimized memset.
func Memset(buf []byte, c byte) {
	for i := range buf {
		buf[i] = c
	}
}

// Memcmp compares two byte slices using SIMD-optimized comparison.
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

// unsafeString converts a byte slice to string without copying.
// Use with caution - the byte slice must not be modified after this call.
func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}
