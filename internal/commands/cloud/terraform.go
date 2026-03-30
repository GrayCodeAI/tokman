package cloud

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var terraformCmd = &cobra.Command{
	Use:   "terraform [command] [args...]",
	Short: "Terraform commands with compact output",
	Long: `Execute Terraform commands with token-optimized output.

Specialized filters for:
  - plan: Compact resource changes summary
  - apply: Compact apply output
  - show: Compact state inspection
  - output: Raw output values

Examples:
  tokman terraform plan
  tokman terraform apply
  tokman terraform show`,
	DisableFlagParsing: true,
	RunE:               runTerraform,
}

func init() {
	registry.Add(func() { registry.Register(terraformCmd) })
}

func runTerraform(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	switch args[0] {
	case "plan":
		return runTerraformPlan(args[1:])
	case "apply":
		return runTerraformApply(args[1:])
	case "show":
		return runTerraformShow(args[1:])
	case "output":
		return runTerraformOutput(args[1:])
	case "init":
		return runTerraformInit(args[1:])
	case "validate":
		return runTerraformValidate(args[1:])
	default:
		return runTerraformPassthrough(args)
	}
}

func runTerraformPlan(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: terraform plan %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("terraform", append([]string{"plan"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterTerraformPlanOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "terraform_plan", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("terraform plan", "tokman terraform plan", originalTokens, filteredTokens)

	return err
}

func filterTerraformPlanOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var adds, changes, destroys int
	var resources []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for plan summary
		if strings.Contains(line, "Plan:") {
			// Parse: "Plan: 2 to add, 1 to change, 0 to destroy."
			result = append(result, "📋 "+line)
			// Extract counts
			if strings.Contains(line, "to add") {
				fmt.Sscanf(line, "Plan: %d to add", &adds)
			}
			if strings.Contains(line, "to change") {
				var c int
				fmt.Sscanf(line, "Plan: %d to add, %d to change", &adds, &c)
				changes = c
			}
			if strings.Contains(line, "to destroy") {
				var d int
				fmt.Sscanf(line, "Plan: %d to add, %d to change, %d to destroy", &adds, &changes, &d)
				destroys = d
			}
		} else if strings.HasPrefix(line, "# ") {
			// Resource action marker
			resources = append(resources, shared.TruncateLine(line, 80))
		} else if strings.Contains(line, "will be created") {
			result = append(result, "  + "+shared.TruncateLine(line, 80))
		} else if strings.Contains(line, "will be updated") {
			result = append(result, "  ~ "+shared.TruncateLine(line, 80))
		} else if strings.Contains(line, "will be destroyed") {
			result = append(result, "  - "+shared.TruncateLine(line, 80))
		} else if strings.Contains(line, "Error:") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		if adds == 0 && changes == 0 && destroys == 0 {
			return "No changes"
		}
		return fmt.Sprintf("+%d ~%d -%d", adds, changes, destroys)
	}

	// Add resource summary if we have it
	if len(resources) > 0 && len(result) > 0 {
		result = append(result, "")
		result = append(result, "Resources:")
		for i, r := range resources {
			if i >= 15 {
				result = append(result, fmt.Sprintf("  ... +%d more", len(resources)-15))
				break
			}
			result = append(result, "  "+r)
		}
	}

	if len(result) == 0 {
		return raw
	}

	return strings.Join(result, "\n")
}

func runTerraformApply(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: terraform apply %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("terraform", append([]string{"apply"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterTerraformApplyOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "terraform_apply", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("terraform apply", "tokman terraform apply", originalTokens, filteredTokens)

	return err
}

func filterTerraformApplyOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Apply complete!") {
			result = append(result, "✅ "+line)
		} else if strings.Contains(line, "Resources:") {
			result = append(result, "📋 "+line)
		} else if strings.Contains(line, "Error:") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		} else if strings.HasPrefix(line, "module.") || strings.HasPrefix(line, "data.") {
			// Resource creation/update
			result = append(result, "  "+shared.TruncateLine(line, 80))
		}
	}

	// Ultra-compact mode
	if shared.UltraCompact && len(result) > 0 {
		for _, line := range result {
			if strings.Contains(line, "Apply complete") {
				return "✅ Applied"
			}
		}
	}

	if len(result) == 0 {
		return "✅ Apply complete"
	}

	return strings.Join(result, "\n")
}

func runTerraformShow(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: terraform show %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("terraform", append([]string{"show"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterTerraformShowOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("terraform show", "tokman terraform show", originalTokens, filteredTokens)

	return err
}

func filterTerraformShowOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	resourceCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Resource blocks
		if strings.HasPrefix(line, "# ") {
			resourceCount++
			if resourceCount <= 20 {
				result = append(result, shared.TruncateLine(line, 80))
			}
		} else if resourceCount <= 20 && (strings.HasPrefix(line, "id ") || strings.HasPrefix(line, "name ") || strings.HasPrefix(line, "arn ")) {
			result = append(result, "  "+shared.TruncateLine(line, 80))
		}
	}

	if resourceCount > 20 {
		result = append(result, fmt.Sprintf("  ... +%d more resources", resourceCount-20))
	}

	if len(result) == 0 {
		return raw
	}

	return strings.Join(result, "\n")
}

func runTerraformOutput(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: terraform output %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("terraform", append([]string{"output"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	// For output, we usually want the raw value
	filtered := filterTerraformOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("terraform output", "tokman terraform output", originalTokens, filteredTokens)

	return err
}

func filterTerraformOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, shared.TruncateLine(line, 120))
		}
	}

	if len(result) > 30 {
		return strings.Join(result[:30], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-30)
	}
	return strings.Join(result, "\n")
}

func runTerraformInit(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: terraform init %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("terraform", append([]string{"init"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterTerraformInitOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "terraform_init", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("terraform init", "tokman terraform init", originalTokens, filteredTokens)

	return err
}

func filterTerraformInitOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var providers []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Terraform has been successfully initialized") {
			result = append(result, "✅ Terraform initialized successfully")
		} else if strings.Contains(line, "Initializing provider plugins") {
			result = append(result, "📦 Initializing providers...")
		} else if strings.Contains(line, "- Finding") {
			// Skip verbose provider finding
		} else if strings.Contains(line, "- Installing") {
			providers = append(providers, shared.TruncateLine(line, 60))
		} else if strings.Contains(line, "Error:") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	// Add providers
	if len(providers) > 0 {
		result = append(result, "")
		result = append(result, "Providers installed:")
		for _, p := range providers {
			result = append(result, "  "+p)
		}
	}

	if len(result) == 0 {
		return raw
	}

	return strings.Join(result, "\n")
}

func runTerraformValidate(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: terraform validate %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("terraform", append([]string{"validate"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterTerraformValidateOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("terraform validate", "tokman terraform validate", originalTokens, filteredTokens)

	return err
}

func filterTerraformValidateOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Success!") {
			return "✅ Configuration is valid"
		} else if strings.Contains(line, "Error:") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		} else if strings.Contains(line, "Warning:") {
			result = append(result, "⚠️  "+shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		return raw
	}

	return strings.Join(result, "\n")
}

func runTerraformPassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: terraform %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("terraform", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterTerraformOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("terraform %s", args[0]), "tokman terraform", originalTokens, filteredTokens)

	return err
}
