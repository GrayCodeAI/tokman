// Package agents provides integration with various AI coding agents.
// Supports Claude Code, Cursor, Cline, and other AI assistants.
package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Agent represents an AI coding agent integration
type Agent struct {
	Name        string
	DisplayName string
	ConfigPath  string
	BinaryName  string
	DetectFunc  func() bool
	SetupFunc   func() error
	StatusFunc  func() (*AgentStatus, error)
}

// AgentStatus represents the current status of an agent integration
type AgentStatus struct {
	Name         string `json:"name"`
	Installed    bool   `json:"installed"`
	Configured   bool   `json:"configured"`
	Version      string `json:"version,omitempty"`
	ConfigPath   string `json:"config_path,omitempty"`
	TokenUsage   int64  `json:"token_usage,omitempty"`
	LastActive   string `json:"last_active,omitempty"`
	ErrorMessage string `json:"error,omitempty"`
}

// Known agents
var (
	// Claude Code - uses ccusage for tracking
	ClaudeCode = Agent{
		Name:        "claude-code",
		DisplayName: "Claude Code",
		BinaryName:  "claude",
		ConfigPath:  "~/.claude/settings.json",
		DetectFunc:  detectClaudeCode,
		SetupFunc:   setupClaudeCode,
		StatusFunc:  statusClaudeCode,
	}

	// Cursor - VS Code fork with AI
	Cursor = Agent{
		Name:        "cursor",
		DisplayName: "Cursor",
		BinaryName:  "cursor",
		ConfigPath:  "~/.cursor/settings.json",
		DetectFunc:  detectCursor,
		SetupFunc:   setupCursor,
		StatusFunc:  statusCursor,
	}

	// Cline - VS Code extension
	Cline = Agent{
		Name:        "cline",
		DisplayName: "Cline",
		BinaryName:  "code", // Uses VS Code
		ConfigPath:  "~/.vscode/settings.json",
		DetectFunc:  detectCline,
		SetupFunc:   setupCline,
		StatusFunc:  statusCline,
	}

	// Continue - open source AI assistant
	Continue = Agent{
		Name:        "continue",
		DisplayName: "Continue",
		BinaryName:  "continue",
		ConfigPath:  "~/.continue/config.json",
		DetectFunc:  detectContinue,
		SetupFunc:   setupContinue,
		StatusFunc:  statusContinue,
	}

	// Aider - terminal-based AI pair programmer
	Aider = Agent{
		Name:        "aider",
		DisplayName: "Aider",
		BinaryName:  "aider",
		ConfigPath:  "~/.aider.conf.yml",
		DetectFunc:  detectAider,
		SetupFunc:   setupAider,
		StatusFunc:  statusAider,
	}

	// Codex CLI - OpenAI's coding agent
	CodexCLI = Agent{
		Name:        "codex-cli",
		DisplayName: "Codex CLI",
		BinaryName:  "codex",
		ConfigPath:  "~/.codex/config.json",
		DetectFunc:  detectCodexCLI,
		SetupFunc:   setupCodexCLI,
		StatusFunc:  statusCodexCLI,
	}

	// AdaL - SylphAI's AI Agent
	AdaL = Agent{
		Name:        "adal",
		DisplayName: "AdaL (SylphAI)",
		BinaryName:  "adal",
		ConfigPath:  "~/.adal/config",
		DetectFunc:  detectAdaL,
		SetupFunc:   setupAdaL,
		StatusFunc:  statusAdaL,
	}

	// Kiro - AI coding agent
	Kiro = Agent{
		Name:        "kiro",
		DisplayName: "Kiro",
		BinaryName:  "kiro",
		ConfigPath:  "~/.kilorc",
		DetectFunc:  detectKiro,
		SetupFunc:   setupKiro,
		StatusFunc:  statusKiro,
	}

	// Kilo Code - AI coding agent
	KiloCode = Agent{
		Name:        "kilo-code",
		DisplayName: "Kilo Code",
		BinaryName:  "kilo",
		ConfigPath:  "~/.kilorc",
		DetectFunc:  detectKiloCode,
		SetupFunc:   setupKiloCode,
		StatusFunc:  statusKiloCode,
	}

	// Windsurf - Codeium's AI IDE
	Windsurf = Agent{
		Name:        "windsurf",
		DisplayName: "Windsurf",
		BinaryName:  "windsurf",
		ConfigPath:  "~/.windsurf/settings.json",
		DetectFunc:  detectWindsurf,
		SetupFunc:   setupWindsurf,
		StatusFunc:  statusWindsurf,
	}

	// Replit Agent - Cloud-based AI agent
	ReplitAgent = Agent{
		Name:        "replit-agent",
		DisplayName: "Replit Agent",
		BinaryName:  "replit",
		ConfigPath:  ".replit",
		DetectFunc:  detectReplitAgent,
		SetupFunc:   setupReplitAgent,
		StatusFunc:  statusReplitAgent,
	}

	// Gemini CLI - Google's Gemini CLI
	GeminiCLI = Agent{
		Name:        "gemini-cli",
		DisplayName: "Gemini CLI",
		BinaryName:  "gemini",
		ConfigPath:  "~/.gemini/settings.json",
		DetectFunc:  detectGeminiCLI,
		SetupFunc:   setupGeminiCLI,
		StatusFunc:  statusGeminiCLI,
	}

	// OpenCode - Open source AI coding agent
	OpenCode = Agent{
		Name:        "opencode",
		DisplayName: "OpenCode",
		BinaryName:  "opencode",
		ConfigPath:  "~/.config/opencode/config.toml",
		DetectFunc:  detectOpenCode,
		SetupFunc:   setupOpenCode,
		StatusFunc:  statusOpenCode,
	}

	// AllAgents is the list of supported agents
	AllAgents = []Agent{
		ClaudeCode, Cursor, Cline, Continue, Aider, CodexCLI,
		AdaL, Kiro, KiloCode, Windsurf, ReplitAgent, GeminiCLI, OpenCode,
	}
)

