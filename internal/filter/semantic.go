package filter

import (
	"bufio"
	"math"
	"strings"
	"unicode"
)

// SemanticFilter prunes low-information segments using statistical analysis.
// Based on "Selective Context" (Li et al., 2024) - uses self-information
// and information density to identify low-value content.
type SemanticFilter struct {
	// Importance keywords for different contexts
	highImportanceKeywords   []string
	mediumImportanceKeywords []string
	lowImportanceKeywords    []string
}

// NewSemanticFilter creates a new semantic filter.
func NewSemanticFilter() *SemanticFilter {
	return &SemanticFilter{
		highImportanceKeywords: []string{
			"error", "failed", "failure", "fatal", "panic",
			"exception", "critical", "bug", "issue",
			"test", "assert", "expect", "verify",
			"diff", "change", "modified", "deleted", "added",
		},
		mediumImportanceKeywords: []string{
			"warning", "warn", "deprecated", "caution",
			"info", "note", "todo", "fixme", "hack",
			"function", "class", "method", "variable", "import",
			"build", "compile", "deploy", "release",
		},
		lowImportanceKeywords: []string{
			"success", "ok", "done", "complete", "finished",
			"verbose", "debug", "trace", "silly",
		},
	}
}

// Name returns the filter name.
func (f *SemanticFilter) Name() string {
	return "semantic"
}

// Apply applies semantic pruning to the input.
func (f *SemanticFilter) Apply(input string, mode Mode) (string, int) {
	original := len(input)

	// Don't process very short inputs
	if original < 100 {
		return input, 0
	}

	// Segment the input by logical boundaries
	segments := f.segmentOutput(input)
	if len(segments) == 0 {
		return input, 0
	}

	// Score each segment
	scored := make([]scoredSegment, len(segments))
	for i, seg := range segments {
		scored[i] = scoredSegment{
			content: seg.content,
			score:   f.scoreSegment(seg.content),
			lines:   seg.lines,
		}
	}

	// Determine threshold based on mode
	threshold := f.getThreshold(mode)

	// Filter segments based on scores
	var kept []string
	for _, seg := range scored {
		if seg.score >= threshold {
			// Keep full segment
			kept = append(kept, seg.content)
		} else if seg.score >= threshold*0.5 && seg.lines > 3 {
			// Partial compression: keep first and last lines
			compressed := f.compressSegment(seg.content)
			kept = append(kept, compressed)
		}
		// Below 0.5*threshold: drop entirely
	}

	output := strings.Join(kept, "\n")
	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// segment represents a logical segment of output
type segment struct {
	content string
	lines   int
}

// scoredSegment is a segment with its importance score
type scoredSegment struct {
	content string
	score   float64
	lines   int
}

// segmentOutput splits output into logical segments using streaming
// for memory efficiency with large contexts
func (f *SemanticFilter) segmentOutput(input string) []segment {
	var segments []segment
	var currentSegment strings.Builder
	var prevLine string
	lineInSegment := 0

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for segment boundaries (streaming-compatible)
		isBoundary := f.isSegmentBoundaryStreaming(line, prevLine)

		if isBoundary && currentSegment.Len() > 0 {
			// Save current segment
			segments = append(segments, segment{
				content: currentSegment.String(),
				lines:   lineInSegment,
			})
			currentSegment.Reset()
			lineInSegment = 0
		}

		if currentSegment.Len() > 0 {
			currentSegment.WriteString("\n")
		}
		currentSegment.WriteString(line)
		lineInSegment++
		prevLine = line
	}

	if err := scanner.Err(); err != nil {
		return segments
	}

	// Add final segment
	if currentSegment.Len() > 0 {
		segments = append(segments, segment{
			content: currentSegment.String(),
			lines:   lineInSegment,
		})
	}

	return segments
}

// isSegmentBoundaryStreaming determines if a line is a segment boundary (streaming version)
func (f *SemanticFilter) isSegmentBoundaryStreaming(line, prevLine string) bool {
	trimmed := strings.TrimSpace(line)

	// Empty lines are often boundaries (but not consecutive ones)
	if trimmed == "" && prevLine != "" && strings.TrimSpace(prevLine) != "" {
		return true
	}

	// Structural markers
	if strings.HasPrefix(trimmed, "---") ||
		strings.HasPrefix(trimmed, "===") ||
		strings.HasPrefix(trimmed, "+++") {
		return true
	}

	// Test output boundaries
	if strings.Contains(trimmed, "test result:") ||
		strings.Contains(trimmed, "PASS") ||
		strings.Contains(trimmed, "FAIL") ||
		strings.Contains(trimmed, "running") && strings.Contains(trimmed, "tests") {
		return true
	}

	// Build phase boundaries
	if strings.Contains(trimmed, "Compiling") ||
		strings.Contains(trimmed, "Building") ||
		strings.Contains(trimmed, "Generating") ||
		strings.Contains(trimmed, "Finished") {
		return true
	}

	return false
}

// isSegmentBoundary is the legacy method for backward compatibility with tests
func (f *SemanticFilter) isSegmentBoundary(line string, lines []string, idx int) bool {
	prevLine := ""
	if idx > 0 {
		prevLine = lines[idx-1]
	}
	return f.isSegmentBoundaryStreaming(line, prevLine)
}

