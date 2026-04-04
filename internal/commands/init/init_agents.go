package initpkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/GrayCodeAI/tokman/internal/agents"
	"github.com/GrayCodeAI/tokman/internal/config"
)

// runAllAgentsInit detects and sets up all installed agents
func runAllAgentsInit(global bool, patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("\n%s\n\n", green("Detecting installed agents..."))

	statuses := agents.DetectAll()
	setupCount := 0

	for _, status := range statuses {
		if !status.Installed {
			continue
		}

		fmt.Printf("  %s %s is installed", green("✓"), cyan(status.Name))
		if status.Version != "" {
			fmt.Printf(" (v%s)", status.Version)
		}
		fmt.Println()

		switch status.Name {
		case "claude-code":
			if global {
				if err := runGlobalInit(patchMode); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}
		case "cursor":
			installCursorHook()
		case "gemini-cli":
			runGeminiInit(global, patchMode)
		case "codex-cli":
			runCodexInit(global)
		case "adal":
			runAdaLInit(global, patchMode)
		case "kiro":
			runKiroInit(global, patchMode)
		case "kilo-code":
			runKiloInit(global, patchMode)
		case "windsurf":
			installWindsurfRules()
		case "cline":
			installClineRules()
		case "opencode":
			installOpencodePlugin()
		case "aider":
			runAiderInit(global)
		case "continue":
			runContinueInit(global)
		case "replit-agent":
			runReplitInit(global)
		default:
			fmt.Printf("    %s No automatic setup available\n", yellow("!"))
			continue
		}

		setupCount++
		fmt.Printf("    %s Configured\n", green("✓"))
	}

	fmt.Printf("\n%s\n", green("Setup complete!"))
	fmt.Printf("Configured %d agent(s).\n\n", setupCount)
}

// === Kiro Integration ===

const tokmanSlimKiro = `<!-- tokman-kiro-instructions v1 -->
# TokMan (Token Manager) - Token-Optimized Commands for Kiro

## Golden Rule

**Always prefix commands with ` + "`tokman`" + `** for 60-90% token savings.

## Key Commands

` + "```bash" + `
tokman git status / log / diff / show   # Git (59-80% savings)
tokman cargo build / test / clippy      # Rust (80-90%)
tokman tsc / lint / prettier            # JS/TS (70-87%)
tokman vitest / playwright              # Tests (90-99%)
tokman gain                             # View savings stats
` + "```" + `

<!-- /tokman-kiro-instructions -->
`

