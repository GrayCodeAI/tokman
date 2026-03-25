package toml

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/simd"
)

// TOMLFilterEngine applies TOML-defined filter rules to output
type TOMLFilterEngine struct {
	config      *FilterConfig
	compiledRe  []*regexp.Regexp // pre-compiled replace patterns
	compileOnce sync.Once
}

// NewTOMLFilterEngine creates a new filter engine from a TOML config
func NewTOMLFilterEngine(config *FilterConfig) *TOMLFilterEngine {
	return &TOMLFilterEngine{config: config}
}

// Apply applies the filter rules to the input
func (e *TOMLFilterEngine) Apply(input string, mode filter.Mode) (string, int) {
	if e.config == nil {
		return input, 0
	}

	output := input
	originalLen := len(input)

	// Pipeline stages (in order):
	// 1. Strip ANSI
	// 2. Replace (regex substitutions)
	// 3. Match output (short-circuit)
	// 4. Strip/Keep lines
	// 5. Truncate lines
	// 6. Head/Tail selection
	// 7. Max lines cap
	// 8. On empty handling

	// Stage 1: Strip ANSI
	if e.config.StripANSI {
		output = stripANSI(output)
	}

	// Stage 2: Replace patterns
	e.compileOnce.Do(func() {
		e.compiledRe = make([]*regexp.Regexp, len(e.config.Replace))
		for i, rule := range e.config.Replace {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: invalid regex pattern %q: %v\n", rule.Pattern, err)
				continue
			}
			e.compiledRe[i] = re
		}
	})
	for i, rule := range e.config.Replace {
		if e.compiledRe[i] != nil {
			output = e.compiledRe[i].ReplaceAllString(output, rule.Replacement)
		}
	}

	// Stage 3: Match output (short-circuit)
	// If pattern matches, return message immediately.
	// If `unless` is set and also matches, skip this rule (prevents swallowing errors).
	for _, rule := range e.config.MatchOutput {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid regex pattern %q: %v\n", rule.Pattern, err)
			continue
		}
		if re.MatchString(output) {
			// Check unless clause - if set and matches, skip this rule
			if rule.Unless != "" {
				unlessRe, err := regexp.Compile(rule.Unless)
				if err == nil && unlessRe.MatchString(output) {
					continue // unless pattern matches - skip this rule
				}
			}
			output = rule.Message
			return output, originalLen - len(output)
		}
	}

	// Stage 4: Strip/Keep lines
	output = e.filterLines(output)

	// Stage 5: Truncate lines at position
	if e.config.TruncateLinesAt > 0 {
		output = truncateLines(output, e.config.TruncateLinesAt)
	}

	// Stage 6: Head/Tail selection
	if e.config.Head > 0 || e.config.Tail > 0 {
		output = selectHeadTail(output, e.config.Head, e.config.Tail)
	}

	// Stage 7: Max lines cap
	if e.config.MaxLines > 0 {
		output = capLines(output, e.config.MaxLines)
	}

	// Stage 8: On empty handling
	if strings.TrimSpace(output) == "" && e.config.OnEmpty != "" {
		output = e.config.OnEmpty
	}

	tokensSaved := originalLen - len(output)
	if tokensSaved < 0 {
		tokensSaved = 0
	}

	return output, tokensSaved
}

// filterLines filters lines based on strip/keep patterns
func (e *TOMLFilterEngine) filterLines(input string) string {
	if len(e.config.StripLinesMatching) == 0 && len(e.config.KeepLinesMatching) == 0 {
		return input
	}

	// Compile regex patterns once
	stripPatterns := make([]*regexp.Regexp, 0, len(e.config.StripLinesMatching))
	for _, p := range e.config.StripLinesMatching {
		re, err := regexp.Compile(p)
		if err == nil {
			stripPatterns = append(stripPatterns, re)
		}
	}

	keepPatterns := make([]*regexp.Regexp, 0, len(e.config.KeepLinesMatching))
	for _, p := range e.config.KeepLinesMatching {
		re, err := regexp.Compile(p)
		if err == nil {
			keepPatterns = append(keepPatterns, re)
		}
	}

	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := scanner.Text()
		keep := true

		// Check strip patterns
		for _, re := range stripPatterns {
			if re.MatchString(line) {
				keep = false
				break
			}
		}

		// Check keep patterns (if any defined, line must match at least one)
		if keep && len(keepPatterns) > 0 {
			keep = false
			for _, re := range keepPatterns {
				if re.MatchString(line) {
					keep = true
					break
				}
			}
		}

		if keep {
			result.WriteString(line)
			result.WriteByte('\n')
		}
	}

	if err := scanner.Err(); err != nil {
		return input
	}

	return strings.TrimRight(result.String(), "\n")
}

