package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

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
)

// Embedded hook script
const rewriteHook = `#!/usr/bin/env bash
# tokman-hook-version: 2
# TokMan Claude Code hook — rewrites commands to use tokman for token savings.
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

Examples:
  tokman init              # Local init (config directories only)
  tokman init -g           # Global init with Claude Code integration
  tokman init -g --auto-patch  # Skip confirmation prompt
  tokman init -g --no-patch    # Skip settings.json patching
  tokman init -g --uninstall   # Remove TokMan from ~/.claude/`,
	Run: func(cmd *cobra.Command, args []string) {
		if initUninstall {
			runUninstall()
			return
		}

		if initGlobal {
			mode := PatchModeAsk
			if initAutoPatch {
				mode = PatchModeAuto
			} else if initNoPatch {
				mode = PatchModeSkip
			}
			runGlobalInit(mode)
			return
		}

		runLocalInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initGlobal, "global", "g", false, "Global init with Claude Code integration")
	initCmd.Flags().BoolVar(&initAutoPatch, "auto-patch", false, "Auto-patch settings.json without prompting")
	initCmd.Flags().BoolVar(&initNoPatch, "no-patch", false, "Skip settings.json patching (show manual instructions)")
	initCmd.Flags().BoolVar(&initUninstall, "uninstall", false, "Remove TokMan from ~/.claude/")
	initCmd.Flags().StringVar(&initMode, "mode", "minimal", "Filter mode: none, minimal, aggressive")
}

func runLocalInit() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\n%s\n", green("Initializing TokMan..."))
	fmt.Println()

	// Create config directory
	configDir := filepath.Dir(config.ConfigPath())
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  %s Config directory: %s\n", green("✓"), cyan(configDir))

	// Create data directory
	dataDir := config.DataPath()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  %s Data directory: %s\n", green("✓"), cyan(dataDir))

	// Create hooks directory
	hooksDir := config.HooksPath()
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating hooks directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  %s Hooks directory: %s\n", green("✓"), cyan(hooksDir))

	// Initialize database
	dbPath := config.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	tracker.Close()
	fmt.Printf("  %s Database: %s\n", green("✓"), cyan(dbPath))

	// Create default config if it doesn't exist
	cfgPath := config.ConfigPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg := config.Defaults()
		if err := cfg.Save(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
			os.Exit(1)
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
}

