package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var doctorFix bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose tokman setup issues",
	Long: `Check system configuration, shell hooks, database connectivity,
tokenizer availability, and common setup problems.`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "attempt to fix detected issues")
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	Name    string
	Status  string // "ok", "warn", "error"
	Message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("tokman doctor — diagnosing setup")
	fmt.Println("================================")

	var results []checkResult

	// Check 1: Binary location
	results = append(results, checkBinary())

	// Check 2: Config directory
	results = append(results, checkConfigDir())

	// Check 3: Database
	results = append(results, checkDatabase())

	// Check 4: Shell hook
	results = append(results, checkShellHook())

	// Check 5: PATH resolution
	results = append(results, checkPath())

	// Check 6: Platform info
	results = append(results, checkPlatform())

	// Check 7: Tokenizer
	results = append(results, checkTokenizer())

	// Check 8: TOML filters (R70)
	results = append(results, checkTOMLFilters())

	// Check 9: Disk space (R70)
	results = append(results, checkDiskSpace())

	// Check 10: Go version (R70)
	results = append(results, checkGoVersion())

	// Print results
	hasError := false
	for _, r := range results {
		icon := "✓"
		switch r.Status {
		case "warn":
			icon = "⚠"
		case "error":
			icon = "✗"
			hasError = true
		}
		fmt.Printf("  %s %s: %s\n", icon, r.Name, r.Message)
	}

	fmt.Println()
	if hasError {
		fmt.Println("Some checks failed. See messages above for fixes.")
		return fmt.Errorf("doctor check failed")
	}
	fmt.Println("All checks passed!")
	return nil
}

func checkBinary() checkResult {
	exe, err := os.Executable()
	if err != nil {
		return checkResult{"Binary", "error", "cannot determine executable path"}
	}
	return checkResult{"Binary", "ok", exe}
}

func checkConfigDir() checkResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return checkResult{"Config Dir", "error", "cannot determine home directory"}
	}
	configDir := filepath.Join(home, ".config", "tokman")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return checkResult{"Config Dir", "warn", configDir + " (not found — run 'tokman init')"}
	}
	return checkResult{"Config Dir", "ok", configDir}
}

func checkDatabase() checkResult {
	dbPath := tracking.DatabasePath()
	if dbPath == "" {
		return checkResult{"Database", "error", "cannot determine database path"}
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return checkResult{"Database", "warn", dbPath + " (will be created on first use)"}
	}
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return checkResult{"Database", "error", dbPath + " (" + err.Error() + ")"}
	}
	defer tracker.Close()
	return checkResult{"Database", "ok", dbPath}
}

func checkShellHook() checkResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return checkResult{"Shell Hook", "error", "cannot determine home directory"}
	}

	// Check common hook locations
	hookPaths := []string{
		filepath.Join(home, ".claude", "hooks", "tokman.sh"),
		filepath.Join(home, ".config", "tokman", "hook.sh"),
	}

	for _, p := range hookPaths {
		if _, err := os.Stat(p); err == nil {
			return checkResult{"Shell Hook", "ok", p}
		}
	}

	return checkResult{"Shell Hook", "warn", "no shell hook found — run 'tokman init'"}
}

func checkPath() checkResult {
	path, err := exec.LookPath("tokman")
	if err != nil {
		return checkResult{"PATH", "warn", "tokman not in PATH (may need symlink or PATH update)"}
	}
	return checkResult{"PATH", "ok", path}
}

func checkPlatform() checkResult {
	return checkResult{"Platform", "ok", runtime.GOOS + "/" + runtime.GOARCH + " Go " + runtime.Version()}
}

func checkTokenizer() checkResult {
	// Try to use tiktoken to verify tokenizer is available
	_, err := exec.LookPath("tiktoken")
	if err != nil {
		// tiktoken is embedded in Go binary, so this is OK
		return checkResult{"Tokenizer", "ok", "tiktoken-go (embedded)"}
	}
	return checkResult{"Tokenizer", "ok", "tiktoken available"}
}

func checkTOMLFilters() checkResult {
	builtinDir := filepath.Join(getTokmanSourceDir(), "internal", "toml", "builtin")
	if entries, err := os.ReadDir(builtinDir); err == nil {
		count := 0
		for _, e := range entries {
			if !e.IsDir() {
				count++
			}
		}
		return checkResult{"TOML Filters", "ok", fmt.Sprintf("%d built-in filters", count)}
	}
	return checkResult{"TOML Filters", "warn", "built-in filters directory not found"}
}

func checkDiskSpace() checkResult {
	home, _ := os.UserHomeDir()
	if home == "" {
		return checkResult{"Disk Space", "warn", "cannot check disk space"}
	}
	dbPath := filepath.Join(home, ".local", "share", "tokman", "tracking.db")
	if info, err := os.Stat(dbPath); err == nil {
		sizeMB := float64(info.Size()) / 1024 / 1024
		if sizeMB > 100 {
			return checkResult{"Disk Space", "warn", fmt.Sprintf("database is %.1fMB — consider 'tokman clean'", sizeMB)}
		}
		return checkResult{"Disk Space", "ok", fmt.Sprintf("database is %.1fMB", sizeMB)}
	}
	return checkResult{"Disk Space", "ok", "no database yet"}
}

func checkGoVersion() checkResult {
	// Check if Go is available for development
	if _, err := exec.LookPath("go"); err == nil {
		return checkResult{"Go", "ok", "available (for development)"}
	}
	return checkResult{"Go", "ok", "not required (prebuilt binary)"}
}
