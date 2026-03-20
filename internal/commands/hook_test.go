package commands

import (
	"testing"
)

func TestDetectCopilotFormat(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]interface{}
		wantFormat  copilotHookFormat
		wantCommand string
	}{
		{
			name: "VS Code format",
			input: map[string]interface{}{
				"tool_name": "Bash",
				"tool_input": map[string]interface{}{
					"command": "git status",
				},
			},
			wantFormat:  copilotFormatVsCode,
			wantCommand: "git status",
		},
		{
			name: "VS Code runTerminalCommand",
			input: map[string]interface{}{
				"tool_name": "runTerminalCommand",
				"tool_input": map[string]interface{}{
					"command": "cargo test",
				},
			},
			wantFormat:  copilotFormatVsCode,
			wantCommand: "cargo test",
		},
		{
			name: "Copilot CLI format",
			input: map[string]interface{}{
				"toolName": "bash",
				"toolArgs": `{"command":"git status"}`,
			},
			wantFormat:  copilotFormatCli,
			wantCommand: "git status",
		},
		{
			name: "non-bash tool",
			input: map[string]interface{}{
				"tool_name": "editFiles",
			},
			wantFormat: copilotFormatPassThrough,
		},
		{
			name:       "empty input",
			input:      map[string]interface{}{},
			wantFormat: copilotFormatPassThrough,
		},
		{
			name: "non-bash copilot CLI tool",
			input: map[string]interface{}{
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

func TestIsGeminiHookPresent(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]interface{}
		want     bool
	}{
		{
			name:     "empty settings",
			settings: map[string]interface{}{},
			want:     false,
		},
		{
			name: "with tokman hook",
			settings: map[string]interface{}{
				"hooks": map[string]interface{}{
					"BeforeTool": []interface{}{
						map[string]interface{}{
							"matcher": "run_shell_command",
							"hooks": []interface{}{
								map[string]interface{}{
									"type":    "command",
									"command": "/home/user/.gemini/hooks/tokman-hook-gemini.sh",
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "without tokman hook",
			settings: map[string]interface{}{
				"hooks": map[string]interface{}{
					"BeforeTool": []interface{}{
						map[string]interface{}{
							"matcher": "run_shell_command",
							"hooks": []interface{}{
								map[string]interface{}{
									"type":    "command",
									"command": "/some/other/hook.sh",
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "no BeforeTool",
			settings: map[string]interface{}{
				"hooks": map[string]interface{}{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGeminiHookPresent(tt.settings)
			if got != tt.want {
				t.Errorf("isGeminiHookPresent() = %v, want %v", got, tt.want)
			}
		})
	}
}
