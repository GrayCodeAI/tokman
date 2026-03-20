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
