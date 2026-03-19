package toml

import (
	"testing"
)

func TestParseContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		filters int
	}{
		{
			name: "simple filter",
			content: `schema_version = 1

[docker_ps]
match_command = "^docker(\\s+ps)?(\\s+.*)?$"
strip_ansi = true
max_lines = 50
`,
			wantErr: false,
			filters: 1,
		},
		{
			name: "multiple filters",
			content: `schema_version = 1

[npm_install]
match_command = "^npm(\\s+install)?(\\s+.*)?$"
strip_ansi = true
on_empty = "Packages installed"

[npm_test]
match_command = "^npm(\\s+test)?(\\s+.*)?$"
strip_ansi = true
head = 20
tail = 20
`,
			wantErr: false,
			filters: 2,
		},
		{
			name: "filter with replace rules",
			content: `schema_version = 1

[terraform_plan]
match_command = "^terraform(\\s+plan)?(\\s+.*)?$"
strip_ansi = true
[[replace]]
pattern = "^\\x1b\\[.*m"
replacement = ""

[[strip_lines_matching]]
pattern = "^Initializing"
`,
			wantErr: false,
			filters: 1,
		},
		{
			name: "invalid schema version",
			content: `schema_version = 2

[test]
match_command = "^test$"
`,
			wantErr: true,
		},
		{
			name: "missing match_command",
			content: `schema_version = 1

[test]
strip_ansi = true
`,
			wantErr: true,
		},
		{
			name: "invalid regex",
			content: `schema_version = 1

[test]
match_command = "[invalid(regex"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			filter, err := parser.ParseContent([]byte(tt.content), "test.toml")

			if tt.wantErr {
				if err == nil {
					// Check validation errors too
					if filter != nil {
						if verr := filter.Validate(); verr != nil {
							return // Validation error is expected
						}
					}
					t.Errorf("ParseContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseContent() error = %v", err)
				return
			}

			// Validate the filter
			if err := filter.Validate(); err != nil {
				t.Errorf("Validate() error = %v", err)
				return
			}

			if len(filter.Filters) != tt.filters {
				t.Errorf("ParseContent() got %d filters, want %d", len(filter.Filters), tt.filters)
			}
		})
	}
}

func TestMatchesCommand(t *testing.T) {
	content := `schema_version = 1

[docker_ps]
match_command = "^docker(\\s+ps)?(\\s+.*)?$"

[npm_test]
match_command = "^npm(\\s+test)?(\\s+.*)?$"

[cargo_build]
match_command = "^cargo(\\s+build)?(\\s+.*)?$"
`
	parser := NewParser()
	filter, err := parser.ParseContent([]byte(content), "test.toml")
	if err != nil {
		t.Fatalf("ParseContent() error = %v", err)
	}

	tests := []struct {
		command string
		matches bool
		name    string
	}{
		{"docker ps", true, "docker_ps"},
		{"docker ps -a", true, "docker_ps"},
		{"npm test", true, "npm_test"},
		{"npm test -- --verbose", true, "npm_test"},
		{"cargo build", true, "cargo_build"},
		{"cargo build --release", true, "cargo_build"},
		{"git status", false, ""},
		{"python script.py", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			name, cfg, err := filter.MatchesCommand(tt.command)
			if err != nil {
				t.Errorf("MatchesCommand() error = %v", err)
				return
			}

			if tt.matches {
				if cfg == nil {
					t.Errorf("MatchesCommand(%q) expected match, got none", tt.command)
					return
				}
				if name != tt.name {
					t.Errorf("MatchesCommand(%q) got name %q, want %q", tt.command, name, tt.name)
				}
			} else {
				if cfg != nil {
					t.Errorf("MatchesCommand(%q) expected no match, got %s", tt.command, name)
				}
			}
		})
	}
}

func TestFilterRegistry(t *testing.T) {
	registry := NewFilterRegistry()

	// Test empty registry
	if registry.Count() != 0 {
		t.Errorf("New registry should be empty, got %d filters", registry.Count())
	}

	// Test finding matching filter (empty)
	_, _, cfg := registry.FindMatchingFilter("docker ps")
	if cfg != nil {
		t.Errorf("Empty registry should not match any filter")
	}
}
