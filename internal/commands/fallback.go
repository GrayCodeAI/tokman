package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/toml"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// FallbackHandler handles unknown commands by attempting TOML filter matching
type FallbackHandler struct {
	registry   *toml.FilterRegistry
	loader     *toml.Loader
	tracker    *tracking.Tracker
	runner     core.CommandRunner // T1: Injected command runner
	teeEnabled bool
	teeDir     string
}

// NewFallbackHandler creates a new fallback handler
func NewFallbackHandler() *FallbackHandler {
	loader := toml.GetLoader()

	// Get current working directory as project dir
	projectDir, _ := os.Getwd()

	registry, err := loader.LoadAll(projectDir)
	if err != nil {
		// Log warning but continue with empty registry
		fmt.Fprintf(os.Stderr, "Warning: failed to load TOML filters: %v\n", err)
		registry = toml.NewFilterRegistry()
	}

	return &FallbackHandler{
		registry:   registry,
		loader:     loader,
		tracker:    getGlobalTracker(),
		runner:     core.NewOSCommandRunner(), // T1: Use CommandRunner interface
		teeEnabled: true,
		teeDir:     getTeeDir(),
	}
}

// getGlobalTracker returns the global tracker instance from tracking package
func getGlobalTracker() *tracking.Tracker {
	return tracking.GetGlobalTracker()
}

// Handle attempts to handle an unknown command
// Returns: output, wasHandled, error
func (h *FallbackHandler) Handle(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, nil
	}

	command := strings.Join(args, " ")

	// Find matching TOML filter
	filename, filterName, config := h.registry.FindMatchingFilter(command)
	if config == nil {
		// No filter found - raw passthrough
		return h.rawPassthrough(args)
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "[TOML filter: %s/%s]\n", filename, filterName)
	}

	// Execute the command
	start := time.Now()
	output, exitCode, err := h.executeCommand(args)
	execTime := time.Since(start)

	if err != nil {
		// Save tee on failure
		if h.teeEnabled && len(output) > 500 {
			teePath := h.saveTee(args, output)
			output = output + fmt.Sprintf("\n[full output saved: %s]", teePath)
		}
		return output, true, err
	}

	// Apply multi-layer pipeline compression
	filtered := h.applyPipeline(output, config)

	// Record tracking
	if h.tracker != nil {
		originalTokens := core.EstimateTokens(output) // T22: Unified estimator
		filteredTokens := core.EstimateTokens(filtered)
		saved := originalTokens - filteredTokens
		if saved < 0 {
			saved = 0
		}

		record := &tracking.CommandRecord{
			Command:        command,
			OriginalTokens: originalTokens,
			FilteredTokens: filteredTokens,
			SavedTokens:    saved,
			ProjectPath:    getProjectPath(),
			ExecTimeMs:     execTime.Milliseconds(),
			Timestamp:      start,
			ParseSuccess:   exitCode == 0,
		}
		h.tracker.Record(record)
	}

	// If command failed, add tee hint
	if exitCode != 0 && h.teeEnabled && len(output) > 500 {
		teePath := h.saveTee(args, output)
		filtered = filtered + fmt.Sprintf("\n[full output saved: %s]", teePath)
	}

	return filtered, true, nil
}

// rawPassthrough executes command without filtering
func (h *FallbackHandler) rawPassthrough(args []string) (string, bool, error) {
	output, exitCode, err := h.executeCommand(args)

	// Record parse failure for future improvement
	if h.tracker != nil && len(args) > 0 {
		h.tracker.RecordParseFailure(strings.Join(args, " "), getProjectPath(), false)
	}

	// Save tee on failure
	if exitCode != 0 && h.teeEnabled && len(output) > 500 {
		teePath := h.saveTee(args, output)
		output = output + fmt.Sprintf("\n[full output saved: %s]", teePath)
	}

	return output, true, err
}