// === Detection Functions ===

func detectClaudeCode() bool {
	// Check for claude binary
	if _, err := exec.LookPath("claude"); err == nil {
		return true
	}
	// Check for ccusage (Claude Code usage tracker)
	if _, err := exec.LookPath("ccusage"); err == nil {
		return true
	}
	// Check for config
	if configExists("~/.claude/settings.json") {
		return true
	}
	return false
}

func detectCursor() bool {
	// Check for cursor binary
	if _, err := exec.LookPath("cursor"); err == nil {
		return true
	}
	// Check for AppImage on Linux
	if _, err := os.Stat("/opt/Cursor/cursor"); err == nil {
		return true
	}
	// Check for config
	if configExists("~/.cursor/settings.json") {
		return true
	}
	return false
}

func detectCline() bool {
	// Cline is a VS Code extension
	configPaths := []string{
		"~/.vscode/settings.json",
		"~/.config/Code/User/settings.json",
		"~/.config/VSCodium/User/settings.json",
	}
	for _, p := range configPaths {
		if configExists(p) {
			// Check if cline extension is installed
			data, err := os.ReadFile(expandPath(p))
			if err == nil && strings.Contains(string(data), "cline") {
				return true
			}
		}
	}
	return false
}

func detectContinue() bool {
	if configExists("~/.continue/config.json") {
		return true
	}
	// Check VS Code extensions for Continue
	return checkVSCodeExtension("continue")
}

func detectAider() bool {
	if _, err := exec.LookPath("aider"); err == nil {
		return true
	}
	return configExists("~/.aider.conf.yml")
}

func detectCodexCLI() bool {
	if _, err := exec.LookPath("codex"); err == nil {
		return true
	}
	return configExists("~/.codex/config.json")
}

func detectAdaL() bool {
	// Check for adal binary
	if _, err := exec.LookPath("adal"); err == nil {
		return true
	}
	// Check for ADAL_SESSION_ID env var
	if os.Getenv("ADAL_SESSION_ID") != "" {
		return true
	}
	// Check for config directory
	return configExists("~/.adal/config")
}

func detectKiro() bool {
	if _, err := exec.LookPath("kiro"); err == nil {
		return true
	}
	return configExists("~/.kilorc")
}

func detectKiloCode() bool {
	if _, err := exec.LookPath("kilo"); err == nil {
		return true
	}
	return configExists("~/.kilorc")
}

func detectWindsurf() bool {
	// Check for windsurf binary
	if _, err := exec.LookPath("windsurf"); err == nil {
		return true
	}
	// Check for Windsurf config directory
	if runtime.GOOS == "darwin" {
		return configExists("~/Library/Application Support/Windsurf/User/settings.json")
	}
	return configExists("~/.config/windsurf/User/settings.json")
}

func detectReplitAgent() bool {
	// Replit Agent runs in the cloud, check for .replit file
	return configExists(".replit")
}

func detectGeminiCLI() bool {
	if _, err := exec.LookPath("gemini"); err == nil {
		return true
	}
	return configExists("~/.gemini/settings.json")
}

