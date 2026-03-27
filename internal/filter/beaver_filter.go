package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

var beaverFuncRe = regexp.MustCompile(`(func|function|def|class|struct)\s+`)

// BeaverFilter implements BEAVER-style hierarchical page-level compression.
// Research: "BEAVER: Training-Free Hierarchical Prompt Compression" (arXiv 2603.19635, Mar 2026)
// Key Innovation: Maps variable-length contexts into dense page-level representations.
// Uses anchor pages (high importance) + flow pages (context bridges) for structure-aware selection.
// Maximizes hardware parallelism by working at page level instead of token level.
//
// Unlike linear token pruning, BEAVER selects entire pages/sections, preserving
// structural coherence while enabling parallel processing.
type BeaverFilter struct {
	config BeaverConfig
}

// BeaverConfig holds configuration for BEAVER compression
type BeaverConfig struct {
	Enabled          bool
	PageSize         int     // Number of lines per page
	AnchorRatio      float64 // Ratio of pages to keep as anchors (0-1)
	FlowRatio        float64 // Ratio of flow pages to keep (0-1)
	MinContentLength int
}

// DefaultBeaverConfig returns default configuration
func DefaultBeaverConfig() BeaverConfig {
	return BeaverConfig{
		Enabled:          true,
		PageSize:         10,
		AnchorRatio:      0.3,
		FlowRatio:        0.2,
		MinContentLength: 500,
	}
}

// NewBeaverFilter creates a new BEAVER filter
func NewBeaverFilter() *BeaverFilter {
	return &BeaverFilter{config: DefaultBeaverConfig()}
}

// Name returns the filter name
func (b *BeaverFilter) Name() string { return "beaver" }

// beaverPage represents a page of content
type beaverPage struct {
	content   string
	score     float64
	pageType  string // "anchor", "flow", "filler"
	startLine int
	endLine   int
	lineCount int
}

// Apply applies BEAVER hierarchical compression
func (b *BeaverFilter) Apply(input string, mode Mode) (string, int) {
	if !b.config.Enabled || mode == ModeNone {
		return input, 0
	}

	if len(input) < b.config.MinContentLength {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	// Split into pages
	pages := b.splitIntoPages(input)
	if len(pages) < 3 {
		return input, 0
	}

	// Score pages
	b.scorePages(pages, mode)

	// Select pages (anchor + flow)
	output := b.selectPages(pages, mode)

	finalTokens := core.EstimateTokens(output)
	saved := originalTokens - finalTokens
	if saved < 5 {
		return input, 0
	}

	return output, saved
}

// splitIntoPages splits content into pages
func (b *BeaverFilter) splitIntoPages(input string) []beaverPage {
	lines := strings.Split(input, "\n")
	var pages []beaverPage

	pageSize := b.config.PageSize
	for i := 0; i < len(lines); i += pageSize {
		end := i + pageSize
		if end > len(lines) {
			end = len(lines)
		}

		pageLines := lines[i:end]
		content := strings.Join(pageLines, "\n")

		pages = append(pages, beaverPage{
			content:   content,
			startLine: i,
			endLine:   end,
			lineCount: len(pageLines),
		})
	}

	return pages
}

// scorePages scores each page for importance
func (b *BeaverFilter) scorePages(pages []beaverPage, mode Mode) {
	for i := range pages {
		score := 0.5
		content := pages[i].content
		lower := strings.ToLower(content)

		// Error/warning content
		if strings.Contains(lower, "error") || strings.Contains(lower, "fail") {
			score += 0.4
		}

		// Code structure
		if beaverFuncRe.MatchString(content) {
			score += 0.3
		}

		// First and last pages are often important
		if i == 0 || i == len(pages)-1 {
			score += 0.2
		}

		// Information density
		words := strings.Fields(content)
		if len(words) > 0 {
			uniqueWords := make(map[string]bool)
			for _, w := range words {
				uniqueWords[strings.ToLower(w)] = true
			}
			density := float64(len(uniqueWords)) / float64(len(words))
			score += density * 0.2
		}

		pages[i].score = score

		// Classify page type
		if score > 0.7 {
			pages[i].pageType = "anchor"
		} else if score > 0.4 {
			pages[i].pageType = "flow"
		} else {
			pages[i].pageType = "filler"
		}
	}
}

// selectPages selects anchor and flow pages
func (b *BeaverFilter) selectPages(pages []beaverPage, mode Mode) string {
	anchorCount := int(float64(len(pages)) * b.config.AnchorRatio)
	flowCount := int(float64(len(pages)) * b.config.FlowRatio)

	if mode == ModeAggressive {
		anchorCount = int(float64(anchorCount) * 0.7)
		flowCount = int(float64(flowCount) * 0.5)
	}

	// Keep anchor pages + selected flow pages
	var result strings.Builder
	kept := 0

	for _, page := range pages {
		if page.pageType == "anchor" && kept < anchorCount+flowCount {
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(page.content)
			kept++
		} else if page.pageType == "flow" && kept < anchorCount+flowCount {
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(page.content)
			kept++
		}
	}

	return strings.TrimSpace(result.String())
}
