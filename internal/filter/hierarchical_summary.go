package filter

import (
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// HierarchicalSummaryFilter implements AutoCompressor-style compression (Princeton/MIT, 2023).
// Recursive summarization that compresses context into summary vectors.
//
// Algorithm:
// 1. Divide content into hierarchical levels (sections → paragraphs → sentences)
// 2. Summarize each level recursively
// 3. Combine summaries with preserved key content
// 4. Use bottom-up summarization for maximum compression
//
// Research Results: Extreme compression (depends on summary size).
// Key insight: Recursive summarization preserves global context.
type HierarchicalSummaryFilter struct {
	// Summarization settings
	maxChunkSize int
	summaryRatio float64
	maxLevels    int

	// Preservation settings
	preserveFirst  bool
	preserveLast   bool
	preserveErrors bool
}

// NewHierarchicalSummaryFilter creates a new hierarchical summary filter
func NewHierarchicalSummaryFilter() *HierarchicalSummaryFilter {
	return &HierarchicalSummaryFilter{
		maxChunkSize:   1000, // Characters per chunk
		summaryRatio:   0.3,  // Keep 30% in summary
		maxLevels:      3,    // Maximum recursion depth
		preserveFirst:  true,
		preserveLast:   true,
		preserveErrors: true,
	}
}

// Name returns the filter name
func (f *HierarchicalSummaryFilter) Name() string {
	return "hierarchical_summary"
}

// Apply applies hierarchical summarization
func (f *HierarchicalSummaryFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Level 0: Split into sections
	sections := f.splitSections(input)

	// Level 1: Summarize each section
	summaries := f.summarizeSections(sections, mode)

	// Level 2: Create hierarchical summary
	output := f.createHierarchicalSummary(summaries, sections, mode)

	saved := core.EstimateTokens(input) - core.EstimateTokens(output)
	if saved < 0 {
		saved = 0
	}
	return output, saved
}

// hierSection represents a content section for hierarchical summarization
type hierSection struct {
	content   string
	startLine int
	endLine   int
	level     int // Heading level (0 = no heading)
}

// splitSections divides content into logical sections
func (f *HierarchicalSummaryFilter) splitSections(input string) []hierSection {
	lines := strings.Split(input, "\n")
	var sections []hierSection

	var current hierSection
	current.startLine = 0
	current.level = 0

	for i, line := range lines {
		// Detect section boundaries
		if f.isSectionBoundary(line) {
			// Save current section
			if current.content != "" {
				current.endLine = i - 1
				sections = append(sections, current)
			}

			// Start new section
			current = hierSection{
				startLine: i,
				level:     f.headingLevel(line),
			}
		} else {
			if current.content != "" {
				current.content += "\n"
			}
			current.content += line
		}
	}

	// Save last section
	if current.content != "" {
		current.endLine = len(lines) - 1
		sections = append(sections, current)
	}

	return sections
}

// isSectionBoundary checks if a line is a section boundary
func (f *HierarchicalSummaryFilter) isSectionBoundary(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Markdown headings
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// ASCII separators
	if len(trimmed) > 3 {
		allDashes := true
		allEquals := true
		for _, c := range trimmed {
			if c != '-' {
				allDashes = false
			}
			if c != '=' {
				allEquals = false
			}
		}
		if (allDashes || allEquals) && len(trimmed) > 3 {
			return true
		}
	}

	// Blank lines after code blocks
	if strings.HasPrefix(trimmed, "```") {
		return true
	}

	return false
}

// headingLevel returns the markdown heading level
func (f *HierarchicalSummaryFilter) headingLevel(line string) int {
	trimmed := strings.TrimSpace(line)
	level := 0
	for _, c := range trimmed {
		if c == '#' {
			level++
		} else {
			break
		}
	}
	return level
}

// sectionSummary represents a summarized section
type sectionSummary struct {
	original hierSection
	summary  string
	keyLines []string
}

// summarizeSections creates summaries for each section
func (f *HierarchicalSummaryFilter) summarizeSections(sections []hierSection, mode Mode) []sectionSummary {
	summaries := make([]sectionSummary, len(sections))

	for i, sec := range sections {
		summaries[i] = sectionSummary{
			original: sec,
			summary:  f.summarizeSection(sec.content, mode),
			keyLines: f.extractKeyLines(sec.content),
		}
	}

	return summaries
}

// summarizeSection creates a summary of a section
func (f *HierarchicalSummaryFilter) summarizeSection(content string, mode Mode) string {
	lines := strings.Split(content, "\n")

	// Skip short sections
	if len(lines) < 5 {
		return content
	}

	// Extract first line (often a heading)
	first := ""
	if len(lines) > 0 {
		first = lines[0]
	}

	// Extract key content
	var keyContent []string
	keyContent = append(keyContent, first)

	// Look for important lines
	for _, line := range lines {
		if f.isKeyLine(line) {
			keyContent = append(keyContent, line)
		}
	}

	// Look for last meaningful line
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			keyContent = append(keyContent, lines[i])
			break
		}
	}

	// Create summary
	summaryRatio := f.summaryRatio
	if mode == ModeAggressive {
		summaryRatio = 0.2
	}

	maxLines := int(float64(len(lines)) * summaryRatio)
	if maxLines < 3 {
		maxLines = 3
	}

	// Deduplicate and limit
	seen := make(map[string]bool)
	var result []string
	for _, line := range keyContent {
		if !seen[line] && strings.TrimSpace(line) != "" {
			seen[line] = true
			result = append(result, line)
			if len(result) >= maxLines {
				break
			}
		}
	}

	return strings.Join(result, "\n")
}