func detectOpenCode() bool {
	if _, err := exec.LookPath("opencode"); err == nil {
		return true
	}
	return configExists("~/.config/opencode/config.toml")
}

// === Setup Functions ===

func setupClaudeCode() error {
	// Claude Code setup is handled by ccusage integration
	configDir := expandPath("~/.claude")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	// Create settings.json with tokman integration
	settings := map[string]any{
		"env": map[string]string{
			"CLAUDE_CODE_USE_TOKMAN": "true",
		},
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupCursor() error {
	configDir := expandPath("~/.cursor")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	settings := map[string]any{
		"cursor.ai.tokenOptimization": true,
		"cursor.ai.cacheEnabled":      true,
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupCline() error {
	configDir := expandPath("~/.vscode")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	settings := map[string]any{
		"cline.tokenOptimization": true,
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupContinue() error {
	configDir := expandPath("~/.continue")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	config := map[string]any{
		"models": []map[string]any{
			{
				"title":    "TokMan Optimized",
				"provider": "anthropic",
				"cache":    true,
			},
		},
	}
	return writeJSONConfig(filepath.Join(configDir, "config.json"), config)
}

func setupAider() error {
	configPath := expandPath("~/.aider.conf.yml")
	content := `# Aider configuration with TokMan optimization
cache-prompts: true
map-tokens: 2048
`
	return os.WriteFile(configPath, []byte(content), 0600)
}

func setupCodexCLI() error {
	configDir := expandPath("~/.codex")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	config := map[string]any{
		"optimization": map[string]any{
			"cacheEnabled": true,
		},
	}
	return writeJSONConfig(filepath.Join(configDir, "config.json"), config)
}

func setupAdaL() error {
	configDir := expandPath("~/.adal")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	// AdaL uses YAML config with MCP tool support
	config := `# AdaL Configuration with TokMan Integration
# Token optimization enabled via tokman

mcp:
  servers:
    tokman:
      command: tokman
      args: ["mcp", "serve"]
      tools:
        - compress
        - status
        - suggest

optimization:
  token_reduction: true
  cache_enabled: true
`
	return os.WriteFile(filepath.Join(configDir, "config"), []byte(config), 0600)
}

func setupKiro() error {
	// Kiro uses YAML config with lifecycle hooks
	config := `# Kiro Configuration with TokMan Integration

hooks:
  preToolUse:
    - matcher: "Bash"
      command: "tokman rewrite"
  
optimization:
  cacheEnabled: true
  tokenBudget: 4000
`
	return os.WriteFile(expandPath("~/.kilorc"), []byte(config), 0600)
}

func setupKiloCode() error {
	// Kilo Code uses similar config to Kiro
	config := `# Kilo Code Configuration with TokMan Integration

hooks:
  preToolUse:
    - matcher: "Bash"
      command: "tokman rewrite"
  
optimization:
  cacheEnabled: true
  tokenBudget: 4000
`
	return os.WriteFile(expandPath("~/.kilorc"), []byte(config), 0600)
}

func setupWindsurf() error {
	var configDir string
	if runtime.GOOS == "darwin" {
		configDir = expandPath("~/Library/Application Support/Windsurf/User")
	} else {
		configDir = expandPath("~/.config/windsurf/User")
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	settings := map[string]any{
		"windsurf.ai.tokenOptimization": true,
		"windsurf.ai.cacheEnabled":      true,
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupReplitAgent() error {
	// Replit Agent uses .replit file
	config := `# Replit Configuration with TokMan Integration

[agent]
token_optimization = true

[env]
TOKMAN_ENABLED = "true"
`
	return os.WriteFile(".replit", []byte(config), 0600)
}

func setupGeminiCLI() error {
	configDir := expandPath("~/.gemini")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	// Gemini CLI uses hooks in settings.json
	settings := map[string]any{
		"hooks": map[string]any{
			"BeforeTool": []any{
				map[string]any{
					"matcher": "run_shell_command",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "tokman hook gemini",
						},
					},
				},
			},
		},
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupOpenCode() error {
	configDir := expandPath("~/.config/opencode")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	// OpenCode uses TOML config
	config := `# OpenCode Configuration with TokMan Integration

[tools.shell]
command = "tokman proxy"

[plugins]
tokman = { enabled = true, path = "~/.config/opencode/plugins/tokman.ts" }

[optimization]
cache_enabled = true
token_budget = 4000
`
	return os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(config), 0600)
}

// === Status Functions ===

func statusClaudeCode() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "claude-code",
		Installed:  detectClaudeCode(),
		ConfigPath: expandPath("~/.claude/settings.json"),
	}

	if !status.Installed {
		return status, nil
	}

	// Get version
	if output, err := exec.Command("claude", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	// Check configuration
	status.Configured = configExists("~/.claude/settings.json")

	return status, nil
}

func statusCursor() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "cursor",
		Installed:  detectCursor(),
		ConfigPath: expandPath("~/.cursor/settings.json"),
	}

	if status.Installed {
		status.Configured = configExists("~/.cursor/settings.json")
	}

	return status, nil
}

func statusCline() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "cline",
		Installed:  detectCline(),
		ConfigPath: expandPath("~/.vscode/settings.json"),
	}

	if status.Installed {
		status.Configured = true
	}

	return status, nil
}

func statusContinue() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "continue",
		Installed:  detectContinue(),
		ConfigPath: expandPath("~/.continue/config.json"),
	}

	if status.Installed {
		status.Configured = configExists("~/.continue/config.json")
	}

	return status, nil
}

func statusAider() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "aider",
		Installed:  detectAider(),
		ConfigPath: expandPath("~/.aider.conf.yml"),
	}

	if !status.Installed {
		return status, nil
	}

	// Get version
	if output, err := exec.Command("aider", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	status.Configured = configExists("~/.aider.conf.yml")

	return status, nil
}

