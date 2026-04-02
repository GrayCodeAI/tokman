// Package core provides goal-driven context analysis for intelligent compression.
package core

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/utils"
)

// GoalType represents the type of user goal.
type GoalType int

const (
	GoalUnderstand GoalType = iota // Understand code structure
	GoalDebug                      // Debug an issue
	GoalRefactor                   // Refactor code
	GoalReview                     // Code review
	GoalTest                       // Write tests
	GoalDocument                   // Generate documentation
	GoalSearch                     // Find specific information
	GoalOptimize                   // Performance optimization
)

// Goal represents a user's intent.
type Goal struct {
	Type        GoalType
	Keywords    []string
	PriorityMap map[string]float64 // Keyword -> importance score
}

// GoalAnalyzer analyzes content against user goals.
type GoalAnalyzer struct {
	goal     Goal
	keywords map[string]float64
}

// NewGoalAnalyzer creates a new goal analyzer.
func NewGoalAnalyzer(goal Goal) *GoalAnalyzer {
	return &GoalAnalyzer{
		goal:     goal,
		keywords: buildKeywordMap(goal),
	}
}

// DefaultGoals provides predefined goals.
func DefaultGoals() map[GoalType]Goal {
	return map[GoalType]Goal{
		GoalUnderstand: {
			Type: GoalUnderstand,
			Keywords: []string{"function", "method", "class", "interface", "struct",
				"package", "import", "export", "public", "private"},
			PriorityMap: map[string]float64{
				"function": 0.9, "func": 0.9, "method": 0.9,
				"class": 0.8, "struct": 0.8, "interface": 0.8,
				"package": 0.7, "module": 0.7,
				"import": 0.6, "export": 0.6,
			},
		},
		GoalDebug: {
			Type: GoalDebug,
			Keywords: []string{"error", "exception", "panic", "fail", "bug",
				"stack trace", "log", "debug", "breakpoint"},
			PriorityMap: map[string]float64{
				"error": 1.0, "err": 1.0, "exception": 1.0,
				"panic": 0.95, "fatal": 0.95,
				"stack": 0.9, "trace": 0.9,
				"log": 0.8, "debug": 0.8,
				"assert": 0.7, "test": 0.7,
			},
		},
		GoalRefactor: {
			Type: GoalRefactor,
			Keywords: []string{"duplicate", "extract", "inline", "rename",
				"move", "replace", "simplify", "cleanup"},
			PriorityMap: map[string]float64{
				"duplicate": 0.95, "copy": 0.9,
				"refactor": 0.9, "extract": 0.9,
				"TODO": 0.8, "FIXME": 0.8,
				"deprecated": 0.7, "legacy": 0.7,
			},
		},
		GoalReview: {
			Type: GoalReview,
			Keywords: []string{"security", "performance", "style", "lint",
				"comment", "doc", "review", "approval"},
			PriorityMap: map[string]float64{
				"security": 1.0, "vulnerability": 1.0, "injection": 0.95,
				"password": 0.9, "secret": 0.9, "token": 0.9,
				"performance": 0.85, "optimize": 0.85,
				"TODO": 0.6, "FIXME": 0.6,
			},
		},
		GoalTest: {
			Type: GoalTest,
			Keywords: []string{"test", "spec", "assert", "expect",
				"mock", "stub", "coverage", "fixture"},
			PriorityMap: map[string]float64{
				"test": 1.0, "Test": 1.0, "spec": 0.95,
				"assert": 0.9, "expect": 0.9, "should": 0.9,
				"mock": 0.8, "stub": 0.8, "fake": 0.8,
				"coverage": 0.7, "benchmark": 0.7,
			},
		},
		GoalDocument: {
			Type: GoalDocument,
			Keywords: []string{"doc", "comment", "README", "API",
				"example", "usage", "guide", "tutorial"},
			PriorityMap: map[string]float64{
				"documentation": 1.0, "README": 0.95, "API": 0.9,
				"example": 0.85, "examples": 0.85,
				"usage": 0.8, "guide": 0.8,
				"comment": 0.6, "//": 0.6,
			},
		},
	}
}

// ContextItem represents a single item of context.
type ContextItem struct {
	Content   string
	Source    string
	LineNum   int
	Relevance float64
	Priority  float64
}

// AnalyzeResult contains the analysis results.
type AnalyzeResult struct {
	Items         []ContextItem
	TotalItems    int
	FilteredItems int
	RelevanceAvg  float64
}

// Analyze analyzes content and returns relevant items based on goal.
func (a *GoalAnalyzer) Analyze(content string, source string) AnalyzeResult {
	lines := strings.Split(content, "\n")
	items := make([]ContextItem, 0, len(lines))

	for i, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		relevance := a.calculateRelevance(line)
		priority := a.calculatePriority(line)

		items = append(items, ContextItem{
			Content:   line,
			Source:    source,
			LineNum:   i + 1,
			Relevance: relevance,
			Priority:  priority,
		})
	}

	// Filter by relevance threshold
	var filtered []ContextItem
	threshold := 0.3 // Minimum relevance score
	for _, item := range items {
		if item.Relevance >= threshold {
			filtered = append(filtered, item)
		}
	}

	// Sort by combined score (relevance * priority)
	sortByScore(filtered)

	// Calculate average relevance
	avgRelevance := 0.0
	if len(filtered) > 0 {
		for _, item := range filtered {
			avgRelevance += item.Relevance
		}
		avgRelevance /= float64(len(filtered))
	}

	return AnalyzeResult{
		Items:         filtered,
		TotalItems:    len(items),
		FilteredItems: len(filtered),
		RelevanceAvg:  avgRelevance,
	}
}