// scoreSegment calculates information density for a segment
// Higher score = more important to keep
func (f *SemanticFilter) scoreSegment(segment string) float64 {
	if strings.TrimSpace(segment) == "" {
		return 0.0
	}

	// Calculate multiple metrics
	uniqueRatio := f.uniqueTokenRatio(segment)
	keywordScore := f.keywordDensity(segment)
	structuralScore := f.structuralMarkers(segment)
	entropy := f.charEntropy(segment)

	// Weighted combination
	// Keywords are most important for coding agent context
	score := 0.3*uniqueRatio + 0.35*keywordScore + 0.2*structuralScore + 0.15*entropy

	return score
}

// uniqueTokenRatio measures vocabulary diversity
// Higher ratio = more unique information
func (f *SemanticFilter) uniqueTokenRatio(segment string) float64 {
	words := strings.Fields(segment)
	if len(words) == 0 {
		return 0.0
	}

	seen := make(map[string]bool)
	for _, word := range words {
		// Normalize: lowercase and strip punctuation
		normalized := strings.ToLower(strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		}))
		if normalized != "" {
			seen[normalized] = true
		}
	}

	ratio := float64(len(seen)) / float64(len(words))
	return ratio
}

// keywordDensity scores based on importance keywords
func (f *SemanticFilter) keywordDensity(segment string) float64 {
	lower := strings.ToLower(segment)

	highCount := 0
	mediumCount := 0
	lowCount := 0

	// Count high importance keywords (weight: 1.0)
	for _, kw := range f.highImportanceKeywords {
		highCount += strings.Count(lower, kw)
	}

	// Count medium importance keywords (weight: 0.5)
	for _, kw := range f.mediumImportanceKeywords {
		mediumCount += strings.Count(lower, kw)
	}

	// Count low importance keywords (weight: -0.3, these reduce score)
	for _, kw := range f.lowImportanceKeywords {
		lowCount += strings.Count(lower, kw)
	}

	// Normalize by segment length
	length := float64(len(segment))
	if length == 0 {
		return 0.0
	}

	// Calculate weighted density
	score := (float64(highCount)*1.0 + float64(mediumCount)*0.5 - float64(lowCount)*0.3) / (length / 100.0)

	// Clamp to [0, 1]
	if score < 0 {
		return 0.0
	}
	if score > 1 {
		return 1.0
	}

	return score
}

// structuralMarkers detects important structural elements
func (f *SemanticFilter) structuralMarkers(segment string) float64 {
	score := 0.0

	// File paths and function names (very important for context)
	if strings.Contains(segment, ".go:") ||
		strings.Contains(segment, ".rs:") ||
		strings.Contains(segment, ".py:") ||
		strings.Contains(segment, ".js:") ||
		strings.Contains(segment, ".ts:") {
		score += 0.3
	}

	// Line numbers (indicates actionable error locations)
	if strings.Contains(segment, "line ") ||
		strings.Contains(segment, ":line") {
		score += 0.2
	}

	// Stack traces (critical for debugging)
	if strings.Contains(segment, "at ") ||
		strings.Contains(segment, "Stack trace") ||
		strings.Contains(segment, "Traceback") {
		score += 0.4
	}

	// Diff hunks (essential for code review)
	if strings.HasPrefix(strings.TrimSpace(segment), "@@") ||
		strings.HasPrefix(strings.TrimSpace(segment), "diff --git") {
		score += 0.3
	}

	// Clamp to [0, 1]
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// charEntropy measures information content via character distribution
// Higher entropy = more random/information-rich
func (f *SemanticFilter) charEntropy(segment string) float64 {
	if len(segment) == 0 {
		return 0.0
	}

	// Count character frequencies
	freq := make(map[rune]int)
	for _, r := range segment {
		freq[r]++
	}

	// Calculate Shannon entropy
	total := float64(len(segment))
	entropy := 0.0

	for _, count := range freq {
		p := float64(count) / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	// Normalize (max entropy for ASCII is ~7 bits, for printable chars ~6 bits)
	normalized := entropy / 6.0
	if normalized > 1.0 {
		normalized = 1.0
	}

	return normalized
}

// getThreshold returns the score threshold for keeping segments
func (f *SemanticFilter) getThreshold(mode Mode) float64 {
	switch mode {
	case ModeAggressive:
		return 0.4 // Keep only high-value segments
	case ModeMinimal:
		return 0.25 // Moderate filtering
	default:
		return 0.15 // Light filtering (ModeNone)
	}
}

// compressSegment creates a compressed version of a segment
// Keeps first and last lines with an ellipsis
func (f *SemanticFilter) compressSegment(segment string) string {
	lines := strings.Split(segment, "\n")
	if len(lines) <= 3 {
		return segment
	}

	// Keep first line, ellipsis, last line
	var result []string
	result = append(result, lines[0])
	result = append(result, "... ["+itoa(len(lines)-2)+" lines omitted]")
	result = append(result, lines[len(lines)-1])

	return strings.Join(result, "\n")
}

// itoa converts int to string (avoids strconv import)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}

	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if neg {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
