package linter

import (
	"fmt"
	"os/exec"
	"strings"
)

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func compactPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")

	prefixes := []string{"/src/", "/lib/", "/tests/", "/test/"}
	for _, prefix := range prefixes {
		if idx := strings.LastIndex(path, prefix); idx >= 0 {
			return path[idx+1:]
		}
	}

	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}

	return path
}

func filterPrettierOutput(output string) string {
	var filesToFormat []string
	var filesChecked int

	outputLower := strings.ToLower(output)
	if strings.Contains(outputLower, "all matched files use prettier") {
		return "✓ Prettier: All files formatted correctly"
	}

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		lower := strings.ToLower(trimmed)

		if strings.Contains(lower, "checking") || strings.Contains(lower, "parsing") {
			continue
		}

		if !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "+") {
			if !strings.Contains(lower, "error") && !strings.Contains(lower, "warning") {
				filesToFormat = append(filesToFormat, trimmed)
				filesChecked++
			}
		}
	}

	if len(filesToFormat) == 0 {
		return "✓ Prettier: All files formatted correctly"
	}

	if len(filesToFormat) > 0 {
		var result strings.Builder
		result.WriteString(fmt.Sprintf("Prettier: %d files need formatting\n", len(filesToFormat)))
		result.WriteString("═══════════════════════════════════════\n")

		for i, file := range filesToFormat {
			if i >= 10 {
				result.WriteString(fmt.Sprintf("\n... +%d more files\n", len(filesToFormat)-10))
				break
			}
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, compactPath(file)))
		}

		if filesChecked > len(filesToFormat) {
			result.WriteString(fmt.Sprintf("\n✓ %d files already formatted\n", filesChecked-len(filesToFormat)))
		}

		result.WriteString("\n💡 Run `prettier --write .` to format these files\n")
		return result.String()
	}

	if strings.Contains(strings.ToLower(output), "modified") || strings.Contains(strings.ToLower(output), "formatted") {
		return "✓ Prettier: Files formatted"
	}

	return "✓ Format: All files formatted correctly"
}

func packageManagerExec(cmd string) *exec.Cmd {
	npxCmd := exec.Command("npx", cmd)
	if _, err := exec.LookPath("npx"); err == nil {
		return npxCmd
	}
	pnpmCmd := exec.Command("pnpm", "exec", cmd)
	if _, err := exec.LookPath("pnpm"); err == nil {
		return pnpmCmd
	}
	return exec.Command(cmd)
}
