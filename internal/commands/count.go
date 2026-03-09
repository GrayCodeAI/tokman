package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/tokenizer"
)

var (
	countModel    string
	countEncoding string
	countCompare  bool
	countFiles    bool
)

var countCmd = &cobra.Command{
	Use:   "count [text|file...]",
	Short: "Count tokens using tiktoken (OpenAI tokenizer)",
	Long: `Count tokens in text or files using OpenAI's tiktoken tokenizer.

Supports multiple encoding schemes:
  - cl100k_base: GPT-4, GPT-3.5-turbo, text-embedding-ada-002
  - o200k_base: GPT-4o, GPT-4o-mini
  - p50k_base: GPT-3 (davinci, curie, babbage, ada)

Examples:
  # Count tokens in text
  tokman count "Hello, world!"
  
  # Count tokens from stdin
  echo "Hello, world!" | tokman count
  
  # Count tokens in a file
  tokman count file.txt
  
  # Count tokens with specific model encoding
  tokman count --model gpt-4o "Hello, world!"
  
  # Compare heuristic vs actual count
  tokman count --compare "Hello, world!"
  
  # Count multiple files
  tokman count *.go`,
	RunE: runCount,
}

func init() {
	rootCmd.AddCommand(countCmd)

	countCmd.Flags().StringVarP(&countModel, "model", "m", "gpt-4",
		"model to use for encoding (gpt-4, gpt-4o, gpt-3.5-turbo, claude-3-opus, etc.)")
	countCmd.Flags().StringVarP(&countEncoding, "encoding", "e", "",
		"encoding to use (cl100k_base, o200k_base, p50k_base). Overrides --model")
	countCmd.Flags().BoolVarP(&countCompare, "compare", "c", false,
		"compare heuristic vs actual token count")
	countCmd.Flags().BoolVarP(&countFiles, "files", "f", false,
		"treat arguments as files to count")
}

func runCount(cmd *cobra.Command, args []string) error {
	// Determine encoding
	var enc tokenizer.Encoding
	var t *tokenizer.Tokenizer
	var err error

	if countEncoding != "" {
		enc = tokenizer.Encoding(countEncoding)
		t, err = tokenizer.New(enc)
	} else {
		enc, _ = tokenizer.ModelToEncoding[countModel]
		t, err = tokenizer.NewForModel(countModel)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize tokenizer: %w", err)
	}

	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	hasStdin := (stat.Mode() & os.ModeCharDevice) == 0

	// Mode 1: stdin
	if hasStdin && len(args) == 0 {
		return countFromReader(t, os.Stdin, enc)
	}

	// Mode 2: no args - show help
	if len(args) == 0 {
		return cmd.Help()
	}

	// Mode 3: files mode
	if countFiles {
		return countFilesMode(t, args, enc)
	}

	// Mode 4: compare mode
	if countCompare {
		text := strings.Join(args, " ")
		return showComparison(text)
	}

	// Mode 5: single text argument
	if len(args) == 1 {
		// Check if it's a file
		if _, err := os.Stat(args[0]); err == nil {
			return countFile(t, args[0], enc)
		}
	}

	// Mode 6: text argument(s)
	text := strings.Join(args, " ")
	return countText(t, text, enc)
}

func countFromReader(t *tokenizer.Tokenizer, r io.Reader, enc tokenizer.Encoding) error {
	stats := &tokenizer.CountStats{Encoding: enc}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		stats.TotalTokens += t.Count(line)
		stats.TotalChars += len(line)
		stats.TotalLines++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading: %w", err)
	}

	fmt.Print(stats.Summary())
	return nil
}

func countText(t *tokenizer.Tokenizer, text string, enc tokenizer.Encoding) error {
	count := t.Count(text)
	lines := strings.Count(text, "\n") + 1

	stats := &tokenizer.CountStats{
		TotalTokens: count,
		TotalChars:  len(text),
		TotalLines:  lines,
		Encoding:    enc,
	}

	fmt.Print(stats.Summary())
	return nil
}

func countFile(t *tokenizer.Tokenizer, path string, enc tokenizer.Encoding) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	count := t.Count(string(content))
	lines := strings.Count(string(content), "\n")

	stats := &tokenizer.CountStats{
		TotalTokens: count,
		TotalChars:  len(content),
		TotalLines:  lines,
		FilesCount:  1,
		Encoding:    enc,
	}

	fmt.Printf("📁 File: %s\n\n", path)
	fmt.Print(stats.Summary())
	return nil
}

func countFilesMode(t *tokenizer.Tokenizer, files []string, enc tokenizer.Encoding) error {
	stats := &tokenizer.CountStats{Encoding: enc}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			continue
		}

		stats.TotalTokens += t.Count(string(content))
		stats.TotalChars += len(content)
		stats.TotalLines += strings.Count(string(content), "\n")
		stats.FilesCount++
	}

	fmt.Print(stats.Summary())
	return nil
}

func showComparison(text string) error {
	heuristic, actual, diff := tokenizer.CompareCounts(text)

	fmt.Println("📊 Token Count Comparison")
	fmt.Println("────────────────────────────────────")
	fmt.Printf("Text:           \"%.50s%s\"\n", text, func() string {
		if len(text) > 50 {
			return "..."
		}
		return ""
	}())
	fmt.Printf("Characters:     %d\n", len(text))
	fmt.Println("────────────────────────────────────")
	fmt.Printf("Heuristic (÷4): %d tokens\n", heuristic)
	fmt.Printf("Actual (tiktoken): %d tokens\n", actual)
	fmt.Println("────────────────────────────────────")

	if diff > 0 {
		fmt.Printf("Heuristic overestimates by %.1f%%\n", diff)
	} else if diff < 0 {
		fmt.Printf("Heuristic underestimates by %.1f%%\n", -diff)
	} else {
		fmt.Println("Both methods agree!")
	}

	return nil
}
