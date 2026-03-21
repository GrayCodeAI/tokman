package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var layersCmd = &cobra.Command{
	Use:   "layers",
	Short: "Show the 10-layer compression pipeline architecture",
	Long: `Display information about Tokman's 10-layer compression pipeline.

Each layer is based on cutting-edge research from 2023-2026:

Layer 1: Entropy Filtering (Selective Context, Mila 2023)
Layer 2: Perplexity Pruning (LLMLingua, Microsoft 2023)
Layer 3: Goal-Driven Selection (SWE-Pruner, Shanghai Jiao Tong 2025)
Layer 4: AST Preservation (LongCodeZip, NUS 2025)
Layer 5: Contrastive Ranking (LongLLMLingua, Microsoft 2024)
Layer 6: N-gram Abbreviation (CompactPrompt 2025)
Layer 7: Evaluator Heads (EHPC, Tsinghua/Huawei 2025)
Layer 8: Gist Compression (Stanford/Berkeley 2023)
Layer 9: Hierarchical Summary (AutoCompressor, Princeton/MIT 2023)
Layer 10: Budget Enforcement (Industry standard)

Use --verbose for detailed algorithm explanations.
`,
	Run: runLayers,
}

func init() {
	rootCmd.AddCommand(layersCmd)
	layersCmd.Flags().BoolP("verbose", "v", false, "show detailed layer descriptions")
}

func runLayers(cmd *cobra.Command, args []string) {
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Println("╔═══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           TOKMAN 10-LAYER COMPRESSION PIPELINE                        ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════════════╣")

	layers := []struct {
		num         int
		name        string
		research    string
		compression string
		desc        string
	}{
		{1, "Entropy Filtering", "Selective Context (Mila 2023)", "2-3x",
			"Removes low-information tokens based on entropy scores. Tokens that appear frequently with little variation are pruned."},
		{2, "Perplexity Pruning", "LLMLingua (Microsoft 2023)", "20x",
			"Uses iterative perplexity scoring to identify and remove less important tokens while preserving semantic meaning."},
		{3, "Goal-Driven Selection", "SWE-Pruner (Shanghai Jiao Tong 2025)", "14.8x",
			"CRF-style line scoring based on query intent. Prioritizes content relevant to the task (debug, review, deploy)."},
		{4, "AST Preservation", "LongCodeZip (NUS 2025)", "4-8x",
			"Syntax-aware compression that preserves abstract syntax tree structure while removing redundant code."},
		{5, "Contrastive Ranking", "LongLLMLingua (Microsoft 2024)", "4-10x",
			"Question-relevance scoring using n-gram contrastive analysis between query and context."},
		{6, "N-gram Abbreviation", "CompactPrompt (2025)", "2.5x",
			"Lossless compression of repeated n-grams using dictionary-based abbreviation."},
		{7, "Evaluator Heads", "EHPC (Tsinghua/Huawei 2025)", "5-7x",
			"Simulates early-layer attention heads to identify important tokens without full model inference."},
		{8, "Gist Compression", "Stanford/Berkeley 2023", "20x+",
			"Compresses prompts into 'gist tokens' - virtual tokens representing semantic meaning."},
		{9, "Hierarchical Summary", "AutoCompressor (Princeton/MIT 2023)", "Extreme",
			"Recursive summarization that compresses context into hierarchical summary vectors."},
		{10, "Budget Enforcement", "Industry Standard", "Guaranteed",
			"Strict token limit enforcement with intelligent truncation preserving critical content."},
	}

	for _, l := range layers {
		fmt.Printf("║ Layer %d: %-20s %-15s          ║\n", l.num, l.name, "("+l.compression+")")
		fmt.Printf("║   Research: %-56s ║\n", l.research)
		if verbose {
			fmt.Printf("║   %-68s ║\n", wrapText(l.desc, 68))
		}
		fmt.Println("╟───────────────────────────────────────────────────────────────────────╢")
	}

	fmt.Println("║ PIPELINE ORDER:                                                       ║")
	fmt.Println("║   Statistical → Semantic → Structural → Budget                       ║")
	fmt.Println("║                                                                       ║")
	fmt.Println("║ FEATURES:                                                             ║")
	fmt.Println("║   • Streaming for large inputs (up to 2M tokens)                     ║")
	fmt.Println("║   • Automatic validation and fail-safe mode                          ║")
	fmt.Println("║   • Query-aware compression for specific intents                     ║")
	fmt.Println("║   • Configurable layer thresholds                                     ║")
	fmt.Println("║   • Cache for repeated compressions                                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════╝")

	if !verbose {
		fmt.Println("\n💡 Use --verbose for detailed algorithm explanations")
	}
}

func wrapText(s string, width int) string {
	if len(s) <= width {
		return s
	}

	var result string
	words := splitWords(s)
	lineLen := 0

	for _, word := range words {
		if lineLen+len(word) > width {
			result += "║   " + word
			lineLen = len(word)
		} else {
			if lineLen > 0 {
				result += " "
			}
			result += word
			lineLen += len(word) + 1
		}
	}

	return result
}

func splitWords(s string) []string {
	var words []string
	word := ""
	for _, c := range s {
		if c == ' ' || c == '\n' {
			if word != "" {
				words = append(words, word)
				word = ""
			}
		} else {
			word += string(c)
		}
	}
	if word != "" {
		words = append(words, word)
	}
	return words
}
