package integration

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	cliBuildOnce sync.Once
	cliBuildPath string
	cliBuildErr  error
)

type cliSmokeEnv struct {
	root          string
	home          string
	xdgConfigHome string
	xdgDataHome   string
}

type cliResult struct {
	output   string
	exitCode int
	err      error
}

func TestCLISmokePassthroughRecordedInStatusAndGain(t *testing.T) {
	env := newCLISmokeEnv(t)

	run := runTokmanCLI(t, env, "echo", "hi")
	if run.err != nil {
		t.Fatalf("echo passthrough failed: %v\n%s", run.err, run.output)
	}
	if strings.TrimSpace(run.output) != "hi" {
		t.Fatalf("unexpected passthrough output: %q", run.output)
	}

	status := runTokmanCLI(t, env, "status")
	if status.err != nil {
		t.Fatalf("status failed: %v\n%s", status.err, status.output)
	}
	if !regexp.MustCompile(`Commands:\s+1`).MatchString(status.output) {
		t.Fatalf("status did not report one command:\n%s", status.output)
	}
	if !strings.Contains(status.output, "echo hi") {
		t.Fatalf("status did not include passthrough command:\n%s", status.output)
	}

	gain := runTokmanCLI(t, env, "gain")
	if gain.err != nil {
		t.Fatalf("gain failed: %v\n%s", gain.err, gain.output)
	}
	if !regexp.MustCompile(`(Total commands|Commands processed):\s+1`).MatchString(gain.output) {
		t.Fatalf("gain did not report one command:\n%s", gain.output)
	}
	if !strings.Contains(gain.output, "echo hi") {
		t.Fatalf("gain did not include passthrough command:\n%s", gain.output)
	}
}

func TestCLISmokeConfiguredDatabasePathIsHonored(t *testing.T) {
	env := newCLISmokeEnv(t)
	customDBPath := filepath.Join(env.root, "custom-db", "tracking.sqlite")
	env.writeConfig(t, fmt.Sprintf("[tracking]\ndatabase_path = %q\n", customDBPath))

	run := runTokmanCLI(t, env, "echo", "hi")
	if run.err != nil {
		t.Fatalf("echo passthrough failed: %v\n%s", run.err, run.output)
	}

	if _, err := os.Stat(customDBPath); err != nil {
		t.Fatalf("custom database path was not created: %v", err)
	}

	defaultDBPath := filepath.Join(env.dataDir(), "tracking.db")
	if _, err := os.Stat(defaultDBPath); err == nil {
		t.Fatalf("unexpected default database created at %s", defaultDBPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat default database: %v", err)
	}

	trackerDB, err := tracking.NewTracker(customDBPath)
	if err != nil {
		t.Fatalf("open custom tracker: %v", err)
	}
	defer trackerDB.Close()

	summary, err := trackerDB.GetSavings("")
	if err != nil {
		t.Fatalf("GetSavings() error = %v", err)
	}
	if summary.TotalCommands != 1 {
		t.Fatalf("custom database recorded %d commands, want 1", summary.TotalCommands)
	}

	status := runTokmanCLI(t, env, "status")
	if status.err != nil {
		t.Fatalf("status failed: %v\n%s", status.err, status.output)
	}
	if !regexp.MustCompile(`Commands:\s+1`).MatchString(status.output) {
		t.Fatalf("status did not read configured database:\n%s", status.output)
	}

	top := runTokmanCLI(t, env, "top", "--limit", "5")
	if top.err != nil {
		t.Fatalf("top failed: %v\n%s", top.err, top.output)
	}
	if !strings.Contains(top.output, "echo hi") {
		t.Fatalf("top did not read configured database via global tracker:\n%s", top.output)
	}
}

func TestCLISmokeCustomConfigFlagIsHonoredByReadCommands(t *testing.T) {
	env := newCLISmokeEnv(t)
	customConfigPath := filepath.Join(env.root, "custom-config.toml")
	customDBPath := filepath.Join(env.root, "custom-flag-db", "tracking.sqlite")
	if err := os.WriteFile(customConfigPath, []byte(fmt.Sprintf("[tracking]\ndatabase_path = %q\n", customDBPath)), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", customConfigPath, err)
	}

	run := runTokmanCLI(t, env, "--config", customConfigPath, "echo", "hi")
	if run.err != nil {
		t.Fatalf("echo passthrough failed: %v\n%s", run.err, run.output)
	}

	if _, err := os.Stat(customDBPath); err != nil {
		t.Fatalf("custom database path was not created: %v", err)
	}

	defaultDBPath := filepath.Join(env.dataDir(), "tracking.db")
	if _, err := os.Stat(defaultDBPath); err == nil {
		t.Fatalf("unexpected default database created at %s", defaultDBPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat default database: %v", err)
	}

	stats := runTokmanCLI(t, env, "--config", customConfigPath, "stats")
	if stats.err != nil {
		t.Fatalf("stats failed: %v\n%s", stats.err, stats.output)
	}
	if !regexp.MustCompile(`Commands tracked:\s+1`).MatchString(stats.output) {
		t.Fatalf("stats did not read configured database via --config:\n%s", stats.output)
	}

	history := runTokmanCLI(t, env, "--config", customConfigPath, "history", "--limit", "5")
	if history.err != nil {
		t.Fatalf("history failed: %v\n%s", history.err, history.output)
	}
	if !strings.Contains(history.output, "echo hi") {
		t.Fatalf("history did not read configured database via --config:\n%s", history.output)
	}

	export := runTokmanCLI(t, env, "--config", customConfigPath, "export", "--format", "json", "--limit", "5")
	if export.err != nil {
		t.Fatalf("export failed: %v\n%s", export.err, export.output)
	}
	if !strings.Contains(export.output, "\"command\": \"echo hi\"") {
		t.Fatalf("export did not read configured database via --config:\n%s", export.output)
	}
}

func TestCLISmokeXDGUserFilterAppliesOnNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell executable permissions")
	}

	env := newCLISmokeEnv(t)
	scriptPath := filepath.Join(env.root, "probe-filter")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nprintf 'INFO: skip\\nERROR: keep\\n'\nexit 7\n"), 0755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", scriptPath, err)
	}

	env.writeFilters(t, fmt.Sprintf(`schema_version = 1

[shell_error]
match_command = "probe-filter$"
keep_lines_matching = ["^ERROR:"]
`))

	run := runTokmanCLI(t, env, scriptPath)
	if run.exitCode != 7 {
		t.Fatalf("exit code = %d, want 7\n%s", run.exitCode, run.output)
	}
	if !strings.Contains(run.output, "ERROR: keep") {
		t.Fatalf("filtered output did not keep error line:\n%s", run.output)
	}
	if strings.Contains(run.output, "INFO: skip") {
		t.Fatalf("filtered output still contains stripped line:\n%s", run.output)
	}

	trackerDB, err := tracking.NewTracker(filepath.Join(env.dataDir(), "tracking.db"))
	if err != nil {
		t.Fatalf("open tracker: %v", err)
	}
	defer trackerDB.Close()

	recent, err := trackerDB.GetRecentCommands("", 5)
	if err != nil {
		t.Fatalf("GetRecentCommands() error = %v", err)
	}
	if len(recent) == 0 {
		t.Fatal("expected at least one recorded command")
	}
	if recent[0].Command != scriptPath {
		t.Fatalf("unexpected recorded command: %q", recent[0].Command)
	}
	if recent[0].ParseSuccess {
		t.Fatalf("expected non-zero matched fallback to record parse_success=false: %+v", recent[0])
	}
}

