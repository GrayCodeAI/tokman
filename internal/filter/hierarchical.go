package filter

import (
	"strings"
)

// HierarchicalFilter implements multi-level summarization for large outputs.
// Based on "Hierarchical Context Compression" research - creates a tree-like
// structure where each level provides progressively more detail.
//
// For outputs exceeding a threshold (default 10K lines), this filter:
// 1. Segments the output into logical sections
// 2. Generates summaries at multiple abstraction levels
// 3. Preserves the most important sections verbatim
// 4. Compresses mid-importance sections into summaries
// 5. Drops low-importance sections entirely
type HierarchicalFilter struct {
	// Threshold for triggering hierarchical compression (in lines)
	lineThreshold int
	// Maximum depth of summarization hierarchy
	maxDepth int
	// Whether to use semantic scoring for section importance
	useSemanticScoring bool
}

// NewHierarchicalFilter creates a new hierarchical summarization filter.
func NewHierarchicalFilter() *HierarchicalFilter {
	return &HierarchicalFilter{
		lineThreshold:      500, // 500 lines = ~10K tokens
		maxDepth:           3,   // 3 levels: overview, summary, detail
		useSemanticScoring: true,
	}
}

// Name returns the filter name.
func (f *HierarchicalFilter) Name() string {
	return "hierarchical"
}

// Apply applies hierarchical summarization to large outputs.
func (f *HierarchicalFilter) Apply(input string, mode Mode) (string, int) {
	lines := strings.Split(input, "\n")
	lineCount := len(lines)

	// Don't process small outputs
	if lineCount < f.lineThreshold {
		return input, 0
	}

	// Segment into logical sections
	sections := f.segmentIntoSections(lines)
	if len(sections) == 0 {
		return input, 0
	}

	// Score each section
	scored := f.scoreSections(sections)

	// Build hierarchical output based on mode
	output := f.buildHierarchicalOutput(scored, mode, lineCount)

	tokensSaved := EstimateTokens(input) - EstimateTokens(output)
	if tokensSaved < 0 {
		tokensSaved = 0
	}

	return output, tokensSaved
}

// section represents a logical section of the output
type section struct {
	content   string
	startLine int
	endLine   int
	level     int // 0 = top-level, 1 = nested, etc.
	score     float64
	summary   string
}

// segmentIntoSections divides output into logical sections
func (f *HierarchicalFilter) segmentIntoSections(lines []string) []section {
	var sections []section
	var currentSection []string
	sectionStart := 0
	currentLevel := 0

	for i, line := range lines {
		level := f.detectSectionLevel(line)

		// Check for section boundary
		isBoundary := f.isSectionBoundary(line, i, lines)

		if isBoundary && len(currentSection) > 0 {
			// Save current section
			sections = append(sections, section{
				content:   strings.Join(currentSection, "\n"),
				startLine: sectionStart,
				endLine:   i - 1,
				level:     currentLevel,
			})
			currentSection = nil
			sectionStart = i
			currentLevel = level
		}

		currentSection = append(currentSection, line)
	}

	// Add final section
	if len(currentSection) > 0 {
		sections = append(sections, section{
			content:   strings.Join(currentSection, "\n"),
			startLine: sectionStart,
			endLine:   len(lines) - 1,
			level:     currentLevel,
		})
	}

	return sections
}

// detectSectionLevel determines the nesting level of a section
func (f *HierarchicalFilter) detectSectionLevel(line string) int {
	trimmed := strings.TrimSpace(line)

	// Headers and dividers indicate top-level sections
	if strings.HasPrefix(trimmed, "===") ||
		strings.HasPrefix(trimmed, "---") ||
		strings.HasPrefix(trimmed, "##") {
		return 0
	}

	// Subsection markers
	if strings.HasPrefix(trimmed, "---") ||
		strings.HasPrefix(trimmed, "###") {
		return 1
	}

	return 2 // Default to detail level
}

// isSectionBoundary detects if a line starts a new section
func (f *HierarchicalFilter) isSectionBoundary(line string, idx int, lines []string) bool {
	trimmed := strings.TrimSpace(line)

	// Visual dividers
	if strings.HasPrefix(trimmed, "===") ||
		strings.HasPrefix(trimmed, "---") ||
		strings.HasPrefix(trimmed, "+++") {
		return true
	}

	// Markdown headers
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// Test output boundaries
	if strings.Contains(trimmed, "test result:") ||
		strings.Contains(trimmed, "running ") && strings.Contains(trimmed, " tests") {
		return true
	}

	// Build phase boundaries
	if strings.Contains(trimmed, "Compiling ") ||
		strings.Contains(trimmed, "Building ") ||
		strings.Contains(trimmed, "Finished ") {
		return true
	}

	// File markers (git diff, error messages)
	if strings.HasPrefix(trimmed, "diff --git") ||
		strings.Contains(trimmed, "error[") ||
		strings.Contains(trimmed, "error: ") {
		return true
	}

	// Empty line after substantial content
	if trimmed == "" && idx > 0 && idx < len(lines)-1 {
		prevTrimmed := strings.TrimSpace(lines[idx-1])
		nextTrimmed := strings.TrimSpace(lines[idx+1])
		if prevTrimmed != "" && nextTrimmed != "" {
			// Check if this is a paragraph break (not just spacing)
			if len(prevTrimmed) > 50 || len(nextTrimmed) > 50 {
				return true
			}
		}
	}

	return false
}