// ExtractRelevant extracts only the relevant content based on goal.
func (a *GoalAnalyzer) ExtractRelevant(content string, source string) string {
	result := a.Analyze(content, source)

	var lines []string
	for _, item := range result.Items {
		lines = append(lines, item.Content)
	}

	return strings.Join(lines, "\n")
}

// RankFiles ranks multiple files by relevance to goal.
func (a *GoalAnalyzer) RankFiles(files map[string]string) []RankedFile {
	var ranked []RankedFile

	for path, content := range files {
		result := a.Analyze(content, path)
		score := result.RelevanceAvg * float64(result.FilteredItems) / float64(utils.Max(result.TotalItems, 1))

		ranked = append(ranked, RankedFile{
			Path:      path,
			Content:   content,
			Score:     score,
			Relevance: result.RelevanceAvg,
		})
	}

	// Sort by score descending
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].Score > ranked[i].Score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	return ranked
}

// RankedFile represents a file with relevance score.
type RankedFile struct {
	Path      string
	Content   string
	Score     float64
	Relevance float64
}

// GetGoalFromQuery infers a goal from a natural language query.
func GetGoalFromQuery(query string) GoalType {
	query = strings.ToLower(query)

	patterns := map[GoalType][]string{
		GoalDebug:    {"debug", "fix", "error", "bug", "problem", "issue", "panic", "crash", "exception"},
		GoalRefactor: {"refactor", "rename", "extract", "move", "clean", "restructure", "simplify"},
		GoalTest:     {"test", "spec", "coverage", "unit test", "integration test", "assert"},
		GoalReview:   {"review", "audit", "check", "validate", "inspect"},
		GoalDocument: {"document", "doc", "comment", "explain", "describe"},
		GoalOptimize: {"optimize", "performance", "speed", "slow", "bottleneck", "memory", "cpu"},
		GoalSearch:   {"find", "search", "locate", "where", "which"},
	}

	scores := make(map[GoalType]int)
	for goal, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(query, keyword) {
				scores[goal]++
			}
		}
	}

	// Find highest scoring goal
	bestGoal := GoalUnderstand
	bestScore := 0
	for goal, score := range scores {
		if score > bestScore {
			bestScore = score
			bestGoal = goal
		}
	}

	return bestGoal
}

// calculateRelevance calculates how relevant a line is to the current goal.
func (a *GoalAnalyzer) calculateRelevance(line string) float64 {
	line = strings.ToLower(line)
	score := 0.0
	matches := 0

	for keyword, weight := range a.keywords {
		if strings.Contains(line, keyword) {
			score += weight
			matches++
		}
	}

	// Bonus for multiple matches
	if matches > 1 {
		score *= (1.0 + float64(matches)*0.1)
	}

	// Normalize
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculatePriority calculates priority based on structural indicators.
func (a *GoalAnalyzer) calculatePriority(line string) float64 {
	priority := 0.5

	// Higher priority for definitions
	if definitionPattern.MatchString(line) {
		priority += 0.3
	}

	// Higher priority for exported/public items
	if exportedPattern.MatchString(line) {
		priority += 0.2
	}

	// Higher priority for important comments
	if strings.Contains(line, "TODO") || strings.Contains(line, "FIXME") || strings.Contains(line, "NOTE") {
		priority += 0.25
	}

	// Lower priority for imports in most goals (except understanding)
	if importPattern.MatchString(line) && a.goal.Type != GoalUnderstand {
		priority -= 0.2
	}

	// Cap priority
	if priority > 1.0 {
		priority = 1.0
	}
	if priority < 0.1 {
		priority = 0.1
	}

	return priority
}

var (
	definitionPattern = regexp.MustCompile(`^\s*(func|class|struct|interface|def|fn)\s+`)
	exportedPattern   = regexp.MustCompile(`^\s*(export|pub|public)\s+`)
	importPattern     = regexp.MustCompile(`^\s*(import|use|require)\s+`)
)

func buildKeywordMap(goal Goal) map[string]float64 {
	keywords := make(map[string]float64)

	// Add priority mapped keywords
	for k, v := range goal.PriorityMap {
		keywords[strings.ToLower(k)] = v
	}

	// Add default weight for non-mapped keywords
	for _, keyword := range goal.Keywords {
		if _, exists := keywords[keyword]; !exists {
			keywords[keyword] = 0.5
		}
	}

	return keywords
}

func sortByScore(items []ContextItem) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			scoreI := items[i].Relevance * items[i].Priority
			scoreJ := items[j].Relevance * items[j].Priority
			if scoreJ > scoreI {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

