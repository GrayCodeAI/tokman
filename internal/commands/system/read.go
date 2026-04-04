package system

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/contextread"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	readLevel        string
	readMode         string
	readMaxLines     int
	readMaxTokens    int
	readLineNums     bool
	readStartLine    int
	readEndLine      int
	readSaveSnapshot bool
	readRelatedFiles int
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read [file]",
	Short: "Read file with intelligent filtering",
	Long: `Read a file and apply token-optimized filtering.

Supports multiple filter levels:
  - none: No filtering, raw output
  - minimal: Remove comments, collapse blank lines (default)
  - aggressive: Strip imports, function bodies, keep signatures

Examples:
  tokman read main.go
  tokman read main.go --level aggressive --max-lines 50
  tokman read main.go --mode map
  tokman read main.go --mode delta
  tokman read main.go -n  # show line numbers`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRead,
}

func init() {
	registry.Add(func() { registry.Register(readCmd) })
	readCmd.Flags().StringVarP(&readLevel, "level", "l", "minimal", "Filter level: none, minimal, aggressive")
	readCmd.Flags().StringVar(&readMode, "mode", "", "Context read mode: auto, full, map, signatures, aggressive, entropy, lines, delta, graph")
	readCmd.Flags().IntVarP(&readMaxLines, "max-lines", "m", 0, "Maximum lines to output (0 = no limit)")
	readCmd.Flags().IntVar(&readMaxTokens, "max-tokens", 0, "Approximate token budget for read output (0 = no limit)")
	readCmd.Flags().BoolVarP(&readLineNums, "line-numbers", "n", false, "Show line numbers")
	readCmd.Flags().IntVar(&readStartLine, "start-line", 0, "Start line for line-oriented reads")
	readCmd.Flags().IntVar(&readEndLine, "end-line", 0, "End line for line-oriented reads")
	readCmd.Flags().BoolVar(&readSaveSnapshot, "save-snapshot", true, "Persist file snapshot for future delta reads")
	readCmd.Flags().IntVar(&readRelatedFiles, "related-files", 3, "Number of related files to include in graph mode")
}

func runRead(cmd *cobra.Command, args []string) error {
	var content string
	var filePath string

	if len(args) == 0 {
		// Read from stdin
		if shared.Verbose > 0 {
			fmt.Fprintln(os.Stderr, "Reading from stdin")
		}
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		content = strings.Join(lines, "\n")
		filePath = "stdin"
	} else {
		// Read from file
		filePath = args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		content = string(data)
	}

	start := time.Now()
	opts := readOptions{
		level:        readLevel,
		mode:         readMode,
		maxLines:     readMaxLines,
		maxTokens:    readMaxTokens,
		lineNumbers:  readLineNums,
		startLine:    readStartLine,
		endLine:      readEndLine,
		saveSnapshot: readSaveSnapshot && filePath != "stdin",
		relatedFiles: readRelatedFiles,
	}
	filtered, originalTokens, filteredTokens, err := buildReadOutput(filePath, content, opts)
	if err != nil {
		return err
	}

	// Output
	fmt.Print(filtered)
	if !strings.HasSuffix(filtered, "\n") {
		fmt.Println()
	}

	recordSmartRead("tokman read", filePath, content, opts, originalTokens, filteredTokens, time.Since(start).Milliseconds())

	if shared.Verbose > 0 {
		originalLines := len(strings.Split(content, "\n"))
		filteredLines := len(strings.Split(filtered, "\n"))
		reduction := 0.0
		if originalLines > 0 {
			reduction = float64(originalLines-filteredLines) / float64(originalLines) * 100
		}
		fmt.Fprintf(os.Stderr, "Lines: %d -> %d (%.1f%% reduction, %d tokens saved)\n",
			originalLines, filteredLines, reduction, originalTokens-filteredTokens)
	}

	return nil
}

type readOptions struct {
	level        string
	mode         string
	maxLines     int
	maxTokens    int
	lineNumbers  bool
	startLine    int
	endLine      int
	saveSnapshot bool
	relatedFiles int
}

func buildReadOutput(filePath, content string, opts readOptions) (string, int, int, error) {
	return contextread.Build(filePath, content, contextread.Options{
		Level:        opts.level,
		Mode:         opts.mode,
		MaxLines:     opts.maxLines,
		MaxTokens:    opts.maxTokens,
		LineNumbers:  opts.lineNumbers,
		StartLine:    opts.startLine,
		EndLine:      opts.endLine,
		SaveSnapshot: opts.saveSnapshot,
		RelatedFiles: opts.relatedFiles,
	})
}

func recordSmartRead(commandName, filePath, content string, opts readOptions, originalTokens, filteredTokens int, execTimeMs int64) {
	tracker, err := shared.OpenTracker()
	if err != nil {
		return
	}
	defer tracker.Close()

	projectPath := shared.GetProjectPath()

	savedTokens := originalTokens - filteredTokens
	if savedTokens < 0 {
		savedTokens = 0
	}

	meta := contextread.Describe("read", filePath, content, contextread.Options{
		Level:        opts.level,
		Mode:         opts.mode,
		MaxLines:     opts.maxLines,
		MaxTokens:    opts.maxTokens,
		LineNumbers:  opts.lineNumbers,
		StartLine:    opts.startLine,
		EndLine:      opts.endLine,
		SaveSnapshot: opts.saveSnapshot,
		RelatedFiles: opts.relatedFiles,
	})

	if err := tracker.Record(&tracking.CommandRecord{
		Command:             fmt.Sprintf("%s %s", commandName, filePath),
		OriginalTokens:      originalTokens,
		FilteredTokens:      filteredTokens,
		SavedTokens:         savedTokens,
		ProjectPath:         projectPath,
		ExecTimeMs:          execTimeMs,
		ParseSuccess:        true,
		ContextKind:         meta.Kind,
		ContextMode:         meta.RequestedMode,
		ContextResolvedMode: meta.ResolvedMode,
		ContextTarget:       meta.Target,
		ContextRelatedFiles: meta.RelatedFiles,
		ContextBundle:       meta.Bundle,
	}); err != nil {
		log.Printf("failed to record smart read: %v", err)
	}
}
