package initpkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/integrity"
	"github.com/GrayCodeAI/tokman/internal/tracking"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

var (
	initGlobal    bool
	initAutoPatch bool
	initNoPatch   bool
	initUninstall bool
	initMode      string
	initAll       bool
	// Existing agent flags
	initCursor   bool
	initCopilot  bool
	initWindsurf bool
	initCline    bool
	initOpencode bool
	initGemini   bool
	initCodex    bool
	// New agent flags
	initAdaL     bool
	initKiro     bool
	initKilo     bool
	initReplit   bool
	initAider    bool
	initContinue bool
)

// Embedded hook script
const rewriteHook = `#!/usr/bin/env bash
# tokman-hook-version: 2
# TokMan Code hook — rewrites commands to use tokman for token savings.
# Requires: tokman >= 0.2.0, jq

if ! command -v jq &>/dev/null; then
  exit 0
fi

if ! command -v tokman &>/dev/null; then
  exit 0
fi

TOKMAN_VERSION=$(tokman --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
if [ -n "$TOKMAN_VERSION" ]; then
  MAJOR=$(echo "$TOKMAN_VERSION" | cut -d. -f1)
  MINOR=$(echo "$TOKMAN_VERSION" | cut -d. -f2)
  if [ "$MAJOR" -eq 0 ] && [ "$MINOR" -lt 2 ]; then
    echo "[tokman] WARNING: tokman $TOKMAN_VERSION is too old (need >= 0.2.0). Upgrade: go install github.com/GrayCodeAI/tokman@latest" >&2
    exit 0
  fi
fi

# Check if TokMan is enabled globally
if [ ! -f "$HOME/.local/share/tokman/.enabled" ]; then
  exit 0
fi

INPUT=$(cat)
CMD=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [ -z "$CMD" ]; then
  exit 0
fi

REWRITTEN=$(tokman rewrite "$CMD" 2>/dev/null) || exit 0

if [ "$CMD" = "$REWRITTEN" ]; then
  exit 0
fi

ORIGINAL_INPUT=$(echo "$INPUT" | jq -c '.tool_input')
UPDATED_INPUT=$(echo "$ORIGINAL_INPUT" | jq --arg cmd "$REWRITTEN" '.command = $cmd')

jq -n \
  --argjson updated "$UPDATED_INPUT" \
  '{
    "hookSpecificOutput": {
      "hookEventName": "PreToolUse",
      "permissionDecision": "allow",
      "permissionDecisionReason": "TokMan auto-rewrite",
      "updatedInput": $updated
    }
  }'
`

// getTokmanSlim returns the embedded TokMan awareness instructions
func getTokmanSlim() string {
	return `<!-- tokman-instructions v1 -->
# TokMan (Token Manager) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with ` + "`tokman`" + `**. If TokMan has a dedicated filter, it uses it. If not, it passes through unchanged.

**Important**: Even in command chains with ` + "`&&`" + `, use ` + "`tokman`" + `:
` + "```bash" + `
# Wrong
git add . && git commit -m "msg" && git push

# Correct
tokman git add . && tokman git commit -m "msg" && tokman git push
` + "```" + `

## TokMan Commands by Workflow

### Build & Compile (80-90% savings)
` + "```bash" + `
tokman cargo build         # Cargo build output
tokman cargo check         # Cargo check output
tokman cargo clippy        # Clippy warnings grouped by file
tokman tsc                 # TypeScript errors grouped by file/code
tokman lint                # ESLint/Biome violations grouped
tokman prettier --check    # Files needing format only
tokman next build          # Next.js build with route metrics
` + "```" + `

### Test (90-99% savings)
` + "```bash" + `
tokman cargo test          # Cargo test failures only
tokman vitest run          # Vitest failures only
tokman playwright test     # Playwright failures only
tokman test <cmd>          # Generic test wrapper - failures only
` + "```" + `

### Git (59-80% savings)
` + "```bash" + `
tokman git status          # Compact status
tokman git log             # Compact log
tokman git diff            # Compact diff
tokman git show            # Compact show
tokman git add             # Ultra-compact confirmations
tokman git commit          # Ultra-compact confirmations
tokman git push            # Ultra-compact confirmations
` + "```" + `

### GitHub (26-87% savings)
` + "```bash" + `
tokman gh pr view <num>    # Compact PR view
tokman gh pr checks        # Compact PR checks
tokman gh run list         # Compact workflow runs
tokman gh issue list       # Compact issue list
` + "```" + `

### Files & Search (60-75% savings)
` + "```bash" + `
tokman ls <path>           # Tree format, compact
tokman head <file>         # First N lines
tokman grep <pattern>      # Search grouped by file
` + "```" + `

### Meta Commands
` + "```bash" + `
tokman gain                # View token savings statistics
tokman gain --history      # View command history with savings
tokman discover            # Analyze sessions for missed usage
tokman proxy <cmd>         # Run command without filtering
` + "```" + `

## Token Savings Overview

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |

Overall average: **60-90% token reduction** on common development operations.
<!-- /tokman-instructions -->
`
}

