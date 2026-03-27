package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// NumericCompressFilter compresses large numeric arrays and sequences in output.
// Detects:
//  1. Arithmetic sequences → range notation: [1,2,3,4,5] → range(1,6)
//  2. Repeated values → × notation: [0,0,0,0,0] → [0 ×5]
//  3. Sorted runs in larger arrays → abbreviated with ellipsis
//  4. Long float arrays (benchmark/profiling output) → statistical summary
type NumericCompressFilter struct{}

var (
	// Inline numeric array: [1, 2, 3, ...] or (1, 2, 3, ...) — at least 6 numbers
	numericArrayRe = regexp.MustCompile(`[\[(](\s*-?\d+(?:\.\d+)?\s*(?:,\s*-?\d+(?:\.\d+)?\s*){5,})[\])]`)
	// Standalone line that is just numbers separated by commas/spaces/tabs (data tables)
	numericDataLineRe = regexp.MustCompile(`^[\s\t]*-?\d+(?:\.\d+)?(?:[\s,\t]+-?\d+(?:\.\d+)?){4,}[\s\t]*$`)
	// Benchmark/profiling lines: "  1234567 ns/op  1024 B/op  3 allocs/op"
	benchLineRe = regexp.MustCompile(`^\s*\d[\d,]*\s+(?:ns/op|µs/op|ms/op|B/op|allocs/op)`)
)

// NewNumericCompressFilter creates a new numeric data compression filter.
func NewNumericCompressFilter() *NumericCompressFilter {
	return &NumericCompressFilter{}
}

// Name returns the filter name.
func (f *NumericCompressFilter) Name() string {
	return "numeric_compress"
}

// Apply compresses numeric sequences and arrays.
func (f *NumericCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input

	// Compress inline numeric arrays
	output = numericArrayRe.ReplaceAllStringFunc(output, func(m string) string {
		bracket := m[0]
		closeBracket := m[len(m)-1]
		inner := m[1 : len(m)-1]
		compressed := f.compressNumericList(inner, mode)
		if compressed == inner {
			return m
		}
		return string(bracket) + compressed + string(closeBracket)
	})

	// Compress standalone numeric data lines (tables/matrices)
	lines := strings.Split(output, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		line := lines[i]
		if numericDataLineRe.MatchString(line) {
			// Collect run of numeric data lines
			runStart := i
			for i < len(lines) && numericDataLineRe.MatchString(lines[i]) {
				i++
			}
			runLen := i - runStart
			if runLen > 5 || mode == ModeAggressive && runLen > 2 {
				// Show first 2 and last 1 with summary
				result = append(result, lines[runStart])
				if runLen > 1 {
					result = append(result, lines[runStart+1])
				}
				if runLen > 3 {
					result = append(result, fmt.Sprintf("    ... [%d more data rows omitted] ...", runLen-3))
					result = append(result, lines[i-1])
				}
			} else {
				result = append(result, lines[runStart:i]...)
			}
		} else {
			result = append(result, line)
			i++
		}
	}
	output = strings.Join(result, "\n")

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// compressNumericList compresses a comma-separated list of numbers.
func (f *NumericCompressFilter) compressNumericList(inner string, mode Mode) string {
	// Parse numbers
	parts := strings.Split(inner, ",")
	nums := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return inner // not all numbers — don't compress
		}
		nums = append(nums, n)
	}
	if len(nums) < 6 {
		return inner
	}

	// Check for repeated values
	if compressed := tryRepeat(nums); compressed != "" {
		return compressed
	}

	// Check for arithmetic sequence (integers only)
	if compressed := tryArithmeticSeq(nums); compressed != "" {
		return compressed
	}

	// For aggressive mode: abbreviate long lists with stats
	if mode == ModeAggressive && len(nums) > 10 {
		return abbreviateNums(nums)
	}

	return inner
}

// tryRepeat checks if all values are identical.
func tryRepeat(nums []float64) string {
	v := nums[0]
	for _, n := range nums[1:] {
		if n != v {
			return ""
		}
	}
	s := formatNum(v)
	return fmt.Sprintf("%s ×%d", s, len(nums))
}

// tryArithmeticSeq checks for integer arithmetic sequences.
func tryArithmeticSeq(nums []float64) string {
	if len(nums) < 4 {
		return ""
	}
	// Must be integers
	for _, n := range nums {
		if n != float64(int64(n)) {
			return ""
		}
	}
	step := nums[1] - nums[0]
	for i := 2; i < len(nums); i++ {
		if nums[i]-nums[i-1] != step {
			return ""
		}
	}
	start := int64(nums[0])
	end := int64(nums[len(nums)-1])
	if step == 1 {
		return fmt.Sprintf("range(%d, %d)", start, end+1)
	}
	if step == -1 {
		return fmt.Sprintf("range(%d, %d, -1)", start, end-1)
	}
	return fmt.Sprintf("range(%d, %d, %d)", start, end+int64(step), int64(step))
}

// abbreviateNums produces a brief stats summary for a long float array.
func abbreviateNums(nums []float64) string {
	min, max, sum := nums[0], nums[0], 0.0
	for _, n := range nums {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
		sum += n
	}
	mean := sum / float64(len(nums))
	return fmt.Sprintf("%s...%s (mean=%s, n=%d)",
		formatNum(nums[0]), formatNum(nums[len(nums)-1]), formatNum(mean), len(nums))
}

func formatNum(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'g', 4, 64)
}
