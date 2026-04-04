package filter

import (
	"strings"
	"testing"
)

func TestPipelineManager_BasicProcess(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens:   2000000,
		ChunkSize:          100000,
		StreamThreshold:    500000,
		TeeOnFailure:       true,
		FailSafeMode:       true,
		ValidateOutput:     true,
		ShortCircuitBudget: true,
		CacheEnabled:       true,
		CacheMaxSize:       100,
		PipelineCfg: PipelineConfig{
			Mode:            ModeMinimal,
			SessionTracking: true,
			NgramEnabled:    true,
		},
	})

	input := "This is test content with some repeated repeated repeated text."
	ctx := CommandContext{Command: "test"}

	result, err := manager.Process(input, ModeMinimal, ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Output == "" {
		t.Error("output should not be empty")
	}

	if result.OriginalTokens <= 0 {
		t.Error("original tokens should be positive")
	}

	if result.SavedTokens < 0 {
		t.Error("saved tokens should be non-negative")
	}
}

func TestPipelineManager_LargeContext(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens: 2000000,
		ChunkSize:        10000, // Small chunk for testing
		StreamThreshold:  1000,  // Low threshold to trigger streaming
		TeeOnFailure:     true,
		FailSafeMode:     true,
		ValidateOutput:   true,
		CacheEnabled:     false,
		PipelineCfg: PipelineConfig{
			Mode:            ModeAggressive,
			SessionTracking: true,
			NgramEnabled:    true,
		},
	})

	// Create large input (simulate 100K tokens)
	line := "This is a test line with some content that will be processed.\n"
	largeInput := strings.Repeat(line, 3000) // ~150K tokens

	ctx := CommandContext{Command: "test"}

	result, err := manager.Process(largeInput, ModeAggressive, ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Output == "" {
		t.Error("output should not be empty")
	}

	// Should have used chunked processing
	if result.Chunks == 0 {
		t.Error("should have processed in chunks for large input")
	}

	// Verify compression occurred
	if len(result.Output) >= len(largeInput) {
		t.Error("output should be smaller than input")
	}
}

func TestPipelineManager_BudgetEnforcement(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens:   2000000,
		ChunkSize:          100000,
		StreamThreshold:    500000,
		TeeOnFailure:       true,
		FailSafeMode:       true,
		ValidateOutput:     true,
		ShortCircuitBudget: true,
		CacheEnabled:       false,
		PipelineCfg: PipelineConfig{
			Mode:            ModeAggressive,
			Budget:          100,
			SessionTracking: true,
		},
	})

	// Create input larger than budget
	input := strings.Repeat("Test content line number that should be compressed.\n", 100)

	ctx := CommandContext{Command: "test"}

	result, err := manager.ProcessWithBudget(input, ModeAggressive, 100, ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Budget enforcement should kick in
	tokens := EstimateTokens(result.Output)
	if tokens > 150 { // Allow some slack
		t.Errorf("output tokens (%d) should be close to budget (100)", tokens)
	}
}

func TestPipelineManager_QueryAware(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens: 2000000,
		ChunkSize:        100000,
		StreamThreshold:  500000,
		TeeOnFailure:     true,
		FailSafeMode:     true,
		ValidateOutput:   true,
		CacheEnabled:     false,
		PipelineCfg: PipelineConfig{
			Mode:            ModeMinimal,
			SessionTracking: true,
			NgramEnabled:    true,
		},
	})

	input := `# Project README
This is a project about authentication.
Error: Failed to connect to database at line 42.
Warning: Deprecated function call.
The authentication module handles user login.`

	ctx := CommandContext{
		Command: "git",
		Intent:  "debug",
	}

	result, err := manager.ProcessWithQuery(input, ModeMinimal, "debug authentication error", ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify output is valid
	if result.Output == "" {
		t.Error("output should not be empty")
	}

	// Verify compression occurred
	if len(result.Output) > len(input) {
		t.Error("output should not be larger than input")
	}
}

func TestPipelineManager_ProcessWithQueryUpdatesFilters(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens: 2000000,
		ChunkSize:        100000,
		StreamThreshold:  500000,
		TeeOnFailure:     false,
		FailSafeMode:     true,
		ValidateOutput:   false,
		CacheEnabled:     false,
		PipelineCfg: PipelineConfig{
			Mode:              ModeMinimal,
			QueryIntent:       "initial",
			EnableGoalDriven:  true,
			EnableContrastive: true,
		},
	})

	input := strings.Join([]string{
		"authentication failure at login boundary",
		"payment timeout during invoice capture",
		"authentication token refresh failed",
		"payment gateway returned 502",
	}, "\n")
	ctx := CommandContext{Command: "test"}

	if _, err := manager.ProcessWithQuery(input, ModeMinimal, "authentication failure", ctx); err != nil {
		t.Fatalf("first ProcessWithQuery() error = %v", err)
	}
	if manager.coordinator.goalDrivenFilter == nil || manager.coordinator.goalDrivenFilter.goal != "authentication failure" {
		t.Fatal("goal-driven filter was not updated to first query")
	}
	if manager.coordinator.contrastiveFilter == nil || manager.coordinator.contrastiveFilter.question != "authentication failure" {
		t.Fatal("contrastive filter was not updated to first query")
	}

	if _, err := manager.ProcessWithQuery(input, ModeMinimal, "payment timeout", ctx); err != nil {
		t.Fatalf("second ProcessWithQuery() error = %v", err)
	}
	if manager.coordinator.goalDrivenFilter == nil || manager.coordinator.goalDrivenFilter.goal != "payment timeout" {
		t.Fatal("goal-driven filter was not refreshed for second query")
	}
	if manager.coordinator.contrastiveFilter == nil || manager.coordinator.contrastiveFilter.question != "payment timeout" {
		t.Fatal("contrastive filter was not refreshed for second query")
	}
}

