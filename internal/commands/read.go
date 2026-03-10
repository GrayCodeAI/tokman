package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	readLevel    string
	readMaxLines int
	readLineNums bool
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
  tokman read main.go -n  # show line numbers`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRead,
}

func init() {
	rootCmd.AddCommand(readCmd)
	readCmd.Flags().StringVarP(&readLevel, "level", "l", "minimal", "Filter level: none, minimal, aggressive")
	readCmd.Flags().IntVarP(&readMaxLines, "max-lines", "m", 0, "Maximum lines to output (0 = no limit)")
	readCmd.Flags().BoolVarP(&readLineNums, "line-numbers", "n", false, "Show line numbers")
}

func runRead(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	var content string
	var filePath string

	if len(args) == 0 {
		// Read from stdin
		if verbose > 0 {
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

	// Detect language from extension and content
	lang := detectLanguage(filePath, content)
	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Detected language: %s\n", lang)
	}

	// Parse filter level
	mode := filter.Mode(readLevel)
	if mode != filter.ModeMinimal && mode != filter.ModeAggressive && readLevel != "none" {
		return fmt.Errorf("invalid filter level: %s (use: none, minimal, aggressive)", readLevel)
	}

	var filtered string
	var tokensSaved int

	if readLevel == "none" {
		filtered = content
		tokensSaved = 0
	} else {
		engine := filter.NewEngine(mode)
		filtered, tokensSaved = engine.ProcessWithLang(content, string(lang))
	}

	// Apply max lines if specified
	if readMaxLines > 0 {
		filtered = truncateLines(filtered, readMaxLines)
	}

	// Add line numbers if requested
	if readLineNums {
		filtered = addLineNumbers(filtered)
	}

	// Output
	fmt.Print(filtered)
	if !strings.HasSuffix(filtered, "\n") {
		fmt.Println()
	}

	// Track savings
	originalTokens := filter.EstimateTokens(content)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(filePath, "tokman read", originalTokens, filteredTokens)

	if verbose > 0 {
		originalLines := len(strings.Split(content, "\n"))
		filteredLines := len(strings.Split(filtered, "\n"))
		reduction := 0.0
		if originalLines > 0 {
			reduction = float64(originalLines-filteredLines) / float64(originalLines) * 100
		}
		fmt.Fprintf(os.Stderr, "Lines: %d -> %d (%.1f%% reduction, %d tokens saved)\n",
			originalLines, filteredLines, reduction, tokensSaved)
	}

	return nil
}

// detectLanguage returns the language from file extension with content fallback
func detectLanguage(path string, content string) filter.Language {
	lang := detectLanguageFromExtension(path)
	if lang != filter.LangUnknown {
		return lang
	}
	return filter.DetectLanguageFromInput(content)
}

// detectLanguageFromExtension returns the language from file extension
func detectLanguageFromExtension(path string) filter.Language {
	if path == "stdin" {
		return filter.LangUnknown
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return filter.LangGo
	case ".rs":
		return filter.LangRust
	case ".py", ".pyw":
		return filter.LangPython
	case ".js", ".mjs", ".cjs":
		return filter.LangJavaScript
	case ".ts", ".tsx":
		return filter.LangTypeScript
	case ".java":
		return filter.LangJava
	case ".c", ".h":
		return filter.LangC
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh":
		return filter.LangCpp
	case ".rb":
		return filter.LangRuby
	case ".sh", ".bash", ".zsh":
		return filter.LangShell
	case ".sql":
		return filter.LangSQL
	case ".yaml", ".yml":
		return filter.LangUnknown
	case ".toml":
		return filter.LangUnknown
	case ".json":
		return filter.LangUnknown
	case ".md":
		return filter.LangUnknown
	default:
		return filter.LangUnknown
	}
}

// truncateLines limits output to maxLines
func truncateLines(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}

	// Keep first half and last quarter
	keepStart := maxLines / 2
	keepEnd := maxLines / 4

	var result []string
	result = append(result, lines[:keepStart]...)
	result = append(result, fmt.Sprintf("// ... %d lines omitted ...", len(lines)-keepStart-keepEnd))
	result = append(result, lines[len(lines)-keepEnd:]...)

	return strings.Join(result, "\n")
}

// addLineNumbers prefixes each line with its number
func addLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	width := len(fmt.Sprintf("%d", len(lines)))

	var result strings.Builder
	for i, line := range lines {
		result.WriteString(fmt.Sprintf("%*d │ %s\n", width, i+1, line))
	}
	return result.String()
}