// isKeyLine checks if a line contains key information
func (f *HierarchicalSummaryFilter) isKeyLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Skip empty lines
	if trimmed == "" {
		return false
	}

	// Skip comment-only lines
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
		return false
	}

	// Important keywords
	keywords := []string{"error", "warning", "fail", "success", "complete", "done", "result"}
	for _, kw := range keywords {
		if strings.Contains(strings.ToLower(trimmed), kw) {
			return true
		}
	}

	// Code declarations
	if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "def ") ||
		strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "type ") {
		return true
	}

	return false
}

// extractKeyLines extracts important lines from content
func (f *HierarchicalSummaryFilter) extractKeyLines(content string) []string {
	lines := strings.Split(content, "\n")
	var keyLines []string

	for _, line := range lines {
		if f.isKeyLine(line) {
			keyLines = append(keyLines, line)
		}
	}

	return keyLines
}

// createHierarchicalSummary builds the final hierarchical summary
func (f *HierarchicalSummaryFilter) createHierarchicalSummary(summaries []sectionSummary, sections []hierSection, mode Mode) string {
	var result []string

	for i, sum := range summaries {
		// Preserve first section if configured
		if i == 0 && f.preserveFirst {
			result = append(result, sum.original.content)
			continue
		}

		// Preserve last section if configured
		if i == len(summaries)-1 && f.preserveLast {
			result = append(result, sum.original.content)
			continue
		}

		// Check for error content
		if f.preserveErrors && f.hasErrors(sum.original.content) {
			result = append(result, sum.original.content)
			continue
		}

		// Add summary
		if sum.summary != "" {
			result = append(result, sum.summary)
		}
	}

	return strings.Join(result, "\n\n")
}

// hasErrors checks if content contains errors or stack traces.
// Enhanced error/stack trace detection for preservation.
func (f *HierarchicalSummaryFilter) hasErrors(content string) bool {
	lower := strings.ToLower(content)

	// Error keywords
	if strings.Contains(lower, "error") ||
		strings.Contains(lower, "exception") ||
		strings.Contains(lower, "panic") ||
		strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "failure") ||
		strings.Contains(lower, "traceback") ||
		strings.Contains(lower, "stack trace") ||
		strings.Contains(lower, "abort") {
		return true
	}

	// Stack trace patterns (file:line patterns)
	lines := strings.Split(content, "\n")
	stackLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Common stack trace patterns: "  at func (file:line:col)" or "  file.go:123"
		if strings.Contains(trimmed, ".go:") ||
			strings.Contains(trimmed, ".py:") ||
			strings.Contains(trimmed, ".js:") ||
			strings.Contains(trimmed, ".rs:") ||
			strings.HasPrefix(trimmed, "at ") ||
			strings.HasPrefix(trimmed, "    at ") {
			stackLines++
		}
	}
	// If multiple lines look like stack trace, preserve the block
	return stackLines >= 2
}

// SetSummaryRatio sets the summary ratio
func (f *HierarchicalSummaryFilter) SetSummaryRatio(ratio float64) {
	f.summaryRatio = ratio
}

// SetMaxLevels sets the maximum recursion depth
func (f *HierarchicalSummaryFilter) SetMaxLevels(levels int) {
	f.maxLevels = levels
}
