package filter

import (
	"sort"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// FilterPriority holds a filter with its computed priority score.
type FilterPriority struct {
	Filter   Filter
	Priority int // higher = applied first
}

// FilterOrderOptimizer determines the optimal filter application order
// based on content-type probing and historical effectiveness data.
// Task #188: Priority-based filter ordering optimization.
type FilterOrderOptimizer struct {
	// staticPriorities maps filter names to static priority levels.
	// Filters that run cheap checks first to skip expensive ones later.
	staticPriorities map[string]int
}

// defaultFilterPriorities defines the canonical ordering.
// Priority 100 = first (cheap, high-impact), Priority 10 = last (expensive).
var defaultFilterPriorities = map[string]int{
	// Structural guards (always first — prevent wasted work)
	"binary_passthrough": 100,
	"skip_list":          95,
	"ansi_strip":         90,
	"ansi":              90,

	// Fast, high-impact content-independent filters
	"boilerplate":        80,
	"comment_strip":      78,
	"comment":            78,
	"whitespace":         76,
	"line_truncate":      74,
	"import_collapse":    72,

	// Content-type specific (run after type is known)
	"error_dedup":        65,
	"smart_log":          63,
	"rle_compress":       60,
	"url_compress":       58,
	"numeric_compress":   55,
	"numerical_quant":    55,
	"sql_compress":       50,
	"proto_compress":     50,
	"git_diff":           50,
	"test_output":        50,
	"agent_compress":     48,
	"csv_compress":       46,
	"html_compress":      44,
	"shell_output":       42,

	// Semantic / content-aware (heavier)
	"importance_scoring": 35,
	"sentence_boundary":  33,
	"ast_skeleton":       30,
	"pattern_dict":       28,
	"crossrun_dedup":     25,

	// Budget enforcement (always last)
	"context_window":     15,
	"smart_truncate":     12,
	"sliding_window":     10,
}

// NewFilterOrderOptimizer creates an optimizer with default priorities.
func NewFilterOrderOptimizer() *FilterOrderOptimizer {
	return &FilterOrderOptimizer{staticPriorities: defaultFilterPriorities}
}

// Sort returns filters sorted by priority (highest first) for the given input.
// Content analysis is used to bump priorities of filters likely to be effective.
func (o *FilterOrderOptimizer) Sort(filters []Filter, input string) []Filter {
	priorities := make(map[string]int, len(filters))

	// Start with static priorities
	for _, f := range filters {
		name := f.Name()
		if p, ok := o.staticPriorities[name]; ok {
			priorities[name] = p
		} else {
			priorities[name] = 50 // default mid-priority
		}
	}

	// Content-based priority adjustments
	lower := strings.ToLower(input[:min(len(input), 512)])

	// Bump error filters if content has errors
	if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "panic") {
		priorities["error_dedup"] += 20
		priorities["smart_log"] += 15
	}

	// Bump log filter for timestamped content
	if logTimestampFullRe.MatchString(input[:min(len(input), 200)]) {
		priorities["smart_log"] += 25
	}

	// Bump ANSI filter for terminal output
	if strings.Contains(input[:min(len(input), 200)], "\x1b[") {
		priorities["ansi_strip"] += 30
		priorities["ansi"] += 30
	}

	// Bump code filters for code content
	if strings.Contains(lower, "func ") || strings.Contains(lower, "def ") ||
		strings.Contains(lower, "class ") || strings.Contains(lower, "import ") {
		priorities["comment_strip"] += 10
		priorities["ast_skeleton"] += 10
		priorities["import_collapse"] += 10
	}

	// Sort by priority descending
	sorted := make([]Filter, len(filters))
	copy(sorted, filters)
	sort.Slice(sorted, func(i, j int) bool {
		pi := priorities[sorted[i].Name()]
		pj := priorities[sorted[j].Name()]
		if pi != pj {
			return pi > pj
		}
		return sorted[i].Name() < sorted[j].Name()
	})

	return sorted
}

// min helper for int (avoid math import)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// OrderedPipeline wraps a set of filters and applies them in optimized order.
type OrderedPipeline struct {
	optimizer *FilterOrderOptimizer
	filters   []Filter
}

// NewOrderedPipeline creates a pipeline that auto-orders filters.
func NewOrderedPipeline(filters []Filter) *OrderedPipeline {
	return &OrderedPipeline{
		optimizer: NewFilterOrderOptimizer(),
		filters:   filters,
	}
}

// Apply runs all filters in optimized order and returns the final output.
func (p *OrderedPipeline) Apply(input string, mode Mode) (string, int) {
	ordered := p.optimizer.Sort(p.filters, input)
	origTokens := core.EstimateTokens(input)

	current := input
	totalSaved := 0
	for _, f := range ordered {
		out, saved := f.Apply(current, mode)
		if saved > 0 {
			current = out
			totalSaved += saved
		}
	}

	// Sanity check
	finalSaved := origTokens - core.EstimateTokens(current)
	if finalSaved < 0 {
		finalSaved = 0
	}
	return current, finalSaved
}
