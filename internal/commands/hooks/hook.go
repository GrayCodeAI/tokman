package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/discover"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook processors for AI coding agents",
	Long: `Native hook processors that read JSON from stdin and output
rewritten commands for various AI coding agent platforms.

Supported platforms:
  gemini   — Gemini CLI BeforeTool hook
  copilot  — GitHub Copilot (VS Code Chat + Copilot CLI)`,
}

func init() {
	registry.Add(func() { registry.Register(hookCmd) })
	hookCmd.AddCommand(hookGeminiCmd)
	hookCmd.AddCommand(hookCopilotCmd)
}

// ── Gemini CLI hook ───────────────────────────────────────────

var hookGeminiCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Process Gemini CLI BeforeTool hook",
	Long: `Reads JSON from stdin (Gemini CLI hook format), rewrites
run_shell_command tool calls to TokMan equivalents, and outputs
Gemini CLI JSON format to stdout.

Used as a Gemini CLI BeforeTool hook — install with: tokman init -g --gemini`,
	Run: func(cmd *cobra.Command, args []string) {
		runGeminiHook()
	},
}

func runGeminiHook() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}

	inputStr := strings.TrimSpace(string(input))
	if inputStr == "" {
		printGeminiAllow()
		return
	}

	var v map[string]any
	if err := json.Unmarshal([]byte(inputStr), &v); err != nil {
		printGeminiAllow()
		return
	}

	toolName, _ := v["tool_name"].(string)
	if toolName != "run_shell_command" {
		printGeminiAllow()
		return
	}

	toolInput, ok := v["tool_input"].(map[string]any)
	if !ok {
		printGeminiAllow()
		return
	}

	cmdStr, _ := toolInput["command"].(string)
	if cmdStr == "" {
		printGeminiAllow()
		return
	}

	rewritten, changed := discover.RewriteCommand(cmdStr, nil)
	if !changed {
		printGeminiAllow()
		return
	}

	printGeminiRewrite(rewritten)
}

func printGeminiAllow() {
	fmt.Println(`{"decision":"allow"}`)
}

func printGeminiRewrite(cmd string) {
	output := map[string]any{
		"decision": "allow",
		"hookSpecificOutput": map[string]any{
			"tool_input": map[string]any{
				"command": cmd,
			},
		},
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))
}

// ── Copilot hook (VS Code + Copilot CLI) ──────────────────────

type copilotHookFormat int

const (
	copilotFormatVsCode copilotHookFormat = iota
	copilotFormatCli
	copilotFormatPassThrough
)

var hookCopilotCmd = &cobra.Command{
	Use:   "copilot",
	Short: "Process Copilot preToolUse hook",
	Long: `Reads JSON from stdin, auto-detects VS Code Copilot Chat format
(snake_case) vs Copilot CLI format (camelCase), and outputs the
appropriate response.

Used as a Copilot preToolUse hook — install with: tokman init -g --copilot`,
	Run: func(cmd *cobra.Command, args []string) {
		runCopilotHook()
	},
}

func runCopilotHook() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}

	inputStr := strings.TrimSpace(string(input))
	if inputStr == "" {
		return
	}

	var v map[string]any
	if err := json.Unmarshal([]byte(inputStr), &v); err != nil {
		fmt.Fprintln(os.Stderr, "[tokman hook] Failed to parse JSON input:", err)
		return
	}

	format, command := detectCopilotFormat(v)
	switch format {
	case copilotFormatVsCode:
		handleCopilotVsCode(command)
	case copilotFormatCli:
		handleCopilotCli(command)
	}
}

func detectCopilotFormat(v map[string]any) (copilotHookFormat, string) {
	// VS Code Copilot Chat / Claude Code: snake_case keys
	if toolName, ok := v["tool_name"].(string); ok {
		switch toolName {
		case "runTerminalCommand", "Bash", "bash":
			if toolInput, ok := v["tool_input"].(map[string]any); ok {
				if cmd, ok := toolInput["command"].(string); ok && cmd != "" {
					return copilotFormatVsCode, cmd
				}
			}
		}
		return copilotFormatPassThrough, ""
	}

	// Copilot CLI: camelCase keys, toolArgs is a JSON-encoded string
	if toolName, ok := v["toolName"].(string); ok && toolName == "bash" {
		if toolArgsStr, ok := v["toolArgs"].(string); ok {
			var toolArgs map[string]any
			if err := json.Unmarshal([]byte(toolArgsStr), &toolArgs); err == nil {
				if cmd, ok := toolArgs["command"].(string); ok && cmd != "" {
					return copilotFormatCli, cmd
				}
			}
		}
		return copilotFormatPassThrough, ""
	}

	return copilotFormatPassThrough, ""
}

func handleCopilotVsCode(cmd string) {
	rewritten, changed := discover.RewriteCommand(cmd, nil)
	if !changed {
		return
	}

	output := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "allow",
			"permissionDecisionReason": "TokMan auto-rewrite",
			"updatedInput": map[string]any{
				"command": rewritten,
			},
		},
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))
}

func handleCopilotCli(cmd string) {
	rewritten, changed := discover.RewriteCommand(cmd, nil)
	if !changed {
		fmt.Println("{}")
		return
	}

	output := map[string]any{
		"permissionDecision":       "deny",
		"permissionDecisionReason": fmt.Sprintf("Token savings: use `%s` instead (tokman saves 60-90%% tokens)", rewritten),
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))
}