// runGlobalInit performs global initialization with Claude Code integration
func runGlobalInit(patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// First run local init
	runLocalInit()

	// Get Claude directory
	claudeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	claudeDir = filepath.Join(claudeDir, ".claude")

	// Create Claude hooks directory
	claudeHooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(claudeHooksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Claude hooks directory: %v\n", err)
		os.Exit(1)
	}

	// Install hook
	hookPath := filepath.Join(claudeHooksDir, "tokman-rewrite.sh")
	hookChanged, err := ensureHookInstalled(hookPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing hook: %v\n", err)
		os.Exit(1)
	}

	hookStatus := "already up to date"
	if hookChanged {
		hookStatus = "installed/updated"
	}

	// Create TOKMAN.md
	tokmanMdPath := filepath.Join(claudeDir, "TOKMAN.md")
	if err := writeIfChanged(tokmanMdPath, getTokmanSlim(), "TOKMAN.md"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating TOKMAN.md: %v\n", err)
		os.Exit(1)
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
}

// runUninstall removes all TokMan artifacts from ~/.claude/
func runUninstall() {
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Get Claude directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
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
	if removedHash, _ := integrity.RemoveHash(hookPath); removedHash {
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
}

// ensureHookInstalled writes the hook file if missing or outdated
func ensureHookInstalled(hookPath string) (bool, error) {
	changed := false

	if content, err := os.ReadFile(hookPath); err == nil {
		if string(content) == rewriteHook {
			return false, nil
		}
	}

	// Write hook
	if err := os.WriteFile(hookPath, []byte(rewriteHook), 0755); err != nil {
		return false, err
	}
	changed = true

	// Store integrity hash
	if err := integrity.StoreHash(hookPath); err != nil {
		return changed, err
	}

	return changed, nil
}

// writeIfChanged writes content to file if it differs from existing content
func writeIfChanged(path, content, name string) error {
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == content {
			return nil
		}
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// patchClaudeMd adds @TOKMAN.md reference to CLAUDE.md
func patchClaudeMd(path string) (bool, error) {
	content := ""
	if data, err := os.ReadFile(path); err == nil {
		content = string(data)
	}

	migrated := false

	// Check for old block and migrate
	if strings.Contains(content, "<!-- tokman-instructions") {
		newContent, didMigrate := removeTokmanBlock(content)
		if didMigrate {
			content = newContent
			migrated = true
		}
	}

	// Check if @TOKMAN.md already present
	if strings.Contains(content, "@TOKMAN.md") {
		if migrated {
			os.WriteFile(path, []byte(content), 0644)
		}
		return migrated, nil
	}

	// Add @TOKMAN.md
	var newContent string
	if content == "" {
		newContent = "@TOKMAN.md\n"
	} else {
		newContent = fmt.Sprintf("%s\n\n@TOKMAN.md\n", strings.TrimSpace(content))
	}

	return migrated, os.WriteFile(path, []byte(newContent), 0644)
}

// removeTokmanBlock removes old TokMan block from content
func removeTokmanBlock(content string) (string, bool) {
	startMarker := "<!-- tokman-instructions"
	endMarker := "<!-- /tokman-instructions -->"

	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return content, false
	}

	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx == -1 {
		// Malformed - opening without closing
		return content, false
	}

	endPos := startIdx + endIdx + len(endMarker)
	before := strings.TrimRight(content[:startIdx], "\n")
	after := strings.TrimLeft(content[endPos:], "\n")

	if after == "" {
		return before, true
	}
	return before + "\n\n" + after, true
}

// removeTokmanMdReference removes @TOKMAN.md lines from content
func removeTokmanMdReference(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "@TOKMAN.md") {
			result = append(result, line)
		}
	}
	// Clean up double blanks
	return cleanDoubleBlanks(strings.Join(result, "\n"))
}

// cleanDoubleBlanks collapses 3+ consecutive blank lines to 2
func cleanDoubleBlanks(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "" {
			blankCount := 0
			for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				blankCount++
				i++
			}
			keep := blankCount
			if keep > 2 {
				keep = 2
			}
			for j := 0; j < keep; j++ {
				result = append(result, "")
			}
		} else {
			result = append(result, lines[i])
			i++
		}
	}
	return strings.Join(result, "\n")
}

// patchSettingsJson patches settings.json with TokMan hook
func patchSettingsJson(hookPath string, mode PatchMode) PatchResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		return PatchResultSkipped
	}

	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")

	// Read or create settings.json
	var root map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &root); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing settings.json: %v\n", err)
			return PatchResultSkipped
		}
	} else {
		root = make(map[string]interface{})
	}

	// Check if hook already present
	if hookAlreadyPresent(root, hookPath) {
		return PatchResultAlreadyPresent
	}

	// Handle mode
	switch mode {
	case PatchModeSkip:
		printManualInstructions(hookPath)
		return PatchResultSkipped
	case PatchModeAsk:
		if !promptUserConsent(settingsPath) {
			printManualInstructions(hookPath)
			return PatchResultDeclined
		}
	case PatchModeAuto:
		// Proceed without prompting
	}

	// Insert hook entry
	insertHookEntry(root, hookPath)

	// Backup original
	if data, err := os.ReadFile(settingsPath); err == nil {
		backupPath := settingsPath + ".bak"
		os.WriteFile(backupPath, data, 0644)
	}

	// Write updated settings
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serializing settings.json: %v\n", err)
		return PatchResultSkipped
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing settings.json: %v\n", err)
		return PatchResultSkipped
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n  settings.json: %s\n", green("hook added"))
	fmt.Println("  Restart Claude Code. Test with: tokman git status")

	return PatchResultPatched
}

