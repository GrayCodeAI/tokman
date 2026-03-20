package filter

import (
	"regexp"
	"strings"
)

// progressRegex matches common progress bar patterns.
var progressRegex = regexp.MustCompile(`(\d+/\d+)|(\d+\.\d+\s+[KMG]?B/s)`)

// progressLineRegex matches lines that are entirely progress indicators.
var progressLineRegex = regexp.MustCompile(`^\s*(\[?[#█░▒▓∎▸▹●○━─═]+\]\s*\d+%?\s*)+\s*$`)

// FilterProgressBars removes progress bar lines from output.
// R62: Progress bars are noisy and consume tokens.
func FilterProgressBars(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	for _, line := range lines {
		// Skip lines that are entirely progress bars
		if progressLineRegex.MatchString(line) {
			continue
		}

		// Clean inline progress indicators
		cleaned := cleanInlineProgress(line)
		if strings.TrimSpace(cleaned) != "" {
			result = append(result, cleaned)
		}
	}

	return strings.Join(result, "\n")
}

// cleanInlineProgress removes progress indicators from within a line.
func cleanInlineProgress(line string) string {
	// Remove ANSI first
	line = StripANSI(line)

	// Remove progress bar patterns
	line = progressRegex.ReplaceAllString(line, "")

	// Clean up extra whitespace
	line = strings.Join(strings.Fields(line), " ")

	return line
}

// FilterNoisyOutput removes common noise from terminal output.
func FilterNoisyOutput(input string) string {
	input = StripANSI(input)
	input = FilterProgressBars(input)

	// Remove carriage returns (common in progress output)
	input = strings.ReplaceAll(input, "\r", "")

	// Remove excessive blank lines
	lines := strings.Split(input, "\n")
	var result []string
	blankCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
