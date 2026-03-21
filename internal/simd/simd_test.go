package simd

import (
	"strings"
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no ANSI codes",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "simple SGR",
			input:    "\x1b[31mRed\x1b[0m text",
			expected: "Red text",
		},
		{
			name:     "complex SGR",
			input:    "\x1b[1;31;47mBold Red on White\x1b[0m",
			expected: "Bold Red on White",
		},
		{
			name:     "CSI sequence",
			input:    "\x1b[2J\x1b[HClear screen",
			expected: "Clear screen",
		},
		{
			name:     "OSC sequence with BEL",
			input:    "\x1b]0;Title\x07Content",
			expected: "Content",
		},
		{
			name:     "OSC sequence with ESC backslash",
			input:    "\x1b]8;;http://example.com\x1b\\Link\x1b]8;;\x1b\\",
			expected: "Link",
		},
		{
			name:     "multiple codes",
			input:    "\x1b[32m\x1b[1mBold Green\x1b[0m \x1b[4mUnderline\x1b[0m",
			expected: "Bold Green Underline",
		},
		{
			name:     "mixed content",
			input:    "Start \x1b[31m\x1b[1mCOLOR\x1b[0m end",
			expected: "Start COLOR end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHasANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"Hello", false},
		{"\x1b[31mRed\x1b[0m", true},
		{"Color \x1b[32mgreen\x1b[0m text", true},
		{"\x1b]0;Title\x07", true},
	}

	for _, tt := range tests {
		result := HasANSI(tt.input)
		if result != tt.expected {
			t.Errorf("HasANSI(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestIndexByte(t *testing.T) {
	tests := []struct {
		s        string
		c        byte
		expected int
	}{
		{"", 'a', -1},
		{"hello", 'h', 0},
		{"hello", 'o', 4},
		{"hello", 'x', -1},
		{"hello world", ' ', 5},
	}

	for _, tt := range tests {
		result := IndexByte(tt.s, tt.c)
		if result != tt.expected {
			t.Errorf("IndexByte(%q, %q) = %d, want %d", tt.s, tt.c, result, tt.expected)
		}
	}
}

func TestIndexByteSet(t *testing.T) {
	tests := []struct {
		s        string
		set      []byte
		expected int
	}{
		{"", []byte{'a', 'b'}, -1},
		{"hello", []byte{'x', 'y', 'z'}, -1},
		{"hello", []byte{'h', 'l'}, 0},
		{"hello", []byte{'l', 'o'}, 2},
		{"world", []byte{'r', 'd'}, 2},
	}

	for _, tt := range tests {
		result := IndexByteSet(tt.s, tt.set)
		if result != tt.expected {
			t.Errorf("IndexByteSet(%q, %v) = %d, want %d", tt.s, tt.set, result, tt.expected)
		}
	}
}

func TestCountByte(t *testing.T) {
	tests := []struct {
		s        string
		c        byte
		expected int
	}{
		{"", 'a', 0},
		{"hello", 'l', 2},
		{"hello", 'o', 1},
		{"aaa", 'a', 3},
		{"abc", 'x', 0},
	}

	for _, tt := range tests {
		result := CountByte(tt.s, tt.c)
		if result != tt.expected {
			t.Errorf("CountByte(%q, %q) = %d, want %d", tt.s, tt.c, result, tt.expected)
		}
	}
}

func TestCountByteSet(t *testing.T) {
	tests := []struct {
		s        string
		set      []byte
		expected int
	}{
		{"", []byte{'a', 'b'}, 0},
		{"hello", []byte{'l', 'o'}, 3},
		{"hello world", []byte{' ', 'l'}, 4},
		{"abc", []byte{'x', 'y'}, 0},
	}

	for _, tt := range tests {
		result := CountByteSet(tt.s, tt.set)
		if result != tt.expected {
			t.Errorf("CountByteSet(%q, %v) = %d, want %d", tt.s, tt.set, result, tt.expected)
		}
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"", []string{"a"}, false},
		{"hello", []string{}, false},
		{"hello world", []string{"world"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"hello world", []string{"foo", "world"}, true},
		{"hello", []string{"he", "lo"}, true},
	}

	for _, tt := range tests {
		result := ContainsAny(tt.s, tt.substrs)
		if result != tt.expected {
			t.Errorf("ContainsAny(%q, %v) = %v, want %v", tt.s, tt.substrs, result, tt.expected)
		}
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		c        byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'-', false},
		{' ', false},
		{'.', false},
	}

	for _, tt := range tests {
		result := IsWordChar(tt.c)
		if result != tt.expected {
			t.Errorf("IsWordChar(%q) = %v, want %v", tt.c, result, tt.expected)
		}
	}
}

func TestFindWordBoundary(t *testing.T) {
	tests := []struct {
		s        string
		pos      int
		expected int
	}{
		{"hello world", 0, 5},
		{"hello world", 6, 11},
		{"hello world", 5, 5},
		{"abc123", 0, 6},
		{"abc def", 0, 3},
		{"", 0, 0},
	}

	for _, tt := range tests {
		result := FindWordBoundary(tt.s, tt.pos)
		if result != tt.expected {
			t.Errorf("FindWordBoundary(%q, %d) = %d, want %d", tt.s, tt.pos, result, tt.expected)
		}
	}
}

func TestCountBrackets(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		pairs      []BracketPair
		expectedOp int
		expectedCl int
	}{
		{"empty", "", nil, 0, 0},
		{"simple braces", "{ }", nil, 1, 1},
		{"nested braces", "{{ }}", nil, 2, 2},
		{"code with all brackets", "func() { return []int{1, 2, 3} }", nil, 4, 4},
		{"braces only", "func() { return []int{1, 2, 3} }", []BracketPair{{'{', '}'}}, 2, 2},
		{"square brackets only", "arr[0] = arr[1]", []BracketPair{{'[', ']'}}, 2, 2},
		{"parentheses only", "func(a, b) call()", []BracketPair{{'(', ')'}}, 2, 2},
		{"html angle brackets", "<html><body></body></html>", []BracketPair{{'<', '>'}}, 4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opens, closes := CountBrackets(tt.s, tt.pairs)
			if opens != tt.expectedOp || closes != tt.expectedCl {
				t.Errorf("CountBrackets(%q) = (%d, %d), want (%d, %d)", tt.s, opens, closes, tt.expectedOp, tt.expectedCl)
			}
		})
	}
}

