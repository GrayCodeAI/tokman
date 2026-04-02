// Package reversible provides pipeline integration for reversible compression.
package reversible

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Pipeline integrates reversible compression with command execution.
type Pipeline struct {
	store      Store
	classifier Classifier
	config     Config
}

// NewPipeline creates a new reversible compression pipeline.
func NewPipeline(store Store, config Config) *Pipeline {
	classifier := &SimpleClassifier{}
	return &Pipeline{
		store:      store,
		classifier: classifier,
		config:     config,
	}
}

// Execute runs a command with reversible compression.
func (p *Pipeline) Execute(command string, args []string, input io.Reader) (string, *Entry, error) {
	start := time.Now()

	// Run the command
	cmd := exec.Command(command, args...)
	if input != nil {
		cmd.Stdin = input
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	duration := time.Since(start)

	if err != nil && len(output) == 0 {
		return "", nil, fmt.Errorf("command failed: %w", err)
	}

	// Classify content
	classification := p.classifier.Classify(outputStr)

	// Create entry
	entry := &Entry{
		Original:       outputStr,
		Command:        fmt.Sprintf("%s %s", command, strings.Join(args, " ")),
		ContentType:    classification.Type,
		CompressionAlg: p.config.DefaultAlgorithm,
		CreatedAt:      start,
		AccessedAt:     start,
		SizeOriginal:   int64(len(outputStr)),
	}

	// Store entry
	hash, err := p.store.Save(entry)
	if err != nil {
		// Return output even if storage fails
		return outputStr, nil, err
	}

	entry.Hash = hash

	// Record in command history if SQLiteStore
	if sqliteStore, ok := p.store.(*SQLiteStore); ok {
		_ = sqliteStore.RecordCommand(&CommandRecord{
			Command:    entry.Command,
			Hash:       hash,
			Timestamp:  start,
			Duration:   duration,
			Compressed: true,
		})
	}

	// Return output with marker
	marker := Marker{Hash: hash[:16]}
	markedOutput := fmt.Sprintf("%s\n\n[Compressed: %s]", outputStr, marker.String())

	return markedOutput, entry, nil
}

// Restore retrieves original content from a marker.
func (p *Pipeline) Restore(markerStr string) (*Entry, error) {
	marker, err := ParseMarker(markerStr)
	if err != nil {
		// Try to extract marker from text
		if idx := strings.Index(markerStr, "[rewind:"); idx >= 0 {
			endIdx := strings.Index(markerStr[idx:], "]")
			if endIdx > 0 {
				marker, err = ParseMarker(markerStr[idx : idx+endIdx+1])
			}
		}
		if err != nil {
			return nil, err
		}
	}

	return p.store.Retrieve(marker.Hash)
}

// ProcessOutput processes output text and replaces markers with original content.
func (p *Pipeline) ProcessOutput(output string) (string, error) {
	var result strings.Builder
	lastIdx := 0

	for {
		idx := strings.Index(output[lastIdx:], "[rewind:")
		if idx < 0 {
			result.WriteString(output[lastIdx:])
			break
		}
		idx += lastIdx

		endIdx := strings.Index(output[idx:], "]")
		if endIdx < 0 {
			result.WriteString(output[lastIdx:])
			break
		}
		endIdx += idx + 1

		// Write text before marker
		result.WriteString(output[lastIdx:idx])

		// Try to restore
		markerStr := output[idx:endIdx]
		if entry, err := p.Restore(markerStr); err == nil {
			result.WriteString(entry.Original)
		} else {
			// Keep marker if restore fails
			result.WriteString(markerStr)
		}

		lastIdx = endIdx
	}

	return result.String(), nil
}

// ShouldCompress determines if content should be compressed based on size and type.
func (p *Pipeline) ShouldCompress(content string, cmd string) bool {
	// Minimum size threshold (1KB)
	if len(content) < 1024 {
		return false
	}

	// Maximum size threshold (100MB)
	if len(content) > 100*1024*1024 {
		return false
	}

	// Classify content
	classification := p.classifier.Classify(content)

	// Don't compress if confidence is low and content is small
	if classification.Confidence < 0.5 && len(content) < 10*1024 {
		return false
	}

	return true
}

// CLI provides command-line interface for reversible compression.
type CLI struct {
	pipeline *Pipeline
}

// NewCLI creates a new CLI.
func NewCLI(store Store, config Config) *CLI {
	return &CLI{
		pipeline: NewPipeline(store, config),
	}
}

// StoreCommand handles the `rewind store` command.
func (c *CLI) StoreCommand(content string, command string) (string, error) {
	entry := &Entry{
		Original:       content,
		Command:        command,
		CreatedAt:      time.Now(),
		SizeOriginal:   int64(len(content)),
	}

	hash, err := c.pipeline.store.Save(entry)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("[rewind:%s]", hash[:16]), nil
}

// RetrieveCommand handles the `rewind retrieve` command.
func (c *CLI) RetrieveCommand(hash string) (*Entry, error) {
	return c.pipeline.store.Retrieve(hash)
}

// ListCommand handles the `rewind list` command.
func (c *CLI) ListCommand(filter ListFilter) ([]*Entry, error) {
	return c.pipeline.store.List(filter)
}

// ClearCommand handles the `rewind clear` command.
func (c *CLI) ClearCommand(olderThan time.Duration) (int64, error) {
	if sqliteStore, ok := c.pipeline.store.(*SQLiteStore); ok {
		return sqliteStore.DeleteOlderThan(olderThan)
	}
	return 0, fmt.Errorf("store does not support bulk delete")
}

// StatsCommand handles the `rewind stats` command.
func (c *CLI) StatsCommand() (StoreStats, error) {
	return c.pipeline.store.Stats()
}

// VacuumCommand handles the `rewind vacuum` command.
func (c *CLI) VacuumCommand() error {
	return c.pipeline.store.Vacuum()
}

// Integration provides integration with TokMan's existing filter pipeline.
type Integration struct {
	pipeline *Pipeline
	enabled  bool
}

// NewIntegration creates a new pipeline integration.
func NewIntegration(store Store, config Config) *Integration {
	return &Integration{
		pipeline: NewPipeline(store, config),
		enabled:  true,
	}
}

// Enable enables the integration.
func (i *Integration) Enable() {
	i.enabled = true
}

// Disable disables the integration.
func (i *Integration) Disable() {
	i.enabled = false
}

// IsEnabled returns true if integration is enabled.
func (i *Integration) IsEnabled() bool {
	return i.enabled
}

// ProcessFilterOutput processes output from the filter pipeline.
func (i *Integration) ProcessFilterOutput(original, compressed string, command string) (string, error) {
	if !i.enabled {
		return compressed, nil
	}

	// Only store if compression was effective
	if len(compressed) >= int(float64(len(original))*0.9) {
		return compressed, nil
	}

	// Check if we should compress
	if !i.pipeline.ShouldCompress(original, command) {
		return compressed, nil
	}

	// Store the original
	entry := &Entry{
		Original:       original,
		Command:        command,
		Compressed:     compressed,
		CreatedAt:      time.Now(),
		SizeOriginal:   int64(len(original)),
		SizeCompressed: int64(len(compressed)),
	}

	hash, err := i.pipeline.store.Save(entry)
	if err != nil {
		// Return compressed without marker on error
		return compressed, nil
	}

	// Return marker instead of compressed content
	marker := Marker{Hash: hash[:16]}
	return marker.String(), nil
}

// RestoreInput restores original content from input containing markers.
func (i *Integration) RestoreInput(input string) (string, error) {
	if !i.enabled {
		return input, nil
	}

	return i.pipeline.ProcessOutput(input)
}

// SetupDefault creates a default integration with SQLite store.
func SetupDefault() (*Integration, error) {
	config := DefaultConfig()
	store, err := NewSQLiteStore(config)
	if err != nil {
		return nil, err
	}

	return NewIntegration(store, config), nil
}

// InjectMarker injects a marker into text at a good position.
func InjectMarker(text string, marker Marker, maxLen int) string {
	if len(text) <= maxLen {
		return text + "\n\n" + marker.String()
	}

	// Find a good breakpoint (end of line)
	breakpoint := maxLen
	for i := maxLen; i > maxLen-100 && i > 0; i-- {
		if text[i] == '\n' {
			breakpoint = i
			break
		}
	}

	return text[:breakpoint] + "\n\n... [truncated]\n" + marker.String()
}

// ExtractMarkers extracts all markers from text.
func ExtractMarkers(text string) []Marker {
	var markers []Marker
	searchStart := 0

	for {
		idx := strings.Index(text[searchStart:], "[rewind:")
		if idx < 0 {
			break
		}
		idx += searchStart

		endIdx := strings.Index(text[idx:], "]")
		if endIdx < 0 {
			break
		}
		endIdx += idx + 1

		if marker, err := ParseMarker(text[idx:endIdx]); err == nil {
			markers = append(markers, marker)
		}

		searchStart = endIdx
	}

	return markers
}

