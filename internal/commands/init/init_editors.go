package initpkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

func installCursorHook() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	cursorDir, err := os.UserHomeDir()
	if err != nil {
		cursorDir = "."
	}
	cursorDir = filepath.Join(cursorDir, ".cursor")
	hooksDir := filepath.Join(cursorDir, "hooks")

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Cursor hooks directory: %v\n", err)
		return
	}

	hookPath := filepath.Join(hooksDir, "tokman-rewrite.sh")
	hookContent := "#!/usr/bin/env bash\n" +
		"# TokMan Cursor Agent hook\n" +
		"if ! command -v jq &>/dev/null || ! command -v tokman &>/dev/null; then exit 0; fi\n" +
		"INPUT=$(cat)\n" +
		"CMD=$(echo \"$INPUT\" | jq -r '.tool_input.command // empty')\n" +
		"[ -z \"$CMD\" ] && echo '{}' && exit 0\n" +
		"REWRITTEN=$(tokman rewrite \"$CMD\" 2>/dev/null) || { echo '{}'; exit 0; }\n" +
		"[ \"$CMD\" = \"$REWRITTEN\" ] && echo '{}' && exit 0\n" +
		"jq -n --arg cmd \"$REWRITTEN\" '{\"permission\":\"allow\",\"updated_input\":{\"command\":$cmd}}'\n"

	if err := os.WriteFile(hookPath, []byte(hookContent), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Cursor hook: %v\n", err)
		return
	}

	fmt.Printf("\n  %s Cursor hook: %s\n", green("✓"), cyan(hookPath))
	fmt.Println("  Add to ~/.cursor/hooks.json:")
	fmt.Printf("    %s\n", cyan(`{"PreToolUse":{"Bash":{"command":"bash","args":["`+hookPath+`"]}}}`))
}

func installCopilotHook() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	hooksDir := filepath.Join(home, ".tokman", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}

	hookPath := filepath.Join(hooksDir, "copilot-tokman-rewrite.sh")
	hookContent := "#!/usr/bin/env bash\n" +
		"# TokMan Copilot hook - supports VS Code Copilot Chat + Copilot CLI\n" +
		"if ! command -v tokman &>/dev/null; then exit 0; fi\n" +
		"INPUT=$(cat)\n" +
		"TOOL_NAME=$(echo \"$INPUT\" | jq -r '.tool_name // empty')\n" +
		"CMD=$(echo \"$INPUT\" | jq -r '.tool_input.command // empty')\n" +
		"if [ -n \"$TOOL_NAME\" ] && [ -n \"$CMD\" ]; then\n" +
		"  REWRITTEN=$(tokman rewrite \"$CMD\" 2>/dev/null) || { echo '{}'; exit 0; }\n" +
		"  [ \"$CMD\" = \"$REWRITTEN\" ] && echo '{}' && exit 0\n" +
		"  jq -n --arg cmd \"$REWRITTEN\" '{\"hookSpecificOutput\":{\"hookEvent\":\"PreToolUse\",\"updatedInput\":{\"command\":$cmd}}}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"TOOL_NAME_CAMEL=$(echo \"$INPUT\" | jq -r '.toolName // empty')\n" +
		"TOOL_ARGS=$(echo \"$INPUT\" | jq -r '.toolArgs // empty')\n" +
		"if [ \"$TOOL_NAME_CAMEL\" = \"bash\" ] && [ -n \"$TOOL_ARGS\" ]; then\n" +
		"  CMD=$(echo \"$TOOL_ARGS\" | jq -r '.command // empty')\n" +
		"  [ -n \"$CMD\" ] && { REWRITTEN=$(tokman rewrite \"$CMD\" 2>/dev/null) || { echo '{}'; exit 0; }; [ \"$CMD\" != \"$REWRITTEN\" ] && jq -n --arg r \"Token savings: use '$REWRITTEN' instead\" '{\"permissionDecision\":\"deny\",\"permissionDecisionReason\":$r}'; }\n" +
		"fi\n" +
		"echo '{}'\n"

	if err := os.WriteFile(hookPath, []byte(hookContent), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Copilot hook: %v\n", err)
		return
	}

	// Create .github/hooks/tokman-rewrite.json template (Task 67)
	githubHooksDir := filepath.Join(home, ".github", "hooks")
	if err := os.MkdirAll(githubHooksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create .github/hooks: %v\n", err)
	}
	jsonConfigPath := filepath.Join(githubHooksDir, "tokman-rewrite.json")
	jsonConfig := "{\n" +
		"  \"version\": \"1.0\",\n" +
		"  \"name\": \"TokMan Rewrite Hook\",\n" +
		"  \"description\": \"Automatically rewrites commands to use tokman for token optimization\",\n" +
		"  \"hook\": {\n" +
		"    \"type\": \"PreToolUse\",\n" +
		"    \"tools\": [\"Bash\", \"Shell\"],\n" +
		"    \"command\": \"" + hookPath + "\",\n" +
		"    \"timeout\": 5000\n" +
		"  },\n" +
		"  \"settings\": {\n" +
		"    \"enabled\": true,\n" +
		"    \"logRewrites\": true,\n" +
		"    \"excludedCommands\": [\"cd\", \"export\", \"source\", \"alias\"]\n" +
		"  }\n" +
		"}\n"
	if err := os.WriteFile(jsonConfigPath, []byte(jsonConfig), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", jsonConfigPath, err)
	}

	instructionsDir := filepath.Join(home, ".github")
	if err := os.MkdirAll(instructionsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}
	instructionsPath := filepath.Join(instructionsDir, "copilot-instructions.md")
	instructions := "# TokMan - Token-Optimized CLI Proxy\n\n" +
		"Always prefix shell commands with `tokman` to minimize token consumption.\n\n" +
		"## Quick Reference\n\n" +
		"**Build & Compile:** `tokman cargo build`, `tokman tsc`, `tokman lint`\n" +
		"**Tests:** `tokman cargo test`, `tokman vitest run`, `tokman playwright test`\n" +
		"**Git:** `tokman git status`, `tokman git log`, `tokman git diff`\n" +
		"**GitHub:** `tokman gh pr view`, `tokman gh run list`\n" +
		"**Files:** `tokman ls`, `tokman grep`, `tokman head`\n\n" +
		"## Meta Commands\n\n" +
		"- `tokman gain` - View token savings statistics\n" +
		"- `tokman discover` - Analyze sessions for missed usage\n" +
		"- `tokman proxy <cmd>` - Run command without filtering\n\n" +
		"**Average savings: 60-90% token reduction** on common development operations.\n"
	if err := os.WriteFile(instructionsPath, []byte(instructions), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", instructionsPath, err)
	}

	fmt.Printf("\n  %s Copilot hook: %s\n", green("✓"), cyan(hookPath))
	fmt.Printf("  %s Hook config: %s\n", green("✓"), cyan(jsonConfigPath))
	fmt.Printf("  %s Instructions: %s\n", green("✓"), cyan(instructionsPath))
}