// scoreSections assigns importance scores to each section
func (f *HierarchicalFilter) scoreSections(sections []section) []section {
	if !f.useSemanticScoring {
		// Uniform scoring without semantic analysis
		for i := range sections {
			sections[i].score = 0.5
		}
		return sections
	}

	// Use semantic scoring
	sf := NewSemanticFilter()
	for i := range sections {
		sections[i].score = f.calculateSectionScore(sections[i], sf)
		sections[i].summary = f.generateSectionSummary(sections[i])
	}

	return sections
}

// calculateSectionScore computes importance for a section
func (f *HierarchicalFilter) calculateSectionScore(s section, sf *SemanticFilter) float64 {
	score := 0.0
	content := s.content
	lower := strings.ToLower(content)

	// High importance indicators (weight: 1.0)
	highKeywords := []string{
		"error", "failed", "failure", "fatal", "panic",
		"exception", "critical", "bug", "security",
		"diff --git", "deleted", "added", "modified",
	}
	for _, kw := range highKeywords {
		if strings.Contains(lower, kw) {
			score += 0.2
		}
	}

	// Medium importance indicators (weight: 0.5)
	mediumKeywords := []string{
		"warning", "deprecated", "todo", "fixme",
		"test", "assert", "expect", "verify",
		"function", "class", "struct", "interface",
	}
	for _, kw := range mediumKeywords {
		if strings.Contains(lower, kw) {
			score += 0.1
		}
	}

	// File references (very important for debugging)
	if strings.Contains(content, ".go:") ||
		strings.Contains(content, ".rs:") ||
		strings.Contains(content, ".py:") ||
		strings.Contains(content, ".js:") ||
		strings.Contains(content, ".ts:") {
		score += 0.3
	}

	// Stack traces
	if strings.Contains(content, "at ") ||
		strings.Contains(content, "Traceback") ||
		strings.Contains(content, "stack trace") {
		score += 0.4
	}

	// Length penalty (longer sections are less dense)
	lineCount := s.endLine - s.startLine + 1
	if lineCount > 100 {
		score *= 0.8
	}

	// Clamp to [0, 1]
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// generateSectionSummary creates a one-line summary of a section
func (f *HierarchicalFilter) generateSectionSummary(s section) string {
	lines := strings.Split(s.content, "\n")
	if len(lines) == 0 {
		return "[empty section]"
	}

	// Find the most representative line
	var bestLine string
	bestScore := -1.0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Score this line
		score := f.scoreLineForSummary(trimmed)
		if score > bestScore {
			bestScore = score
			bestLine = trimmed
		}
	}

	if bestLine == "" {
		return "[section]"
	}

	// Truncate if needed
	if len(bestLine) > 80 {
		return bestLine[:77] + "..."
	}

	return bestLine
}

// scoreLineForSummary rates how good a line is as a summary
func (f *HierarchicalFilter) scoreLineForSummary(line string) float64 {
	lower := strings.ToLower(line)
	score := 0.0

	// Prefer lines with key information
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
		score += 0.5
	}
	if strings.Contains(lower, "test") || strings.Contains(lower, "pass") {
		score += 0.3
	}

	// Prefer shorter lines
	if len(line) < 60 {
		score += 0.2
	}

	// Avoid pure symbols or numbers
	hasLetters := false
	for _, r := range line {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetters = true
			break
		}
	}
	if !hasLetters {
		score -= 0.5
	}

	return score
}

// buildHierarchicalOutput constructs the compressed output
func (f *HierarchicalFilter) buildHierarchicalOutput(sections []section, mode Mode, totalLines int) string {
	var output []string

	// Add header showing compression stats
	output = append(output, f.formatHeader(totalLines, len(sections)))

	// Determine thresholds based on mode
	highThreshold, midThreshold := f.getThresholds(mode)

	for _, s := range sections {
		switch {
		case s.score >= highThreshold:
			// Keep full content
			output = append(output, s.content)

		case s.score >= midThreshold:
			// Keep summary with line range
			output = append(output, f.formatSummarySection(s))

		default:
			// Skip low-importance sections
			// Optionally: add a one-liner indicating skipped content
		}
	}

	return strings.Join(output, "\n")
}

// formatHeader creates the compression header
func (f *HierarchicalFilter) formatHeader(totalLines, sectionCount int) string {
	return "\n[Hierarchical Summary: " + itoa(totalLines) + " lines → " + itoa(sectionCount) + " sections]\n"
}

// formatSummarySection formats a section as a summary
func (f *HierarchicalFilter) formatSummarySection(s section) string {
	lineCount := s.endLine - s.startLine + 1
	return "\n├─ [L" + itoa(s.startLine+1) + "-" + itoa(s.endLine+1) + "] " + s.summary + " (" + itoa(lineCount) + " lines, score: " + f.formatScore(s.score) + ")\n"
}

// formatScore formats a score to 2 decimal places
func (f *HierarchicalFilter) formatScore(score float64) string {
	// Simple formatting without strconv
	intPart := int(score * 100)
	return itoa(intPart/100) + "." + itoaPad(intPart%100, 2)
}

// itoaPad pads an integer with zeros
func itoaPad(n, width int) string {
	s := itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

// getThresholds returns score thresholds for different compression levels
func (f *HierarchicalFilter) getThresholds(mode Mode) (high, mid float64) {
	switch mode {
	case ModeAggressive:
		return 0.7, 0.4 // Only high-value content
	case ModeMinimal:
		return 0.5, 0.25
	default:
		return 0.6, 0.3
	}
}

// SetLineThreshold configures the line threshold for hierarchical compression
func (f *HierarchicalFilter) SetLineThreshold(threshold int) {
	f.lineThreshold = threshold
}

// SetMaxDepth configures the maximum summarization depth
func (f *HierarchicalFilter) SetMaxDepth(depth int) {
	f.maxDepth = depth
}
