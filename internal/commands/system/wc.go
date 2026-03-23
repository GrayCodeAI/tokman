package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var wcCmd = &cobra.Command{
	Use:   "wc [args...]",
	Short: "Word/line/byte count with compact output",
	Long: `Word/line/byte count with token-optimized output.

Strips paths and padding for minimal token usage.

Examples:
  tokman wc -l file.go
  tokman wc -w *.go
  tokman wc *.py  (shows: 30L 96W 978B file.py)`,
	RunE: runWc,
}

func init() {
	registry.Add(func() { registry.Register(wcCmd) })
}

// WcMode represents which columns the user requested
type WcMode int

const (
	WcModeFull  WcMode = iota // Default: lines, words, bytes (3 columns)
	WcModeLines               // Lines only (-l)
	WcModeWords               // Words only (-w)
	WcModeBytes               // Bytes only (-c)
	WcModeChars               // Chars only (-m)
	WcModeMixed               // Multiple flags combined
)

func runWc(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	c := exec.Command("wc", args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String()
	if output == "" && stderr.Len() > 0 {
		output = stderr.String()
	}

	// Detect mode from args
	mode := detectWcMode(args)

	// Compact output with smart formatting
	filtered := compactWcOutput(output, mode)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("wc %s", strings.Join(args, " ")), "tokman wc", originalTokens, filteredTokens)

	return err
}

// detectWcMode determines which columns the user requested
func detectWcMode(args []string) WcMode {
	flags := 0
	hasL, hasW, hasC, hasM := false, false, false, false

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		// Handle combined flags like -lw
		for _, ch := range arg[1:] {
			switch ch {
			case 'l':
				hasL = true
				flags++
			case 'w':
				hasW = true
				flags++
			case 'c':
				hasC = true
				flags++
			case 'm':
				hasM = true
				flags++
			}
		}
	}

	if flags == 0 {
		return WcModeFull
	}
	if flags > 1 {
		return WcModeMixed
	}

	switch {
	case hasL:
		return WcModeLines
	case hasW:
		return WcModeWords
	case hasC:
		return WcModeBytes
	case hasM:
		return WcModeChars
	default:
		return WcModeFull
	}
}

// compactWcOutput formats wc output for minimal token usage
func compactWcOutput(output string, mode WcMode) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return ""
	}

	// Single file (one output line)
	if len(lines) == 1 {
		return formatSingleLine(lines[0], mode) + "\n"
	}

	// Multiple files — compact table
	return formatMultiLine(lines, mode) + "\n"
}

// formatSingleLine formats a single wc output line
func formatSingleLine(line string, mode WcMode) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}

	switch mode {
	case WcModeLines, WcModeWords, WcModeBytes, WcModeChars:
		// First number is the only requested column
		if len(fields) >= 1 {
			return fields[0]
		}
		return line

	case WcModeFull:
		if len(fields) >= 3 {
			// Check if last field is a path
			last := fields[len(fields)-1]
			if isNumeric(last) {
				// No path, just numbers (stdin)
				return fmt.Sprintf("%sL %sW %sB", fields[0], fields[1], fields[2])
			}
			return fmt.Sprintf("%sL %sW %sB %s", fields[0], fields[1], fields[2], last)
		}
		return line

	case WcModeMixed:
		// Strip file path, keep numbers only
		if len(fields) >= 2 {
			last := fields[len(fields)-1]
			if !isNumeric(last) {
				return strings.Join(fields[:len(fields)-1], " ") + " " + last
			}
		}
		return strings.Join(fields, " ")

	default:
		return line
	}
}

// formatMultiLine formats multiple files as a compact table
func formatMultiLine(lines []string, mode WcMode) string {
	// Extract paths and find common prefix
	var paths []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			last := fields[len(fields)-1]
			if last != "total" && !isNumeric(last) {
				paths = append(paths, last)
			}
		}
	}

	commonPrefix := findCommonPrefix(paths)

	var result []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		isTotal := fields[len(fields)-1] == "total"

		switch mode {
		case WcModeLines, WcModeWords, WcModeBytes, WcModeChars:
			if isTotal {
				result = append(result, fmt.Sprintf("Σ %s", fields[0]))
			} else if len(fields) >= 2 {
				name := stripPrefix(fields[len(fields)-1], commonPrefix)
				result = append(result, fmt.Sprintf("%s %s", fields[0], name))
			}

		case WcModeFull:
			if isTotal {
				if len(fields) >= 3 {
					result = append(result, fmt.Sprintf("Σ %sL %sW %sB", fields[0], fields[1], fields[2]))
				}
			} else if len(fields) >= 4 {
				name := stripPrefix(fields[3], commonPrefix)
				result = append(result, fmt.Sprintf("%sL %sW %sB %s", fields[0], fields[1], fields[2], name))
			}

		case WcModeMixed:
			if isTotal {
				nums := fields[:len(fields)-1]
				result = append(result, fmt.Sprintf("Σ %s", strings.Join(nums, " ")))
			} else if len(fields) >= 2 {
				last := fields[len(fields)-1]
				if !isNumeric(last) {
					name := stripPrefix(last, commonPrefix)
					nums := fields[:len(fields)-1]
					result = append(result, fmt.Sprintf("%s %s", strings.Join(nums, " "), name))
				}
			}
		}
	}

	return strings.Join(result, "\n")
}

// findCommonPrefix finds the common directory prefix among paths
func findCommonPrefix(paths []string) string {
	if len(paths) <= 1 {
		return ""
	}

	// Find the longest common prefix ending with /
	first := paths[0]
	var prefix string
	if idx := strings.LastIndex(first, "/"); idx >= 0 {
		prefix = first[:idx+1]
	}

	// Check if all paths start with this prefix
	for _, p := range paths {
		if !strings.HasPrefix(p, prefix) {
			// Try shorter prefixes
			for i := len(prefix) - 1; i >= 0; i-- {
				if prefix[i] == '/' {
					candidate := prefix[:i+1]
					allMatch := true
					for _, p2 := range paths {
						if !strings.HasPrefix(p2, candidate) {
							allMatch = false
							break
						}
					}
					if allMatch {
						return candidate
					}
				}
			}
			return ""
		}
	}

	return prefix
}

// stripPrefix removes the common prefix from a path
func stripPrefix(path, prefix string) string {
	if prefix == "" {
		return path
	}
	return strings.TrimPrefix(path, prefix)
}

// isNumeric checks if a string is a number
func isNumeric(s string) bool {
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return len(s) > 0
}
