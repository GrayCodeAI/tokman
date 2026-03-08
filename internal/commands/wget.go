package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var wgetStdout bool

var wgetCmd = &cobra.Command{
	Use:   "wget <url> [args...]",
	Short: "Download with compact output (strips progress bars)",
	Long: `Download files with token-optimized output.

Strips progress bars and shows only essential result information.

Examples:
  tokman wget https://example.com/file.zip
  tokman wget -O output.txt https://example.com/data.txt
  tokman wget --stdout https://example.com/data.txt`,
	Args: cobra.MinimumNArgs(1),
	RunE: runWget,
}

func init() {
	rootCmd.AddCommand(wgetCmd)
	wgetCmd.Flags().BoolVarP(&wgetStdout, "stdout", "O", false, "Output to stdout instead of file")
}

func runWget(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()
	url := args[0]
	userArgs := args[1:]

	if wgetStdout {
		return runWgetStdout(url, userArgs, timer)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "wget: %s\n", url)
	}

	// Run wget and capture output
	wgetArgs := userArgs
	wgetArgs = append(wgetArgs, url)

	execCmd := exec.Command("wget", wgetArgs...)
	output, err := execCmd.CombinedOutput()

	stderr := string(output)
	raw := stderr

	if err == nil {
		filename := extractFilename(stderr, url, userArgs)
		size := getFileSize(filename)
		msg := fmt.Sprintf("⬇️  %s ok | %s | %s", compactURL(url), filename, formatSize(size))
		fmt.Println(msg)

		originalTokens := filter.EstimateTokens(raw)
		filteredTokens := filter.EstimateTokens(msg)
		timer.Track(fmt.Sprintf("wget %s", url), "tokman wget", originalTokens, filteredTokens)
	} else {
		errorMsg := parseWgetError(stderr)
		msg := fmt.Sprintf("⬇️  %s FAILED: %s", compactURL(url), errorMsg)
		fmt.Println(msg)

		originalTokens := filter.EstimateTokens(raw)
		filteredTokens := filter.EstimateTokens(msg)
		timer.Track(fmt.Sprintf("wget %s", url), "tokman wget", originalTokens, filteredTokens)
		return err
	}

	return nil
}

func runWgetStdout(url string, userArgs []string, timer *tracking.TimedExecution) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "wget: %s -> stdout\n", url)
	}

	// wget -q -O - for stdout output
	wgetArgs := []string{"-q", "-O", "-"}
	wgetArgs = append(wgetArgs, userArgs...)
	wgetArgs = append(wgetArgs, url)

	execCmd := exec.Command("wget", wgetArgs...)
	output, err := execCmd.CombinedOutput()

	if err != nil {
		errorMsg := parseWgetError(string(output))
		msg := fmt.Sprintf("⬇️  %s FAILED: %s", compactURL(url), errorMsg)
		fmt.Println(msg)
		return err
	}

	content := string(output)
	lines := strings.Split(content, "\n")
	total := len(lines)
	raw := content

	var rtkOutput string
	if total > 20 {
		rtkOutput = fmt.Sprintf("⬇️  %s ok | %d lines | %s\n", compactURL(url), total, formatSize(uint64(len(output))))
		rtkOutput += "--- first 10 lines ---\n"
		for i := 0; i < 10 && i < len(lines); i++ {
			rtkOutput += truncateLine(lines[i], 100) + "\n"
		}
		rtkOutput += fmt.Sprintf("... +%d more lines", total-10)
	} else {
		rtkOutput = fmt.Sprintf("⬇️  %s ok | %d lines\n", compactURL(url), total)
		rtkOutput += content
	}

	fmt.Print(rtkOutput)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(rtkOutput)
	timer.Track(fmt.Sprintf("wget -O - %s", url), "tokman wget -O", originalTokens, filteredTokens)

	return nil
}

// extractFilename parses wget output or args to find the saved filename.
func extractFilename(stderr, url string, args []string) string {
	// Check for -O argument first
	for i, arg := range args {
		if arg == "-O" || arg == "--output-document" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		if strings.HasPrefix(arg, "-O") && len(arg) > 2 {
			return arg[2:]
		}
	}

	// Parse wget output for "Saving to" or similar patterns
	for _, line := range strings.Split(stderr, "\n") {
		if strings.Contains(line, "Saving to") || strings.Contains(line, "saved") {
			// Extract filename from quotes
			start := strings.IndexAny(line, "'\"«")
			end := strings.LastIndexAny(line, "'\"»")
			if start >= 0 && end > start {
				return strings.TrimSpace(line[start+1 : end])
			}
		}
	}

	// Fallback: extract from URL
	path := url
	if idx := strings.Index(url, "://"); idx >= 0 {
		path = url[idx+3:]
	}
	filename := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		filename = path[idx+1:]
	}
	if idx := strings.Index(filename, "?"); idx >= 0 {
		filename = filename[:idx]
	}
	if filename == "" || !strings.Contains(filename, ".") {
		return "index.html"
	}
	return filename
}

// getFileSize returns the size of a file in bytes.
func getFileSize(filename string) uint64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return uint64(info.Size())
}

// formatSize formats bytes into human-readable format.
func formatSize(bytes uint64) string {
	if bytes == 0 {
		return "?"
	}
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1fGB", float64(bytes)/(1024*1024*1024))
}

// compactURL shortens a URL for display.
func compactURL(url string) string {
	// Remove protocol
	withoutProto := url
	if strings.HasPrefix(url, "https://") {
		withoutProto = url[8:]
	} else if strings.HasPrefix(url, "http://") {
		withoutProto = url[7:]
	}

	// Truncate if too long
	if len(withoutProto) <= 50 {
		return withoutProto
	}
	return withoutProto[:25] + "..." + withoutProto[len(withoutProto)-20:]
}

// parseWgetError extracts a meaningful error message from wget output.
func parseWgetError(stderr string) string {
	combined := stderr

	if strings.Contains(combined, "404") {
		return "404 Not Found"
	}
	if strings.Contains(combined, "403") {
		return "403 Forbidden"
	}
	if strings.Contains(combined, "401") {
		return "401 Unauthorized"
	}
	if strings.Contains(combined, "500") {
		return "500 Server Error"
	}
	if strings.Contains(combined, "Connection refused") {
		return "Connection refused"
	}
	if strings.Contains(combined, "unable to resolve") || strings.Contains(combined, "Name or service not known") {
		return "DNS lookup failed"
	}
	if strings.Contains(combined, "timed out") {
		return "Connection timed out"
	}
	if strings.Contains(combined, "SSL") || strings.Contains(combined, "certificate") {
		return "SSL/TLS error"
	}

	// Return first meaningful line
	for _, line := range strings.Split(stderr, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			if len(trimmed) > 60 {
				return trimmed[:57] + "..."
			}
			return trimmed
		}
	}

	return "Unknown error"
}
