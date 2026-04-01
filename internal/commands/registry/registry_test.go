package registry

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterSkipsDuplicateCommandNames(t *testing.T) {
	root := &cobra.Command{Use: "tokman"}
	Init(root)

	first := &cobra.Command{Use: "gain", Short: "first"}
	second := &cobra.Command{Use: "gain", Short: "second"}

	Register(first)
	Register(second)

	cmds := root.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command after duplicate registration, got %d", len(cmds))
	}
	if cmds[0].Short != "first" {
		t.Fatalf("expected first registration to win, got %q", cmds[0].Short)
	}
}