func TestMemset(t *testing.T) {
	buf := make([]byte, 10)
	Memset(buf, 'a')
	for i, c := range buf {
		if c != 'a' {
			t.Errorf("Memset: buf[%d] = %q, want 'a'", i, c)
		}
	}
}

func TestMemcmp(t *testing.T) {
	tests := []struct {
		a, b     []byte
		expected int
	}{
		{[]byte("abc"), []byte("abc"), 0},
		{[]byte("abc"), []byte("abd"), -1},
		{[]byte("abd"), []byte("abc"), 1},
		{[]byte("ab"), []byte("abc"), -1},
		{[]byte("abc"), []byte("ab"), 1},
	}

	for _, tt := range tests {
		result := Memcmp(tt.a, tt.b)
		if (result == 0 && tt.expected != 0) ||
			(result < 0 && tt.expected >= 0) ||
			(result > 0 && tt.expected <= 0) {
			t.Errorf("Memcmp(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// Benchmark: SIMD vs Regex ANSI stripping
func BenchmarkStripANSISIMD(b *testing.B) {
	// Generate large input with ANSI codes
	input := strings.Repeat("\x1b[31mHello\x1b[0m World ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		StripANSI(input)
	}
}

func BenchmarkStripANSIRegex(b *testing.B) {
	input := strings.Repeat("\x1b[31mHello\x1b[0m World ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate regex-based stripping (simplified)
		strings.ReplaceAll(input, "\x1b[31m", "")
	}
}

func BenchmarkIndexByteSet(b *testing.B) {
	s := strings.Repeat("hello world ", 1000)
	set := []byte{'a', 'e', 'i', 'o', 'u'}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IndexByteSet(s, set)
	}
}

func BenchmarkCountByteSet(b *testing.B) {
	s := strings.Repeat("hello world ", 1000)
	set := []byte{'a', 'e', 'i', 'o', 'u'}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CountByteSet(s, set)
	}
}

func BenchmarkContainsAny(b *testing.B) {
	s := strings.Repeat("hello world ", 1000)
	substrs := []string{"world", "foo", "bar", "baz"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ContainsAny(s, substrs)
	}
}
