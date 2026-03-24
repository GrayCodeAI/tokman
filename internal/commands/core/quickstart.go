package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var quickstartAll bool

// quickstartCmd provides one-command setup for TokMan
var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "One-command setup for TokMan",
	Long: `Automatically detect installed AI agents, install hooks,
apply sensible defaults, and verify the setup works.

This is the fastest way to get started with TokMan:
  - Detects Claude Code, Cursor, Windsurf, Cline, etc.
  - Installs appropriate hooks for each agent
  - Creates default configuration
  - Runs doctor to verify setup`,
	RunE: runQuickstart,
}

func init() {
	quickstartCmd.Flags().BoolVarP(&quickstartAll, "all", "a", false, "setup for all detected agents")
	registry.Add(func() { registry.Register(quickstartCmd) })
}

type agentInfo struct {
	Name        string
	ConfigPath  string
	HookPath    string
	Detected    bool
	Configured  bool
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	if shared.IsVerbose() {
		fmt.Println("TokMan Quickstart")
		fmt.Println("=================")
		fmt.Println()
	}

	// Step 1: Detect agents
	fmt.Println("🔍 Detecting AI agents...")
	agents := detectAgents()
	
	detectedCount := 0
	for _, agent := range agents {
		if agent.Detected {
			detectedCount++
			fmt.Printf("   ✓ %s detected\n", agent.Name)
		}
	}
	
	if detectedCount == 0 {
		fmt.Println("   ℹ No AI agents detected in standard locations")
		fmt.Println()
		fmt.Println("You can manually run:")
		fmt.Println("  tokman init --claude     # For Claude Code")
		fmt.Println("  tokman init --cursor     # For Cursor")
		fmt.Println("  tokman init --windsurf   # For Windsurf")
		return nil
	}
	fmt.Println()

	// Step 2: Install hooks
	fmt.Println("📦 Installing hooks...")
	installedCount := 0
	for _, agent := range agents {
		if agent.Detected {
			if quickstartAll || detectedCount == 1 {
				if err := installHookForAgent(agent); err != nil {
					fmt.Printf("   ✗ %s: %v\n", agent.Name, err)
				} else {
					installedCount++
					fmt.Printf("   ✓ %s hook installed\n", agent.Name)
				}
			}
		}
	}
	
	if installedCount == 0 && detectedCount > 0 && !quickstartAll {
		fmt.Println("   ℹ Run 'tokman quickstart --all' to install hooks for all detected agents")
	}
	fmt.Println()

	// Step 3: Create default config
	fmt.Println("⚙️  Setting up configuration...")
	if err := createDefaultConfig(); err != nil {
		fmt.Printf("   ✗ Config setup failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Default configuration applied")
	}
	fmt.Println()

	// Step 4: Run doctor
	fmt.Println("🏥 Running diagnostics...")
	doctorCmd := exec.Command("tokman", "doctor")
	doctorCmd.Stdout = os.Stdout
	doctorCmd.Stderr = os.Stderr
	if err := doctorCmd.Run(); err != nil {
		fmt.Println()
		fmt.Println("⚠️  Some issues detected. See above for details.")
		return nil
	}
	
	fmt.Println()
	fmt.Println("🎉 Quickstart complete!")
	fmt.Println()
	fmt.Println("TokMan is now active and will compress CLI output automatically.")
	fmt.Println()
	fmt.Println("Quick commands:")
	fmt.Println("  tokman status          # View current stats")
	fmt.Println("  tokman gain            # See token savings")
	fmt.Println("  tokman discover        # Find optimization opportunities")
	return nil
}

func detectAgents() []agentInfo {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	agents := []agentInfo{
		{
			Name:       "Claude Code",
			ConfigPath: home + "/.claude/settings.json",
			HookPath:   home + "/.claude/hooks/tokman.sh",
		},
		{
			Name:       "Cursor",
			ConfigPath: home + "/.cursor/rules/tokman",
			HookPath:   home + "/.cursor/hooks/tokman.sh",
		},
		{
			Name:       "Windsurf",
			ConfigPath: home + "/.windsurf/settings.json",
			HookPath:   home + "/.windsurf/hooks/tokman.sh",
		},
		{
			Name:       "Cline",
			ConfigPath: home + "/.cline/config.json",
			HookPath:   home + "/.cline/hooks/tokman.sh",
		},
		{
			Name:       "OpenCode",
			ConfigPath: home + "/.opencode/config.json",
			HookPath:   home + "/.opencode/hooks/tokman.sh",
		},
		{
			Name:       "OpenClaw",
			ConfigPath: home + "/.openclaw/config.json",
			HookPath:   home + "/.openclaw/hooks/tokman.sh",
		},
	}

	for i := range agents {
		if _, err := os.Stat(agents[i].ConfigPath); err == nil {
			agents[i].Detected = true
		}
		// Check if hook already exists
		if _, err := os.Stat(agents[i].HookPath); err == nil {
			agents[i].Configured = true
		}
	}

	return agents
}

func installHookForAgent(agent agentInfo) error {
	// Create hooks directory if needed
	hookDir := filepath.Dir(agent.HookPath)
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return fmt.Errorf("cannot create hooks directory: %w", err)
	}

	// Generate hook script
	hookScript := generateHookScript()
	if err := os.WriteFile(agent.HookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("cannot write hook: %w", err)
	}

	return nil
}

func generateHookScript() string {
	return `#!/bin/bash
# TokMan compression hook
# Auto-generated by 'tokman quickstart'

# Pass command through tokman for compression
exec tokman "$@"
`
}

func createDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "tokman")
	configPath := filepath.Join(configDir, "config.toml")

	// Create directory if needed
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Already exists, don't overwrite
	}

	// Write default config
	defaultConfig := `# TokMan Configuration
# Auto-generated by 'tokman quickstart'

[tracking]
enabled = true

[filter]
# Preset: fast, balanced, full, or auto
preset = "auto"

[pipeline]
# Maximum context tokens
max_context_tokens = 2000000
`
	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}
