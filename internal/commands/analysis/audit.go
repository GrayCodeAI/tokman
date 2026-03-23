package analysis

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var auditCmd = &cobra.Command{
	Use:   "audit [file]",
	Short: "Analyze compression performance on a file or stdin",
	Long: `Audit evaluates the 10-layer compression pipeline on input content.

Shows layer-by-layer statistics, compression ratios, and warnings.

Examples:
  tokman audit output.txt
  git status | tokman audit -
  tokman audit --mode aggressive --budget 500 large_file.txt
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAudit,
}

func init() {
	registry.Add(func() { registry.Register(auditCmd) })
	auditCmd.Flags().StringP("mode", "m", "aggressive", "compression mode: minimal, aggressive")
	auditCmd.Flags().IntP("budget", "b", 0, "token budget (0 = unlimited)")
	auditCmd.Flags().String("query", "", "query intent for query-aware compression")
	auditCmd.Flags().Bool("layers", true, "show layer-by-layer breakdown")
	auditCmd.Flags().Bool("validate", true, "validate output quality")
	auditCmd.Flags().Bool("json", false, "output in JSON format")
}

func runAudit(cmd *cobra.Command, args []string) error {
	start := time.Now()

	// Parse flags
	modeStr, _ := cmd.Flags().GetString("mode")
	budget, _ := cmd.Flags().GetInt("budget")
	query, _ := cmd.Flags().GetString("query")
	showLayers, _ := cmd.Flags().GetBool("layers")
	validate, _ := cmd.Flags().GetBool("validate")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Convert mode
	var mode filter.Mode
	switch modeStr {
	case "minimal":
		mode = filter.ModeMinimal
	case "aggressive":
		mode = filter.ModeAggressive
	default:
		mode = filter.ModeAggressive
	}

	// Read input
	var input string
	var err error

	if len(args) > 0 && args[0] != "-" {
		// Read from file
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		input = string(data)
	} else {
		// Read from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("no input provided (use - for stdin or specify a file)")
		}
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		input = string(data)
	}

	if input == "" {
		return fmt.Errorf("input is empty")
	}

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create pipeline manager
	pipelineCfg := convertConfigToPipeline(cfg.Pipeline)
	manager := filter.NewPipelineManager(filter.ManagerConfig{
		MaxContextTokens:   cfg.Pipeline.MaxContextTokens,
		ChunkSize:          cfg.Pipeline.ChunkSize,
		StreamThreshold:    cfg.Pipeline.StreamThreshold,
		TeeOnFailure:       cfg.Pipeline.TeeOnFailure,
		FailSafeMode:       cfg.Pipeline.FailSafeMode,
		ValidateOutput:     validate,
		ShortCircuitBudget: cfg.Pipeline.ShortCircuitBudget,
		CacheEnabled:       false, // Disable cache for accurate audit
		PipelineCfg:        pipelineCfg,
	})

	// Create command context
	ctx := filter.CommandContext{
		Intent: query,
	}

	// Process with budget if specified
	var result *filter.ProcessResult
	if budget > 0 {
		result, err = manager.ProcessWithBudget(input, mode, budget, ctx)
	} else if query != "" {
		result, err = manager.ProcessWithQuery(input, mode, query, ctx)
	} else {
		result, err = manager.Process(input, mode, ctx)
	}

	if err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	duration := time.Since(start)

	// Output results
	if jsonOutput {
		auditOutputJSON(result, duration)
	} else {
		outputText(result, duration, showLayers)
	}

	return nil
}

func outputText(result *filter.ProcessResult, duration time.Duration, showLayers bool) {
	// Header
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    TOKMAN COMPRESSION AUDIT                     ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	// Token stats
	fmt.Printf("║ Original Tokens:     %10d                           ║\n", result.OriginalTokens)
	fmt.Printf("║ Final Tokens:        %10d                           ║\n", result.FinalTokens)
	fmt.Printf("║ Tokens Saved:        %10d                           ║\n", result.SavedTokens)
	fmt.Printf("║ Compression Ratio:   %10.1f%%                          ║\n", result.ReductionPercent)
	fmt.Printf("║ Processing Time:     %10v                           ║\n", duration.Round(time.Microsecond))

	// Additional info
	if result.CacheHit {
		fmt.Println("║ Cache:               HIT (cached result)                   ║")
	}
	if result.Chunks > 1 {
		fmt.Printf("║ Chunks Processed:    %10d                           ║\n", result.Chunks)
	}
	if result.Validated {
		fmt.Println("║ Validation:          PASSED                               ║")
	}
	if result.Warning != "" {
		fmt.Printf("║ Warning:             %-36s ║\n", auditTruncate(result.Warning, 36))
	}
	if result.TeeFile != "" {
		fmt.Printf("║ Tee File:            %-36s ║\n", auditTruncate(result.TeeFile, 36))
	}

	// Layer breakdown
	if showLayers && len(result.LayerStats) > 0 {
		fmt.Println("╠════════════════════════════════════════════════════════════════╣")
		fmt.Println("║ LAYER BREAKDOWN:                                               ║")
		fmt.Println("╠════════════════════════════════════════════════════════════════╣")

		// Ordered layers
		layerOrder := []string{
			"1_entropy", "2_perplexity", "3_goal_driven", "4_ast_preserve",
			"5_contrastive", "6_ngram", "7_evaluator", "8_gist", "9_hierarchical",
			"neural", "10_session", "10_budget", "10_total",
		}

		totalLayers := 0
		for _, layer := range layerOrder {
			if stat, ok := result.LayerStats[layer]; ok && stat.TokensSaved > 0 {
				fmt.Printf("║   %-24s: %10d tokens saved      ║\n", layer, stat.TokensSaved)
				totalLayers++
			}
		}

		if totalLayers == 0 {
			fmt.Println("║   (No layers contributed significant savings)                 ║")
		}
	}

	// Footer
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	// Quality assessment
	if result.ReductionPercent > 50 {
		fmt.Println("\n✅ Excellent compression! Significant token savings.")
	} else if result.ReductionPercent > 20 {
		fmt.Println("\n👍 Good compression. Moderate token savings.")
	} else if result.ReductionPercent > 0 {
		fmt.Println("\n💡 Minimal compression. Consider using aggressive mode.")
	} else {
		fmt.Println("\n⚠️  No compression achieved. Input may already be optimized.")
	}
}

func auditOutputJSON(result *filter.ProcessResult, duration time.Duration) {
	fmt.Printf(`{
  "original_tokens": %d,
  "final_tokens": %d,
  "saved_tokens": %d,
  "compression_percent": %.1f,
  "processing_time_ms": %d,
  "cache_hit": %v,
  "chunks": %d,
  "validated": %v,
  "warning": %q,
  "tee_file": %q,
  "layer_stats": {
`, result.OriginalTokens, result.FinalTokens, result.SavedTokens,
		result.ReductionPercent, duration.Milliseconds(),
		result.CacheHit, result.Chunks, result.Validated,
		result.Warning, result.TeeFile)

	first := true
	for layer, stat := range result.LayerStats {
		if !first {
			fmt.Printf(",\n")
		}
		fmt.Printf(`    "%s": {"tokens_saved": %d}`, layer, stat.TokensSaved)
		first = false
	}

	fmt.Println("\n  }\n}")
}

func auditTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// convertConfigToPipeline converts config.PipelineConfig to filter.PipelineConfig
func convertConfigToPipeline(cfg config.PipelineConfig) filter.PipelineConfig {
	return filter.PipelineConfig{
		Mode:               filter.ModeMinimal, // Will be overridden by manager
		QueryIntent:        "",
		Budget:             cfg.DefaultBudget,
		LLMEnabled:         false,
		SessionTracking:    true,
		NgramEnabled:       cfg.EnableNgram,
		PromptTemplate:     "",
		EnableEntropy:      cfg.EnableEntropy,
		EnablePerplexity:   cfg.EnablePerplexity,
		EnableGoalDriven:   cfg.EnableGoalDriven,
		EnableAST:          cfg.EnableAST,
		EnableContrastive:  cfg.EnableContrastive,
		EnableEvaluator:    cfg.EnableEvaluator,
		EnableGist:         cfg.EnableGist,
		EnableHierarchical: cfg.EnableHierarchical,
	}
}
