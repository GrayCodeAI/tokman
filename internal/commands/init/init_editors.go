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

	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
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

	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Copilot hook: %v\n", err)
		return
	}

	instructionsDir := filepath.Join(home, ".github")
	if err := os.MkdirAll(instructionsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}
	instructionsPath := filepath.Join(instructionsDir, "copilot-instructions.md")
	instructions := "# TokMan - Token-Optimized CLI Proxy\n\n" +
		"Always prefix shell commands with `tokman` to minimize token consumption.\n\n" +
		"Examples: tokman git status, tokman cargo test, tokman ls, tokman grep \"pattern\" .\n" +
		"Meta: tokman gain (savings), tokman discover (missed opportunities), tokman proxy <cmd> (raw)\n"
	if err := os.WriteFile(instructionsPath, []byte(instructions), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", instructionsPath, err)
	}

	fmt.Printf("\n  %s Copilot hook: %s\n", green("✓"), cyan(hookPath))
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
