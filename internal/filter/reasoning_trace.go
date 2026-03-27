package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ReasoningTraceFilter compresses chain-of-thought and reasoning model outputs.
// Research Source: "R-KV: Redundancy-aware KV Cache Compression for Reasoning Models" (2025)
// Key Innovation: Detect self-reflection loops in CoT traces and compress repetitive
// reasoning while keeping conclusions and key reasoning steps.
// This is critical for modern reasoning models (o1, Claude thinking, DeepSeek-R1).
type ReasoningTraceFilter struct {
	config ReasoningTraceConfig
}

// ReasoningTraceConfig holds configuration for reasoning trace compression
type ReasoningTraceConfig struct {
	// Enabled controls whether the filter is active
	Enabled bool

	// MaxReflectionLoops is max number of reflection loops to keep
	MaxReflectionLoops int

	// MinTraceLength is minimum tokens to consider CoT compression
	MinTraceLength int

	// PreserveConclusions always keeps conclusion/final answer sections
	PreserveConclusions bool

	// ReflectionPatterns are patterns that indicate self-reflection
	ReflectionPatterns []string
}

// DefaultReasoningTraceConfig returns default configuration
func DefaultReasoningTraceConfig() ReasoningTraceConfig {
	return ReasoningTraceConfig{
		Enabled:             true,
		MaxReflectionLoops:  2,
		MinTraceLength:      50,
		PreserveConclusions: true,
		ReflectionPatterns: []string{
			"let me reconsider",
			"actually,",
			"on second thought",
			"wait,",
			"hmm,",
			"rethinking",
			"revisiting",
			"let me re-examine",
			"correction:",
			"i made a mistake",
			"that's not right",
			"let me try again",
			"alternatively,",
			"another approach",
			"let me think about this differently",
			"reconsidering",
			"actually wait",
			"no, that's wrong",
			"i need to reconsider",
			"let me re-evaluate",
		},
	}
}

// NewReasoningTraceFilter creates a new reasoning trace filter
func NewReasoningTraceFilter() *ReasoningTraceFilter {
	return &ReasoningTraceFilter{
		config: DefaultReasoningTraceConfig(),
	}
}

// Name returns the filter name
func (f *ReasoningTraceFilter) Name() string {
	return "reasoning_trace"
}

// Apply applies reasoning trace compression
func (f *ReasoningTraceFilter) Apply(input string, mode Mode) (string, int) {
	if !f.config.Enabled || mode == ModeNone {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)
	if originalTokens < f.config.MinTraceLength {
		return input, 0
	}

	// Detect if this is a reasoning trace
	if !f.isReasoningTrace(input) {
		return input, 0
	}

	// Split into reasoning blocks
	blocks := f.splitReasoningBlocks(input)

	// Identify reflection loops
	loops := f.detectReflectionLoops(blocks)

	// Compress: keep first N loops + conclusion, summarize the rest
	output := f.compressLoops(blocks, loops, mode)

	finalTokens := core.EstimateTokens(output)
	saved := originalTokens - finalTokens
	if saved < 5 {
		return input, 0
	}

	return output, saved
}

// isReasoningTrace detects if content looks like a CoT/reasoning trace
func (f *ReasoningTraceFilter) isReasoningTrace(input string) bool {
	lower := strings.ToLower(input)

	// Check for reasoning indicators
	indicators := []string{
		"step 1", "step 2", "first,", "second,", "third,",
		"let me", "thinking about", "reasoning:",
		"therefore", "thus", "conclusion:",
		"approach:", "solution:",
	}

	matchCount := 0
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			matchCount++
		}
	}

	// Also check for reflection patterns
	for _, pattern := range f.config.ReflectionPatterns {
		if strings.Contains(lower, pattern) {
			matchCount++
		}
	}

	return matchCount >= 2
}

// reasoningBlock represents a block of reasoning text
type reasoningBlock struct {
	content       string
	isReflection  bool
	isConclusion  bool
	reflectionIdx int
}

// splitReasoningBlocks splits reasoning trace into logical blocks
func (f *ReasoningTraceFilter) splitReasoningBlocks(input string) []reasoningBlock {
	// Split by paragraphs or double newlines
	paragraphs := regexp.MustCompile(`\n\s*\n`).Split(input, -1)

	var blocks []reasoningBlock
	reflectionCount := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		block := reasoningBlock{content: para}

		lower := strings.ToLower(para)

		// Check if this is a conclusion
		if f.config.PreserveConclusions {
			conclusionMarkers := []string{
				"conclusion:", "final answer:", "result:", "therefore,",
				"in summary:", "to summarize:", "the answer is",
				"so,", "thus,",
			}
			for _, marker := range conclusionMarkers {
				if strings.Contains(lower, marker) {
					block.isConclusion = true
					break
				}
			}
		}

		// Check if this is a reflection/reconsideration
		for _, pattern := range f.config.ReflectionPatterns {
			if strings.Contains(lower, pattern) {
				block.isReflection = true
				block.reflectionIdx = reflectionCount
				reflectionCount++
				break
			}
		}

		blocks = append(blocks, block)
	}

	return blocks
}

// detectReflectionLoops groups reflection blocks into loops
func (f *ReasoningTraceFilter) detectReflectionLoops(blocks []reasoningBlock) []int {
	var loopStarts []int
	inLoop := false
	loopStart := -1

	for i, block := range blocks {
		if block.isReflection {
			if !inLoop {
				loopStart = i
				inLoop = true
			}
		} else if inLoop {
			// End of reflection loop
			loopStarts = append(loopStarts, loopStart)
			inLoop = false
		}
	}

	// Handle case where loop extends to end
	if inLoop {
		loopStarts = append(loopStarts, loopStart)
	}

	return loopStarts
}

// compressLoops compresses reasoning by keeping first N loops + conclusions
func (f *ReasoningTraceFilter) compressLoops(blocks []reasoningBlock, loopStarts []int, mode Mode) string {
	maxLoops := f.config.MaxReflectionLoops
	if mode == ModeAggressive {
		maxLoops = 1
	}

	// Determine which blocks to keep
	keep := make([]bool, len(blocks))

	// Always keep the first occurrence of each reasoning step
	for i := range blocks {
		keep[i] = true
	}

	// For each reflection loop beyond maxLoops, replace with summary
	loopsCompressed := 0
	for _, start := range loopStarts {
		loopsCompressed++
		if loopsCompressed > maxLoops {
			// Mark this loop's blocks for compression (except conclusion)
			for j := start; j < len(blocks); j++ {
				if blocks[j].isConclusion {
					break
				}
				if blocks[j].isReflection {
					keep[j] = false
				}
			}
		}
	}

	// Reconstruct output
	var result strings.Builder
	for i, block := range blocks {
		if keep[i] {
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(block.content)
		} else if i > 0 && !keep[i-1] {
			// First compressed block in a sequence - add summary marker
			continue
		}
	}

	// Add compression indicator if we compressed anything
	totalCompressed := 0
	for _, k := range keep {
		if !k {
			totalCompressed++
		}
	}
	if totalCompressed > 0 {
		result.WriteString("\n[")
		result.WriteString(fmt.Sprintf("%d", totalCompressed))
		result.WriteString(" reasoning steps compressed]")
	}

	return result.String()
}