// PatchMode controls settings.json patching behavior
type PatchMode int

const (
	PatchModeAsk  PatchMode = iota // Default: prompt user [y/N]
	PatchModeAuto                  // --auto-patch: no prompt
	PatchModeSkip                  // --no-patch: manual instructions
)

// PatchResult describes the outcome of settings.json patching
type PatchResult int

const (
	PatchResultPatched        PatchResult = iota // Hook was added successfully
	PatchResultAlreadyPresent                    // Hook was already in settings.json
	PatchResultDeclined                          // User declined when prompted
	PatchResultSkipped                           // --no-patch flag used
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TokMan",
	Long: `Initialize TokMan by creating the necessary directories,
configuration files, and database.

With --global (-g), also sets up Claude Code integration:
  - Installs hook to ~/.claude/hooks/tokman-rewrite.sh
  - Creates TOKMAN.md with usage instructions
  - Patches ~/.claude/settings.json with PreToolUse hook

Editor-specific flags install hooks for other editors:
  --cursor   Install Cursor Agent hook
  --copilot  Install GitHub Copilot hook + instructions
  --windsurf Install Windsurf rules
  --cline    Install Cline rules
  --opencode Install OpenCode plugin
  --gemini   Install Gemini CLI hook (requires -g)
  --codex    Configure Codex CLI integration

Examples:
  tokman init              # Local init (config directories only)
  tokman init -g           # Global init with Claude Code integration
  tokman init -g --cursor  # Also install Cursor hook
  tokman init -g --copilot # Also install Copilot hook
  tokman init -g --gemini  # Also install Gemini CLI hook
  tokman init -g --codex   # Also configure Codex CLI
  tokman init -g --auto-patch  # Skip confirmation prompt
  tokman init -g --no-patch    # Skip settings.json patching
  tokman init -g --uninstall   # Remove TokMan from ~/.claude/`,
	Run: func(cmd *cobra.Command, args []string) {
		if initUninstall {
			if err := runUninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			return
		}

		if initGlobal {
			mode := PatchModeAsk
			if initAutoPatch {
				mode = PatchModeAuto
			} else if initNoPatch {
				mode = PatchModeSkip
			}
			if err := runGlobalInit(mode); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return
			}
		} else {
			if err := runLocalInit(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return
			}
		}

		// Install editor-specific hooks
		if initCursor {
			installCursorHook()
		}
		if initCopilot {
			installCopilotHook()
		}
		if initWindsurf {
			installWindsurfRules()
		}
		if initCline {
			installClineRules()
		}
		if initOpencode {
			installOpencodePlugin()
		}
		if initGemini {
			mode := PatchModeAsk
			if initAutoPatch {
				mode = PatchModeAuto
			} else if initNoPatch {
				mode = PatchModeSkip
			}
			runGeminiInit(initGlobal, mode)
		}
		if initCodex {
			runCodexInit(initGlobal)
		}

		// New agent hooks
		mode := PatchModeAsk
		if initAutoPatch {
			mode = PatchModeAuto
		} else if initNoPatch {
			mode = PatchModeSkip
		}

		if initAdaL {
			runAdaLInit(initGlobal, mode)
		}
		if initKiro {
			runKiroInit(initGlobal, mode)
		}
		if initKilo {
			runKiloInit(initGlobal, mode)
		}
		if initReplit {
			runReplitInit(initGlobal)
		}
		if initAider {
			runAiderInit(initGlobal)
		}
		if initContinue {
			runContinueInit(initGlobal)
		}

		// --all flag: detect and setup all installed agents
		if initAll {
			runAllAgentsInit(initGlobal, mode)
		}
	},
}

