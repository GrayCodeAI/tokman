package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ProtoCompressFilter compresses Protobuf, Thrift, and Avro schema files.
// Removes boilerplate and normalizes field definitions to reduce token usage
// while preserving the schema structure.
//
// Steps:
//  1. Strip block comments (/* ... */) and line comments (// ...)
//  2. Remove blank lines between fields inside message/struct blocks
//  3. Collapse multi-line field definitions to single lines
//  4. In aggressive mode: remove option/annotation lines, collapse enum values
type ProtoCompressFilter struct{}

var (
	// Proto/Thrift block comments
	protoBlockCommentRe = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	// Proto/Thrift/Avro line comments
	protoLineCommentRe = regexp.MustCompile(`(?m)^\s*//[^\n]*$`)
	// Thrift doc comments
	thriftDocCommentRe = regexp.MustCompile(`(?m)^\s*#[^\n]*$`)
	// Proto option lines (generate redundant noise)
	protoOptionRe = regexp.MustCompile(`(?m)^\s*option\s+[^\n]+;\s*$`)
	// Proto reserved lines
	protoReservedRe = regexp.MustCompile(`(?m)^\s*reserved\s+[^\n]+;\s*$`)
	// Avro/Thrift annotation lines
	thriftAnnotationRe = regexp.MustCompile(`(?m)^\s*@[A-Za-z][^\n]*$`)
	// Detect proto/thrift content
	protoDetectRe = regexp.MustCompile(`(?:message|syntax|enum|service|rpc|struct|union|exception|namespace)\s+\w+`)
)

// NewProtoCompressFilter creates a new proto/thrift compression filter.
func NewProtoCompressFilter() *ProtoCompressFilter {
	return &ProtoCompressFilter{}
}

// Name returns the filter name.
func (f *ProtoCompressFilter) Name() string {
	return "proto_compress"
}

// Apply compresses proto/thrift/avro schema content.
func (f *ProtoCompressFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Only apply to schema-like content
	if !protoDetectRe.MatchString(input) {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input

	// Strip block comments
	output = protoBlockCommentRe.ReplaceAllString(output, "")

	// Strip line comments
	output = protoLineCommentRe.ReplaceAllString(output, "")
	output = thriftDocCommentRe.ReplaceAllString(output, "")

	if mode == ModeAggressive {
		// Remove option lines (proto) and annotations (thrift)
		output = protoOptionRe.ReplaceAllString(output, "")
		output = protoReservedRe.ReplaceAllString(output, "")
		output = thriftAnnotationRe.ReplaceAllString(output, "")

		// Collapse enum blocks
		output = f.collapseEnumBlocks(output)
	}

	// Compact field lines inside message/struct blocks
	output = f.compactFieldLines(output)

	// Collapse blank lines
	output = collapseBlankLines(output, 1)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// compactFieldLines removes extra blank lines inside message/struct bodies
// and normalizes whitespace on field definition lines.
func (f *ProtoCompressFilter) compactFieldLines(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	depth := 0
	prevBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track brace depth
		opens := strings.Count(trimmed, "{")
		closes := strings.Count(trimmed, "}")
		depth += opens - closes
		if depth < 0 {
			depth = 0
		}

		isBlank := trimmed == ""

		// Inside a block: suppress consecutive blank lines
		if depth > 0 && isBlank {
			if prevBlank {
				continue
			}
		}

		// Normalize indentation inside blocks to 2-space
		if depth > 0 && !isBlank && !strings.HasPrefix(trimmed, "{") && !strings.HasSuffix(trimmed, "{") {
			line = strings.Repeat("  ", depth) + trimmed
		}

		result = append(result, line)
		prevBlank = isBlank
	}
	return strings.Join(result, "\n")
}

// collapseEnumBlocks collapses enum bodies with many values into compact form.
func (f *ProtoCompressFilter) collapseEnumBlocks(input string) string {
	// Match enum { ... } blocks with many (>6) values
	enumBlockRe := regexp.MustCompile(`(?s)(enum\s+\w+\s*\{)([^{}]+)(\})`)
	return enumBlockRe.ReplaceAllStringFunc(input, func(m string) string {
		parts := enumBlockRe.FindStringSubmatch(m)
		if len(parts) < 4 {
			return m
		}
		header, body, footer := parts[1], parts[2], parts[3]

		// Count enum value lines
		valueLines := []string{}
		for _, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				valueLines = append(valueLines, trimmed)
			}
		}
		if len(valueLines) <= 6 {
			return m
		}
		// Show first 3 and last 1 with summary
		shown := append(valueLines[:3], fmt.Sprintf("/* ... %d more values ... */", len(valueLines)-4))
		shown = append(shown, valueLines[len(valueLines)-1])
		return header + "\n  " + strings.Join(shown, "\n  ") + "\n" + footer
	})
}
