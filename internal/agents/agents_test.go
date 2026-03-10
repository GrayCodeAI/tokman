package agents

import (
	"testing"
)

func TestDetectAll(t *testing.T) {
	statuses := DetectAll()

	if len(statuses) == 0 {
		t.Error("Expected at least one agent status")
	}

	// Check that all agents have required fields
	for _, s := range statuses {
		if s.Name == "" {
			t.Error("Agent status missing name")
		}
		// Installed should be true or false, not undefined
		if s.Installed && s.ErrorMessage != "" {
			t.Errorf("Agent %s marked as installed but has error: %s", s.Name, s.ErrorMessage)
		}
	}
}

func TestGetAgent(t *testing.T) {
	// Test getting known agent
	agent := GetAgent("claude-code")
	if agent == nil {
		t.Error("Expected to find claude-code agent")
	}
	if agent.DisplayName != "Claude Code" {
		t.Errorf("Expected display name 'Claude Code', got '%s'", agent.DisplayName)
	}

	// Test getting unknown agent
	agent = GetAgent("unknown-agent")
	if agent != nil {
		t.Error("Expected nil for unknown agent")
	}
}

func TestGetAgentBinaryPath(t *testing.T) {
	// This tests the function exists and doesn't panic
	// The actual result depends on what's installed
	path := GetAgentBinaryPath("claude-code")
	// Path can be empty if claude isn't installed
	_ = path
}

func TestInstallInstructions(t *testing.T) {
	// Test known agent
	inst := InstallInstructions("aider")
	if inst == "" {
		t.Error("Expected installation instructions for aider")
	}
	if inst == "No installation instructions available." {
		t.Error("Expected specific instructions for aider")
	}

	// Test unknown agent
	inst = InstallInstructions("unknown")
	if inst != "No installation instructions available." {
		t.Error("Expected 'No installation instructions available' for unknown agent")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"~/.claude", ".claude"},
		{"/absolute/path", "/absolute/path"},
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if tt.input[0] == '~' {
			// Should have expanded ~ to home directory
			if result[0] != '/' {
				t.Errorf("Path not expanded: %s", result)
			}
		}
	}
}

func TestConfigExists(t *testing.T) {
	// Test with non-existent path
	if configExists("/nonexistent/path/config.json") {
		t.Error("Expected false for non-existent path")
	}
}

func TestAllAgentsList(t *testing.T) {
	if len(AllAgents) < 6 {
		t.Errorf("Expected at least 6 agents, got %d", len(AllAgents))
	}

	// Check each agent has required fields
	for _, agent := range AllAgents {
		if agent.Name == "" {
			t.Error("Agent missing name")
		}
		if agent.DisplayName == "" {
			t.Errorf("Agent %s missing display name", agent.Name)
		}
		if agent.DetectFunc == nil {
			t.Errorf("Agent %s missing detect function", agent.Name)
		}
		if agent.StatusFunc == nil {
			t.Errorf("Agent %s missing status function", agent.Name)
		}
	}
}

func TestSetupAgentValidation(t *testing.T) {
	// Test setup for unknown agent
	err := SetupAgent("unknown-agent")
	if err == nil {
		t.Error("Expected error for unknown agent")
	}
}
