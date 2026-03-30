package lang

import (
	"encoding/json"
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

var rubyCmd = &cobra.Command{
	Use:   "ruby [args...]",
	Short: "Ruby commands with compact output",
	Long: `Execute Ruby commands with token-optimized output.

Provides compact output for rspec, rubocop, rake, and bundle commands.

Examples:
  tokman ruby rspec
  tokman ruby rubocop
  tokman ruby rake test
  tokman ruby bundle install`,
	DisableFlagParsing: true,
	RunE:               runRuby,
}

func init() {
	registry.Add(func() { registry.Register(rubyCmd) })
}

func runRuby(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"--help"}
	}

	// Route to specialized handlers
	switch args[0] {
	case "rspec":
		return runRspecCmd(args[1:])
	case "rubocop":
		return runRubocopCmd(args[1:])
	case "rake":
		return runRakeCmd(args[1:])
	case "bundle":
		return runBundleCmd(args[1:])
	case "rails":
		return runRailsCmd(args[1:])
	default:
		return runRubyPassthrough(args)
	}
}

// =============================================================================
// RSpec Commands (Task 58)
// =============================================================================

func runRspecCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rspec %s\n", strings.Join(args, " "))
	}

	// Use --format json for structured output
	jsonArgs := append([]string{"--format", "json"}, args...)
	execCmd := exec.Command("rspec", jsonArgs...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRspecOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rspec", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("rspec %s", strings.Join(args, " ")), "tokman ruby rspec", originalTokens, filteredTokens)

	return err
}

// RSpecJSON represents the JSON output from rspec --format json
type RSpecJSON struct {
	Version  string        `json:"version"`
	Examples []RSpecExample `json:"examples"`
	Summary  RSpecSummary  `json:"summary"`
}

