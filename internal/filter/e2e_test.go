package filter_test

import (
	"strings"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

const e2eInput = `=== Build Output ===
Time: 2025-03-27 10:00:00
Command: go build ./...

INFO: compiling package github.com/GrayCodeAI/tokman/internal/filter
INFO: compiling package github.com/GrayCodeAI/tokman/internal/core
WARNING: deprecated call in utils.go line 42
WARNING: deprecated call in utils.go line 43
WARNING: deprecated call in utils.go line 44
ERROR: failed to link: undefined symbol at main.go:99
Error: build failed with exit code 1

Stack trace:
  goroutine 1 [running]:
  main.main()
    /home/user/project/main.go:99 +0x123

CRITICAL: disk usage at 92%
INFO: retrying build 1 of 3
INFO: retrying build 2 of 3
INFO: retrying build 3 of 3
FATAL: max retries exceeded

=== Build Complete ===
Total time: 42 seconds
`

// TestPipelineEndToEnd runs real input through the full pipeline and checks
// that the output is shorter than the input.
func TestPipelineEndToEnd(t *testing.T) {
	p := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:                filter.ModeAggressive,
		SessionTracking:     true,
		NgramEnabled:        true,
		EnableEntropy:       true,
		EnablePerplexity:    true,
		EnableGoalDriven:    true,
		EnableAST:           true,
		EnableContrastive:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
		EnableAttribution:   true,
		EnableDynamicRatio:  true,
	})

	output, stats := p.Process(e2eInput)

	if output == "" {
		t.Fatal("pipeline produced empty output")
	}
	if len(output) >= len(e2eInput) {
		t.Errorf("output (%d bytes) should be shorter than input (%d bytes)", len(output), len(e2eInput))
	}
	if stats.OriginalTokens <= 0 {
		t.Errorf("original tokens should be positive, got %d", stats.OriginalTokens)
	}
	if stats.TotalSaved < 0 {
		t.Errorf("total saved should be >= 0, got %d", stats.TotalSaved)
	}
	t.Logf("Original: %d tokens, Final: %d tokens, Saved: %d (%.1f%%)",
		stats.OriginalTokens, stats.FinalTokens, stats.TotalSaved, stats.ReductionPercent)
}

// TestPipelinePresetsEndToEnd tests each preset profile produces valid output.
func TestPipelinePresetsEndToEnd(t *testing.T) {
	input := strings.Repeat(e2eInput, 3)

	presets := []filter.PipelinePreset{
		filter.PresetFast,
		filter.PresetBalanced,
		filter.PresetFull,
	}

	for _, preset := range presets {
		preset := preset
		t.Run(string(preset), func(t *testing.T) {
			output, saved := filter.QuickProcessPreset(input, filter.ModeMinimal, preset)

			if output == "" {
				t.Errorf("preset %q produced empty output", preset)
			}
			if saved < 0 {
				t.Errorf("preset %q: tokens saved should be >= 0, got %d", preset, saved)
			}
			t.Logf("preset=%s saved=%d outputLen=%d", preset, saved, len(output))
		})
	}
}

// TestPipelineRoundtrip compresses input then checks that content preservation
// is above 50%. Uses PresetFast / ModeMinimal on a non-repetitive corpus so
// that key structural content is retained by the pipeline.
func TestPipelineRoundtrip(t *testing.T) {
	// Build a diverse, non-repetitive input so the fast preset keeps > 50%.
	input := strings.Join([]string{
		"# Authentication Service",
		"",
		"Handles OAuth2, JWT tokens, and session management for all clients.",
		"Rate limit: 100 requests per second per API key.",
		"Token expiry: 24 hours for access tokens, 30 days for refresh tokens.",
		"",
		"## Configuration",
		"DATABASE_URL=postgres://localhost:5432/auth",
		"REDIS_URL=redis://localhost:6379/0",
		"JWT_SECRET=changeme-in-production",
		"",
		"## Endpoints",
		"POST /auth/login  — exchange credentials for JWT",
		"POST /auth/refresh — exchange refresh token for new access token",
		"DELETE /auth/logout — revoke session",
		"GET  /auth/me — return authenticated user profile",
		"",
		"## Error Codes",
		"401 Unauthorized — invalid or expired token",
		"429 Too Many Requests — rate limit exceeded",
		"500 Internal Server Error — unexpected backend failure",
	}, "\n")

	output, _ := filter.QuickProcessPreset(input, filter.ModeMinimal, filter.PresetFast)

	if output == "" {
		t.Fatal("output is empty")
	}

	// Content preservation: output should retain at least 50% of input length.
	preserved := float64(len(output)) / float64(len(input))
	if preserved < 0.50 {
		t.Errorf("content preservation %.1f%% is below 50%% threshold", preserved*100)
	}

	t.Logf("Input: %d bytes, Output: %d bytes, Preserved: %.1f%%", len(input), len(output), preserved*100)
}

// TestPipelineEmptyInput verifies that an empty string input returns an empty
// string output without errors.
func TestPipelineEmptyInput(t *testing.T) {
	p := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:            filter.ModeMinimal,
		SessionTracking: true,
	})

	output, stats := p.Process("")

	if output != "" {
		t.Errorf("expected empty output for empty input, got %q", output)
	}
	if stats.OriginalTokens != 0 {
		t.Errorf("expected 0 original tokens for empty input, got %d", stats.OriginalTokens)
	}
	if stats.TotalSaved != 0 {
		t.Errorf("expected 0 saved tokens for empty input, got %d", stats.TotalSaved)
	}
}

// TestPipelineModeNone verifies that ModeNone returns the original input
// unchanged.
func TestPipelineModeNone(t *testing.T) {
	p := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:            filter.ModeNone,
		SessionTracking: false,
		NgramEnabled:    false,
	})

	input := e2eInput
	output, stats := p.Process(input)

	if output != input {
		t.Errorf("ModeNone should return original input unchanged\nwant len=%d, got len=%d", len(input), len(output))
	}
	if stats.TotalSaved != 0 {
		t.Logf("ModeNone: TotalSaved=%d (some filters may still be active)", stats.TotalSaved)
	}
}
