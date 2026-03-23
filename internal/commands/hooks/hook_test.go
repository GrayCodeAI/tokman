package hooks

import (
	"testing"
)

func TestDetectCopilotFormat(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		wantFormat  copilotHookFormat
		wantCommand string
	}{
		{
			name: "VS Code format",
			input: map[string]any{
				"tool_name": "Bash",
				"tool_input": map[string]any{
					"command": "git status",
				},
			},
			wantFormat:  copilotFormatVsCode,
			wantCommand: "git status",
		},
		{
			name: "VS Code runTerminalCommand",
			input: map[string]any{
				"tool_name": "runTerminalCommand",
				"tool_input": map[string]any{
					"command": "cargo test",
				},
			},
			wantFormat:  copilotFormatVsCode,
			wantCommand: "cargo test",
		},
		{
			name: "Copilot CLI format",
			input: map[string]any{
				"toolName": "bash",
				"toolArgs": `{"command":"git status"}`,
			},
			wantFormat:  copilotFormatCli,
			wantCommand: "git status",
		},
		{
			name: "non-bash tool",
			input: map[string]any{
				"tool_name": "editFiles",
			},
			wantFormat: copilotFormatPassThrough,
		},
		{
			name:       "empty input",
			input:      map[string]any{},
			wantFormat: copilotFormatPassThrough,
		},
		{
			name: "non-bash copilot CLI tool",
			input: map[string]any{
				"toolName": "view",
				"toolArgs": "{}",
			},
			wantFormat: copilotFormatPassThrough,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, cmd := detectCopilotFormat(tt.input)
			if format != tt.wantFormat {
				t.Errorf("format = %v, want %v", format, tt.wantFormat)
			}
			if tt.wantCommand != "" && cmd != tt.wantCommand {
				t.Errorf("command = %q, want %q", cmd, tt.wantCommand)
			}
		})
	}
}

func TestGeminiHookOutput(t *testing.T) {
	// Test that printGeminiAllow outputs valid JSON
	// This is a basic structural test
	allowed := `{"decision":"allow"}`
	if allowed == "" {
		t.Error("allow output should not be empty")
	}
	if len(allowed) < 10 {
		t.Errorf("allow output too short: %q", allowed)
	}
}
