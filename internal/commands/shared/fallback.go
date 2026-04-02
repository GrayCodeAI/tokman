package shared

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/toml"
	"github.com/GrayCodeAI/tokman/internal/tracking"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

// Fallback handler for TOML-based command filtering.
// This is the main entry point for commands not explicitly defined in Go.

// FallbackHandler handles commands via TOML filter system.
type FallbackHandler struct {
	registry   *toml.FilterRegistry
	loader     *toml.Loader
	tracker    *tracking.Tracker
	runner     core.CommandRunner
	teeEnabled bool
	teeDir     string
}

var (
	globalFallback *FallbackHandler
	fallbackOnce   sync.Once
)

// GetFallback returns the global fallback handler (singleton).
func GetFallback() *FallbackHandler {
	fallbackOnce.Do(func() {
		globalFallback = NewFallbackHandler()
	})
	return globalFallback
}

// NewFallbackHandler creates a new fallback handler.
func NewFallbackHandler() *FallbackHandler {
	loader := toml.GetLoader()

	projectDir, _ := os.Getwd()

	registry, err := loader.LoadAll(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load TOML filters: %v\n", err)
		registry = toml.NewFilterRegistry()
	}

	return &FallbackHandler{
		registry:   registry,
		loader:     loader,
		tracker:    getGlobalTracker(),
		runner:     core.NewOSCommandRunner(),
		teeEnabled: true,
		teeDir:     getTeeDir(),
	}
}

// Handle processes a command through the TOML filter system.
func (h *FallbackHandler) Handle(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, nil
	}

	command := strings.Join(args, " ")

	filename, filterName, config := h.registry.FindMatchingFilter(command)
	if config == nil {
		return h.rawPassthrough(args)
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "[TOML filter: %s/%s]\n", filename, filterName)
	}

	start := time.Now()
	output, exitCode, err := h.executeCommand(args)
	execTime := time.Since(start)

	if err != nil {
		if h.teeEnabled && len(output) > 500 {
			teePath := h.saveTee(args, output)
			output = output + fmt.Sprintf("\n[full output saved: %s]", teePath)
		}
		return output, true, err
	}

	filtered := h.applyPipeline(output, config)

	if h.tracker != nil {
		originalTokens := core.EstimateTokens(output)
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
			// AI Agent attribution from environment
			AgentName:   os.Getenv("TOKMAN_AGENT"),
			ModelName:   os.Getenv("TOKMAN_MODEL"),
			Provider:    os.Getenv("TOKMAN_PROVIDER"),
			ModelFamily: utils.GetModelFamily(os.Getenv("TOKMAN_MODEL")),
		}
		h.tracker.Record(record)
	}

	if exitCode != 0 && h.teeEnabled && len(output) > 500 {
		teePath := h.saveTee(args, output)
		filtered = filtered + fmt.Sprintf("\n[full output saved: %s]", teePath)
	}

	return filtered, true, nil
}

func (h *FallbackHandler) rawPassthrough(args []string) (string, bool, error) {
	output, exitCode, err := h.executeCommand(args)

	// Apply remote compression even without TOML filter
	if IsRemoteMode() && len(output) > 100 {
		mode := filter.ModeMinimal
		if IsUltraCompact() {
			mode = filter.ModeAggressive
		}
		filtered, saved, rerr := RemoteCompress(output, string(mode), GetTokenBudget())
		if rerr == nil {
			if IsVerbose() && saved > 0 {
				fmt.Fprintf(os.Stderr, "[remote compression: %d tokens saved]\n", saved)
			}
			output = filtered
		} else if IsVerbose() {
			fmt.Fprintf(os.Stderr, "[remote compression failed: %v]\n", rerr)
		}
	}

	if h.tracker != nil && len(args) > 0 {
		h.tracker.RecordParseFailure(strings.Join(args, " "), "no filter matched", err == nil)
	}

	if exitCode != 0 && h.teeEnabled && len(output) > 500 {
		teePath := h.saveTee(args, output)
		output = output + fmt.Sprintf("\n[full output saved: %s]", teePath)
	}

	return output, true, err
}