func init() {
	registry.Add(func() { registry.Register(initCmd) })
	initCmd.Flags().BoolVarP(&initGlobal, "global", "g", false, "Global init with Claude Code integration")
	initCmd.Flags().BoolVar(&initAutoPatch, "auto-patch", false, "Auto-patch settings.json without prompting")
	initCmd.Flags().BoolVar(&initNoPatch, "no-patch", false, "Skip settings.json patching (show manual instructions)")
	initCmd.Flags().BoolVar(&initUninstall, "uninstall", false, "Remove TokMan from ~/.claude/")
	initCmd.Flags().StringVar(&initMode, "mode", "minimal", "Filter mode: none, minimal, aggressive")
	initCmd.Flags().BoolVar(&initAll, "all", false, "Setup all detected agents")
	// Existing agent flags
	initCmd.Flags().BoolVar(&initCursor, "cursor", false, "Install Cursor Agent hook")
	initCmd.Flags().BoolVar(&initCopilot, "copilot", false, "Install GitHub Copilot hook + instructions")
	initCmd.Flags().BoolVar(&initWindsurf, "windsurf", false, "Install Windsurf rules file")
	initCmd.Flags().BoolVar(&initCline, "cline", false, "Install Cline rules file")
	initCmd.Flags().BoolVar(&initOpencode, "opencode", false, "Install OpenCode plugin")
	initCmd.Flags().BoolVar(&initGemini, "gemini", false, "Install Gemini CLI hook")
	initCmd.Flags().BoolVar(&initCodex, "codex", false, "Configure Codex CLI integration")
	// New agent flags
	initCmd.Flags().BoolVar(&initAdaL, "adal", false, "Install AdaL (SylphAI) integration")
	initCmd.Flags().BoolVar(&initKiro, "kiro", false, "Install Kiro integration")
	initCmd.Flags().BoolVar(&initKilo, "kilo", false, "Install Kilo Code integration")
	initCmd.Flags().BoolVar(&initReplit, "replit", false, "Install Replit Agent integration")
	initCmd.Flags().BoolVar(&initAider, "aider", false, "Install Aider native integration")
	initCmd.Flags().BoolVar(&initContinue, "continue", false, "Install Continue integration")
}

func runLocalInit() error {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\n%s\n", green("Initializing TokMan..."))
	fmt.Println()

	// Create config directory
	configDir := filepath.Dir(config.ConfigPath())
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	fmt.Printf("  %s Config directory: %s\n", green("✓"), cyan(configDir))

	// Create data directory
	dataDir := config.DataPath()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}
	fmt.Printf("  %s Data directory: %s\n", green("✓"), cyan(dataDir))

	// Create hooks directory
	hooksDir := config.HooksPath()
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}
	fmt.Printf("  %s Hooks directory: %s\n", green("✓"), cyan(hooksDir))

	// Initialize database
	dbPath := config.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	tracker.Close()
	fmt.Printf("  %s Database: %s\n", green("✓"), cyan(dbPath))

	// Create default config if it doesn't exist
	cfgPath := config.ConfigPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg := config.Defaults()
		if err := cfg.Save(cfgPath); err != nil {
			return fmt.Errorf("creating config file: %w", err)
		}
		fmt.Printf("  %s Config file: %s\n", green("✓"), cyan(cfgPath))
	} else {
		fmt.Printf("  %s Config file exists: %s\n", green("✓"), cyan(cfgPath))
	}

	// Create log file
	logPath := config.LogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		if err := utils.InitLogger(logPath, utils.LevelInfo); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create log file: %v\n", err)
		} else {
			fmt.Printf("  %s Log file: %s\n", green("✓"), cyan(logPath))
		}
	}

	// Store integrity hash for hook if it exists
	hookPath := filepath.Join(hooksDir, "tokman-rewrite.sh")
	if _, err := os.Stat(hookPath); err == nil {
		if err := integrity.StoreHash(hookPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to store hook integrity hash: %v\n", err)
		} else {
			fmt.Printf("  %s Hook integrity hash stored\n", green("✓"))
		}
	}

	fmt.Println()
	fmt.Println(green("✓ TokMan initialized successfully!"))
	fmt.Println()
	fmt.Println("To enable shell hooks, add to your .bashrc or .zshrc:")
	fmt.Printf("  %s\n", cyan("source ~/.local/share/tokman/hooks/tokman-rewrite.sh"))
	fmt.Println()
	return nil
}

