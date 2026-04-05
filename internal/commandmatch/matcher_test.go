package commandmatch

import "testing"

func TestExtractBasename_Method(t *testing.T) {
	m := NewCommandMatcher()

	tests := []struct{ input, want string }{
		{"git status", "git"},
		{"/usr/bin/grep", "/usr/bin/grep"}, // ExtractBasename returns first field, not basename
		{"npm run test", "npm"},
		{"", ""},
		{"single", "single"},
	}

	for _, tt := range tests {
		got := m.ExtractBasename(tt.input)
		if got != tt.want {
			t.Errorf("ExtractBasename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripFlags_Method(t *testing.T) {
	m := NewCommandMatcher()

	tests := []struct{ input, want string }{
		{"git status --short", "git status"},
		{"npm install --save-dev", "npm install"},
		{"tokman -v --json", "tokman"},
		{"ls -la", "ls"},
		{"tokman", "tokman"},
	}

	for _, tt := range tests {
		got := m.StripFlags(tt.input)
		if got != tt.want {
			t.Errorf("StripFlags(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCommandMatcher_Match(t *testing.T) {
	m := NewCommandMatcher()

	// Basic match - command matcher has built-in rules
	got := m.Match("git status")
	if got == nil {
		t.Error("Match should return a rule for 'git status'")
	}
}

func TestBranchingEngine(t *testing.T) {
	be := NewBranchingEngine()
	if be == nil {
		t.Fatal("NewBranchingEngine returned nil")
	}
}

func TestBranchingEngine_AddSuccess(t *testing.T) {
	be := NewBranchingEngine()
	be.AddSuccess("git status", "handler1")
	be.AddFailure("git status", "handler2")
}
