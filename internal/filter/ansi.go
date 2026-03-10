package filter

import "regexp"

// ANSI escape sequence pattern
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Additional ANSI patterns for comprehensive stripping
var (
	// OSC sequences (Operating System Command)
	oscPattern = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)

	// CSI sequences (Control Sequence Introducer)
	csiPattern = regexp.MustCompile(`\x1b\[[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]`)

	// SGR sequences (Select Graphic Rendition) - most common
	sgrPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

	// Link escape sequences (used by modern terminals)
	linkPattern = regexp.MustCompile(`\x1b]8;;[^\x1b]*\x1b\\`)
)

// ANSIFilter strips ANSI escape sequences from output.
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

	// Strip all ANSI sequences
	output := ansiPattern.ReplaceAllString(input, "")
	output = oscPattern.ReplaceAllString(output, "")
	output = csiPattern.ReplaceAllString(output, "")
	output = sgrPattern.ReplaceAllString(output, "")
	output = linkPattern.ReplaceAllString(output, "")

	// Calculate tokens saved (ANSI codes don't count as meaningful tokens)
	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4 // heuristic

	return output, tokensSaved
}

// StripANSI is a utility function to strip ANSI codes from a string.
func StripANSI(input string) string {
	output := ansiPattern.ReplaceAllString(input, "")
	output = oscPattern.ReplaceAllString(output, "")
	output = linkPattern.ReplaceAllString(output, "")
	return output
}

// HasANSI checks if the input contains ANSI escape sequences.
func HasANSI(input string) bool {
	return ansiPattern.MatchString(input) ||
		oscPattern.MatchString(input) ||
		linkPattern.MatchString(input)
}
