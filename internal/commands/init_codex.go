package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

const tokmanSlimCodex = `<!-- tokman-codex-instructions v1 -->
# TokMan (Token Manager) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with ` + "`tokman`" + `** for 60-90% token savings.

## Key Commands

` + "```bash" + `
tokman git status / log / diff / show   # Git (59-80% savings)
tokman cargo build / test / clippy      # Rust (80-90%)
tokman tsc / lint / prettier            # JS/TS (70-87%)
tokman vitest / playwright              # Tests (90-99%)
tokman docker / kubectl                 # Infra (85%)
tokman ls / grep / find / tree          # Files (60-75%)
tokman gain                             # View savings stats
tokman discover                         # Find missed savings
tokman proxy <cmd>                      # Run without filtering
` + "```" + `

<!-- /tokman-codex-instructions -->
`

// runCodexInit sets up Codex CLI integration
func runCodexInit(global bool) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	var agentsMdPath, tokmanMdPath string

	if global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			return
		}
		codexDir := filepath.Join(homeDir, ".codex")
		if err := os.MkdirAll(codexDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Codex config directory: %v\n", err)
			return
		}
		agentsMdPath = filepath.Join(codexDir, "AGENTS.md")
		tokmanMdPath = filepath.Join(codexDir, "TOKMAN.md")
	} else {
		agentsMdPath = "AGENTS.md"
		tokmanMdPath = "TOKMAN.md"
	}

	// Write TOKMAN.md with Codex-specific awareness
	if err := writeIfChanged(tokmanMdPath, tokmanSlimCodex, "TOKMAN.md"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating TOKMAN.md: %v\n", err)
		return
	}

	// Patch AGENTS.md to add @TOKMAN.md reference
	addedRef := patchAgentsMd(agentsMdPath)

	fmt.Printf("\n%s\n\n", green("TokMan configured for Codex CLI."))
	fmt.Printf("  TOKMAN.md: %s\n", cyan(tokmanMdPath))
	if addedRef {
		fmt.Printf("  AGENTS.md: %s\n", green("@TOKMAN.md reference added"))
	} else {
		fmt.Printf("  AGENTS.md: %s\n", green("@TOKMAN.md reference already present"))
	}

	if global {
		fmt.Printf("\n  Codex global instructions path: %s\n", cyan(agentsMdPath))
	} else {
		fmt.Printf("\n  Codex project instructions path: %s\n", cyan(agentsMdPath))
	}
	fmt.Println()
}

// patchAgentsMd adds @TOKMAN.md reference to AGENTS.md if not present
func patchAgentsMd(path string) bool {
	content := ""
	if data, err := os.ReadFile(path); err == nil {
		content = string(data)
	}

	if strings.Contains(content, "@TOKMAN.md") {
		return false
	}

	var newContent string
	if content == "" {
		newContent = "@TOKMAN.md\n"
	} else {
		newContent = fmt.Sprintf("%s\n\n@TOKMAN.md\n", strings.TrimSpace(content))
	}

	os.WriteFile(path, []byte(newContent), 0644)
	return true
}

// uninstallCodex removes Codex artifacts
func uninstallCodex() []string {
	var removed []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return removed
	}
	codexDir := filepath.Join(homeDir, ".codex")

	// Remove TOKMAN.md
	tokmanMdPath := filepath.Join(codexDir, "TOKMAN.md")
	if _, err := os.Stat(tokmanMdPath); err == nil {
		os.Remove(tokmanMdPath)
		removed = append(removed, fmt.Sprintf("TOKMAN.md: %s", tokmanMdPath))
	}

	// Remove @TOKMAN.md reference from AGENTS.md
	agentsMdPath := filepath.Join(codexDir, "AGENTS.md")
	if data, err := os.ReadFile(agentsMdPath); err == nil {
		content := string(data)
		if strings.Contains(content, "@TOKMAN.md") {
			newContent := removeTokmanMdReference(content)
			os.WriteFile(agentsMdPath, []byte(newContent), 0644)
			removed = append(removed, "Codex AGENTS.md: removed @TOKMAN.md reference")
		}
	}

	return removed
}
