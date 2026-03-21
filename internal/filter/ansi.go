package filter

import "github.com/GrayCodeAI/tokman/internal/simd"

// ANSIFilter strips ANSI escape sequences from output.
// Uses SIMD-optimized byte scanning for ~10-40x speedup over regex.
type ANSIFilter struct{}

// NewANSIFilter creates a new ANSI filter.
func NewANSIFilter() *ANSIFilter {
	return &ANSIFilter{}
}

// Name returns the filter name.
func (f *ANSIFilter) Name() string {
	return "ansi"
}

// Apply strips ANSI sequences and returns token savings.
func (f *ANSIFilter) Apply(input string, mode Mode) (string, int) {
	original := len(input)

	// Use SIMD-optimized ANSI stripping (10-40x faster than regex)
	output := simd.StripANSI(input)

	// Calculate tokens saved (ANSI codes don't count as meaningful tokens)
	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4 // heuristic

	return output, tokensSaved
}

// StripANSI is a utility function to strip ANSI codes from a string.
// Delegates to SIMD-optimized implementation.
func StripANSI(input string) string {
	return simd.StripANSI(input)
}

// HasANSI checks if the input contains ANSI escape sequences.
func HasANSI(input string) bool {
	return simd.HasANSI(input)
}