func TestPipelineManager_FailSafe(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens: 2000000,
		ChunkSize:        100000,
		StreamThreshold:  500000,
		TeeOnFailure:     true,
		FailSafeMode:     true,
		ValidateOutput:   true,
		CacheEnabled:     false,
		PipelineCfg: PipelineConfig{
			Mode:            ModeMinimal,
			SessionTracking: true,
		},
	})

	// Input with balanced structure
	input := "Test input with (balanced) brackets [like this] {and this}."

	ctx := CommandContext{Command: "test"}

	result, err := manager.Process(input, ModeMinimal, ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// With fail-safe mode, output should be valid
	if result.Output == "" && input != "" {
		t.Error("fail-safe should ensure non-empty output")
	}
}

func TestPipelineManager_Cache(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens: 2000000,
		ChunkSize:        100000,
		StreamThreshold:  500000,
		TeeOnFailure:     false,
		FailSafeMode:     true,
		ValidateOutput:   false,
		CacheEnabled:     true,
		CacheMaxSize:     100,
		PipelineCfg: PipelineConfig{
			Mode:            ModeMinimal,
			SessionTracking: true,
			NgramEnabled:    true,
		},
	})

	input := "Cache test content with unique identifier 12345."
	ctx := CommandContext{Command: "test"}

	// First call - should not be cache hit
	result1, err := manager.Process(input, ModeMinimal, ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if result1.CacheHit {
		t.Error("first call should not be cache hit")
	}

	// Second call - should be cache hit
	result2, err := manager.Process(input, ModeMinimal, ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if !result2.CacheHit {
		t.Error("second call should be cache hit")
	}

	// Outputs should match
	if result1.Output != result2.Output {
		t.Error("cached output should match original")
	}
}

func TestPipelineManager_CacheKeyIncludesBudget(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens: 2000000,
		ChunkSize:        100000,
		StreamThreshold:  500000,
		TeeOnFailure:     false,
		FailSafeMode:     true,
		ValidateOutput:   false,
		CacheEnabled:     true,
		CacheMaxSize:     100,
		PipelineCfg: PipelineConfig{
			Mode:            ModeAggressive,
			SessionTracking: true,
		},
	})

	input := strings.Repeat("Budget-sensitive compression line.\n", 50)
	ctx := CommandContext{Command: "test"}

	result1, err := manager.ProcessWithBudget(input, ModeAggressive, 400, ctx)
	if err != nil {
		t.Fatalf("ProcessWithBudget() first call error = %v", err)
	}
	if result1.CacheHit {
		t.Fatal("first budgeted call should not be a cache hit")
	}

	result2, err := manager.ProcessWithBudget(input, ModeAggressive, 50, ctx)
	if err != nil {
		t.Fatalf("ProcessWithBudget() second call error = %v", err)
	}
	if result2.CacheHit {
		t.Fatal("different budget should not reuse prior cached result")
	}
}

func TestPipelineManager_Validation(t *testing.T) {
	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens: 2000000,
		ChunkSize:        100000,
		StreamThreshold:  500000,
		TeeOnFailure:     true,
		FailSafeMode:     true,
		ValidateOutput:   true,
		CacheEnabled:     false,
		PipelineCfg: PipelineConfig{
			Mode:            ModeMinimal,
			SessionTracking: true,
		},
	})

	// Error output context
	input := `Building project...
Error: Compilation failed at main.go:42
Stack trace: goroutine 1 [running]
main.main()
	/home/user/project/main.go:42 +0x123`

	ctx := CommandContext{
		Command:    "go",
		Subcommand: "build",
		ExitCode:   1,
		IsError:    true,
	}

	result, err := manager.Process(input, ModeMinimal, ctx)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Validation should pass
	if !result.Validated {
		t.Error("validation should pass for valid output")
	}

	// Error should be preserved
	if !strings.Contains(result.Output, "Error:") {
		t.Error("errors should be preserved")
	}
}

func TestCompressionCache(t *testing.T) {
	cache := NewCompressionCache(3)

	// Add entries
	cache.Set("key1", &CachedResult{Output: "output1", Tokens: 10})
	cache.Set("key2", &CachedResult{Output: "output2", Tokens: 20})
	cache.Set("key3", &CachedResult{Output: "output3", Tokens: 30})

	// Verify size
	if cache.Size() != 3 {
		t.Errorf("cache size should be 3, got %d", cache.Size())
	}

	// Retrieve entries
	if result, ok := cache.Get("key1"); !ok || result.Output != "output1" {
		t.Error("should retrieve key1")
	}

	// Add entry beyond capacity (should evict oldest)
	cache.Set("key4", &CachedResult{Output: "output4", Tokens: 40})

	if cache.Size() > 3 {
		t.Error("cache should not exceed max size")
	}
}

func TestCommandContext(t *testing.T) {
	ctx := CommandContext{
		Command:    "git",
		Subcommand: "status",
		ExitCode:   0,
		Intent:     "review",
		IsTest:     false,
		IsBuild:    false,
		IsError:    false,
	}

	if ctx.Command != "git" {
		t.Error("command should be git")
	}

	if ctx.Intent != "review" {
		t.Error("intent should be review")
	}
}