// runGlobalInit performs global initialization with Claude Code integration
func runGlobalInit(patchMode PatchMode) error {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// First run local init
	if err := runLocalInit(); err != nil {
		return fmt.Errorf("local init failed: %w", err)
	}

	// Get Claude directory
	claudeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	claudeDir = filepath.Join(claudeDir, ".claude")

	// Create Claude hooks directory
	claudeHooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(claudeHooksDir, 0755); err != nil {
		return fmt.Errorf("creating Claude hooks directory: %w", err)
	}

	// Install hook
	hookPath := filepath.Join(claudeHooksDir, "tokman-rewrite.sh")
	hookChanged, err := ensureHookInstalled(hookPath)
	if err != nil {
		return fmt.Errorf("installing hook: %w", err)
	}

	hookStatus := "already up to date"
	if hookChanged {
		hookStatus = "installed/updated"
	}

	// Create TOKMAN.md
	tokmanMdPath := filepath.Join(claudeDir, "TOKMAN.md")
	if err := writeIfChanged(tokmanMdPath, getTokmanSlim(), "TOKMAN.md"); err != nil {
		return fmt.Errorf("creating TOKMAN.md: %w", err)
	}

	// Patch CLAUDE.md to add @TOKMAN.md reference
	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")
	migrated, err := patchClaudeMd(claudeMdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to patch CLAUDE.md: %v\n", err)
	}

	// Print success message
	fmt.Printf("\nTokMan hook %s (global).\n\n", hookStatus)
	fmt.Printf("  Hook:      %s\n", cyan(hookPath))
	fmt.Printf("  TOKMAN.md: %s\n", cyan(tokmanMdPath))
	fmt.Printf("  CLAUDE.md: %s\n", green("@TOKMAN.md reference added"))

	if migrated {
		fmt.Printf("\n  %s Migrated: removed old TokMan block from CLAUDE.md\n", yellow("!"))
		fmt.Println("              replaced with @TOKMAN.md reference")
	}

	// Patch settings.json
	patchResult := patchSettingsJson(hookPath, patchMode)

	// Report result
	switch patchResult {
	case PatchResultPatched:
		// Already printed by patchSettingsJson
	case PatchResultAlreadyPresent:
		fmt.Printf("\n  settings.json: %s\n", green("hook already present"))
		fmt.Println("  Restart Claude Code. Test with: tokman git status")
	case PatchResultDeclined, PatchResultSkipped:
		// Manual instructions already printed by patchSettingsJson
	}

	fmt.Println()
	return nil
}

// runUninstall removes all TokMan artifacts from ~/.claude/
func runUninstall() error {
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Get Claude directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	claudeDir := filepath.Join(homeDir, ".claude")

	var removed []string

	// 1. Remove hook file
	hookPath := filepath.Join(claudeDir, "hooks", "tokman-rewrite.sh")
	if _, err := os.Stat(hookPath); err == nil {
		if err := os.Remove(hookPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing hook: %v\n", err)
		} else {
			removed = append(removed, fmt.Sprintf("Hook: %s", hookPath))
		}
	}

	// 1b. Remove integrity hash file
	removedHash, err := integrity.RemoveHash(hookPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove hook hash: %v\n", err)
	}
	if removedHash {
		removed = append(removed, "Integrity hash: removed")
	}

	// 2. Remove TOKMAN.md
	tokmanMdPath := filepath.Join(claudeDir, "TOKMAN.md")
	if _, err := os.Stat(tokmanMdPath); err == nil {
		if err := os.Remove(tokmanMdPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing TOKMAN.md: %v\n", err)
		} else {
			removed = append(removed, fmt.Sprintf("TOKMAN.md: %s", tokmanMdPath))
		}
	}

	// 3. Remove @TOKMAN.md reference from CLAUDE.md
	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")
	if content, err := os.ReadFile(claudeMdPath); err == nil {
		if strings.Contains(string(content), "@TOKMAN.md") {
			newContent := removeTokmanMdReference(string(content))
			if err := os.WriteFile(claudeMdPath, []byte(newContent), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating CLAUDE.md: %v\n", err)
			} else {
				removed = append(removed, "CLAUDE.md: removed @TOKMAN.md reference")
			}
		}
	}

	// 4. Remove hook entry from settings.json
	if removedHook := removeHookFromSettings(claudeDir); removedHook {
		removed = append(removed, "settings.json: removed TokMan hook entry")
	}

	// 5. Remove Gemini artifacts
	removed = append(removed, uninstallGemini()...)

	// 6. Remove Codex artifacts
	removed = append(removed, uninstallCodex()...)

	// 7. Remove AdaL artifacts
	removed = append(removed, uninstallAdaL()...)

	// 8. Remove Kiro artifacts
	removed = append(removed, uninstallKiro()...)

	// 9. Remove Kilo artifacts
	removed = append(removed, uninstallKilo()...)

	// 10. Remove Replit artifacts
	removed = append(removed, uninstallReplit()...)

	// 11. Remove Aider artifacts
	removed = append(removed, uninstallAider()...)

	// 12. Remove Continue artifacts
	removed = append(removed, uninstallContinue()...)

	// Report results
	fmt.Println()
	if len(removed) == 0 {
		fmt.Println("TokMan was not installed (nothing to remove)")
	} else {
		fmt.Println("TokMan uninstalled:")
		for _, item := range removed {
			fmt.Printf("  %s %s\n", red("-"), cyan(item))
		}
		fmt.Println("\nRestart Claude Code to apply changes.")
	}
	fmt.Println()
	return nil
}
