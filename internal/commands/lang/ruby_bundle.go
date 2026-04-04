package lang

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func runBundleCmd(args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle %s\n", strings.Join(args, " "))
	}

	// Route to specialized handlers
	switch args[0] {
	case "install":
		return runBundleInstallCmd(args[1:])
	case "update":
		return runBundleUpdateCmd(args[1:])
	case "outdated":
		return runBundleOutdatedCmd(args[1:])
	default:
		return runBundlePassthrough(args)
	}
}

func runBundleInstallCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle install %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", append([]string{"install"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleInstallOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "bundle_install", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("bundle install", "tokman ruby bundle install", originalTokens, filteredTokens)

	return err
}

func filterBundleInstallOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var installed, updated int
	gems := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Track gem installations
		if strings.Contains(line, "Installing") {
			installed++
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				gems = append(gems, parts[1])
			}
		} else if strings.Contains(line, "Using") {
			updated++
		} else if strings.Contains(line, "Bundle complete!") {
			result = append(result, "✅ "+line)
		} else if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		var parts []string
		if installed > 0 {
			parts = append(parts, fmt.Sprintf("I:%d", installed))
		}
		if updated > 0 {
			parts = append(parts, fmt.Sprintf("U:%d", updated))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
		return "✅ Done"
	}

	// Normal output
	if len(result) == 0 {
		if installed > 0 || updated > 0 {
			result = append(result, "📋 Bundle Install Summary:")
			if installed > 0 {
				result = append(result, fmt.Sprintf("   📦 %d gems installed", installed))
				if len(gems) > 0 && len(gems) <= 5 {
					result = append(result, fmt.Sprintf("      %s", strings.Join(gems, ", ")))
				} else if len(gems) > 5 {
					result = append(result, fmt.Sprintf("      %s ... +%d more",
						strings.Join(gems[:5], ", "), len(gems)-5))
				}
			}
			if updated > 0 {
				result = append(result, fmt.Sprintf("   ✓ %d gems unchanged", updated))
			}
		} else {
			result = append(result, "✅ Bundle already up to date")
		}
	}

	return strings.Join(result, "\n")
}

func runBundleUpdateCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle update %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", append([]string{"update"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleUpdateOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("bundle update", "tokman ruby bundle update", originalTokens, filteredTokens)

	return err
}

func filterBundleUpdateOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var updated int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Installing") || strings.Contains(line, "Updating") {
			updated++
		} else if strings.Contains(line, "Bundle updated!") {
			result = append(result, "✅ "+line)
		} else if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		if updated > 0 {
			return fmt.Sprintf("✅ Updated %d gems", updated)
		}
		return "✅ Bundle update complete"
	}

	return strings.Join(result, "\n")
}

func runBundleOutdatedCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle outdated %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", append([]string{"outdated"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleOutdatedOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("bundle outdated", "tokman ruby bundle outdated", originalTokens, filteredTokens)

	return err
}

func filterBundleOutdatedOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var outdated []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Gem") && strings.Contains(line, "Current") && strings.Contains(line, "Latest") {
			// Header line - skip
			continue
		} else if strings.Contains(line, "(") && strings.Contains(line, ">") {
			// Outdated gem line like "rails (6.1.0 > 7.0.0)"
			outdated = append(outdated, shared.TruncateLine(line, 60))
		} else if strings.Contains(line, "Bundle up to date") {
			return "✅ All gems are up to date"
		}
	}

	if len(outdated) == 0 {
		return "✅ All gems are up to date"
	}

	result = append(result, fmt.Sprintf("📋 %d outdated gems:", len(outdated)))
	for i, gem := range outdated {
		if i >= 15 {
			result = append(result, fmt.Sprintf("   ... +%d more", len(outdated)-15))
			break
		}
		result = append(result, fmt.Sprintf("   • %s", gem))
	}

	return strings.Join(result, "\n")
}

func runBundlePassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("bundle %s", args[0]), "tokman ruby bundle", originalTokens, filteredTokens)

	return err
}

func filterBundleOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, shared.TruncateLine(line, 120))
		}
	}

	if len(result) > 30 {
		return strings.Join(result[:30], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-30)
	}
	return strings.Join(result, "\n")
}
