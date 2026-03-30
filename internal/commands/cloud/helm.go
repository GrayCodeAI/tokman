package cloud

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var helmCmd = &cobra.Command{
	Use:   "helm [command] [args...]",
	Short: "Helm commands with compact output",
	Long: `Execute Helm commands with token-optimized output.

Specialized filters for:
  - list: Compact releases table
  - status: Compact release status
  - history: Compact revision history
  - upgrade: Compact upgrade output

Examples:
  tokman helm list
  tokman helm status my-release
  tokman helm upgrade my-release ./chart`,
	DisableFlagParsing: true,
	RunE:               runHelm,
}

func init() {
	registry.Add(func() { registry.Register(helmCmd) })
}

func runHelm(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	switch args[0] {
	case "list", "ls":
		return runHelmList(args[1:])
	case "status":
		return runHelmStatus(args[1:])
	case "history":
		return runHelmHistory(args[1:])
	case "upgrade":
		return runHelmUpgrade(args[1:])
	case "install":
		return runHelmInstall(args[1:])
	case "rollback":
		return runHelmRollback(args[1:])
	case "uninstall", "delete":
		return runHelmUninstall(args[1:])
	case "repo":
		return runHelmRepo(args[1:])
	default:
		return runHelmPassthrough(args)
	}
}

func runHelmList(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm list %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"list"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmListOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("helm list", "tokman helm list", originalTokens, filteredTokens)

	return err
}

func filterHelmListOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	releaseCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Header line
		if strings.HasPrefix(line, "NAME") {
			if !shared.UltraCompact {
				result = append(result, "📋 Helm Releases:")
			}
			continue
		}

		// Release line
		releaseCount++
		if shared.UltraCompact {
			// Ultra-compact: just name and status
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				result = append(result, fmt.Sprintf("%s (%s)", fields[0], fields[3]))
			}
		} else if releaseCount <= 20 {
			// Normal: show full line but truncated
			result = append(result, "  "+shared.TruncateLine(line, 100))
		}
	}

	if releaseCount > 20 {
		result = append(result, fmt.Sprintf("  ... +%d more releases", releaseCount-20))
	}

	if len(result) == 0 {
		return "No releases found"
	}

	return strings.Join(result, "\n")
}

func runHelmStatus(args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		fmt.Println("Usage: helm status <release>")
		return nil
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm status %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"status"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmStatusOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("helm status %s", args[0]), "tokman helm status", originalTokens, filteredTokens)

	return err
}

func filterHelmStatusOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var resources []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Release info
		if strings.HasPrefix(line, "NAME:") {
			result = append(result, "📋 "+line)
		} else if strings.HasPrefix(line, "STATUS:") {
			status := strings.TrimPrefix(line, "STATUS: ")
			status = strings.TrimSpace(status)
			if status == "deployed" {
				result = append(result, "✅ STATUS: deployed")
			} else if status == "failed" {
				result = append(result, "❌ STATUS: failed")
			} else {
				result = append(result, "  STATUS: "+status)
			}
		} else if strings.HasPrefix(line, "REVISION:") {
			result = append(result, "  "+line)
		} else if strings.HasPrefix(line, "LAST DEPLOYED:") {
			result = append(result, "  "+line)
		} else if strings.Contains(line, "NAMESPACE") && strings.Contains(line, "NAME") {
			// Resource header - skip
		} else if strings.HasPrefix(line, "NOTES:") {
			result = append(result, "")
			result = append(result, "NOTES:")
		} else if strings.Contains(line, "/") && len(strings.Fields(line)) >= 2 {
			// Resource line like "Deployment/apps/my-app"
			resources = append(resources, shared.TruncateLine(line, 80))
		}
	}

	// Add resources summary
	if len(resources) > 0 {
		result = append(result, "")
		result = append(result, "Resources:")
		for i, r := range resources {
			if i >= 10 {
				result = append(result, fmt.Sprintf("  ... +%d more", len(resources)-10))
				break
			}
			result = append(result, "  "+r)
		}
	}

	if len(result) == 0 {
		return raw
	}

	return strings.Join(result, "\n")
}

func runHelmHistory(args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		fmt.Println("Usage: helm history <release>")
		return nil
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm history %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"history"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmHistoryOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("helm history %s", args[0]), "tokman helm history", originalTokens, filteredTokens)

	return err
}

func filterHelmHistoryOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Header line
		if strings.HasPrefix(line, "REVISION") {
			if !shared.UltraCompact {
				result = append(result, "📋 Revision History:")
			}
			continue
		}

		// Revision line
		if shared.UltraCompact {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				result = append(result, fmt.Sprintf("#%s %s", fields[0], fields[2]))
			}
		} else {
			result = append(result, "  "+shared.TruncateLine(line, 80))
		}
	}

	if len(result) == 0 {
		return raw
	}

	return strings.Join(result, "\n")
}

