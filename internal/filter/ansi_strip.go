package filter

import (
	"regexp"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ansiStripRe matches the three main families of ANSI / VT escape sequences:
//
//  1. CSI sequences: ESC [ <params> <final>   e.g. color codes, cursor moves
//  2. OSC sequences: ESC ] <text> BEL          e.g. window title, hyperlinks
//  3. Charset sequences: ESC ( or ESC ) <code> e.g. VT100 character-set select
var ansiStripRe = regexp.MustCompile(
	`\x1b\[[0-9;]*[mGKHFABCDsu]` + // CSI: colors, cursor, erase
		`|\x1b\][^\x07]*\x07` + // OSC: title / hyperlinks
		`|\x1b[()][AB012]`, // Charset: G0/G1 select
)

// ANSIStripFilter removes ANSI escape sequences from terminal output. Task #116.
// It complements the existing ANSIFilter (which uses a SIMD path) by providing
// a pure-regex implementation that covers the three sequence families listed above.
type ANSIStripFilter struct{}

// NewANSIStripFilter creates a new ANSIStripFilter.
func NewANSIStripFilter() *ANSIStripFilter {
	return &ANSIStripFilter{}
}

// Name returns the filter name.
func (f *ANSIStripFilter) Name() string {
	return "ansi_strip"
}

// Apply removes ANSI escape sequences from input and returns token savings.
// The filter runs in all modes including ModeNone because escape codes should
// always be removed before further processing.
func (f *ANSIStripFilter) Apply(input string, mode Mode) (string, int) {
	originalTokens := core.EstimateTokens(input)

	output := ansiStripRe.ReplaceAllString(input, "")

	saved := originalTokens - core.EstimateTokens(output)
	if saved < 0 {
		saved = 0
	}
	return output, saved
}
