package system

import (
	"os"
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

var lsCmd = &cobra.Command{
	Use:   "ls [path] [args...]",
	Short: "List directory contents (filtered)",
	Long: `List directory with noise filtering:
- Hides .git, node_modules, target, etc.
- Groups directories and files
- Shows human-readable sizes`,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		if err := shared.ExecuteAndRecord("ls", func() (string, string, error) {
			return runLS(args)
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	registry.Add(func() { registry.Register(lsCmd) })
}

// loadNoiseDirs returns a set of noise directory names from config or defaults.
func loadNoiseDirs() map[string]bool {
	cfg, err := config.Load(shared.CfgFile)
	noiseDirs := make(map[string]bool)

	// Use loaded config noise dirs, or fall back to defaults
	var dirs []string
	if err == nil && len(cfg.Filter.NoiseDirs) > 0 {
		dirs = cfg.Filter.NoiseDirs
	} else {
		// Fall back to default config values (centralized in config.go)
		dirs = config.Defaults().Filter.NoiseDirs
	}

	for _, dir := range dirs {
		noiseDirs[dir] = true
	}
	return noiseDirs
}

// runLS executes ls with noise filtering, returns (raw, filtered, error)
func runLS(args []string) (string, string, error) {
	path := "."
	lsArgs := []string{}

	// Parse args: separate flags from path
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			lsArgs = append(lsArgs, arg)
		} else {
			path = arg
		}
	}

	// Default to -la if no flags provided
	if len(lsArgs) == 0 {
		lsArgs = []string{"-la"}
	}

	lsArgs = append(lsArgs, path)

	cmd := exec.Command("ls", lsArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("ls failed: %w", err)
	}

	raw := out.String()

	// Use ultra-compact format by default
	if shared.UltraCompact {
		return raw, filterLSOutputUltraCompact(raw), nil
	}
	return raw, filterLSOutput(raw), nil
}

// filterLSOutputUltraCompact returns compact output
func filterLSOutputUltraCompact(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return output
	}

	// Load noise dirs from config
	noiseDirs := loadNoiseDirs()

	var dirs []string
	var files []struct{ name, size string }
	extCount := make(map[string]int)

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
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		perms := fields[0]
		sizeStr := fields[4]
		name := strings.Join(fields[8:], " ")

		// Skip . and ..
		if name == "." || name == ".." {
			continue
		}

		// Check if noise directory
		if noiseDirs[name] {
			continue
		}

		// Parse size
		size := parseSize(sizeStr)
		sizeFormatted := utils.FormatBytes(size)

		// Check if directory
		isDir := strings.HasPrefix(perms, "d")

		if isDir {
			dirs = append(dirs, fmt.Sprintf("%s/", name))
		} else {
			files = append(files, struct{ name, size string }{name, sizeFormatted})
			// Track extension
			ext := "no ext"
			if dotIdx := strings.LastIndex(name, "."); dotIdx >= 0 {
				ext = name[dotIdx:]
			}
			extCount[ext]++
		}
	}

	// Sort alphabetically
	sort.Strings(dirs)
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	// Build output: dirs first, then files with sizes
	var result []string
	result = append(result, dirs...)
	for _, f := range files {
		result = append(result, fmt.Sprintf("%s  %s", f.name, f.size))
	}

	// Summary line with extension counts
	if len(files) > 0 || len(dirs) > 0 {
		summary := fmt.Sprintf("\n%d files, %d dirs", len(files), len(dirs))
		if len(extCount) > 0 {
			// Sort extensions by count
			type extPair struct {
				ext   string
				count int
			}
			var extPairs []extPair
			for e, c := range extCount {
				extPairs = append(extPairs, extPair{e, c})
			}
			sort.Slice(extPairs, func(i, j int) bool { return extPairs[i].count > extPairs[j].count })

			var parts []string
			limit := 5
			if len(extPairs) < limit {
				limit = len(extPairs)
			}
			for _, ep := range extPairs[:limit] {
				parts = append(parts, fmt.Sprintf("%d %s", ep.count, ep.ext))
			}
			summary += " (" + strings.Join(parts, ", ")
			if len(extPairs) > 5 {
				summary += fmt.Sprintf(", +%d more", len(extPairs)-5)
			}
			summary += ")"
		}
		result = append(result, summary)
	}

	return strings.Join(result, "\n")
}

// filterLSOutput filters and formats ls output
func filterLSOutput(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return output
	}

	// Load noise dirs from config
	noiseDirs := loadNoiseDirs()

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
			sizeStr := utils.FormatBytes(size)
			dirs = append(dirs, fmt.Sprintf("📁 %s (%s)", name, sizeStr))
		} else {
			totalFileSize += size
			sizeStr := utils.FormatBytes(size)
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
	if _, err := fmt.Sscanf(s, "%d", &size); err != nil {
		return 0
	}
	return size
}
