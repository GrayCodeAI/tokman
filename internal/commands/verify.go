package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/integrity"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify hook integrity",
	Long: `Verify the integrity of the TokMan hook script.

This command checks that the hook file (~/.claude/hooks/tokman-rewrite.sh)
matches its stored SHA-256 hash to detect any unauthorized modifications.

The integrity check protects against command injection attacks where
an attacker might modify the hook to execute malicious commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		result, err := integrity.VerifyHook()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying hook: %v\n", err)
			os.Exit(1)
		}

		if verbose > 0 {
			fmt.Printf("Hook:  %s\n", result.HookPath)
			fmt.Printf("Hash:  %s\n", result.HashPath)
			fmt.Println()
		}

		switch result.Status {
		case integrity.StatusVerified:
			hash, _ := integrity.ComputeHash(result.HookPath)
			fmt.Printf("%s  hook integrity verified\n", green("PASS"))
			fmt.Printf("      sha256:%s\n", hash)
			fmt.Printf("      %s\n", cyan(result.HookPath))

		case integrity.StatusTampered:
			fmt.Fprintf(os.Stderr, "%s  hook integrity check FAILED\n", red("FAIL"))
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "  Expected: %s\n", result.Expected)
			fmt.Fprintf(os.Stderr, "  Actual:   %s\n", result.Actual)
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "  The hook file has been modified outside of `tokman init`.")
			fmt.Fprintln(os.Stderr, "  This could indicate tampering or a manual edit.")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "  To restore: tokman init")
			fmt.Fprintf(os.Stderr, "  To inspect: cat %s\n", result.HookPath)
			os.Exit(1)

		case integrity.StatusNoBaseline:
			fmt.Printf("%s  no baseline hash found\n", yellow("WARN"))
			fmt.Println("      Hook exists but was installed before integrity checks.")
			fmt.Println("      Run `tokman init` to establish baseline.")

		case integrity.StatusNotInstalled:
			fmt.Printf("%s  TokMan hook not installed\n", yellow("SKIP"))
			fmt.Println("      Run `tokman init` to install.")

		case integrity.StatusOrphanedHash:
			fmt.Fprintf(os.Stderr, "%s  hash file exists but hook is missing\n", yellow("WARN"))
			fmt.Fprintln(os.Stderr, "      Run `tokman init` to reinstall.")
		}

		// Print security reference
		fmt.Println()
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println("Security: This check prevents command injection via hook tampering.")
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
