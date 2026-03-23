package core

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for tokman.

To load completions:

Bash:
  source <(tokman completion bash)
  # Add to ~/.bashrc for permanent effect

Zsh:
  source <(tokman completion zsh)
  # Add to ~/.zshrc for permanent effect

Fish:
  tokman completion fish | source
  # Add to ~/.config/fish/config.fish for permanent effect

PowerShell:
  tokman completion powershell | Out-String | Invoke-Expression
  # Add to profile for permanent effect`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:      runCompletion,
}

func init() {
	registry.Add(func() { registry.Register(completionCmd) })
}

func runCompletion(cmd *cobra.Command, args []string) error {
	root := shared.RootCmd().(*cobra.Command)
	switch args[0] {
	case "bash":
		return root.GenBashCompletion(os.Stdout)
	case "zsh":
		return root.GenZshCompletion(os.Stdout)
	case "fish":
		return root.GenFishCompletion(os.Stdout, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell: %s", args[0])
	}
}
