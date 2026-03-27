package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// AgentCompressFilter applies agent-aware compression optimized for content
// prepared for AI agent consumption: tool outputs, file reads, search results.
//
// Agent-specific strategy:
//   - Preserve structured data (JSON, code, tables) at high fidelity
//   - Compress prose/description sections more aggressively
//   - Remove redundant tool call scaffolding and verbose headers
//   - Collapse repeated tool output patterns (same tool, similar results)
//   - Preserve error messages and numerical data completely
//
// This filter should run early in the pipeline for agent-mode inputs.
type AgentCompressFilter struct{}

var (
	// Tool output headers: common patterns in agentic frameworks
	toolHeaderRe = regexp.MustCompile(
		`(?m)^(?:Tool output:|Result from \w+:|Calling tool:|<tool_result>|<result>|` +
			`RESULT:|OUTPUT:|RESPONSE:)\s*`,
	)
	// Verbose XML/JSON wrapper tags common in tool use
	toolWrapperRe = regexp.MustCompile(
		`(?s)<(?:tool_result|tool_use|function_calls|function_results|invoke)[^>]*>.*?</(?:tool_result|tool_use|function_calls|function_results|invoke)>`,
	)
	// Repeated "Searching..." / "Reading..." progress lines from agents
	agentProgressRe = regexp.MustCompile(
		`(?m)^(?:Searching for|Reading file|Fetching|Retrieving|Loading|Scanning)[^\n]*\.\.\.\s*$`,
	)
	// Verbose search result headers ("Result 1 of 20", "Source: https://...")
	searchHeaderRe = regexp.MustCompile(
		`(?m)^(?:Result \d+ of \d+|Source: https?://\S+|Relevance score: \d+\.\d+|Rank: \d+)\s*$`,
	)
	// Redundant observation prefixes in ReAct/tool loops
	observationRe = regexp.MustCompile(
		`(?m)^(?:Observation:|Action Input:|Thought:|Final Answer:)\s*`,
	)
)

// NewAgentCompressFilter creates an agent-aware compression filter.
func NewAgentCompressFilter() *AgentCompressFilter {
	return &AgentCompressFilter{}
}

// Name returns the filter name.
func (f *AgentCompressFilter) Name() string {
	return "agent_compress"
}

// Apply applies agent-specific compression.
func (f *AgentCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input

	// Always: remove tool output headers
	output = toolHeaderRe.ReplaceAllString(output, "")

	// Always: remove agent progress lines
	output = agentProgressRe.ReplaceAllString(output, "")

	// Always: remove search result scaffolding
	output = searchHeaderRe.ReplaceAllString(output, "")

	// Remove ReAct observation/thought prefixes (they add tokens without value)
	output = observationRe.ReplaceAllString(output, "")

	if mode == ModeAggressive {
		// Aggressive: remove verbose XML tool wrappers, keeping inner content
		output = toolWrapperRe.ReplaceAllStringFunc(output, func(m string) string {
			// Extract inner content between tags
			open := strings.Index(m, ">")
			close := strings.LastIndex(m, "</")
			if open >= 0 && close > open {
				return strings.TrimSpace(m[open+1 : close])
			}
			return m
		})

		// Compress repeated identical tool outputs
		output = f.deduplicateToolOutputs(output)
	}

	// Clean up blank lines
	output = collapseBlankLines(output, 2)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// deduplicateToolOutputs collapses identical consecutive tool output blocks.
func (f *AgentCompressFilter) deduplicateToolOutputs(input string) string {
	// Split on blank lines (each block = one tool output)
	blocks := strings.Split(input, "\n\n")
	if len(blocks) < 3 {
		return input
	}

	seen := make(map[string]int) // block hash → first occurrence index
	result := make([]string, 0, len(blocks))

	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if len(trimmed) < 50 {
			result = append(result, block)
			continue
		}
		// Use first 100 chars as a quick fingerprint
		key := trimmed
		if len(key) > 100 {
			key = key[:100]
		}
		if count, ok := seen[key]; ok {
			// Already seen: skip and note the count
			seen[key] = count + 1
			continue
		}
		seen[key] = 1
		result = append(result, block)
	}

	return strings.Join(result, "\n\n")
}
