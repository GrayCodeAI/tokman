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

	// AllAgents is the list of supported agents
	AllAgents = []Agent{ClaudeCode, Cursor, Cline, Continue, Aider, CodexCLI}
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

// === Setup Functions ===

func setupClaudeCode() error {
	// Claude Code setup is handled by ccusage integration
	configDir := expandPath("~/.claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create settings.json with tokman integration
	settings := map[string]interface{}{
		"env": map[string]string{
			"CLAUDE_CODE_USE_TOKMAN": "true",
		},
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupCursor() error {
	configDir := expandPath("~/.cursor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	settings := map[string]interface{}{
		"cursor.ai.tokenOptimization": true,
		"cursor.ai.cacheEnabled":      true,
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupCline() error {
	configDir := expandPath("~/.vscode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	settings := map[string]interface{}{
		"cline.tokenOptimization": true,
	}
	return writeJSONConfig(filepath.Join(configDir, "settings.json"), settings)
}

func setupContinue() error {
	configDir := expandPath("~/.continue")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	config := map[string]interface{}{
		"models": []map[string]interface{}{
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
	return os.WriteFile(configPath, []byte(content), 0644)
}

func setupCodexCLI() error {
	configDir := expandPath("~/.codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	config := map[string]interface{}{
		"optimization": map[string]interface{}{
			"cacheEnabled": true,
		},
	}
	return writeJSONConfig(filepath.Join(configDir, "config.json"), config)
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

// === Utility Functions ===

func configExists(path string) bool {
	expanded := expandPath(path)
	_, err := os.Stat(expanded)
	return err == nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

func writeJSONConfig(path string, data interface{}) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
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
	}

	if inst, ok := instructions[name]; ok {
		return inst
	}
	return "No installation instructions available."
}

// GetAgentBinaryPath returns the path to an agent's binary
func GetAgentBinaryPath(name string) string {
	binaries := map[string]string{
		"claude-code": "claude",
		"cursor":      "cursor",
		"aider":       "aider",
		"codex-cli":   "codex",
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
	home, _ := os.UserHomeDir()

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