func statusCodexCLI() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "codex-cli",
		Installed:  detectCodexCLI(),
		ConfigPath: expandPath("~/.codex/config.json"),
	}

	if !status.Installed {
		return status, nil
	}

	if output, err := exec.Command("codex", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	status.Configured = configExists("~/.codex/config.json")

	return status, nil
}

func statusAdaL() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "adal",
		Installed:  detectAdaL(),
		ConfigPath: expandPath("~/.adal/config"),
	}

	if !status.Installed {
		return status, nil
	}

	// Check for ADAL_SESSION_ID to confirm active session
	if os.Getenv("ADAL_SESSION_ID") != "" {
		status.LastActive = "current session"
	}

	// Get version
	if output, err := exec.Command("adal", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	status.Configured = configExists("~/.adal/config")

	return status, nil
}

func statusKiro() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "kiro",
		Installed:  detectKiro(),
		ConfigPath: expandPath("~/.kilorc"),
	}

	if !status.Installed {
		return status, nil
	}

	if output, err := exec.Command("kiro", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	status.Configured = configExists("~/.kilorc")

	return status, nil
}

func statusKiloCode() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "kilo-code",
		Installed:  detectKiloCode(),
		ConfigPath: expandPath("~/.kilorc"),
	}

	if !status.Installed {
		return status, nil
	}

	if output, err := exec.Command("kilo", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	status.Configured = configExists("~/.kilorc")

	return status, nil
}

func statusWindsurf() (*AgentStatus, error) {
	var configPath string
	if runtime.GOOS == "darwin" {
		configPath = "~/Library/Application Support/Windsurf/User/settings.json"
	} else {
		configPath = "~/.config/windsurf/User/settings.json"
	}

	status := &AgentStatus{
		Name:       "windsurf",
		Installed:  detectWindsurf(),
		ConfigPath: expandPath(configPath),
	}

	if status.Installed {
		status.Configured = configExists(configPath)
	}

	return status, nil
}

func statusReplitAgent() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "replit-agent",
		Installed:  detectReplitAgent(),
		ConfigPath: ".replit",
	}

	if status.Installed {
		status.Configured = configExists(".replit")
	}

	return status, nil
}

func statusGeminiCLI() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "gemini-cli",
		Installed:  detectGeminiCLI(),
		ConfigPath: expandPath("~/.gemini/settings.json"),
	}

	if !status.Installed {
		return status, nil
	}

	if output, err := exec.Command("gemini", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	status.Configured = configExists("~/.gemini/settings.json")

	return status, nil
}

func statusOpenCode() (*AgentStatus, error) {
	status := &AgentStatus{
		Name:       "opencode",
		Installed:  detectOpenCode(),
		ConfigPath: expandPath("~/.config/opencode/config.toml"),
	}

	if !status.Installed {
		return status, nil
	}

	if output, err := exec.Command("opencode", "--version").Output(); err == nil {
		status.Version = strings.TrimSpace(string(output))
	}

	status.Configured = configExists("~/.config/opencode/config.toml")

	return status, nil
}

// === Utility Functions ===

func configExists(path string) bool {
	expanded := expandPath(path)
	_, err := os.Stat(expanded)
	return err == nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func writeJSONConfig(path string, data any) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0600)
}