func newCLISmokeEnv(t *testing.T) cliSmokeEnv {
	t.Helper()

	root := t.TempDir()
	env := cliSmokeEnv{
		root:          root,
		home:          filepath.Join(root, "home"),
		xdgConfigHome: filepath.Join(root, "xdg-config"),
		xdgDataHome:   filepath.Join(root, "xdg-data"),
	}

	dirs := []string{
		env.home,
		env.xdgConfigHome,
		env.xdgDataHome,
		env.configDir(),
		filepath.Join(env.home, ".config", "tokman"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}

	return env
}

func (e cliSmokeEnv) configDir() string {
	return filepath.Join(e.xdgConfigHome, "tokman")
}

func (e cliSmokeEnv) dataDir() string {
	return filepath.Join(e.xdgDataHome, "tokman")
}

func (e cliSmokeEnv) writeConfig(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(e.configDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func (e cliSmokeEnv) writeFilters(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(e.configDir(), "filters.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func runTokmanCLI(t *testing.T, env cliSmokeEnv, args ...string) cliResult {
	t.Helper()

	cmd := exec.Command(cliBinary(t), args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = mergeEnv(os.Environ(), map[string]string{
		"HOME":            env.home,
		"NO_COLOR":        "1",
		"XDG_CONFIG_HOME": env.xdgConfigHome,
		"XDG_DATA_HOME":   env.xdgDataHome,
	})

	output, err := cmd.CombinedOutput()
	result := cliResult{
		output: string(output),
		err:    err,
	}
	if err == nil {
		return result
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.exitCode = exitErr.ExitCode()
		return result
	}

	t.Fatalf("command %v failed before exit status resolution: %v\n%s", args, err, output)
	return cliResult{}
}

func cliBinary(t *testing.T) string {
	t.Helper()

	cliBuildOnce.Do(func() {
		buildDir, err := os.MkdirTemp("", "tokman-cli-smoke-*")
		if err != nil {
			cliBuildErr = err
			return
		}

		cliBuildPath = filepath.Join(buildDir, "tokman")
		buildCmd := exec.Command("go", "build", "-o", cliBuildPath, "./cmd/tokman")
		buildCmd.Dir = repoRoot(t)
		output, err := buildCmd.CombinedOutput()
		if err != nil {
			cliBuildErr = fmt.Errorf("failed to build tokman: %w\n%s", err, output)
		}
	})

	if cliBuildErr != nil {
		t.Fatal(cliBuildErr)
	}
	return cliBuildPath
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	return root
}

func mergeEnv(base []string, overrides map[string]string) []string {
	prefixes := make([]string, 0, len(overrides))
	for key := range overrides {
		prefixes = append(prefixes, key+"=")
	}

	env := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		skip := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(entry, prefix) {
				skip = true
				break
			}
		}
		if !skip {
			env = append(env, entry)
		}
	}

	for key, value := range overrides {
		env = append(env, key+"="+value)
	}

	return env
}
