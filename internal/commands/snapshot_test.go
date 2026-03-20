package commands

import (
	"testing"
)

func TestSanitizeSnapshotName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"../etc/passwd", "passwd"},
		{"name.txt", "name"},
		{"foo/bar", "bar"},
		{"..", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeSnapshotName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSnapshotName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
