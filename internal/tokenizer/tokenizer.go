package tokenizer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	tiktoken "github.com/tiktoken-go/tokenizer"
)

// Encoding represents a tokenizer encoding type.
type Encoding string

const (
	// Cl100kBase is the encoding for GPT-4, GPT-3.5-turbo, text-embedding-ada-002.
	Cl100kBase Encoding = "cl100k_base"
	// O200kBase is the encoding for GPT-4o, GPT-4o-mini.
	O200kBase Encoding = "o200k_base"
	// P50kBase is the encoding for GPT-3 (davinci, curie, babbage, ada).
	P50kBase Encoding = "p50k_base"
	// R50kBase is the encoding for GPT-3 (davinci, curie, babbage, ada) without regex splitting.
	R50kBase Encoding = "r50k_base"
)

// ModelToEncoding maps model names to their encodings.
var ModelToEncoding = map[string]Encoding{
	// GPT-4o family
	"gpt-4o":        O200kBase,
	"gpt-4o-mini":   O200kBase,
	"gpt-4o-2024-05-13": O200kBase,
	"gpt-4o-mini-2024-07-18": O200kBase,
	// GPT-4 family
	"gpt-4":         Cl100kBase,
	"gpt-4-turbo":   Cl100kBase,
	"gpt-4-turbo-preview": Cl100kBase,
	"gpt-4-0125-preview":  Cl100kBase,
	"gpt-4-1106-preview":  Cl100kBase,
	"gpt-4-0613":          Cl100kBase,
	"gpt-4-0314":          Cl100kBase,
	// GPT-3.5 family
	"gpt-3.5-turbo":     Cl100kBase,
	"gpt-3.5-turbo-0125": Cl100kBase,
	"gpt-3.5-turbo-1106": Cl100kBase,
	"gpt-3.5-turbo-0613": Cl100kBase,
	"gpt-3.5-turbo-0301": Cl100kBase,
	// Embedding models
	"text-embedding-ada-002":     Cl100kBase,
	"text-embedding-3-small":     Cl100kBase,
	"text-embedding-3-large":     Cl100kBase,
	// Legacy GPT-3
	"davinci": P50kBase,
	"curie":   P50kBase,
	"babbage": P50kBase,
	"ada":     P50kBase,
	// Claude (approximation - uses similar tokenization)
	"claude-3-opus":   Cl100kBase,
	"claude-3-sonnet": Cl100kBase,
	"claude-3-haiku":  Cl100kBase,
	"claude-3.5-sonnet": Cl100kBase,
	"claude-3.5-haiku":  Cl100kBase,
}

// Tokenizer wraps the tiktoken tokenizer.
type Tokenizer struct {
	codec   tiktoken.Codec
	encName Encoding
}

// New creates a new Tokenizer with the specified encoding.
func New(enc Encoding) (*Tokenizer, error) {
	codec, err := tiktoken.Get(tiktoken.Encoding(enc))
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding %s: %w", enc, err)
	}

	return &Tokenizer{
		codec:   codec,
		encName: enc,
	}, nil
}

// NewForModel creates a Tokenizer for a specific model.
func NewForModel(model string) (*Tokenizer, error) {
	enc, ok := ModelToEncoding[model]
	if !ok {
		// Default to cl100k_base for unknown models
		enc = Cl100kBase
	}
	return New(enc)
}

// Count returns the number of tokens in the given text.
func (t *Tokenizer) Count(text string) int {
	if text == "" {
		return 0
	}
	count, err := t.codec.Count(text)
	if err != nil {
		return EstimateTokens(text)
	}
	return count
}

// CountWithDetails returns token count and tokens.
func (t *Tokenizer) CountWithDetails(text string) (int, []string) {
	if text == "" {
		return 0, nil
	}
	ids, tokens, err := t.codec.Encode(text)
	if err != nil {
		return EstimateTokens(text), nil
	}
	return len(ids), tokens
}

// EncodingName returns the encoding name.
func (t *Tokenizer) EncodingName() Encoding {
	return t.encName
}

// CountFile counts tokens in a file.
func (t *Tokenizer) CountFile(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return t.CountReader(file)
}

// CountReader counts tokens from an io.Reader.
func (t *Tokenizer) CountReader(r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	total := 0

	for scanner.Scan() {
		line := scanner.Text()
		total += t.Count(line)
		// Add 1 for the newline character (models see newlines as tokens)
		total += 1
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading: %w", err)
	}

	return total, nil
}

// CountMessages counts tokens in OpenAI chat message format.
func (t *Tokenizer) CountMessages(messages []Message) int {
	// Every message follows <im_start>{role/name}\n{content}<im_end>\n
	// This adds approximately 4 tokens per message
	tokensPerMessage := 4

	total := 0
	for _, msg := range messages {
		total += tokensPerMessage
		total += t.Count(msg.Role)
		total += t.Count(msg.Content)
		if msg.Name != "" {
			total += t.Count(msg.Name)
			// Name field adds 1 token
			total += 1
		}
	}
	// Every reply is primed with <im_start>assistant
	total += 3
	return total
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// EstimateTokens provides a quick heuristic token count.
// Uses the formula: ceil(text.length / 4.0)
// This is a fallback when tiktoken is not needed.
func EstimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// CompareCounts compares heuristic vs actual token count.
func CompareCounts(text string) (heuristic, actual int, diff float64) {
	heuristic = EstimateTokens(text)
	
	t, err := New(Cl100kBase)
	if err != nil {
		return heuristic, heuristic, 0
	}
	
	actual = t.Count(text)
	
	if actual > 0 {
		diff = float64(heuristic-actual) / float64(actual) * 100
	}
	
	return
}

// FormatCount formats a token count with model context limits.
func FormatCount(count int, model string) string {
	var limit int
	switch {
	case strings.HasPrefix(model, "gpt-4o") || strings.HasPrefix(model, "claude-3"):
		limit = 128000
	case strings.HasPrefix(model, "gpt-4"):
		limit = 128000
	case strings.HasPrefix(model, "gpt-3.5"):
		limit = 16385
	default:
		limit = 8192
	}

	pct := float64(count) / float64(limit) * 100
	return fmt.Sprintf("%d tokens (%.1f%% of %d context)", count, pct, limit)
}

// CountStats holds statistics about token counting.
type CountStats struct {
	TotalTokens int
	TotalChars  int
	TotalLines  int
	FilesCount  int
	Encoding    Encoding
}

// Summary returns a formatted summary of the stats.
func (s *CountStats) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "📊 Token Count Summary\n")
	fmt.Fprintf(&b, "────────────────────────────────────\n")
	fmt.Fprintf(&b, "Encoding: %s\n", s.Encoding)
	fmt.Fprintf(&b, "Tokens:   %d\n", s.TotalTokens)
	fmt.Fprintf(&b, "Chars:    %d\n", s.TotalChars)
	fmt.Fprintf(&b, "Lines:    %d\n", s.TotalLines)
	if s.FilesCount > 0 {
		fmt.Fprintf(&b, "Files:    %d\n", s.FilesCount)
	}
	fmt.Fprintf(&b, "────────────────────────────────────\n")
	if s.TotalChars > 0 {
		ratio := float64(s.TotalTokens) / float64(s.TotalChars)
		fmt.Fprintf(&b, "Token/Char ratio: %.3f\n", ratio)
	}
	return b.String()
}
