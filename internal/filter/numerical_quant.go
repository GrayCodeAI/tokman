package filter

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// NumericalQuantizer compresses numerical data in structured output.
// Research Source: "CompactPrompt: A Unified Pipeline for Prompt Data Compression" (Oct 2025)
// Key Innovation: Apply uniform quantization to numerical columns while preserving
// semantic relationships, achieving significant token savings on structured data.
//
// This filter detects tables, metrics, statistics, and numerical data in output
// and applies precision reduction and formatting compression.
type NumericalQuantizer struct {
	config NumericalConfig
}

// NumericalConfig holds configuration for numerical quantization
type NumericalConfig struct {
	// Enabled controls whether the filter is active
	Enabled bool

	// DecimalPlaces limits decimal precision (e.g., 2 = max 2 decimal places)
	DecimalPlaces int

	// CompressLargeNumbers replaces large numbers with K/M/B suffixes
	CompressLargeNumbers bool

	// LargeNumberThreshold is the threshold for large number compression
	LargeNumberThreshold int

	// CompressPercentages simplifies percentage display
	CompressPercentages bool

	// MinContentLength is minimum content length to apply
	MinContentLength int
}

// DefaultNumericalConfig returns default configuration
func DefaultNumericalConfig() NumericalConfig {
	return NumericalConfig{
		Enabled:              true,
		DecimalPlaces:        2,
		CompressLargeNumbers: true,
		LargeNumberThreshold: 1000,
		CompressPercentages:  true,
		MinContentLength:     50,
	}
}

// NewNumericalQuantizer creates a new numerical quantizer
func NewNumericalQuantizer() *NumericalQuantizer {
	return &NumericalQuantizer{
		config: DefaultNumericalConfig(),
	}
}

// Name returns the filter name
func (n *NumericalQuantizer) Name() string {
	return "numerical_quant"
}

// Apply applies numerical quantization to the input
func (n *NumericalQuantizer) Apply(input string, mode Mode) (string, int) {
	if !n.config.Enabled || mode == ModeNone {
		return input, 0
	}

	if len(input) < n.config.MinContentLength {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	output := input

	// Compress decimal numbers
	if n.config.DecimalPlaces > 0 {
		output = n.compressDecimals(output, mode)
	}

	// Compress large numbers
	if n.config.CompressLargeNumbers {
		output = n.compressLargeNumbers(output)
	}

	// Compress percentages
	if n.config.CompressPercentages {
		output = n.compressPercentages(output)
	}

	// Compress byte sizes (e.g., "1048576 bytes" → "1MB")
	output = n.compressByteSizes(output)

	// Compress timestamps
	output = n.compressTimestamps(output, mode)

	finalTokens := core.EstimateTokens(output)
	saved := originalTokens - finalTokens
	if saved < 2 {
		return input, 0
	}

	return output, saved
}

// compressDecimals reduces decimal precision using pre-compiled regexes.
func (n *NumericalQuantizer) compressDecimals(input string, mode Mode) string {
	maxDecimals := n.config.DecimalPlaces
	re := numDecimalMinRe
	if mode == ModeAggressive {
		maxDecimals = 1
		re = numDecimalAggrRe
	}
	return re.ReplaceAllStringFunc(input, func(match string) string {
		val, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return match
		}
		return strconv.FormatFloat(val, 'f', maxDecimals, 64)
	})
}

// numLargeRe matches a number token which may be a plain integer or a float.
// We capture the whole potential float (digits, optional dot+digits) so we can
// skip entries that have a fractional part in compressLargeNumbers.
var numLargeRe = regexp.MustCompile(`\b(\d{4,})(\.\d+)?\b`)

// numDecimalMinRe / numDecimalAggrRe match floats with more precision than needed.
var (
	numDecimalMinRe  = regexp.MustCompile(`\d+\.\d{4,}`)  // >3 decimals → keep 2
	numDecimalAggrRe = regexp.MustCompile(`\d+\.\d{3,}`)  // >2 decimals → keep 1
	numPercentRe     = regexp.MustCompile(`(\d+\.\d{3,})%`)
)

// compressLargeNumbers replaces large integers with K/M/B suffixes.
// Floats (e.g. "12345.68") are left unchanged because their decimal part is
// already handled by the decimal-precision passes.
func (n *NumericalQuantizer) compressLargeNumbers(input string) string {
	threshold := n.config.LargeNumberThreshold
	return numLargeRe.ReplaceAllStringFunc(input, func(match string) string {
		// If the matched text contains a dot it's a float — skip it.
		if strings.Contains(match, ".") {
			return match
		}
		val, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return match
		}

		abs := math.Abs(val)
		sign := ""
		if val < 0 {
			sign = "-"
		}

		switch {
		case abs >= 1e9:
			return sign + strconv.FormatFloat(abs/1e9, 'f', 1, 64) + "B"
		case abs >= 1e6:
			return sign + strconv.FormatFloat(abs/1e6, 'f', 1, 64) + "M"
		case abs >= float64(threshold):
			return sign + strconv.FormatFloat(abs/1e3, 'f', 1, 64) + "K"
		default:
			return match
		}
	})
}

// compressPercentages simplifies percentage display.
func (n *NumericalQuantizer) compressPercentages(input string) string {
	// "95.123456%" → "95.1%"
	return numPercentRe.ReplaceAllStringFunc(input, func(match string) string {
		val, err := strconv.ParseFloat(match[:len(match)-1], 64)
		if err != nil {
			return match
		}
		return strconv.FormatFloat(val, 'f', 1, 64) + "%"
	})
}

// byteSizeRe matches byte/octet size expressions (case-insensitive, single pass)
var byteSizeRe = regexp.MustCompile(`(?i)(\d+)\s*(?:bytes|octets)`)

// compressByteSizes replaces verbose byte size displays
func (n *NumericalQuantizer) compressByteSizes(input string) string {
	return byteSizeRe.ReplaceAllStringFunc(input, func(match string) string {
		sub := byteSizeRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		val, err := strconv.ParseFloat(sub[1], 64)
		if err != nil {
			return match
		}

		switch {
		case val >= 1073741824:
			return strconv.FormatFloat(val/1073741824, 'f', 1, 64) + "GB"
		case val >= 1048576:
			return strconv.FormatFloat(val/1048576, 'f', 1, 64) + "MB"
		case val >= 1024:
			return strconv.FormatFloat(val/1024, 'f', 1, 64) + "KB"
		default:
			return match
		}
	})
}

var (
	numTimestampRe = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):\d{2}[\d.Z]*`)
	numMonthNameRe = regexp.MustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),?\s+(\d{4})`)
)

// compressTimestamps simplifies verbose timestamps (aggressive mode only).
func (n *NumericalQuantizer) compressTimestamps(input string, mode Mode) string {
	if mode != ModeAggressive {
		return input
	}

	// "2024-01-15T10:30:45.123Z" → "01-15 10:30"
	result := numTimestampRe.ReplaceAllString(input, "$2-$3 $4:$5")

	// Compress verbose date strings
	re2 := numMonthNameRe
	monthMap := map[string]string{
		"january": "Jan", "february": "Feb", "march": "Mar",
		"april": "Apr", "may": "May", "june": "Jun",
		"july": "Jul", "august": "Aug", "september": "Sep",
		"october": "Oct", "november": "Nov", "december": "Dec",
	}
	return re2.ReplaceAllStringFunc(result, func(match string) string {
		sub := re2.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		abbr := monthMap[strings.ToLower(sub[1])]
		return abbr + " " + sub[2] + " " + sub[3]
	})
}
