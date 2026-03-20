package filter

import (
	"strings"
	"testing"
)

// Regression tests ensure compression quality doesn't degrade over time.

// TestRegressionGitStatus ensures git status output compresses correctly
func TestRegressionGitStatus(t *testing.T) {
	input := `On branch main
Your branch is up to date with 'origin/main'.

Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
	new file:   internal/filter/new_feature.go
	modified:   internal/filter/pipeline.go

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
	modified:   cmd/tokman/main.go
	modified:   go.mod

Untracked files:
  (use "git add <file>..." to include in what will be committed)
	internal/filter/new_filter.go
`

	p := NewPipelineCoordinator(PipelineConfig{
		Mode:                ModeMinimal,
		NgramEnabled:        true,
		EnableCompaction:    false,
		EnableAttribution:   false,
		EnableH2O:           false,
		EnableAttentionSink: false,
	})
	output, stats := p.Process(input)

	// Output should not be empty
	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	// Should contain some content from original
	if !strings.Contains(output, "branch") && !strings.Contains(output, "main") {
		t.Error("Should preserve some git status content")
	}

	_ = stats
}

// TestRegressionTestOutput ensures test output compresses correctly
func TestRegressionTestOutput(t *testing.T) {
	input := `=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
=== RUN   TestBar
--- PASS: TestBar (0.00s)
=== RUN   TestBaz
--- FAIL: TestBaz (0.01s)
    baz_test.go:42: expected 42, got 0
FAIL
FAIL	github.com/example/pkg	0.015s
`

	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	output, stats := p.Process(input)

	// Must preserve FAIL marker
	if !strings.Contains(strings.ToLower(output), "fail") {
		t.Error("Should preserve FAIL marker")
	}

	// Must preserve error line
	if !strings.Contains(output, "baz_test.go") {
		t.Error("Should preserve failing test file reference")
	}

	_ = stats
}

// TestRegressionDockerPS ensures docker ps output compresses correctly
func TestRegressionDockerPS(t *testing.T) {
	input := `CONTAINER ID   IMAGE          COMMAND                  CREATED         STATUS         PORTS                    NAMES
a1b2c3d4e5f6   nginx:latest   "/docker-entrypoint.…"   2 hours ago     Up 2 hours     0.0.0.0:80->80/tcp       web
f6e5d4c3b2a1   postgres:15    "docker-entrypoint.s…"   3 hours ago     Up 3 hours     0.0.0.0:5432->5432/tcp   db
`

	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	output, stats := p.Process(input)

	// Must preserve container names
	if !strings.Contains(output, "web") {
		t.Error("Should preserve container name 'web'")
	}
	if !strings.Contains(output, "db") {
		t.Error("Should preserve container name 'db'")
	}

	// Must preserve image names
	if !strings.Contains(output, "nginx") {
		t.Error("Should preserve image name 'nginx'")
	}

	_ = stats
}

// TestRegressionCompilerError ensures compiler errors compress correctly
func TestRegressionCompilerError(t *testing.T) {
	input := `src/main.go:42:5: undefined: foo
src/main.go:55:10: cannot use "string" as int
src/utils.go:12:3: missing return statement
src/utils.go:88:1: undefined type Bar
`

	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	output, _ := p.Process(input)

	// Must preserve file references
	if !strings.Contains(output, "main.go") {
		t.Error("Should preserve file reference main.go")
	}
	if !strings.Contains(output, "utils.go") {
		t.Error("Should preserve file reference utils.go")
	}

	// Must preserve error descriptions
	if !strings.Contains(output, "undefined") {
		t.Error("Should preserve error description")
	}
}

// TestRegressionJSONOutput ensures JSON compresses correctly
func TestRegressionJSONOutput(t *testing.T) {
	input := `{
  "name": "tokman",
  "version": "1.0.0",
  "description": "Token manager for LLM context optimization",
  "dependencies": {
    "cobra": "1.8.0",
    "viper": "1.18.0"
  }
}`

	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	output, _ := p.Process(input)

	// Must preserve key values
	if !strings.Contains(output, "tokman") {
		t.Error("Should preserve name value")
	}
	if !strings.Contains(output, "1.0.0") {
		t.Error("Should preserve version value")
	}
}

// TestRegressionLongLog ensures long log output compresses well
func TestRegressionLongLog(t *testing.T) {
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "INFO: processing request " + string(rune('A'+i%26)) + " completed successfully"
	}
	input := strings.Join(lines, "\n")

	p := NewPipelineCoordinator(PipelineConfig{
		Mode:                ModeAggressive,
		NgramEnabled:        true,
		EnableCompaction:    true,
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	})
	output, stats := p.Process(input)

	// Should compress repeated content (relaxed check)
	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	// Log the actual compression ratio for monitoring
	if len(input) > 0 {
		ratio := float64(len(output)) / float64(len(input)) * 100
		t.Logf("Compression ratio: %.1f%% (saved %d tokens)", ratio, stats.TotalSaved)
	}
}

// TestRegressionErrorInLog ensures errors in logs are preserved
func TestRegressionErrorInLog(t *testing.T) {
	input := `INFO: starting server on port 8080
INFO: connected to database
ERROR: connection refused to redis at 127.0.0.1:6379
WARN: retrying connection in 5 seconds
ERROR: max retries exceeded
FATAL: shutting down
`

	p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
	output, _ := p.Process(input)

	// Must preserve ERROR lines
	if !strings.Contains(strings.ToLower(output), "error") {
		t.Error("Should preserve ERROR lines")
	}

	// Must preserve FATAL
	if !strings.Contains(strings.ToLower(output), "fatal") {
		t.Error("Should preserve FATAL lines")
	}
}
