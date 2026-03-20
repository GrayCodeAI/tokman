package commands

import (
	"strings"
	"testing"
)

func TestFilterBranchOutput(t *testing.T) {
	input := "  main\n  feature\n  develop\n"
	got := filterBranchOutput(input)
	if !strings.Contains(got, "main") {
		t.Errorf("filterBranchOutput should contain 'main', got:\n%s", got)
	}
}

func TestFilterStashList(t *testing.T) {
	// Test with empty input
	got := filterStashList("")
	_ = got // just check no panic

	// Test with some input
	got = filterStashList("stash@{0}: WIP on main\n")
	if got == "" {
		t.Error("filterStashList should return something for non-empty input")
	}
}

func TestFilterWorktreeList(t *testing.T) {
	input := "/home/user/repo  abc1234 [main]\n"
	got := filterWorktreeList(input)
	if got == "" {
		t.Error("filterWorktreeList should return something")
	}
}