func installWindsurfRules() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	rulesPath := ".windsurfrules"
	rules := "# TokMan - Token-Optimized CLI Proxy\n\n" +
		"Always prefix shell commands with `tokman` to minimize token consumption.\n\n" +
		"Examples: tokman git status, tokman cargo test, tokman ls, tokman grep \"pattern\" .\n" +
		"Meta: tokman gain (savings), tokman discover (missed opportunities), tokman proxy <cmd> (raw)\n"

	if err := os.WriteFile(rulesPath, []byte(rules), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Windsurf rules: %v\n", err)
		return
	}

	fmt.Printf("\n  %s Windsurf rules: %s\n", green("✓"), cyan(rulesPath))
}

func installClineRules() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	rulesPath := ".clinerules"
	rules := "# TokMan - Token-Optimized CLI Proxy\n\n" +
		"Always prefix shell commands with `tokman` to minimize token consumption.\n\n" +
		"Examples: tokman git status, tokman cargo test, tokman ls, tokman grep \"pattern\" .\n" +
		"Meta: tokman gain (savings), tokman discover (missed opportunities), tokman proxy <cmd> (raw)\n"

	if err := os.WriteFile(rulesPath, []byte(rules), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Cline rules: %v\n", err)
		return
	}

	fmt.Printf("\n  %s Cline rules: %s\n", green("✓"), cyan(rulesPath))
}

func installOpencodePlugin() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	pluginDir := filepath.Join(home, ".config", "opencode", "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}

	pluginPath := filepath.Join(pluginDir, "tokman.ts")
	plugin := "import type { Plugin } from \"@opencode-ai/plugin\"\n\n" +
		"export const TokmanPlugin: Plugin = async ({ $ }) => {\n" +
		"  try { await $`which tokman`.quiet() } catch { return {} }\n" +
		"  return {\n" +
		"    \"tool.execute.before\": async (input, output) => {\n" +
		"      const tool = String(input?.tool ?? \"\").toLowerCase()\n" +
		"      if (tool !== \"bash\" && tool !== \"shell\") return\n" +
		"      const args = output?.args\n" +
		"      if (!args || typeof args !== \"object\") return\n" +
		"      const command = (args as Record<string, unknown>).command\n" +
		"      if (typeof command !== \"string\" || !command) return\n" +
		"      try {\n" +
		"        const result = await $`tokman rewrite ${command}`.quiet().nothrow()\n" +
		"        const rewritten = String(result.stdout).trim()\n" +
		"        if (rewritten && rewritten !== command) {\n" +
		"          (args as Record<string, unknown>).command = rewritten\n" +
		"        }\n" +
		"      } catch {}\n" +
		"    },\n" +
		"  }\n" +
		"}\n"

	if err := os.WriteFile(pluginPath, []byte(plugin), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing OpenCode plugin: %v\n", err)
		return
	}

	fmt.Printf("\n  %s OpenCode plugin: %s\n", green("✓"), cyan(pluginPath))
}

// installMistralVibePlaceholder creates a placeholder for Mistral Vibe integration
// Track upstream: https://mistral.ai/vibe (or equivalent when available)
func installMistralVibePlaceholder() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	configDir := filepath.Join(home, ".config", "mistral-vibe")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}

	// Create placeholder config
	placeholderPath := filepath.Join(configDir, "tokman-placeholder.json")
	placeholder := "{\n" +
		"  \"_comment\": \"TokMan integration placeholder for Mistral Vibe\",\n" +
		"  \"_status\": \"pending-upstream-support\",\n" +
		"  \"_tracked\": \"https://mistral.ai/vibe\",\n" +
		"  \"hook\": {\n" +
		"    \"enabled\": false,\n" +
		"    \"type\": \"PreToolUse\",\n" +
		"    \"command\": \"tokman rewrite\"\n" +
		"  },\n" +
		"  \"instructions\": {\n" +
		"    \"prefix\": \"tokman\",\n" +
		"    \"description\": \"Token-optimized CLI proxy for 60-90% savings\"\n" +
		"  }\n" +
		"}\n"

	if err := os.WriteFile(placeholderPath, []byte(placeholder), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Mistral Vibe placeholder: %v\n", err)
		return
	}

	fmt.Printf("\n  %s Mistral Vibe: %s\n", yellow("○"), cyan("(placeholder - pending upstream support)"))
	fmt.Printf("  %s Config: %s\n", green("✓"), cyan(placeholderPath))
}
