package filter

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestConcurrentPipeline runs 10 goroutines each calling Process() on a shared
// PipelineCoordinator concurrently. Should pass under go test -race.
func TestConcurrentPipeline(t *testing.T) {
	t.Parallel()

	coordinator := NewPipelineCoordinator(PipelineConfig{
		Mode:              ModeMinimal,
		SessionTracking:   true,
		NgramEnabled:      true,
		EnableEntropy:     true,
		EnableH2O:         true,
		EnableAttribution: true,
	})

	input := "concurrent pipeline test content with repeated repeated tokens and noise.\n"

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			in := fmt.Sprintf("goroutine %d: %s", i, input)
			out, stats := coordinator.Process(in)
			if out == "" && in != "" {
				t.Errorf("goroutine %d: got empty output for non-empty input", i)
			}
			if stats.OriginalTokens < 0 {
				t.Errorf("goroutine %d: negative original tokens", i)
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentAuditLog records events from 5 goroutines simultaneously.
// Should pass under go test -race.
func TestConcurrentAuditLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.jsonl")

	al, err := NewAuditLog(logPath)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	defer al.Close()

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := al.RecordCompression(
					fmt.Sprintf("filter_%d", i),
					ModeMinimal,
					100+i*j,
					80+i*j,
				); err != nil {
					t.Errorf("goroutine %d record %d: %v", i, j, err)
				}
			}
		}()
	}

	wg.Wait()

	// Verify the log file was written.
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat audit log: %v", err)
	}
	if info.Size() == 0 {
		t.Error("audit log should not be empty after concurrent writes")
	}
}

// TestConcurrentSketchStore exercises concurrent Revive/Apply on a SketchStoreFilter.
// Should pass under go test -race.
func TestConcurrentSketchStore(t *testing.T) {
	t.Parallel()

	store := NewSketchStoreFilter()

	// Seed the store with some sketches via Apply.
	seed := "sketch store concurrent test with varied content for hashing purposes.\n"
	store.Apply(seed, ModeAggressive)

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			in := fmt.Sprintf("goroutine %d: adding content to sketch store for concurrent testing", i)
			// Concurrent Apply (adds sketches).
			out, saved := store.Apply(in, ModeAggressive)
			_ = out
			_ = saved

			// Concurrent reads of all sketches.
			sketches := store.GetAllSketches()
			for hash := range sketches {
				_, _ = store.Revive(hash)
			}

			// Concurrent stats read.
			_ = store.GetStats()
		}()
	}

	wg.Wait()
}

// TestConcurrentManager runs concurrent Process calls on a filter Manager (PipelineManager).
// Should pass under go test -race.
func TestConcurrentManager(t *testing.T) {
	t.Parallel()

	manager := NewPipelineManager(ManagerConfig{
		MaxContextTokens:   2_000_000,
		ChunkSize:          100_000,
		StreamThreshold:    500_000,
		TeeOnFailure:       false,
		FailSafeMode:       true,
		ValidateOutput:     false,
		ShortCircuitBudget: false,
		CacheEnabled:       true,
		CacheMaxSize:       50,
		PipelineCfg: PipelineConfig{
			Mode:              ModeMinimal,
			SessionTracking:   true,
			NgramEnabled:      true,
			EnableEntropy:     true,
			EnableH2O:         true,
			EnableAttribution: true,
		},
	})

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			input := fmt.Sprintf("manager concurrent test goroutine %d: log line with repeated repeated content noise.\n", i)
			ctx := CommandContext{
				Command:  "test",
				Intent:   "debug",
				ExitCode: 0,
			}
			result, err := manager.Process(input, ModeMinimal, ctx)
			if err != nil {
				t.Errorf("goroutine %d: Process error: %v", i, err)
				return
			}
			if result == nil {
				t.Errorf("goroutine %d: nil result", i)
				return
			}
			if result.Output == "" && input != "" {
				t.Errorf("goroutine %d: empty output for non-empty input", i)
			}
		}()
	}

	wg.Wait()
}