func (h *FallbackHandler) executeCommand(args []string) (string, int, error) {
	if len(args) == 0 {
		return "", 0, nil
	}

	if err := SanitizeArgs(args); err != nil {
		return "", 1, fmt.Errorf("invalid arguments: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, exitCode, err := h.runner.Run(ctx, args)
	return output, exitCode, err
}

func (h *FallbackHandler) saveTee(args []string, output string) string {
	if h.teeDir == "" {
		return ""
	}

	if err := os.MkdirAll(h.teeDir, 0700); err != nil {
		return ""
	}

	timestamp := time.Now().Unix()
	slug := strings.ReplaceAll(strings.Join(args, "_"), "/", "_")
	slug = strings.ReplaceAll(slug, "\x00", "")
	slug = strings.ReplaceAll(slug, "\n", "_")
	slug = strings.ReplaceAll(slug, "\r", "_")
	slug = strings.ReplaceAll(slug, "\t", "_")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	filename := fmt.Sprintf("%d_%s.log", timestamp, slug)
	path := filepath.Join(h.teeDir, filename)

	if err := os.WriteFile(path, []byte(output), 0600); err != nil {
		return ""
	}

	h.rotateTeeFiles()

	return path
}

func (h *FallbackHandler) rotateTeeFiles() {
	if h.teeDir == "" {
		return
	}

	entries, err := os.ReadDir(h.teeDir)
	if err != nil {
		return
	}

	if len(entries) > 20 {
		for i := 0; i < len(entries)-20; i++ {
			if err := os.Remove(filepath.Join(h.teeDir, entries[i].Name())); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to remove tee file %s: %v\n", entries[i].Name(), err)
			}
		}
	}
}

func (h *FallbackHandler) applyPipeline(output string, tomlConfig *toml.TOMLFilterRule) string {
	mode := filter.ModeMinimal
	if IsUltraCompact() {
		mode = filter.ModeAggressive
	}

	// Remote mode: use gRPC compression service
	if IsRemoteMode() {
		filtered, saved, err := RemoteCompress(output, string(mode), GetTokenBudget())
		if err != nil {
			if IsVerbose() {
				fmt.Fprintf(os.Stderr, "[remote compression failed: %v, falling back to local]\n", err)
			}
			// Fall through to local processing
		} else {
			if IsVerbose() && saved > 0 {
				fmt.Fprintf(os.Stderr, "[remote pipeline: %d tokens saved]\n", saved)
			}
			return filtered
		}
	}

	preset := GetLayerPreset()
	profile := GetLayerProfile()
	var cfg filter.PipelineConfig

	if profile != "" {
		cfg = filter.ProfileConfig(filter.Profile(profile), mode)
		cfg.QueryIntent = GetQueryIntent()
		cfg.Budget = GetTokenBudget()
		cfg.LLMEnabled = IsLLMEnabled()
	} else if preset != "" {
		cfg = filter.PresetConfig(filter.PipelinePreset(preset), mode)
		cfg.QueryIntent = GetQueryIntent()
		cfg.Budget = GetTokenBudget()
		cfg.LLMEnabled = IsLLMEnabled()
	} else {
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
			EnableTOMLFilter:    true,
		}
	}

	pipeline := filter.NewPipelineCoordinator(cfg)

	// Integrate TOML filter into pipeline (Layer 0) instead of applying separately
	if tomlConfig != nil && (len(tomlConfig.Replace) > 0 || len(tomlConfig.MatchOutput) > 0 || len(tomlConfig.StripLinesMatching) > 0) {
		tomlWrapper := toml.NewTOMLFilterWrapper("toml_pre_filter", tomlConfig)
		pipeline.SetTOMLFilter(tomlWrapper, "toml_pre_filter")
	}

	filtered, stats := pipeline.Process(output)

	if IsVerbose() && stats.TotalSaved > 0 {
		fmt.Fprintf(os.Stderr, "[pipeline: %d -> %d tokens, %.1f%% saved]\n",
			stats.OriginalTokens, stats.FinalTokens, stats.ReductionPercent)
	}

	if IsReversibleEnabled() && len(output) > 100 {
		store := filter.NewReversibleStore()
		layerStats := make(map[string]int)
		for k, v := range stats.LayerStats {
			layerStats[k] = v.TokensSaved
		}
		hash := store.Store("", output, filtered, string(mode), GetTokenBudget(), layerStats)
		if !IsQuietMode() {
			fmt.Fprintf(os.Stderr, "[reversible: %s] ", hash)
		}
	}

	return filtered
}

// Helper functions (package-level)

func getGlobalTracker() *tracking.Tracker {
	return tracking.GetGlobalTracker()
}

func getTeeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".local", "share", "tokman", "tee")
}

func getProjectPath() string {
	path, _ := os.Getwd()
	return path
}

// Root command storage (for CLI integration)

var rootCmd any

// SetRootCmd stores the root command reference.
func SetRootCmd(cmd any) {
	rootCmd = cmd
}

// RootCmd returns the stored root command.
func RootCmd() any {
	return rootCmd
}
