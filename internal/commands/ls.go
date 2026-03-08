package commands

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Patel230/tokman/internal/config"
)

// Default noise directories to filter out (from RTK)
var defaultNoiseDirs = []string{
	".git",
	"node_modules",
	"target",
	"__pycache__",
	".venv",
	"vendor",
	".idea",
	".vscode",
	"dist",
	"build",
	".next",
	"coverage",
	".cache",
}

var lsCmd = &cobra.Command{
	Use:   "ls [path]",
	Short: "List directory contents (filtered)",
	Long: `List directory with noise filtering:
- Hides .git, node_modules, target, etc.
- Groups directories and files
- Shows human-readable sizes`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runLS(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		fmt.Print(output)

		// Record to tracker
		if err := recordCommand("ls", output, output, execTime, true); err != nil && verbose {
			fmt.Fprintf(cmd.OutOrStderr(), "Warning: failed to record: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

// runLS executes ls with noise filtering
func runLS(args []string) (string, error) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Get ls -la output
	cmd := exec.Command("ls", "-la", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ls failed: %w", err)
	}

	return filterLSOutput(out.String()), nil
}

// filterLSOutput filters and formats ls output
func filterLSOutput(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return output
	}

	// Load noise dirs from config
	cfg, err := config.Load(cfgFile)
	noiseDirs := make(map[string]bool)
	if err == nil && len(cfg.Filter.NoiseDirs) > 0 {
		for _, dir := range cfg.Filter.NoiseDirs {
			noiseDirs[dir] = true
		}
	} else {
		// Use defaults
		for _, dir := range defaultNoiseDirs {
			noiseDirs[dir] = true
		}
	}

	var dirs []string
	var files []string
	var totalDirSize int64
	var totalFileSize int64
	var noiseCount int

	// Skip the "total X" line at the start
	startIdx := 0
	if len(lines) > 0 && strings.HasPrefix(lines[0], "total ") {
		startIdx = 1
	}

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		if len(line) == 0 {
			continue
		}

		// Parse ls -la output
		// Format: permissions links owner group size month day time name
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		perms := fields[0]
		sizeStr := fields[4]
		name := strings.Join(fields[8:], " ") // Handle names with spaces

		// Skip . and ..
		if name == "." || name == ".." {
			continue
		}

		// Check if noise directory
		if noiseDirs[name] {
			noiseCount++
			continue
		}

		// Parse size
		size := parseSize(sizeStr)

		// Check if directory
		isDir := strings.HasPrefix(perms, "d")

		if isDir {
			totalDirSize += size
			sizeStr := humanSize(size)
			dirs = append(dirs, fmt.Sprintf("📁 %s (%s)", name, sizeStr))
		} else {
			totalFileSize += size
			sizeStr := humanSize(size)
			files = append(files, fmt.Sprintf("📄 %s (%s)", name, sizeStr))
		}
	}

	// Sort alphabetically
	sort.Strings(dirs)
	sort.Strings(files)

	// Build output
	var result strings.Builder

	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	dim := color.New(color.FgHiBlack).SprintFunc()

	// Summary header
	result.WriteString(cyan("📂 Directory Listing\n"))
	result.WriteString(dim(strings.Repeat("─", 40)) + "\n")

	// Directories
	if len(dirs) > 0 {
		result.WriteString(green("\nDirectories:\n"))
		for _, d := range dirs {
			result.WriteString("  " + d + "\n")
		}
	}

	// Files
	if len(files) > 0 {
		result.WriteString(green("\nFiles:\n"))
		for _, f := range files {
			result.WriteString("  " + f + "\n")
		}
	}

	// Footer stats
	result.WriteString(dim("\n"+strings.Repeat("─", 40)) + "\n")
	result.WriteString(fmt.Sprintf("  %d dirs, %d files", len(dirs), len(files)))
	if noiseCount > 0 {
		result.WriteString(dim(fmt.Sprintf(" (%d noise dirs hidden)", noiseCount)))
	}
	result.WriteString("\n")

	return result.String()
}

// parseSize converts size string to int64
func parseSize(s string) int64 {
	var size int64
	fmt.Sscanf(s, "%d", &size)
	return size
}

// humanSize converts bytes to human readable format
func humanSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
