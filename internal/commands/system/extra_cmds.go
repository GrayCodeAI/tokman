package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/graph"
	"github.com/GrayCodeAI/tokman/internal/memory"
)

var graphCmd = &cobra.Command{
	Use:   "graph [directory]",
	Short: "Analyze project dependency graph",
	Long: `Analyze the project dependency graph to find related files,
impact analysis, and project statistics.

Examples:
  tokman graph .
  tokman graph ./src --related main.go
  tokman graph ./src --impact utils.go`,
	RunE: runGraph,
}

var (
	graphRelated string
	graphImpact  string
	graphStats   bool
)

func init() {
	registry.Add(func() { registry.Register(graphCmd) })
	graphCmd.Flags().StringVar(&graphRelated, "related", "", "Find files related to given file")
	graphCmd.Flags().StringVar(&graphImpact, "impact", "", "Show impact analysis for given file")
	graphCmd.Flags().BoolVar(&graphStats, "stats", false, "Show project statistics")
}

func runGraph(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	g := graph.NewProjectGraph(dir)
	if err := g.Analyze(); err != nil {
		return fmt.Errorf("failed to analyze project: %w", err)
	}

	if graphStats {
		stats := g.Stats()
		fmt.Println(graph.FormatGraphStats(stats))
		if byLang, ok := stats["by_language"].(map[string]int); ok {
			fmt.Println("\nBy Language:")
			for lang, count := range byLang {
				fmt.Printf("  %-15s %d files\n", lang, count)
			}
		}
		return nil
	}

	if graphRelated != "" {
		related := g.FindRelatedFiles(graphRelated, 10)
		if len(related) == 0 {
			fmt.Printf("No related files found for %s\n", graphRelated)
			return nil
		}
		fmt.Printf("Files related to %s:\n", graphRelated)
		for _, f := range related {
			fmt.Printf("  %s\n", f)
		}
		return nil
	}

	if graphImpact != "" {
		affected := g.ImpactAnalysis(graphImpact)
		if len(affected) == 0 {
			fmt.Printf("No files affected by changes to %s\n", graphImpact)
			return nil
		}
		fmt.Printf("Files affected by changes to %s:\n", graphImpact)
		for _, f := range affected {
			fmt.Printf("  %s\n", f)
		}
		return nil
	}

	// Default: show graph overview
	stats := g.Stats()
	fmt.Println(graph.FormatGraphStats(stats))
	return nil
}

var toonCmd = &cobra.Command{
	Use:   "toon <file>",
	Short: "Apply TOON columnar encoding to JSON output",
	Long: `Compress JSON arrays using TOON columnar encoding.
Achieves 40-80% compression on structured data.

Examples:
  cat data.json | tokman toon
  tokman toon data.json`,
	RunE: runToon,
}

func runToon(cmd *cobra.Command, args []string) error {
	var input string
	if len(args) > 0 {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		input = string(data)
	} else {
		data := make([]byte, 1024*1024)
		n, _ := os.Stdin.Read(data)
		input = string(data[:n])
	}

	encoder := filter.NewTOONEncoder(filter.DefaultTOONConfig())
	result, orig, comp, isToon := encoder.Encode(strings.TrimSpace(input))
	if !isToon {
		fmt.Print(input)
		return nil
	}

	fmt.Println(result)
	fmt.Fprintf(os.Stderr, "TOON: %d → %d tokens (%.1f%% saved)\n",
		orig, comp, float64(orig-comp)/float64(orig)*100)
	return nil
}

var tddCmd = &cobra.Command{
	Use:   "tdd <file>",
	Short: "Apply Token Dense Dialect encoding",
	Long: `Replace common programming terms with Unicode symbols
for 8-25% extra token savings.

Examples:
  cat code.go | tokman tdd
  tokman tdd code.go`,
	RunE: runTDD,
}

var tddDecode bool

func init() {
	tddCmd.Flags().BoolVar(&tddDecode, "decode", false, "Decode TDD back to original")
}

func runTDD(cmd *cobra.Command, args []string) error {
	var input string
	if len(args) > 0 {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		input = string(data)
	} else {
		data := make([]byte, 1024*1024)
		n, _ := os.Stdin.Read(data)
		input = string(data[:n])
	}

	tdd := filter.NewTokenDenseDialect(filter.DefaultTDDConfig())

	if tddDecode {
		fmt.Print(tdd.Decode(input))
		return nil
	}

	result, stats := tdd.EncodeWithStats(input)
	fmt.Print(result)
	fmt.Fprintf(os.Stderr, "\n%s\n", filter.FormatTDDStats(stats))
	return nil
}

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Cross-session memory for AI agents",
	Long: `Manage cross-session memory for AI agents.
Persists tasks, findings, decisions, and facts across chats.

Examples:
  tokman memory add task "Fix auth bug" --tag security
  tokman memory query task
  tokman memory stats`,
	RunE: runMemory,
}

var (
	memoryAction   string
	memoryContent  string
	memoryCategory string
	memoryTags     []string
	memoryQuery    string
)

func init() {
	registry.Add(func() { registry.Register(toonCmd) })
	registry.Add(func() { registry.Register(tddCmd) })
	registry.Add(func() { registry.Register(memoryCmd) })

	memoryCmd.Flags().StringVar(&memoryAction, "action", "query", "Action: add, query, stats, clear")
	memoryCmd.Flags().StringVar(&memoryContent, "content", "", "Content to add")
	memoryCmd.Flags().StringVar(&memoryCategory, "category", "task", "Category: task, finding, decision, fact")
	memoryCmd.Flags().StringSliceVar(&memoryTags, "tag", nil, "Tags for memory item")
	memoryCmd.Flags().StringVar(&memoryQuery, "query", "", "Query memory by tags")
}

func runMemory(cmd *cobra.Command, args []string) error {
	storePath := memoryStorePath()
	store := memory.NewMemoryStore(storePath)

	switch memoryAction {
	case "add":
		if memoryContent == "" {
			return fmt.Errorf("--content is required")
		}
		var id string
		switch memoryCategory {
		case "task":
			id = store.AddTask(memoryContent, memoryTags...)
		case "finding":
			id = store.AddFinding(memoryContent, memoryTags...)
		case "decision":
			id = store.AddDecision(memoryContent, memoryTags...)
		case "fact":
			id = store.AddFact(memoryContent, memoryTags...)
		default:
			return fmt.Errorf("unknown category: %s", memoryCategory)
		}
		fmt.Printf("Added %s (ID: %s)\n", memoryCategory, id)

	case "query":
		items := store.Query(memoryCategory, memoryTags...)
		if len(items) == 0 {
			fmt.Println("No items found")
			return nil
		}
		for _, item := range items {
			fmt.Printf("[%s] %s (tags: %s)\n  %s\n\n",
				item.Category, item.CreatedAt.Format("2006-01-02"),
				strings.Join(item.Tags, ", "), item.Content)
		}

	case "stats":
		stats := store.Stats()
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(data))

	case "clear":
		store.Clear()
		fmt.Println("Memory cleared")

	default:
		return fmt.Errorf("unknown action: %s", memoryAction)
	}

	return nil
}

func memoryStorePath() string {
	return filepath.Join(config.DataPath(), "memory.json")
}
