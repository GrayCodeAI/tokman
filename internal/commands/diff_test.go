package commands

import (
	"strings"
	"testing"
)

func TestCompactDiff(t *testing.T) {
	input := `--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 context line
+added line
-removed line
 another context
`

	got := compactDiff(input)
	// Should contain additions and deletions
	if !strings.Contains(got, "+added") {
		t.Errorf("compactDiff should contain additions, got:\n%s", got)
	}
	if !strings.Contains(got, "-removed") {
		t.Errorf("compactDiff should contain deletions, got:\n%s", got)
	}
	if !strings.Contains(got, "@@") {
		t.Errorf("compactDiff should contain hunk headers, got:\n%s", got)
	}
}

func TestCompactDiffEmpty(t *testing.T) {
	got := compactDiff("")
	_ = got // just check no panic
}
