// Package compression provides the compression service interface.
// This service wraps the 31-layer filter pipeline for microservice deployment.
package compression

import (
	"context"
	"time"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/metrics"
)

// CompressionService defines the interface for the compression service.
type CompressionService interface {
	// Compress applies the 31-layer compression pipeline to input text.
	Compress(ctx context.Context, req *CompressRequest) (*CompressResponse, error)

	// GetStats returns compression statistics.
	GetStats(ctx context.Context) (*StatsResponse, error)

	// GetLayers returns information about all compression layers.
	GetLayers(ctx context.Context) ([]LayerInfo, error)
}

// CompressRequest is the input for compression.
type CompressRequest struct {
	Input       string      // Text to compress
	Mode        filter.Mode // Compression mode (none/minimal/aggressive)
	QueryIntent string      // Query intent for query-aware compression
	Budget      int         // Token budget (0 = unlimited)
	Preset      string      // Pipeline preset (fast/balanced/full)
}

// CompressResponse is the output from compression.
type CompressResponse struct {
	Output         string   // Compressed text
	OriginalSize   int      // Original token count
	CompressedSize int      // Compressed token count
	SavingsPercent float64  // Token savings percentage
	LayersApplied  []string // List of layers that were applied
}

// StatsResponse contains compression statistics.
type StatsResponse struct {
	TotalCompressions  int64
	TotalTokensSaved   int64
	AverageCompression float64
	P99LatencyMs       float64
}

// LayerInfo describes a compression layer.
type LayerInfo struct {
	Name        string // Layer name
	Number      int    // Layer number (1-31)
	Research    string // Research paper source
	Compression string // Typical compression ratio
}

// Service implements CompressionService using the filter pipeline.
type Service struct {
	baseConfig filter.PipelineConfig
}

// NewService creates a new compression service.
func NewService(cfg filter.PipelineConfig) *Service {
	return &Service{
		baseConfig: cfg,
	}
}

// Compress implements CompressionService.
func (s *Service) Compress(ctx context.Context, req *CompressRequest) (*CompressResponse, error) {
	start := time.Now()

	cfg := s.baseConfig
	cfg.Mode = req.Mode
	cfg.QueryIntent = req.QueryIntent
	cfg.Budget = req.Budget
	if req.Preset != "" {
		cfg = filter.PresetConfig(filter.PipelinePreset(req.Preset), req.Mode)
		cfg.QueryIntent = req.QueryIntent
		cfg.Budget = req.Budget
	}

	output, stats := filter.NewPipelineCoordinator(cfg).Process(req.Input)

	// Calculate duration
	duration := time.Since(start)
	durationMs := float64(duration.Nanoseconds()) / 1e6

	// Extract layer names from LayerStats
	var layersApplied []string
	for name := range stats.LayerStats {
		layersApplied = append(layersApplied, name)
	}

	// Record Prometheus metrics
	modeStr := "none"
	if req.Mode == filter.ModeMinimal {
		modeStr = "minimal"
	} else if req.Mode == filter.ModeAggressive {
		modeStr = "aggressive"
	}
	metrics.RecordCompression(modeStr, stats.OriginalTokens, stats.FinalTokens, durationMs)

	// Record individual layer metrics
	for _, layerName := range layersApplied {
		metrics.RecordLayerApplied(layerName)
	}

	return &CompressResponse{
		Output:         output,
		OriginalSize:   stats.OriginalTokens,
		CompressedSize: stats.FinalTokens,
		SavingsPercent: stats.ReductionPercent,
		LayersApplied:  layersApplied,
	}, nil
}

// GetStats implements CompressionService.
func (s *Service) GetStats(ctx context.Context) (*StatsResponse, error) {
	// TODO: Wire up to tracking database
	return &StatsResponse{}, nil
}

// GetLayers implements CompressionService.
func (s *Service) GetLayers(ctx context.Context) ([]LayerInfo, error) {
	return []LayerInfo{
		{Number: 1, Name: "Entropy Filtering", Research: "Selective Context (Mila 2023)", Compression: "2-3x"},
		{Number: 2, Name: "Perplexity Pruning", Research: "LLMLingua (Microsoft 2023)", Compression: "20x"},
		{Number: 3, Name: "Goal-Driven Selection", Research: "SWE-Pruner (Shanghai Jiao Tong 2025)", Compression: "14.8x"},
		{Number: 4, Name: "AST Preservation", Research: "LongCodeZip (NUS 2025)", Compression: "4-8x"},
		{Number: 5, Name: "Contrastive Ranking", Research: "LongLLMLingua (Microsoft 2024)", Compression: "4-10x"},
		{Number: 6, Name: "N-gram Abbreviation", Research: "CompactPrompt (2025)", Compression: "2.5x"},
		{Number: 7, Name: "Evaluator Heads", Research: "EHPC (Tsinghua/Huawei 2025)", Compression: "5-7x"},
		{Number: 8, Name: "Gist Compression", Research: "Stanford/Berkeley (2023)", Compression: "20x+"},
		{Number: 9, Name: "Hierarchical Summary", Research: "AutoCompressor (Princeton/MIT 2023)", Compression: "Extreme"},
		{Number: 10, Name: "Budget Enforcement", Research: "Industry standard", Compression: "Guaranteed"},
		{Number: 11, Name: "Compaction", Research: "MemGPT (UC Berkeley 2023)", Compression: "98%+"},
		{Number: 12, Name: "Attribution Filter", Research: "ProCut (LinkedIn 2025)", Compression: "78%"},
		{Number: 13, Name: "H2O Filter", Research: "Heavy-Hitter Oracle (NeurIPS 2023)", Compression: "30x+"},
		{Number: 14, Name: "Attention Sink", Research: "StreamingLLM (2023)", Compression: "Infinite"},
		{Number: 15, Name: "Meta-Token", Research: "arXiv:2506.00307 (2025)", Compression: "27%"},
		{Number: 16, Name: "Semantic Chunk", Research: "ChunkKV-style", Compression: "Context-aware"},
		{Number: 17, Name: "Sketch Store", Research: "KVReviver (Dec 2025)", Compression: "90% memory"},
		{Number: 18, Name: "Lazy Pruner", Research: "LazyLLM (July 2024)", Compression: "2.34x"},
		{Number: 19, Name: "Semantic Anchor", Research: "Attention Gradient Detection", Compression: "Preserved"},
		{Number: 20, Name: "Agent Memory", Research: "Focus-inspired", Compression: "Knowledge graph"},
	}, nil
}