// hookAlreadyPresent checks if TokMan hook is already in settings.json
func hookAlreadyPresent(root map[string]interface{}, hookPath string) bool {
	hooks, ok := root["hooks"].(map[string]interface{})
	if !ok {
		return false
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		return false
	}

	for _, entry := range preToolUse {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		hooksArray, ok := entryMap["hooks"].([]interface{})
		if !ok {
			continue
		}

		for _, hook := range hooksArray {
			hookMap, ok := hook.(map[string]interface{})
			if !ok {
				continue
			}

			if cmd, ok := hookMap["command"].(string); ok {
				if strings.Contains(cmd, "tokman-rewrite.sh") && strings.Contains(hookPath, "tokman-rewrite.sh") {
					return true
				}
			}
		}
	}

	return false
}

// insertHookEntry adds TokMan hook to settings.json
func insertHookEntry(root map[string]interface{}, hookPath string) {
	// Ensure hooks object exists
	hooks, ok := root["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		root["hooks"] = hooks
	}

	// Ensure PreToolUse array exists
	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		preToolUse = []interface{}{}
	}

	// Append TokMan hook entry
	hookEntry := map[string]interface{}{
		"matcher": "Bash",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookPath,
			},
		},
	}

	preToolUse = append(preToolUse, hookEntry)
	hooks["PreToolUse"] = preToolUse
}

// removeHookFromSettings removes TokMan hook from settings.json
func removeHookFromSettings(claudeDir string) bool {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil || len(data) == 0 {
		return false
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return false
	}

	hooks, ok := root["hooks"].(map[string]interface{})
	if !ok {
		return false
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		return false
	}

	originalLen := len(preToolUse)
	var newPreToolUse []interface{}

	for _, entry := range preToolUse {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}

		hooksArray, ok := entryMap["hooks"].([]interface{})
		if !ok {
			newPreToolUse = append(newPreToolUse, entry)
			continue
		}

		hasTokmanHook := false
		for _, hook := range hooksArray {
			hookMap, ok := hook.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, ok := hookMap["command"].(string); ok {
				if strings.Contains(cmd, "tokman-rewrite.sh") {
					hasTokmanHook = true
					break
				}
			}
		}

		if !hasTokmanHook {
			newPreToolUse = append(newPreToolUse, entry)
		}
	}

	if len(newPreToolUse) == originalLen {
		return false
	}

	// Backup
	os.WriteFile(settingsPath+".bak", data, 0644)

	// Update
	if len(newPreToolUse) == 0 {
		delete(hooks, "PreToolUse")
		if len(hooks) == 0 {
			delete(root, "hooks")
		}
	} else {
		hooks["PreToolUse"] = newPreToolUse
	}

	newData, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false
	}

	return os.WriteFile(settingsPath, newData, 0644) == nil
}

// promptUserConsent asks user for permission to patch settings.json
func promptUserConsent(settingsPath string) bool {
	fmt.Printf("\nPatch existing %s? [y/N] ", settingsPath)

	// Check if stdin is a terminal
	if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		fmt.Println("(non-interactive mode, defaulting to N)")
		return false
	}

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}

// printManualInstructions shows how to manually add the hook
func printManualInstructions(hookPath string) {
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println("\n  MANUAL STEP: Add this to ~/.claude/settings.json:")
	fmt.Println("  {")
	fmt.Println(`    "hooks": { "PreToolUse": [{`)
	fmt.Println(`      "matcher": "Bash",`)
	fmt.Println(`      "hooks": [{ "type": "command",`)
	fmt.Printf("        \"command\": \"%s\"\n", hookPath)
	fmt.Println(`      }]`)
	fmt.Println(`    }]}`)
	fmt.Println("  }")
	fmt.Printf("\n  Then restart Claude Code. Test with: %s\n", cyan("tokman git status"))
	fmt.Println()
}