func runKiroInit(global bool, patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	kiroRcPath := filepath.Join(homeDir, ".kilorc")
	config := `# Kiro Configuration with TokMan Integration

hooks:
  preToolUse:
    - matcher: "Bash"
      command: "tokman rewrite"

optimization:
  cacheEnabled: true
  tokenBudget: 4000
`

	if err := writeIfChanged(kiroRcPath, config, ".kilorc"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kiro config: %v\n", err)
		return
	}

	tokmanMdPath := filepath.Join(homeDir, ".kiro", "TOKMAN.md")
	if err := os.MkdirAll(filepath.Dir(tokmanMdPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}
	writeIfChanged(tokmanMdPath, tokmanSlimKiro, "TOKMAN.md")

	fmt.Printf("\n%s\n\n", green("Kiro integration configured."))
	fmt.Printf("  Config:    %s\n", cyan(kiroRcPath))
	fmt.Printf("  TOKMAN.md: %s\n", cyan(tokmanMdPath))
	fmt.Println()
}

func uninstallKiro() []string {
	var removed []string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	kiroRcPath := filepath.Join(homeDir, ".kilorc")
	if data, err := os.ReadFile(kiroRcPath); err == nil {
		if strings.Contains(string(data), "tokman") {
			newContent := removeKiroTokmanSection(string(data))
			if err := os.WriteFile(kiroRcPath, []byte(newContent), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", kiroRcPath, err)
			}
			removed = append(removed, "Kiro config: removed TokMan hooks")
		}
	}

	tokmanMdPath := filepath.Join(homeDir, ".kiro", "TOKMAN.md")
	if _, err := os.Stat(tokmanMdPath); err == nil {
		if err := os.Remove(tokmanMdPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", tokmanMdPath, err)
		}
		removed = append(removed, fmt.Sprintf("TOKMAN.md: %s", tokmanMdPath))
	}

	return removed
}

func removeKiroTokmanSection(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inHooks := false
	skipNext := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if skipNext {
			skipNext = false
			continue
		}

		if strings.HasPrefix(trimmed, "hooks:") {
			inHooks = true
		}

		if inHooks && strings.Contains(trimmed, "tokman") {
			continue
		}
		if inHooks && strings.Contains(trimmed, "command: tokman rewrite") {
			continue
		}

		if inHooks && strings.HasPrefix(trimmed, "optimization:") {
			inHooks = false
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// === Kilo Code Integration ===

func runKiloInit(global bool, patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	kiloRcPath := filepath.Join(homeDir, ".kilorc")
	config := `# Kilo Code Configuration with TokMan Integration

hooks:
  preToolUse:
    - matcher: "Bash"
      command: "tokman rewrite"

optimization:
  cacheEnabled: true
  tokenBudget: 4000
`

	if err := writeIfChanged(kiloRcPath, config, ".kilorc"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kilo Code config: %v\n", err)
		return
	}

	fmt.Printf("\n%s\n\n", green("Kilo Code integration configured."))
	fmt.Printf("  Config: %s\n", cyan(kiloRcPath))
	fmt.Println()
}

func uninstallKilo() []string {
	return nil
}

// === Replit Agent Integration ===

func runReplitInit(global bool) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	replitPath := ".replit"
	config := `# Replit Configuration with TokMan Integration

[agent]
token_optimization = true

[env]
TOKMAN_ENABLED = "true"

[[hooks]]
event = "preToolUse"
matcher = "Bash"
command = "tokman rewrite"
`

	if err := writeIfChanged(replitPath, config, ".replit"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Replit config: %v\n", err)
		return
	}

	fmt.Printf("\n%s\n\n", green("Replit Agent integration configured."))
	fmt.Printf("  Config: %s\n", cyan(replitPath))
	fmt.Println()
}

func uninstallReplit() []string {
	var removed []string

	if data, err := os.ReadFile(".replit"); err == nil {
		if strings.Contains(string(data), "TOKMAN_ENABLED") {
			newContent := removeReplitTokmanSection(string(data))
			if err := os.WriteFile(".replit", []byte(newContent), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write .replit: %v\n", err)
			}
			removed = append(removed, ".replit: removed TokMan configuration")
		}
	}

	return removed
}

func removeReplitTokmanSection(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	skipSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "TOKMAN_ENABLED") {
			continue
		}
		if strings.Contains(trimmed, "token_optimization") {
			continue
		}
		if strings.HasPrefix(trimmed, "[agent]") {
			skipSection = true
			result = append(result, line)
			continue
		}
		if strings.HasPrefix(trimmed, "[[hooks]]") && skipSection {
			skipSection = false
			continue
		}
		if strings.Contains(trimmed, "tokman rewrite") {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// === Aider Native Integration ===

const aiderConfig = `# Aider Configuration with TokMan Integration

# Enable prompt caching
cache-prompts: true

# Map tokens for context
map-tokens: 2048

# Auto-commits
auto-commits: true

# Shell command mapping (wraps commands with tokman)
# Note: Aider doesn't have native command hooks, so use shell aliases:
# alias git='tokman git'
# alias ls='tokman ls'
# alias cat='tokman read'
`

func runAiderInit(global bool) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configPath := filepath.Join(homeDir, ".aider.conf.yml")

	if err := writeIfChanged(configPath, aiderConfig, ".aider.conf.yml"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Aider config: %v\n", err)
		return
	}

	aliasesPath := filepath.Join(config.DataPath(), "aider-aliases.sh")
	aliases := `# Aider TokMan Aliases
# Add to your .bashrc or .zshrc:
alias git='tokman git'
alias ls='tokman ls'
alias cat='tokman read'
alias grep='tokman grep'
alias find='tokman find'
alias tree='tokman tree'
`
	if err := os.MkdirAll(filepath.Dir(aliasesPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}
	if err := os.WriteFile(aliasesPath, []byte(aliases), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", aliasesPath, err)
	}

	fmt.Printf("\n%s\n\n", green("Aider integration configured."))
	fmt.Printf("  Config:  %s\n", cyan(configPath))
	fmt.Printf("  Aliases: %s\n", cyan(aliasesPath))
	fmt.Println("\n  Add these aliases to your shell for automatic tokman wrapping:")
	fmt.Printf("  %s\n", cyan("source "+aliasesPath))
	fmt.Println()
}

func uninstallAider() []string {
	var removed []string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	configPath := filepath.Join(homeDir, ".aider.conf.yml")
	if data, err := os.ReadFile(configPath); err == nil {
		if strings.Contains(string(data), "TokMan") {
			if err := os.Remove(configPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", configPath, err)
			}
			removed = append(removed, fmt.Sprintf("Aider config: %s", configPath))
		}
	}

	aliasesPath := filepath.Join(config.DataPath(), "aider-aliases.sh")
	if _, err := os.Stat(aliasesPath); err == nil {
		if err := os.Remove(aliasesPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", aliasesPath, err)
		}
		removed = append(removed, "Aider aliases file")
	}

	return removed
}

// === Continue Integration ===

const continueConfig = `{
  "models": [
    {
      "title": "TokMan Optimized",
      "provider": "anthropic",
      "cache": true
    }
  ],
  "experimental": {
    "tokenOptimization": {
      "enabled": true,
      "wrapper": "tokman"
    }
  },
  "contextProviders": [
    {
      "name": "code",
      "params": {
        "cacheResults": true
      }
    }
  ]
}
`

func runContinueInit(global bool) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".continue")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := writeIfChanged(configPath, continueConfig, "config.json"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Continue config: %v\n", err)
		return
	}

	fmt.Printf("\n%s\n\n", green("Continue integration configured."))
	fmt.Printf("  Config: %s\n", cyan(configPath))
	fmt.Println()
}

func uninstallContinue() []string {
	var removed []string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	configPath := filepath.Join(homeDir, ".continue", "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		if strings.Contains(string(data), "TokMan") {
			if err := os.Remove(configPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", configPath, err)
			}
			removed = append(removed, fmt.Sprintf("Continue config: %s", configPath))
		}
	}

	return removed
}