func checkVSCodeExtension(name string) bool {
	// Check VS Code extension directory
	extDirs := []string{
		filepath.Join(os.Getenv("HOME"), ".vscode", "extensions"),
		filepath.Join(os.Getenv("HOME"), ".config", "Code", "User", "extensions"),
	}

	for _, dir := range extDirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "*"+name+"*"))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// DetectAll returns status of all known agents
func DetectAll() []*AgentStatus {
	var statuses []*AgentStatus
	for _, agent := range AllAgents {
		if agent.StatusFunc != nil {
			status, err := agent.StatusFunc()
			if err != nil {
				status = &AgentStatus{
					Name:         agent.Name,
					Installed:    false,
					ErrorMessage: err.Error(),
				}
			}
			statuses = append(statuses, status)
		}
	}
	return statuses
}

// SetupAgent configures a specific agent for TokMan integration
func SetupAgent(name string) error {
	for _, agent := range AllAgents {
		if agent.Name == name && agent.SetupFunc != nil {
			return agent.SetupFunc()
		}
	}
	return fmt.Errorf("unknown agent: %s", name)
}

// GetAgent returns an agent by name
func GetAgent(name string) *Agent {
	for i := range AllAgents {
		if AllAgents[i].Name == name {
			return &AllAgents[i]
		}
	}
	return nil
}

// InstallInstructions returns installation instructions for an agent
func InstallInstructions(name string) string {
	instructions := map[string]string{
		"claude-code": `# Install Claude Code
npm install -g @anthropic/claude-code

# Or download from: https://claude.ai/code`,
		"cursor": `# Install Cursor
# Download from: https://cursor.sh

# On Linux:
wget https://download.cursor.sh/linux/appImage/x64 -O cursor.AppImage
chmod +x cursor.AppImage`,
		"cline": `# Install Cline
# Open VS Code and install the Cline extension
# Or: code --install-extension saoudrizwan.claude-dev`,
		"continue": `# Install Continue
# VS Code: code --install-extension continue.continue
# Or download from: https://continue.dev`,
		"aider": `# Install Aider
pip install aider-chat

# Or with pipx:
pipx install aider-chat`,
		"codex-cli": `# Install Codex CLI (if available)
# Check OpenAI's documentation for installation instructions`,
		"adal": `# Install AdaL (SylphAI's AI Agent)
pip install adal-cli

# Or with pipx:
pipx install adal-cli

# Configure:
adal config --init`,
		"kiro": `# Install Kiro
# Check the official Kiro documentation for installation instructions
# Typically: npm install -g @kiro/cli`,
		"kilo-code": `# Install Kilo Code
# Check the official Kilo Code documentation for installation instructions`,
		"windsurf": `# Install Windsurf
# Download from: https://codeium.com/windsurf

# On Linux:
wget https://windsurf.sh/download/linux -O windsurf.AppImage
chmod +x windsurf.AppImage`,
		"replit-agent": `# Replit Agent is cloud-based
# No local installation needed
# Create a Replit project and enable the Agent feature`,
		"gemini-cli": `# Install Gemini CLI
npm install -g @anthropic/gemini-cli

# Or download from Google's official source`,
		"opencode": `# Install OpenCode
go install github.com/opencode-ai/opencode@latest

# Or:
npm install -g opencode-cli`,
	}

	if inst, ok := instructions[name]; ok {
		return inst
	}
	return "No installation instructions available."
}

// GetAgentBinaryPath returns the path to an agent's binary
func GetAgentBinaryPath(name string) string {
	binaries := map[string]string{
		"claude-code":  "claude",
		"cursor":       "cursor",
		"aider":        "aider",
		"codex-cli":    "codex",
		"adal":         "adal",
		"kiro":         "kiro",
		"kilo-code":    "kilo",
		"windsurf":     "windsurf",
		"replit-agent": "replit",
		"gemini-cli":   "gemini",
		"opencode":     "opencode",
	}

	if bin, ok := binaries[name]; ok {
		if path, err := exec.LookPath(bin); err == nil {
			return path
		}
	}
	return ""
}

// GetDefaultShellRC returns the default shell RC file path
func GetDefaultShellRC() string {
	shell := os.Getenv("SHELL")
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	if strings.Contains(shell, "zsh") {
		return filepath.Join(home, ".zshrc")
	} else if strings.Contains(shell, "bash") {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".bash_profile")
		}
		return filepath.Join(home, ".bashrc")
	} else if strings.Contains(shell, "fish") {
		return filepath.Join(home, ".config", "fish", "config.fish")
	}
	return filepath.Join(home, ".bashrc")
}
