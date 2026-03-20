package commands

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

	// 1. Install hook script
	hookDir := filepath.Join(geminiDir, "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Gemini hooks dir: %v\n", err)
		return
	}

	hookPath := filepath.Join(hookDir, "tokman-hook-gemini.sh")
	if err := os.WriteFile(hookPath, []byte(geminiHookScript), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Gemini hook: %v\n", err)
		return
	}

	// 2. Install GEMINI.md (TokMan awareness for Gemini)
	geminiMdPath := filepath.Join(geminiDir, "GEMINI.md")
	tokmanSlim := getTokmanSlim()
	if err := writeIfChanged(geminiMdPath, tokmanSlim, "GEMINI.md"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating GEMINI.md: %v\n", err)
	}

	// 3. Patch ~/.gemini/settings.json
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

	// Read or create settings.json
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = make(map[string]interface{})
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Check if hook already registered
	if isGeminiHookPresent(settings) {
		fmt.Printf("  %s Gemini settings.json: hook already present\n", green("✓"))
		return
	}

	// Handle patch mode
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
		// Proceed without prompting
	}

	// Build hook entry matching Gemini CLI format
	hookEntry := map[string]interface{}{
		"matcher": "run_shell_command",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookPath,
			},
		},
	}

	// Insert into settings
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	beforeTool, ok := hooks["BeforeTool"].([]interface{})
	if !ok {
		beforeTool = []interface{}{}
	}

	beforeTool = append(beforeTool, hookEntry)
	hooks["BeforeTool"] = beforeTool

	// Write settings
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
func isGeminiHookPresent(settings map[string]interface{}) bool {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return false
	}
	beforeTool, ok := hooks["BeforeTool"].([]interface{})
	if !ok {
		return false
	}
	for _, entry := range beforeTool {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		hooksArr, ok := entryMap["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range hooksArr {
			hMap, ok := h.(map[string]interface{})
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

	// Remove hook
	hookPath := filepath.Join(geminiDir, "hooks", "tokman-hook-gemini.sh")
	if _, err := os.Stat(hookPath); err == nil {
		os.Remove(hookPath)
		removed = append(removed, fmt.Sprintf("Gemini hook: %s", hookPath))
	}

	// Remove GEMINI.md
	geminiMdPath := filepath.Join(geminiDir, "GEMINI.md")
	if _, err := os.Stat(geminiMdPath); err == nil {
		os.Remove(geminiMdPath)
		removed = append(removed, fmt.Sprintf("GEMINI.md: %s", geminiMdPath))
	}

	// Remove hook from settings.json
	settingsPath := filepath.Join(geminiDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]interface{}
		if json.Unmarshal(data, &settings) == nil {
			if hooks, ok := settings["hooks"].(map[string]interface{}); ok {
				if beforeTool, ok := hooks["BeforeTool"].([]interface{}); ok {
					var newBeforeTool []interface{}
					for _, entry := range beforeTool {
						entryMap, ok := entry.(map[string]interface{})
						if !ok {
							newBeforeTool = append(newBeforeTool, entry)
							continue
						}
						hooksArr, ok := entryMap["hooks"].([]interface{})
						if !ok {
							newBeforeTool = append(newBeforeTool, entry)
							continue
						}
						hasTokman := false
						for _, h := range hooksArr {
							hMap, ok := h.(map[string]interface{})
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
						os.WriteFile(settingsPath, newData, 0644)
						removed = append(removed, "Gemini settings.json: removed hook entry")
					}
				}
			}
		}
	}

	return removed
}
