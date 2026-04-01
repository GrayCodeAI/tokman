package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/contextread"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	contextMode         string
	contextLevel        string
	contextMaxLines     int
	contextMaxTokens    int
	contextLineNumbers  bool
	contextStartLine    int
	contextEndLine      int
	contextSaveSnapshot bool
	contextRelatedFiles int
)

var contextCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"ctx"},
	Short:   "Context tools and usage analysis",
	Long:    `Analyze context usage and generate smart file context for AI agents.`,
	RunE:    runContext,
}

var contextReadCmd = &cobra.Command{
	Use:   "read <file>",
	Short: "Produce smart, budgeted file context",
	Long: `Read a file using TokMan's smart context modes.

Examples:
  tokman ctx read main.go --mode auto
  tokman ctx read main.go --mode signatures --max-tokens 300
  tokman ctx read main.go --start-line 20 --end-line 80
  tokman ctx read main.go --mode graph --related-files 4`,
	Args: cobra.ExactArgs(1),
	RunE: runContextRead,
}

var contextDeltaCmd = &cobra.Command{
	Use:   "delta <file>",
	Short: "Show what changed since the last saved file snapshot",
	Long: `Compare a file to TokMan's last saved snapshot and emit a compact delta.

Examples:
  tokman ctx delta main.go
  tokman ctx delta main.go --max-tokens 200`,
	Args: cobra.ExactArgs(1),
	RunE: runContextDelta,
}

func init() {
	addContextReadFlags(contextReadCmd)
	addContextReadFlags(contextDeltaCmd)

	contextCmd.AddCommand(contextReadCmd, contextDeltaCmd)
	registry.Add(func() { registry.Register(contextCmd) })
}

func runContext(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	savings, err := tracker.GetSavings(cwd)
	if err != nil {
		return fmt.Errorf("failed to get context data: %w", err)
	}

	fmt.Println("Context Window Analysis")
	fmt.Println("======================")
	fmt.Println()

	if savings.TotalCommands == 0 {
		fmt.Println("No data yet. Run some commands through tokman first.")
		return nil
	}

	fmt.Printf("Commands analyzed: %d\n", savings.TotalCommands)
	fmt.Printf("Original context:  %d tokens\n", savings.TotalOriginal)
	fmt.Printf("Filtered context:  %d tokens\n", savings.TotalFiltered)
	fmt.Printf("Tokens saved:      %d tokens\n", savings.TotalSaved)
	fmt.Printf("Reduction:         %.1f%%\n\n", savings.ReductionPct)

	readSavings, err := tracker.GetSavingsForContextReads(cwd, "", "")
	if err != nil {
		return fmt.Errorf("failed to get smart read data: %w", err)
	}
	if readSavings.TotalCommands > 0 {
		fmt.Println("Smart context reads")
		fmt.Println("-------------------")
		fmt.Printf("Reads analyzed:    %d\n", readSavings.TotalCommands)
		fmt.Printf("Original context:  %d tokens\n", readSavings.TotalOriginal)
		fmt.Printf("Delivered context: %d tokens\n", readSavings.TotalFiltered)
		fmt.Printf("Tokens saved:      %d tokens\n", readSavings.TotalSaved)
		fmt.Printf("Reduction:         %.1f%%\n\n", readSavings.ReductionPct)
	}

	contextSizes := []struct {
		name  string
		limit int
	}{
		{"GPT-4o-mini (128K)", 128000},
		{"GPT-4o (128K)", 128000},
		{"Claude 3.5 (200K)", 200000},
		{"Claude 3 Opus (200K)", 200000},
		{"Gemini 1.5 (1M)", 1000000},
	}

	fmt.Println("Context window capacity with tokman:")
	fmt.Printf("%-25s %12s %12s %10s\n", "Model", "Without", "With", "Extra")
	fmt.Printf("%-25s %12s %12s %10s\n", "─────────────────────────", "────────────", "────────────", "──────────")

	for _, cs := range contextSizes {
		without := cs.limit / savings.TotalOriginal
		if without == 0 {
			without = 1
		}
		with := cs.limit / savings.TotalFiltered
		if with == 0 {
			with = 1
		}
		extra := with - without
		fmt.Printf("%-25s %10dx %10dx +%dx\n", cs.name, without, with, extra)
	}

	return nil
}

func addContextReadFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&contextMode, "mode", "auto", "Context mode: auto, full, map, signatures, aggressive, entropy, lines, delta, graph")
	cmd.Flags().StringVar(&contextLevel, "level", "minimal", "Legacy filter level: none, minimal, aggressive")
	cmd.Flags().IntVar(&contextMaxLines, "max-lines", 0, "Maximum lines to emit (0 = no limit)")
	cmd.Flags().IntVar(&contextMaxTokens, "max-tokens", 0, "Approximate token budget for emitted context (0 = no limit)")
	cmd.Flags().BoolVar(&contextLineNumbers, "line-numbers", false, "Include line numbers in output")
	cmd.Flags().IntVar(&contextStartLine, "start-line", 0, "Start line for line-oriented reads")
	cmd.Flags().IntVar(&contextEndLine, "end-line", 0, "End line for line-oriented reads")
	cmd.Flags().BoolVar(&contextSaveSnapshot, "save-snapshot", true, "Persist a snapshot for future delta reads")
	cmd.Flags().IntVar(&contextRelatedFiles, "related-files", 3, "Number of related files to include in graph mode")
}

func runContextRead(cmd *cobra.Command, args []string) error {
	return emitContextFile(args[0], contextread.Options{
		Level:        contextLevel,
		Mode:         contextMode,
		MaxLines:     contextMaxLines,
		MaxTokens:    contextMaxTokens,
		LineNumbers:  contextLineNumbers,
		StartLine:    contextStartLine,
		EndLine:      contextEndLine,
		SaveSnapshot: contextSaveSnapshot,
		RelatedFiles: contextRelatedFiles,
	})
}

func runContextDelta(cmd *cobra.Command, args []string) error {
	mode := contextMode
	if mode == "" || mode == "auto" {
		mode = "delta"
	}
	return emitContextFile(args[0], contextread.Options{
		Level:        contextLevel,
		Mode:         mode,
		MaxLines:     contextMaxLines,
		MaxTokens:    contextMaxTokens,
		LineNumbers:  contextLineNumbers,
		StartLine:    contextStartLine,
		EndLine:      contextEndLine,
		SaveSnapshot: contextSaveSnapshot,
		RelatedFiles: contextRelatedFiles,
	})
}

func emitContextFile(path string, opts contextread.Options) error {
	start := time.Now()
	content, rawContent, originalTokens, filteredTokens, err := buildContextFile(path, opts)
	if err != nil {
		return err
	}

	fmt.Print(content)
	if content != "" && !strings.HasSuffix(content, "\n") {
		fmt.Println()
	}

	commandName := "tokman ctx read"
	if strings.EqualFold(opts.Mode, "delta") {
		commandName = "tokman ctx delta"
	}
	recordContextRead(commandName, path, rawContent, opts, originalTokens, filteredTokens, time.Since(start).Milliseconds())
	return nil
}

func buildContextFile(path string, opts contextread.Options) (string, string, int, int, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("failed to read file %s: %w", cleanPath, err)
	}

	content, originalTokens, filteredTokens, err := contextread.Build(cleanPath, string(data), opts)
	if err != nil {
		return "", "", 0, 0, err
	}
	return content, string(data), originalTokens, filteredTokens, nil
}

func recordContextRead(commandName, path, rawContent string, opts contextread.Options, originalTokens, filteredTokens int, execTimeMs int64) {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	savedTokens := originalTokens - filteredTokens
	if savedTokens < 0 {
		savedTokens = 0
	}

	meta := contextread.Describe("read", path, rawContent, opts)
	if strings.Contains(commandName, "ctx delta") {
		meta.Kind = "delta"
	}

	_ = tracker.Record(&tracking.CommandRecord{
		Command:             fmt.Sprintf("%s %s", commandName, filepath.Clean(path)),
		OriginalTokens:      originalTokens,
		FilteredTokens:      filteredTokens,
		SavedTokens:         savedTokens,
		ProjectPath:         cwd,
		ExecTimeMs:          execTimeMs,
		ParseSuccess:        true,
		ContextKind:         meta.Kind,
		ContextMode:         meta.RequestedMode,
		ContextResolvedMode: meta.ResolvedMode,
		ContextTarget:       meta.Target,
		ContextRelatedFiles: meta.RelatedFiles,
		ContextBundle:       meta.Bundle,
	})
}