type RSpecExample struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	FullDescription string `json:"full_description"`
	Status      string `json:"status"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	Exception   *RSpecException `json:"exception,omitempty"`
}

type RSpecException struct {
	Class     string `json:"class"`
	Message   string `json:"message"`
	Backtrace []string `json:"backtrace"`
}

type RSpecSummary struct {
	Duration       float64 `json:"duration"`
	ExampleCount   int     `json:"example_count"`
	FailureCount   int     `json:"failure_count"`
	PendingCount   int     `json:"pending_count"`
}

func filterRspecOutput(raw string) string {
	// Try to parse as JSON
	var rspec RSpecJSON
	if err := json.Unmarshal([]byte(raw), &rspec); err != nil {
		// Fall back to text parsing
		return filterRspecTextOutput(raw)
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		return filterRspecOutputUltraCompact(rspec)
	}

	var result []string
	result = append(result, "📋 RSpec Results:")
	result = append(result, fmt.Sprintf("   ✅ %d passed", rspec.Summary.ExampleCount-rspec.Summary.FailureCount-rspec.Summary.PendingCount))
	if rspec.Summary.FailureCount > 0 {
		result = append(result, fmt.Sprintf("   ❌ %d failed", rspec.Summary.FailureCount))
	}
	if rspec.Summary.PendingCount > 0 {
		result = append(result, fmt.Sprintf("   ⏸️  %d pending", rspec.Summary.PendingCount))
	}
	result = append(result, fmt.Sprintf("   ⏱️  %.2fs", rspec.Summary.Duration))

	// Show failures
	failures := 0
	for _, ex := range rspec.Examples {
		if ex.Status == "failed" && ex.Exception != nil {
			if failures == 0 {
				result = append(result, "")
				result = append(result, "Failures:")
			}
			failures++
			if failures <= 10 {
				result = append(result, fmt.Sprintf("   • %s:%d: %s", 
					shared.TruncateLine(ex.FilePath, 40), 
					ex.LineNumber, 
					shared.TruncateLine(ex.Description, 50)))
				result = append(result, fmt.Sprintf("     %s", 
					shared.TruncateLine(ex.Exception.Message, 80)))
			}
		}
	}
	if failures > 10 {
		result = append(result, fmt.Sprintf("   ... +%d more failures", failures-10))
	}

	return strings.Join(result, "\n")
}

func filterRspecOutputUltraCompact(rspec RSpecJSON) string {
	passed := rspec.Summary.ExampleCount - rspec.Summary.FailureCount - rspec.Summary.PendingCount
	var parts []string

	parts = append(parts, fmt.Sprintf("P:%d", passed))
	if rspec.Summary.FailureCount > 0 {
		parts = append(parts, fmt.Sprintf("F:%d", rspec.Summary.FailureCount))
	}
	if rspec.Summary.PendingCount > 0 {
		parts = append(parts, fmt.Sprintf("S:%d", rspec.Summary.PendingCount))
	}

	var result []string
	result = append(result, strings.Join(parts, " "))

	// Show up to 5 failures
	failures := 0
	for _, ex := range rspec.Examples {
		if ex.Status == "failed" && failures < 5 {
			failures++
			shortFile := ex.FilePath
			if idx := strings.LastIndex(ex.FilePath, "/"); idx >= 0 {
				shortFile = ex.FilePath[idx+1:]
			}
			result = append(result, fmt.Sprintf("FAIL: %s:%d", shortFile, ex.LineNumber))
		}
	}
	if rspec.Summary.FailureCount > 5 {
		result = append(result, fmt.Sprintf("... +%d more", rspec.Summary.FailureCount-5))
	}

	return strings.Join(result, "\n")
}

func filterRspecTextOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var failures []string
	var passed, failed, pending int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for summary patterns
		if strings.Contains(line, " examples, ") {
			// Parse summary line like "5 examples, 0 failures"
			result = append(result, "📋 RSpec: "+line)
		} else if strings.Contains(line, "FAILED -") || strings.Contains(line, "Failure/Error") {
			failures = append(failures, shared.TruncateLine(line, 100))
			failed++
		} else if strings.Contains(line, "pending") && strings.Contains(line, "(PENDING:") {
			pending++
		} else if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "F") {
			// Progress dots - count them
			for _, c := range line {
				if c == '.' {
					passed++
				} else if c == 'F' {
					failed++
				}
			}
		}
	}

	if len(failures) > 0 {
		result = append(result, "")
		result = append(result, "Failures:")
		for i, f := range failures {
			if i >= 10 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(failures)-10))
				break
			}
			result = append(result, fmt.Sprintf("   • %s", f))
		}
	}

	if len(result) == 0 {
		return raw
	}
	return strings.Join(result, "\n")
}

// =============================================================================
// RuboCop Commands (Task 59)
// =============================================================================

func runRubocopCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rubocop %s\n", strings.Join(args, " "))
	}

	// Use --format json for structured output
	jsonArgs := append([]string{"--format", "json"}, args...)
	execCmd := exec.Command("rubocop", jsonArgs...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRubocopOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("rubocop %s", strings.Join(args, " ")), "tokman ruby rubocop", originalTokens, filteredTokens)

	return err
}

// RuboCopJSON represents the JSON output from rubocop --format json
type RuboCopJSON struct {
	Metadata RuboCopMetadata `json:"metadata"`
	Files    []RuboCopFile   `json:"files"`
	Summary  RuboCopSummary  `json:"summary"`
}

type RuboCopMetadata struct {
	RuboCopVersion string `json:"rubocop_version"`
}

type RuboCopFile struct {
	Path     string         `json:"path"`
	Offenses []RuboCopOffense `json:"offenses"`
}

type RuboCopOffense struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	CopName  string `json:"cop_name"`
	Location RuboCopLocation `json:"location"`
}

type RuboCopLocation struct {
	StartLine int `json:"start_line"`
}

type RuboCopSummary struct {
	OffenseCount int `json:"offense_count"`
	TargetFileCount int `json:"target_file_count"`
	InspectedFileCount int `json:"inspected_file_count"`
}

func filterRubocopOutput(raw string) string {
	// Try to parse as JSON
	var rubocop RuboCopJSON
	if err := json.Unmarshal([]byte(raw), &rubocop); err != nil {
		// Fall back to text parsing
		return filterRubocopTextOutput(raw)
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		return filterRubocopOutputUltraCompact(rubocop)
	}

	var result []string
	result = append(result, "📋 RuboCop Results:")
	result = append(result, fmt.Sprintf("   📁 %d files inspected", rubocop.Summary.InspectedFileCount))

	if rubocop.Summary.OffenseCount == 0 {
		result = append(result, "   ✅ No offenses detected")
		return strings.Join(result, "\n")
	}

	// Count by severity
	severityCount := make(map[string]int)
	for _, file := range rubocop.Files {
		for _, offense := range file.Offenses {
			severityCount[offense.Severity]++
		}
	}

	result = append(result, fmt.Sprintf("   ⚠️  %d offenses:", rubocop.Summary.OffenseCount))
	for sev, count := range severityCount {
		result = append(result, fmt.Sprintf("      %s: %d", sev, count))
	}

	// Show offenses by file
	offenseCount := 0
	for _, file := range rubocop.Files {
		if len(file.Offenses) > 0 {
			shortPath := file.Path
			if idx := strings.LastIndex(file.Path, "/"); idx >= 0 {
				shortPath = file.Path[idx+1:]
			}
			result = append(result, "")
			result = append(result, fmt.Sprintf("   📄 %s (%d offenses):", shortPath, len(file.Offenses)))
			for _, offense := range file.Offenses {
				offenseCount++
				if offenseCount > 15 {
					result = append(result, fmt.Sprintf("   ... +%d more offenses", rubocop.Summary.OffenseCount-15))
					break
				}
				severityIcon := "⚠️"
				if offense.Severity == "error" || offense.Severity == "fatal" {
					severityIcon = "❌"
				}
				result = append(result, fmt.Sprintf("      %s L%d: %s", 
					severityIcon, 
					offense.Location.StartLine,
					shared.TruncateLine(offense.Message, 60)))
			}
			if offenseCount > 15 {
				break
			}
		}
	}

	return strings.Join(result, "\n")
}

func filterRubocopOutputUltraCompact(rubocop RuboCopJSON) string {
	if rubocop.Summary.OffenseCount == 0 {
		return "✅ Clean"
	}

	var result []string
	result = append(result, fmt.Sprintf("O:%d in %d files", rubocop.Summary.OffenseCount, rubocop.Summary.InspectedFileCount))

	// Show up to 5 files with offenses
	fileCount := 0
	for _, file := range rubocop.Files {
		if len(file.Offenses) > 0 && fileCount < 5 {
			shortPath := file.Path
			if idx := strings.LastIndex(file.Path, "/"); idx >= 0 {
				shortPath = file.Path[idx+1:]
			}
			result = append(result, fmt.Sprintf("%s: %d", shortPath, len(file.Offenses)))
			fileCount++
		}
	}

	return strings.Join(result, "\n")
}

func filterRubocopTextOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var offenses []string
	fileCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for file markers
		if strings.HasPrefix(line, "Inspecting") {
			fileCount++
		} else if strings.Contains(line, " offenses detected") || strings.Contains(line, " offense detected") {
			result = append(result, line)
		} else if strings.Contains(line, ":") && (strings.Contains(line, "Error") || 
			strings.Contains(line, "Warning") || strings.Contains(line, "Convention") ||
			strings.Contains(line, "Style")) {
			offenses = append(offenses, shared.TruncateLine(line, 100))
		}
	}

	if len(offenses) > 0 {
		result = append(result, "")
		result = append(result, "Offenses:")
		for i, o := range offenses {
			if i >= 15 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(offenses)-15))
				break
			}
			result = append(result, fmt.Sprintf("   • %s", o))
		}
	}

	if len(result) == 0 {
		return raw
	}
	return strings.Join(result, "\n")
}

// =============================================================================
// Rake Commands (Task 57)
// =============================================================================

func runRakeCmd(args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{}
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rake %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rake", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRakeOutput(raw, args)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rake", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	taskName := "rake"
	if len(args) > 0 {
		taskName = fmt.Sprintf("rake %s", args[0])
	}
	timer.Track(taskName, "tokman ruby rake", originalTokens, filteredTokens)

	return err
}

func filterRakeOutput(raw string, args []string) string {
	if raw == "" {
		return "✅ Rake completed"
	}

	// Check if this is a test task
	if len(args) > 0 && (args[0] == "test" || args[0] == "spec") {
		return filterRakeTestOutput(raw)
	}

	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip verbose rake output
		if strings.HasPrefix(line, "rake aborted!") {
			result = append(result, "❌ "+line)
		} else if strings.Contains(line, "Error") || strings.Contains(line, "error:") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		} else if len(result) < 20 {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		return "✅ Rake completed"
	}

	if len(result) > 20 {
		return strings.Join(result[:20], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-20)
	}

	return strings.Join(result, "\n")
}

func filterRakeTestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var testsRun, testsPassed, testsFailed int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for test summary patterns
		if strings.Contains(line, " tests, ") || strings.Contains(line, " assertions, ") {
			result = append(result, "📋 "+line)
		} else if strings.Contains(line, "FAIL") || strings.Contains(line, "Error") {
			testsFailed++
			if testsFailed <= 10 {
				result = append(result, "❌ "+shared.TruncateLine(line, 100))
			}
		} else if strings.Contains(line, "PASS") || strings.Contains(line, ".") {
			testsPassed++
		}

		// Count tests
		if strings.Contains(line, "test_") {
			testsRun++
		}
	}

	if len(result) == 0 {
		return "✅ All tests passed"
	}

	if testsFailed > 10 {
		result = append(result, fmt.Sprintf("... +%d more failures", testsFailed-10))
	}

	return strings.Join(result, "\n")
}

// =============================================================================
// Bundle Commands (Task 60)
// =============================================================================

func runBundleCmd(args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle %s\n", strings.Join(args, " "))
	}

	// Route to specialized handlers
	switch args[0] {
	case "install":
		return runBundleInstallCmd(args[1:])
	case "update":
		return runBundleUpdateCmd(args[1:])
	case "outdated":
		return runBundleOutdatedCmd(args[1:])
	default:
		return runBundlePassthrough(args)
	}
}

func runBundleInstallCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle install %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", append([]string{"install"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleInstallOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "bundle_install", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("bundle install", "tokman ruby bundle install", originalTokens, filteredTokens)

	return err
}

func filterBundleInstallOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var installed, updated int
	gems := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Track gem installations
		if strings.Contains(line, "Installing") {
			installed++
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				gems = append(gems, parts[1])
			}
		} else if strings.Contains(line, "Using") {
			updated++
		} else if strings.Contains(line, "Bundle complete!") {
			result = append(result, "✅ "+line)
		} else if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		var parts []string
		if installed > 0 {
			parts = append(parts, fmt.Sprintf("I:%d", installed))
		}
		if updated > 0 {
			parts = append(parts, fmt.Sprintf("U:%d", updated))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
		return "✅ Done"
	}

	// Normal output
	if len(result) == 0 {
		if installed > 0 || updated > 0 {
			result = append(result, "📋 Bundle Install Summary:")
			if installed > 0 {
				result = append(result, fmt.Sprintf("   📦 %d gems installed", installed))
				if len(gems) > 0 && len(gems) <= 5 {
					result = append(result, fmt.Sprintf("      %s", strings.Join(gems, ", ")))
				} else if len(gems) > 5 {
					result = append(result, fmt.Sprintf("      %s ... +%d more", 
						strings.Join(gems[:5], ", "), len(gems)-5))
				}
			}
			if updated > 0 {
				result = append(result, fmt.Sprintf("   ✓ %d gems unchanged", updated))
			}
		} else {
			result = append(result, "✅ Bundle already up to date")
		}
	}

	return strings.Join(result, "\n")
}

func runBundleUpdateCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle update %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", append([]string{"update"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleUpdateOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("bundle update", "tokman ruby bundle update", originalTokens, filteredTokens)

	return err
}

func filterBundleUpdateOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var updated int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Installing") || strings.Contains(line, "Updating") {
			updated++
		} else if strings.Contains(line, "Bundle updated!") {
			result = append(result, "✅ "+line)
		} else if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		if updated > 0 {
			return fmt.Sprintf("✅ Updated %d gems", updated)
		}
		return "✅ Bundle update complete"
	}

	return strings.Join(result, "\n")
}

func runBundleOutdatedCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle outdated %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", append([]string{"outdated"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleOutdatedOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("bundle outdated", "tokman ruby bundle outdated", originalTokens, filteredTokens)

	return err
}

func filterBundleOutdatedOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var outdated []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Gem") && strings.Contains(line, "Current") && strings.Contains(line, "Latest") {
			// Header line - skip
			continue
		} else if strings.Contains(line, "(") && strings.Contains(line, ">") {
			// Outdated gem line like "rails (6.1.0 > 7.0.0)"
			outdated = append(outdated, shared.TruncateLine(line, 60))
		} else if strings.Contains(line, "Bundle up to date") {
			return "✅ All gems are up to date"
		}
	}

	if len(outdated) == 0 {
		return "✅ All gems are up to date"
	}

	result = append(result, fmt.Sprintf("📋 %d outdated gems:", len(outdated)))
	for i, gem := range outdated {
		if i >= 15 {
			result = append(result, fmt.Sprintf("   ... +%d more", len(outdated)-15))
			break
		}
		result = append(result, fmt.Sprintf("   • %s", gem))
	}

	return strings.Join(result, "\n")
}

func runBundlePassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: bundle %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("bundle", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterBundleOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("bundle %s", args[0]), "tokman ruby bundle", originalTokens, filteredTokens)

	return err
}

func filterBundleOutput(raw string) string {
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

// =============================================================================
// Rails Commands
// =============================================================================

func runRailsCmd(args []string) error {
	if len(args) == 0 {
		args = []string{"--help"}
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails %s\n", strings.Join(args, " "))
	}

	// Route to specialized handlers
	switch args[0] {
	case "test", "t":
		return runRailsTestCmd(args[1:])
	case "db:migrate":
		return runRailsDbMigrateCmd(args[1:])
	default:
		return runRailsPassthrough(args)
	}
}

func runRailsTestCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails test %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rails", append([]string{"test"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRailsTestOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rails_test", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("rails test", "tokman ruby rails test", originalTokens, filteredTokens)

	return err
}

func filterRailsTestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var passed, failed int
	var failures []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for test result lines
		if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "F") || strings.HasPrefix(line, "E") {
			for _, c := range line {
				if c == '.' {
					passed++
				} else if c == 'F' {
					failed++
				} else if c == 'E' {
					failed++
				}
			}
		} else if strings.Contains(line, " runs, ") {
			result = append(result, "📋 "+line)
		} else if strings.Contains(line, "FAIL") || strings.Contains(line, "ERROR") {
			failures = append(failures, shared.TruncateLine(line, 100))
		}
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		var parts []string
		parts = append(parts, fmt.Sprintf("P:%d", passed))
		if failed > 0 {
			parts = append(parts, fmt.Sprintf("F:%d", failed))
		}
		return strings.Join(parts, " ")
	}

	// Normal output
	if len(result) == 0 {
		if passed > 0 || failed > 0 {
			result = append(result, "📋 Rails Test Results:")
			result = append(result, fmt.Sprintf("   ✅ %d passed", passed))
			if failed > 0 {
				result = append(result, fmt.Sprintf("   ❌ %d failed", failed))
			}
		}
	}

	if len(failures) > 0 {
		result = append(result, "")
		result = append(result, "Failures:")
		for i, f := range failures {
			if i >= 10 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(failures)-10))
				break
			}
			result = append(result, fmt.Sprintf("   • %s", f))
		}
	}

	if len(result) == 0 {
		return "✅ Tests passed"
	}

	return strings.Join(result, "\n")
}

func runRailsDbMigrateCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails db:migrate %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rails", append([]string{"db:migrate"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRailsDbMigrateOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rails_db_migrate", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("rails db:migrate", "tokman ruby rails db:migrate", originalTokens, filteredTokens)

	return err
}

func filterRailsDbMigrateOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var migrations []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for migration output
		if strings.Contains(line, "==") && strings.Contains(line, "migrating") {
			// Migration start
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				migrations = append(migrations, parts[0])
			}
		} else if strings.Contains(line, "migrated") {
			result = append(result, "✅ "+shared.TruncateLine(line, 80))
		} else if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result = append(result, "❌ "+shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		if len(migrations) > 0 {
			return fmt.Sprintf("✅ %d migrations completed", len(migrations))
		}
		return "✅ Database migrated"
	}

	return strings.Join(result, "\n")
}

func runRailsPassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rails", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRailsOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("rails %s", args[0]), "tokman ruby rails", originalTokens, filteredTokens)

	return err
}

func filterRailsOutput(raw string) string {
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

// =============================================================================
// Ruby Passthrough
// =============================================================================

func runRubyPassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: ruby %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("ruby", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRubyOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("ruby %s", strings.Join(args, " ")), "tokman ruby", originalTokens, filteredTokens)

	return err
}

func filterRubyOutput(raw string) string {
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