func runHelmUpgrade(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm upgrade %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"upgrade"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmUpgradeOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "helm_upgrade", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("helm upgrade", "tokman helm upgrade", originalTokens, filteredTokens)

	return err
}

func filterHelmUpgradeOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Release") && strings.Contains(line, "has been upgraded") {
			result = append(result, "✅ "+line)
		} else if strings.Contains(line, "NAME:") {
			result = append(result, "📋 "+line)
		} else if strings.Contains(line, "LAST DEPLOYED:") {
			result = append(result, "  "+line)
		} else if strings.Contains(line, "Error:") || strings.Contains(line, "ERROR:") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	if shared.UltraCompact && len(result) > 0 {
		for _, line := range result {
			if strings.Contains(line, "upgraded") {
				return "✅ Upgraded"
			}
		}
	}

	if len(result) == 0 {
		return "✅ Upgrade complete"
	}

	return strings.Join(result, "\n")
}

func runHelmInstall(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm install %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"install"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmInstallOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "helm_install", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("helm install", "tokman helm install", originalTokens, filteredTokens)

	return err
}

func filterHelmInstallOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "NAME:") && !strings.Contains(line, "NAMESPACE") {
			result = append(result, "✅ Release installed: "+strings.TrimPrefix(line, "NAME: "))
		} else if strings.Contains(line, "LAST DEPLOYED:") {
			result = append(result, "  "+line)
		} else if strings.HasPrefix(line, "NOTES:") {
			result = append(result, "")
			result = append(result, "NOTES:")
		} else if strings.Contains(line, "Error:") || strings.Contains(line, "ERROR:") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	if shared.UltraCompact && len(result) > 0 {
		return "✅ Installed"
	}

	if len(result) == 0 {
		return "✅ Install complete"
	}

	return strings.Join(result, "\n")
}

func runHelmRollback(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm rollback %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"rollback"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmRollbackOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("helm rollback", "tokman helm rollback", originalTokens, filteredTokens)

	return err
}

func filterHelmRollbackOutput(raw string) string {
	if strings.Contains(raw, "Rollback was a success") {
		return "✅ Rollback successful"
	}

	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		return "✅ Rollback complete"
	}

	return strings.Join(result, "\n")
}

func runHelmUninstall(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm uninstall %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"uninstall"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmUninstallOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("helm uninstall", "tokman helm uninstall", originalTokens, filteredTokens)

	return err
}

func filterHelmUninstallOutput(raw string) string {
	if strings.Contains(raw, "release") && strings.Contains(raw, "uninstalled") {
		return "✅ Release uninstalled"
	}

	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		return "✅ Uninstall complete"
	}

	return strings.Join(result, "\n")
}

func runHelmRepo(args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	switch args[0] {
	case "update":
		return runHelmRepoUpdate(args[1:])
	case "list":
		return runHelmRepoList(args[1:])
	default:
		return runHelmPassthrough(append([]string{"repo"}, args...))
	}
}

func runHelmRepoUpdate(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm repo update %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"repo", "update"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmRepoUpdateOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("helm repo update", "tokman helm repo update", originalTokens, filteredTokens)

	return err
}

func filterHelmRepoUpdateOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var updated int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "...Successfully") {
			updated++
			if shared.UltraCompact {
				fields := strings.Fields(line)
				if len(fields) >= 1 {
					result = append(result, "✓ "+fields[0])
				}
			} else {
				result = append(result, "  ✓ "+shared.TruncateLine(line, 60))
			}
		} else if strings.Contains(line, "...Unable") {
			result = append(result, "  ✗ "+shared.TruncateLine(line, 60))
		}
	}

	if shared.UltraCompact {
		return fmt.Sprintf("Updated: %d repos", updated)
	}

	if len(result) == 0 {
		return "✅ Repositories updated"
	}

	return "📋 Repository Update:\n" + strings.Join(result, "\n")
}

func runHelmRepoList(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm repo list %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", append([]string{"repo", "list"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmRepoListOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("helm repo list", "tokman helm repo list", originalTokens, filteredTokens)

	return err
}

func filterHelmRepoListOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "NAME") {
			if !shared.UltraCompact {
				result = append(result, "📋 Repositories:")
			}
			continue
		}

		if shared.UltraCompact {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				result = append(result, fields[0])
			}
		} else {
			result = append(result, "  "+shared.TruncateLine(line, 80))
		}
	}

	if len(result) == 0 {
		return "No repositories configured"
	}

	return strings.Join(result, "\n")
}

func runHelmPassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: helm %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("helm", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterHelmOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("helm %s", args[0]), "tokman helm", originalTokens, filteredTokens)

	return err
}

func filterHelmOutput(raw string) string {
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
