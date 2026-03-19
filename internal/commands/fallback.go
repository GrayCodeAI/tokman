package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/toml"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// FallbackHandler handles unknown commands by attempting TOML filter matching
type FallbackHandler struct {
	registry   *toml.FilterRegistry
	loader     *toml.Loader
	tracker    *tracking.Tracker
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

	// Apply TOML filter
	engine := toml.NewTOMLFilterEngine(config)
	filtered, _ := engine.Apply(output, filter.ModeMinimal)

	// Record tracking
	if h.tracker != nil {
		originalTokens := filter.EstimateTokens(output)
		filteredTokens := filter.EstimateTokens(filtered)
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

// executeCommand runs the command and captures output
func (h *FallbackHandler) executeCommand(args []string) (string, int, error) {
	if len(args) == 0 {
		return "", 0, nil
	}

	// Resolve the command path
	cmdPath, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Sprintf("command not found: %s", args[0]), 127, err
	}

	// Execute the command
	cmd := exec.Command(cmdPath, args[1:]...)
	cmd.Env = os.Environ()
	
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return string(output), exitCode, err
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

// Global fallback handler
var globalFallback *FallbackHandler

// GetFallback returns the global fallback handler
func GetFallback() *FallbackHandler {
	if globalFallback == nil {
		globalFallback = NewFallbackHandler()
	}
	return globalFallback
}
