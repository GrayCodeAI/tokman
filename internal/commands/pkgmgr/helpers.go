package pkgmgr

import (
	"encoding/json"
	"fmt"
	"strings"
)

func filterTscOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	var errors, warnings int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "error") {
			errors++
		}
		if strings.Contains(trimmed, "warning") {
			warnings++
		}
		if len(line) > 120 {
			line = line[:117] + "..."
		}
		result = append(result, line)
	}

	if len(result) > 0 {
		result = append(result, fmt.Sprintf("\n%d errors, %d warnings", errors, warnings))
	}
	return strings.Join(result, "\n")
}

func filterEslintJSON(output string) string {
	var data []map[string]any
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return output
	}

	var result []string
	for _, file := range data {
		name, _ := file["filePath"].(string)
		messages, _ := file["messages"].([]any)
		for _, msg := range messages {
			m, _ := msg.(map[string]any)
			line, _ := m["line"].(float64)
			msgStr, _ := m["message"].(string)
			result = append(result, fmt.Sprintf("%s:%d: %s", name, int(line), msgStr))
		}
	}
	return strings.Join(result, "\n")
}

func filterPrettierOutput(output string) string {
	return strings.TrimSpace(output)
}

func filterPrismaOutputCompact(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "✔") || strings.HasPrefix(trimmed, "✓") {
			result = append(result, trimmed)
			continue
		}
		if strings.Contains(strings.ToLower(trimmed), "error") || strings.Contains(strings.ToLower(trimmed), "warn") {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, "\n")
}

func filterNextOutputCompact(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "✓") || strings.Contains(trimmed, "○") || strings.Contains(trimmed, "λ") {
			result = append(result, trimmed)
			continue
		}
		if strings.Contains(strings.ToLower(trimmed), "error") || strings.Contains(strings.ToLower(trimmed), "warn") {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return "Build completed"
	}
	return strings.Join(result, "\n")
}
