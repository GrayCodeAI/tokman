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

var ansibleCmd = &cobra.Command{
	Use:   "ansible-playbook [playbook] [args...]",
	Short: "Ansible playbook commands with compact output",
	Long: `Execute Ansible playbook commands with token-optimized output.

Specialized filters for:
  - play recap: Compact task summary
  - verbose mode: Filter verbose output
  - facts: Compact gathered facts

Examples:
  tokman ansible-playbook site.yml
  tokman ansible-playbook deploy.yml -v
  tokman ansible-playbook --check site.yml`,
	DisableFlagParsing: true,
	RunE:               runAnsible,
}

func init() {
	registry.Add(func() { registry.Register(ansibleCmd) })
}

func runAnsible(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"--help"}
	}

	return runAnsiblePlaybook(args)
}

func runAnsiblePlaybook(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: ansible-playbook %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("ansible-playbook", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterAnsibleOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "ansible_playbook", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("ansible-playbook", "tokman ansible-playbook", originalTokens, filteredTokens)

	return err
}

func filterAnsibleOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var inPlay bool
	var playName string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Detect play start
		if strings.Contains(line, "PLAY [") {
			if shared.UltraCompact {
				playName = extractBracketContent(line)
				result = append(result, fmt.Sprintf("▶ %s", playName))
			} else {
				result = append(result, line)
			}
			inPlay = true
			continue
		}

		// Detect task start
		if strings.Contains(line, "TASK [") {
			if shared.UltraCompact {
				taskName := extractBracketContent(line)
				result = append(result, fmt.Sprintf("  ⚙ %s", taskName))
			} else {
				result = append(result, shared.TruncateLine(line, 100))
			}
			continue
		}

		// Handle task results
		if inPlay && (strings.Contains(trimmed, "ok:") || strings.Contains(trimmed, "changed:") ||
			strings.Contains(trimmed, "failed:") || strings.Contains(trimmed, "skipping:")) {

			if shared.UltraCompact {
				// Just show status icon
				if strings.Contains(trimmed, "failed:") {
					result = append(result, "    ❌")
				} else if strings.Contains(trimmed, "changed:") {
					result = append(result, "    ✏️")
				} else if strings.Contains(trimmed, "skipping:") {
					result = append(result, "    ⏭️")
				} else {
					result = append(result, "    ✓")
				}
			} else {
				// Show truncated result
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					host := strings.TrimSpace(parts[1])
					host = shared.TruncateLine(host, 60)
					result = append(result, fmt.Sprintf("    %s: %s", parts[0], host))
				} else {
					result = append(result, line)
				}
			}
			continue
		}

		// Handle play recap
		if strings.Contains(line, "PLAY RECAP") {
			result = append(result, "")
			result = append(result, line)
			continue
		}

		// Process recap lines
		if strings.Contains(trimmed, ": ok=") || strings.Contains(trimmed, "changed=") {
			if shared.UltraCompact {
				// Ultra compact recap
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					host := strings.TrimSpace(parts[0])
					stats := strings.TrimSpace(parts[1])
					// Extract just key numbers
					result = append(result, fmt.Sprintf("  %s: %s", host, stats))
				}
			} else {
				result = append(result, line)
			}
			continue
		}

		// Skip verbose output in ultra-compact mode unless it's important
		if shared.UltraCompact {
			// Skip debug output
			if strings.Contains(line, "DEBUG") || strings.Contains(line, "debug:") {
				continue
			}
			// Skip fact gathering details
			if strings.Contains(line, "Gathering Facts") {
				result = append(result, "  📋 Gathering Facts...")
				continue
			}
			// Skip deprecation warnings unless verbose
			if strings.Contains(line, "DEPRECATION WARNING") && shared.Verbose == 0 {
				continue
			}
		}

		// Keep warnings and errors
		if strings.Contains(line, "WARN") || strings.Contains(line, "ERROR") ||
			strings.Contains(line, "fatal:") {
			result = append(result, line)
			continue
		}
	}

	return strings.Join(result, "\n")
}

func extractBracketContent(line string) string {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start != -1 && end != -1 && end > start {
		return line[start+1 : end]
	}
	return line
}
