package commands

import (
	"strings"
	"testing"
)

func TestFilterErrorsAdvanced(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string // substrings that should be in output
	}{
		{
			name:  "Rust error",
			input: "Compiling foo v0.1.0\nerror[E0308]: mismatched types\n  --> src/main.rs:10:5",
			want:  []string{"error[E0308]"},
		},
		{
			name:  "Go error",
			input: "src/main.go:10:5: undefined: foo",
			want:  []string{"undefined: foo"},
		},
		{
			name:  "Python error",
			input: "  File \"script.py\", line 10\n    foo\nNameError: name 'foo' is not defined",
			want:  []string{"NameError"},
		},
		{
			name:  "warning",
			input: "warning: unused variable `x`",
			want:  []string{"warning"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterErrorsAdvanced(tt.input, false)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("filterErrorsAdvanced(%q) missing %q, got:\n%s", tt.name, w, got)
				}
			}
		})
	}
}

func TestFilterErrorsEmpty(t *testing.T) {
	got := filterErrorsAdvanced("", false)
	_ = got // just check no panic
}
