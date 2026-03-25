package initpkg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

const geminiHookScript = `#!/usr/bin/env bash
exec tokman hook gemini
`

// runGeminiInit sets up Gemini CLI integration
func runGeminiInit(global bool, patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	if !global {
		fmt.Fprintf(os.Stderr, "Gemini support is global-only. Use: tokman init -g --gemini\n")
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		return
	}
	geminiDir := filepath.Join(homeDir, ".gemini")

	if err := os.MkdirAll(geminiDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Gemini config dir: %v\n", err)
		return
	}

	hookDir := filepath.Join(geminiDir, "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Gemini hooks dir: %v\n", err)
		return
	}

	hookPath := filepath.Join(hookDir, "tokman-hook-gemini.sh")
	if err := os.WriteFile(hookPath, []byte(geminiHookScript), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Gemini hook: %v\n", err)
		return
	}

	geminiMdPath := filepath.Join(geminiDir, "GEMINI.md")
	tokmanSlim := getTokmanSlim()
	if err := writeIfChanged(geminiMdPath, tokmanSlim, "GEMINI.md"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating GEMINI.md: %v\n", err)
	}

	patchGeminiSettings(geminiDir, hookPath, patchMode)

	fmt.Printf("\n%s\n\n", green("Gemini CLI hook installed (global)."))
	fmt.Printf("  Hook:      %s\n", cyan(hookPath))
	fmt.Printf("  GEMINI.md: %s\n", cyan(geminiMdPath))
	fmt.Println("  Restart Gemini CLI. Test with: git status")
	fmt.Println()
}

// patchGeminiSettings patches ~/.gemini/settings.json with the BeforeTool hook
func patchGeminiSettings(geminiDir string, hookPath string, patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	settingsPath := filepath.Join(geminiDir, "settings.json")

	var settings map[string]any
	if data, err := os.ReadFile(settingsPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = make(map[string]any)
		}
	} else {
		settings = make(map[string]any)
	}

	if isGeminiHookPresent(settings) {
		fmt.Printf("  %s Gemini settings.json: hook already present\n", green("✓"))
		return
	}

	switch patchMode {
	case PatchModeSkip:
		fmt.Printf("\n  Manual setup: add TokMan hook to %s\n", cyan(settingsPath))
		return
	case PatchModeAsk:
		fmt.Printf("\nPatch %s with TokMan hook? [y/N] ", settingsPath)
		if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			fmt.Println("(non-interactive, skipping)")
			return
		}
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(response)), "y") {
			fmt.Println("Skipped.")
			return
		}
	case PatchModeAuto:
	}

	hookEntry := map[string]any{
		"matcher": "run_shell_command",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": hookPath,
			},
		},
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	beforeTool, ok := hooks["BeforeTool"].([]any)
	if !ok {
		beforeTool = []any{}
	}

	beforeTool = append(beforeTool, hookEntry)
	hooks["BeforeTool"] = beforeTool

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serializing settings.json: %v\n", err)
		return
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing settings.json: %v\n", err)
		return
	}

	fmt.Printf("  %s Gemini settings.json: hook added\n", green("✓"))
}

// isGeminiHookPresent checks if TokMan hook is already in Gemini settings
func isGeminiHookPresent(settings map[string]any) bool {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return false
	}
	beforeTool, ok := hooks["BeforeTool"].([]any)
	if !ok {
		return false
	}
	for _, entry := range beforeTool {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		hooksArr, ok := entryMap["hooks"].([]any)
		if !ok {
			continue
		}
		for _, h := range hooksArr {
			hMap, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, ok := hMap["command"].(string); ok && strings.Contains(cmd, "tokman") {
				return true
			}
		}
	}
	return false
}

// uninstallGemini removes Gemini artifacts
func uninstallGemini() []string {
	var removed []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return removed
	}
	geminiDir := filepath.Join(homeDir, ".gemini")

	hookPath := filepath.Join(geminiDir, "hooks", "tokman-hook-gemini.sh")
	if _, err := os.Stat(hookPath); err == nil {
		if err := os.Remove(hookPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", hookPath, err)
		}
		removed = append(removed, fmt.Sprintf("Gemini hook: %s", hookPath))
	}

	geminiMdPath := filepath.Join(geminiDir, "GEMINI.md")
	if _, err := os.Stat(geminiMdPath); err == nil {
		if err := os.Remove(geminiMdPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", geminiMdPath, err)
		}
		removed = append(removed, fmt.Sprintf("GEMINI.md: %s", geminiMdPath))
	}

	settingsPath := filepath.Join(geminiDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]any
		if json.Unmarshal(data, &settings) == nil {
			if hooks, ok := settings["hooks"].(map[string]any); ok {
				if beforeTool, ok := hooks["BeforeTool"].([]any); ok {
					var newBeforeTool []any
					for _, entry := range beforeTool {
						entryMap, ok := entry.(map[string]any)
						if !ok {
							newBeforeTool = append(newBeforeTool, entry)
							continue
						}
						hooksArr, ok := entryMap["hooks"].([]any)
						if !ok {
							newBeforeTool = append(newBeforeTool, entry)
							continue
						}
						hasTokman := false
						for _, h := range hooksArr {
							hMap, ok := h.(map[string]any)
							if !ok {
								continue
							}
							if cmd, ok := hMap["command"].(string); ok && strings.Contains(cmd, "tokman") {
								hasTokman = true
								break
							}
						}
						if !hasTokman {
							newBeforeTool = append(newBeforeTool, entry)
						}
					}
					if len(newBeforeTool) != len(beforeTool) {
						if len(newBeforeTool) == 0 {
							delete(hooks, "BeforeTool")
							if len(hooks) == 0 {
								delete(settings, "hooks")
							}
						} else {
							hooks["BeforeTool"] = newBeforeTool
						}
						newData, _ := json.MarshalIndent(settings, "", "  ")
						if err := os.WriteFile(settingsPath, newData, 0644); err != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", settingsPath, err)
						}
						removed = append(removed, "Gemini settings.json: removed hook entry")
					}
				}
			}
		}
	}

	return removed
}
