package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

const tokmanSlimAdaL = `<!-- tokman-adal-instructions v1 -->
# TokMan (Token Manager) - Token-Optimized Commands for AdaL

## Golden Rule

**Always prefix commands with ` + "`tokman`" + `** for 60-90% token savings.

## TokMan MCP Tools Available

When TokMan is configured as an MCP server, these tools are available:

- **tokman_compress** - Compress any text output for token efficiency
- **tokman_status** - Check current token savings statistics
- **tokman_suggest** - Get optimization suggestions for your workflow

## Key Commands

` + "```bash" + `
tokman git status / log / diff / show   # Git (59-80% savings)
tokman cargo build / test / clippy      # Rust (80-90%)
tokman tsc / lint / prettier            # JS/TS (70-87%)
tokman vitest / playwright              # Tests (90-99%)
tokman docker / kubectl                 # Infra (85%)
tokman ls / grep / find / tree          # Files (60-75%)
tokman gain                             # View savings stats
tokman discover                         # Find missed savings
tokman proxy <cmd>                      # Run without filtering
` + "```" + `

## Integration with AdaL

AdaL's subagent system works seamlessly with TokMan:
- Use ` + "`tokman`" + ` prefix for all bash commands
- Context gathering subagents can use tokman-wrapped reads
- Token savings compound across multi-file operations

<!-- /tokman-adal-instructions -->
`

// runAdaLInit sets up AdaL integration
func runAdaLInit(global bool, patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	var adalDir string
	if global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			return
		}
		adalDir = filepath.Join(homeDir, ".adal")
		if err := os.MkdirAll(adalDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating AdaL config directory: %v\n", err)
			return
		}
	} else {
		adalDir = "."
	}

	// 1. Create TOKMAN.md with AdaL-specific awareness
	tokmanMdPath := filepath.Join(adalDir, "TOKMAN.md")
	if err := writeIfChanged(tokmanMdPath, tokmanSlimAdaL, "TOKMAN.md"); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating TOKMAN.md: %v\n", err)
		return
	}

	// 2. Create MCP server configuration for AdaL
	if global {
		mcpConfigPath := filepath.Join(adalDir, "mcp.json")
		mcpConfig := `{
  "mcpServers": {
    "tokman": {
      "command": "tokman",
      "args": ["mcp", "serve"],
      "env": {}
    }
  }
}
`
		if err := writeIfChanged(mcpConfigPath, mcpConfig, "mcp.json"); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating MCP config: %v\n", err)
			return
		}

		// 3. Patch AdaL config file
		configPath := filepath.Join(adalDir, "config")
		patchAdaLConfig(configPath, patchMode)

		fmt.Printf("\n%s\n\n", green("AdaL integration configured (global)."))
		fmt.Printf("  TOKMAN.md:  %s\n", cyan(tokmanMdPath))
		fmt.Printf("  MCP config: %s\n", cyan(mcpConfigPath))
	} else {
		fmt.Printf("\n%s\n\n", green("AdaL integration configured (project)."))
		fmt.Printf("  TOKMAN.md: %s\n", cyan(tokmanMdPath))
	}

	fmt.Println("  Restart AdaL to apply changes.")
	fmt.Println()
}

// patchAdaLConfig patches AdaL config file with TokMan settings
func patchAdaLConfig(configPath string, patchMode PatchMode) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Read existing config or create new
	content := ""
	if data, err := os.ReadFile(configPath); err == nil {
		content = string(data)
	}

	// Check if already configured
	if strings.Contains(content, "tokman") {
		fmt.Printf("  %s AdaL config: already configured\n", green("✓"))
		return
	}

	// Handle patch mode
	if patchMode == PatchModeSkip {
		fmt.Printf("\n  Manual setup: add TokMan to %s\n", cyan(configPath))
		return
	}

	// Add TokMan configuration
	tokmanConfig := `

# TokMan Integration
mcp:
  servers:
    tokman:
      command: tokman
      args: ["mcp", "serve"]

optimization:
  token_reduction: true
`

	// Append to existing config or create new
	newContent := content + tokmanConfig

	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing AdaL config: %v\n", err)
		return
	}

	fmt.Printf("  %s AdaL config: TokMan MCP server added\n", green("✓"))
}

// uninstallAdaL removes AdaL artifacts
func uninstallAdaL() []string {
	var removed []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return removed
	}
	adalDir := filepath.Join(homeDir, ".adal")

	// Remove TOKMAN.md
	tokmanMdPath := filepath.Join(adalDir, "TOKMAN.md")
	if _, err := os.Stat(tokmanMdPath); err == nil {
		os.Remove(tokmanMdPath)
		removed = append(removed, fmt.Sprintf("TOKMAN.md: %s", tokmanMdPath))
	}

	// Remove MCP config
	mcpConfigPath := filepath.Join(adalDir, "mcp.json")
	if _, err := os.Stat(mcpConfigPath); err == nil {
		os.Remove(mcpConfigPath)
		removed = append(removed, fmt.Sprintf("MCP config: %s", mcpConfigPath))
	}

	// Remove TokMan section from config
	configPath := filepath.Join(adalDir, "config")
	if data, err := os.ReadFile(configPath); err == nil {
		content := string(data)
		if strings.Contains(content, "tokman") {
			// Remove TokMan section
			newContent := removeTokmanSection(content)
			os.WriteFile(configPath, []byte(newContent), 0644)
			removed = append(removed, "AdaL config: removed TokMan section")
		}
	}

	return removed
}

// removeTokmanSection removes TokMan configuration from AdaL config
func removeTokmanSection(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTokmanSection := false
	inMcpSection := false
	indentLevel := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect TokMan section start
		if strings.Contains(trimmed, "# TokMan Integration") {
			inTokmanSection = true
			continue
		}

		// Skip lines in TokMan section
		if inTokmanSection {
			if trimmed == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			inTokmanSection = false
		}

		// Track MCP section
		if strings.HasPrefix(trimmed, "mcp:") {
			inMcpSection = true
			indentLevel = len(line) - len(strings.TrimLeft(line, " \t"))
		}

		// Remove tokman entries from MCP
		if inMcpSection && strings.Contains(trimmed, "tokman:") {
			continue
		}
		if inMcpSection && strings.Contains(trimmed, `command: tokman`) {
			continue
		}
		if inMcpSection && strings.Contains(trimmed, `args: ["mcp", "serve"]`) {
			continue
		}

		// Reset MCP section tracking on dedent
		if inMcpSection && len(line)-len(strings.TrimLeft(line, " \t")) <= indentLevel && !strings.HasPrefix(trimmed, "servers:") && !strings.HasPrefix(trimmed, "-") {
			inMcpSection = false
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
