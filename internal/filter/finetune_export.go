package filter

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// FineTuneRecord holds one training example produced by a filter run.
type FineTuneRecord struct {
	Input        string  `json:"input"`
	Output       string  `json:"output"`
	TokensSaved  int     `json:"tokens_saved"`
	ReductionPct float64 `json:"reduction_pct"`
	Filter       string  `json:"filter"`
	Mode         string  `json:"mode"`
}

// FineTuneExporter collects FineTuneRecords as filters run and can export
// them to JSONL for fine-tuning workflows.
type FineTuneExporter struct {
	mu      sync.Mutex
	records []FineTuneRecord
}

// NewFineTuneExporter creates a ready-to-use FineTuneExporter.
func NewFineTuneExporter() *FineTuneExporter {
	return &FineTuneExporter{}
}

// Record captures a single filter application as a FineTuneRecord.
// It is safe to call concurrently from multiple goroutines.
func (e *FineTuneExporter) Record(filterName string, mode Mode, input, output string) {
	inputTokens := core.EstimateTokens(input)
	outputTokens := core.EstimateTokens(output)
	saved := inputTokens - outputTokens
	if saved < 0 {
		saved = 0
	}

	var reductionPct float64
	if inputTokens > 0 {
		reductionPct = float64(saved) / float64(inputTokens) * 100
	}

	rec := FineTuneRecord{
		Input:        input,
		Output:       output,
		TokensSaved:  saved,
		ReductionPct: reductionPct,
		Filter:       filterName,
		Mode:         string(mode),
	}

	e.mu.Lock()
	e.records = append(e.records, rec)
	e.mu.Unlock()
}

// Export writes all collected records to a JSONL file at path (one JSON
// object per line). It creates or truncates the file.
func (e *FineTuneExporter) Export(path string) error {
	e.mu.Lock()
	snapshot := make([]FineTuneRecord, len(e.records))
	copy(snapshot, e.records)
	e.mu.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, rec := range snapshot {
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	return nil
}

// ExportSamples returns records where tokens_saved >= minSaved.
func (e *FineTuneExporter) ExportSamples(minSaved int) []FineTuneRecord {
	e.mu.Lock()
	defer e.mu.Unlock()

	var result []FineTuneRecord
	for _, rec := range e.records {
		if rec.TokensSaved >= minSaved {
			result = append(result, rec)
		}
	}
	return result
}
