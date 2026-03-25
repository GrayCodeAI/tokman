package filter

import (
	"strconv"
	"strings"
)

// SmallKVCompensator implements SmallKV-style small model compensation.
// Research Source: "SmallKV: Small Model Assisted Compensation of KV Cache
// Compression for Efficient LLM Inference" (2025)
// Key Innovation: When aggressive compression removes important tokens, use a
// lightweight reconstruction pass to compensate for lost information.
//
// This works by: after compression, check if critical information patterns
// were broken (unclosed brackets, incomplete statements, missing context)
// and reconstruct minimal bridges to maintain coherence.
type SmallKVCompensator struct {
	config SmallKVConfig
}

// SmallKVConfig holds configuration for SmallKV compensation
type SmallKVConfig struct {
	// Enabled controls whether the compensator is active
	Enabled bool

	// MinContentLength minimum chars to apply
	MinContentLength int

	// MaxBridgeTokens maximum tokens to add as compensation
	MaxBridgeTokens int

	// CheckSyntaxIntegrity verifies bracket/paren matching
	CheckSyntaxIntegrity bool

	// CheckContextContinuity verifies logical flow preservation
	CheckContextContinuity bool
}

// DefaultSmallKVConfig returns default configuration
func DefaultSmallKVConfig() SmallKVConfig {
	return SmallKVConfig{
		Enabled:                true,
		MinContentLength:       100,
		MaxBridgeTokens:        20,
		CheckSyntaxIntegrity:   true,
		CheckContextContinuity: true,
	}
}

// NewSmallKVCompensator creates a new SmallKV compensator
func NewSmallKVCompensator() *SmallKVCompensator {
	return &SmallKVCompensator{
		config: DefaultSmallKVConfig(),
	}
}

// Name returns the filter name
func (s *SmallKVCompensator) Name() string {
	return "smallkv"
}

// Compensate adds bridge tokens to compensate for over-compression.
// This runs AFTER other filters to repair damage from aggressive compression.
func (s *SmallKVCompensator) Compensate(original, compressed string, mode Mode) string {
	if !s.config.Enabled || mode == ModeNone {
		return compressed
	}

	if len(compressed) < s.config.MinContentLength {
		return compressed
	}

	output := compressed

	// Check and fix syntax integrity
	if s.config.CheckSyntaxIntegrity {
		output = s.fixSyntaxIntegrity(output)
	}

	// Check and fix context continuity
	if s.config.CheckContextContinuity {
		output = s.fixContextContinuity(original, output)
	}

	// Add section markers for readability
	output = s.addSectionMarkers(output)

	return output
}

// Apply implements the Filter interface for pipeline integration
func (s *SmallKVCompensator) Apply(input string, mode Mode) (string, int) {
	// SmallKV doesn't compress - it compensates
	// It should run as a post-processing step
	return input, 0
}

// fixSyntaxIntegrity repairs broken bracket/paren matching
func (s *SmallKVCompensator) fixSyntaxIntegrity(input string) string {
	lines := strings.Split(input, "\n")
	var result strings.Builder

	openBrackets := 0
	openParens := 0
	openSquares := 0

	for _, line := range lines {
		for _, ch := range line {
			switch ch {
			case '{':
				openBrackets++
			case '}':
				openBrackets--
			case '(':
				openParens++
			case ')':
				openParens--
			case '[':
				openSquares++
			case ']':
				openSquares--
			}
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	// Add closing brackets if needed
	output := result.String()
	if openBrackets > 0 {
		for i := 0; i < openBrackets && i < s.config.MaxBridgeTokens; i++ {
			output += "}\n"
		}
	}
	if openParens > 0 {
		for i := 0; i < openParens && i < s.config.MaxBridgeTokens; i++ {
			output += ")\n"
		}
	}
	if openSquares > 0 {
		for i := 0; i < openSquares && i < s.config.MaxBridgeTokens; i++ {
			output += "]\n"
		}
	}

	return output
}

// fixContextContinuity adds minimal context bridges where compression broke flow
func (s *SmallKVCompensator) fixContextContinuity(original, compressed string) string {
	origLines := strings.Split(original, "\n")
	compLines := strings.Split(compressed, "\n")

	// Detect if we lost important context markers
	markers := []string{
		"func ", "function ", "def ", "class ", "struct ",
		"if ", "for ", "while ", "switch ", "match ",
		"return ", "yield ", "break ", "continue ",
	}

	var bridges []string
	for _, marker := range markers {
		origHas := false
		compHas := false

		for _, line := range origLines {
			if strings.Contains(line, marker) {
				origHas = true
				break
			}
		}
		for _, line := range compLines {
			if strings.Contains(line, marker) {
				compHas = true
				break
			}
		}

		// If original had the marker but compressed doesn't, add a bridge
		if origHas && !compHas && len(bridges) < s.config.MaxBridgeTokens {
			bridges = append(bridges, "[... "+strings.TrimSpace(marker)+"... ]")
		}
	}

	if len(bridges) == 0 {
		return compressed
	}

	// Insert bridges at logical points
	result := compressed
	for _, bridge := range bridges {
		result += "\n" + bridge
	}

	return result
}

// addSectionMarkers adds minimal section markers for readability
func (s *SmallKVCompensator) addSectionMarkers(input string) string {
	lines := strings.Split(input, "\n")

	// Detect if content was heavily compressed (few lines, many compression markers)
	compressionMarkers := 0
	for _, line := range lines {
		if strings.Contains(line, "[×") ||
			strings.Contains(line, "[...") ||
			strings.Contains(line, "compressed]") {
			compressionMarkers++
		}
	}

	if compressionMarkers > len(lines)/3 {
		// Heavily compressed - add a summary header
		return "[Compressed output - " + strconv.Itoa(len(lines)) + " lines from original]\n" + input
	}

	return input
}