// stripANSI removes ANSI escape sequences from the output
func stripANSI(input string) string {
	return simd.StripANSI(input)
}

// truncateLines truncates each line to maxLen characters
func truncateLines(input string, maxLen int) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > maxLen {
			line = line[:maxLen] + "..."
		}
		result.WriteString(line)
		result.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return input
	}

	return strings.TrimRight(result.String(), "\n")
}

// selectHeadTail selects head and tail portions of the output
func selectHeadTail(input string, head, tail int) string {
	if head == 0 && tail == 0 {
		return input
	}

	lines := strings.Split(input, "\n")
	total := len(lines)

	if head > 0 && tail > 0 {
		if head+tail >= total {
			return input
		}
		selected := make([]string, 0, head+tail+1)
		selected = append(selected, lines[:head]...)
		selected = append(selected, fmt.Sprintf("... [%d lines truncated] ...", total-head-tail))
		selected = append(selected, lines[total-tail:]...)
		return strings.Join(selected, "\n")
	}

	if head > 0 {
		if head >= total {
			return input
		}
		return strings.Join(lines[:head], "\n") + fmt.Sprintf("\n... [%d lines truncated]", total-head)
	}

	if tail > 0 {
		if tail >= total {
			return input
		}
		return fmt.Sprintf("... [%d lines truncated]\n", total-tail) + strings.Join(lines[total-tail:], "\n")
	}

	return input
}

// capLines limits the total number of lines
func capLines(input string, maxLines int) string {
	lines := strings.Split(input, "\n")
	if len(lines) <= maxLines {
		return input
	}

	// Keep half from beginning, half from end
	half := maxLines / 2
	result := make([]string, 0, maxLines+1)
	result = append(result, lines[:half]...)
	result = append(result, fmt.Sprintf("... [%d lines omitted] ...", len(lines)-maxLines))
	result = append(result, lines[len(lines)-half:]...)
	return strings.Join(result, "\n")
}

// ApplyTOMLFilter is a convenience function to apply a TOML filter config
func ApplyTOMLFilter(input string, config *FilterConfig) (string, int) {
	engine := NewTOMLFilterEngine(config)
	return engine.Apply(input, filter.ModeMinimal)
}

// MatchAndFilter finds a matching filter and applies it
func MatchAndFilter(command string, output string, registry *FilterRegistry) (string, int, bool) {
	if registry == nil {
		return output, 0, false
	}

	_, _, config := registry.FindMatchingFilter(command)
	if config == nil {
		return output, 0, false
	}

	filtered, saved := ApplyTOMLFilter(output, config)
	return filtered, saved, true
}

// FilterFilter implements the filter.Filter interface for TOML filters
type TOMLFilterWrapper struct {
	config *FilterConfig
	name   string
}

// NewTOMLFilterWrapper creates a filter.Filter wrapper for a TOML filter
func NewTOMLFilterWrapper(name string, config *FilterConfig) *TOMLFilterWrapper {
	return &TOMLFilterWrapper{
		config: config,
		name:   name,
	}
}

// Name returns the filter name
func (f *TOMLFilterWrapper) Name() string {
	return f.name
}

// Apply implements the filter.Filter interface
func (f *TOMLFilterWrapper) Apply(input string, mode filter.Mode) (string, int) {
	engine := NewTOMLFilterEngine(f.config)
	return engine.Apply(input, mode)
}