// executeCommand runs the command and captures output via CommandRunner (T1)
func (h *FallbackHandler) executeCommand(args []string) (string, int, error) {
	if len(args) == 0 {
		return "", 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, exitCode, err := h.runner.Run(ctx, args)
	return output, exitCode, err
}

// saveTee saves raw output to a file for recovery
func (h *FallbackHandler) saveTee(args []string, output string) string {
	if h.teeDir == "" {
		return ""
	}

	// Ensure tee directory exists
	if err := os.MkdirAll(h.teeDir, 0755); err != nil {
		return ""
	}

	// Generate filename
	timestamp := time.Now().Unix()
	slug := strings.ReplaceAll(strings.Join(args, "_"), "/", "_")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	filename := fmt.Sprintf("%d_%s.log", timestamp, slug)
	path := filepath.Join(h.teeDir, filename)

	// Write output
	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return ""
	}

	// Rotate old files (keep last 20)
	h.rotateTeeFiles()

	return path
}

// rotateTeeFiles removes old tee files, keeping last 20
func (h *FallbackHandler) rotateTeeFiles() {
	if h.teeDir == "" {
		return
	}

	entries, err := os.ReadDir(h.teeDir)
	if err != nil {
		return
	}

	// Remove files older than 20
	if len(entries) > 20 {
		// Sort by name (timestamp prefix)
		for i := 0; i < len(entries)-20; i++ {
			os.Remove(filepath.Join(h.teeDir, entries[i].Name()))
		}
	}
}

// getTeeDir returns the tee directory path
func getTeeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".local", "share", "tokman", "tee")
}

// getProjectPath returns the current project path
func getProjectPath() string {
	path, _ := os.Getwd()
	return path
}

// applyPipeline applies the multi-layer compression pipeline with preset support.
// T81/T90: Supports fast/balanced/full presets and early-exit.
func (h *FallbackHandler) applyPipeline(output string, tomlConfig *toml.FilterConfig) string {
	// Build pipeline configuration from CLI flags
	mode := filter.ModeMinimal
	if IsUltraCompact() {
		mode = filter.ModeAggressive
	}

	preset := GetLayerPreset()
	var cfg filter.PipelineConfig

	if preset != "" {
		// Use preset configuration (T90)
		cfg = filter.PresetConfig(filter.PipelinePreset(preset), mode)
		cfg.QueryIntent = GetQueryIntent()
		cfg.Budget = GetTokenBudget()
		cfg.LLMEnabled = IsLLMEnabled()
	} else {
		// Use full pipeline (default)
		cfg = filter.PipelineConfig{
			Mode:                mode,
			QueryIntent:         GetQueryIntent(),
			Budget:              GetTokenBudget(),
			LLMEnabled:          IsLLMEnabled(),
			SessionTracking:     true,
			NgramEnabled:        true,
			MultiFileEnabled:    true,
			EnableCompaction:    true,
			EnableAttribution:   true,
			EnableH2O:           true,
			EnableAttentionSink: true,
		}
	}

	// Create and run the pipeline
	pipeline := filter.NewPipelineCoordinator(cfg)
	filtered, stats := pipeline.Process(output)

	// Log compression stats in verbose mode
	if IsVerbose() && stats.TotalSaved > 0 {
		fmt.Fprintf(os.Stderr, "[pipeline: %d -> %d tokens, %.1f%% saved]\n",
			stats.OriginalTokens, stats.FinalTokens, stats.ReductionPercent)
	}

	// If TOML has specific rules, apply them as a final pass
	if tomlConfig != nil && (len(tomlConfig.Replace) > 0 || len(tomlConfig.MatchOutput) > 0 || len(tomlConfig.StripLinesMatching) > 0) {
		engine := toml.NewTOMLFilterEngine(tomlConfig)
		if result, _ := engine.Apply(filtered, mode); result != "" {
			filtered = result
		}
	}

	return filtered
}

// Global fallback handler
var globalFallback *FallbackHandler

// GetFallback returns the global fallback handler
func GetFallback() *FallbackHandler {
	if globalFallback == nil {
		globalFallback = NewFallbackHandler()
	}
	return globalFallback
}
