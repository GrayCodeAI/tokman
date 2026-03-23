package vcs

import (
	"testing"
)

func TestFilterDiff(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple diff",
			input: "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -1,3 +1,4 @@\n line1\n+new line\n line2\n line3\n",
		},
		{
			name:  "empty diff",
			input: "",
		},
		{
			name: "large diff",
			input: func() string {
				s := "diff --git a/big.txt b/big.txt\n"
				for i := 0; i < 1000; i++ {
					s += "+line\n"
				}
				return s
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// filterDiff may truncate or reformat, just check no panic
			got := filterDiff(tt.input)
			_ = got
		})
	}
}
