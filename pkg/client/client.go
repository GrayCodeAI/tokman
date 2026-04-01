// Package client provides gRPC clients for TokMan services.
package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	analyticspb "github.com/GrayCodeAI/tokman/pkg/api/proto/analyticsv1"
	compressionpb "github.com/GrayCodeAI/tokman/pkg/api/proto/compressionv1"
)

// Config holds client configuration.
type Config struct {
	CompressionAddr string // Compression service address (e.g., "localhost:50051")
	AnalyticsAddr   string // Analytics service address (e.g., "localhost:50053")
	Timeout         time.Duration
	DialOptions     []grpc.DialOption // Optional extra dial options, primarily for tests
}

// DefaultConfig returns default client configuration.
func DefaultConfig() *Config {
	return &Config{
		CompressionAddr: "localhost:50051",
		AnalyticsAddr:   "localhost:50053",
		Timeout:         30 * time.Second,
	}
}

// Client provides access to all TokMan services.
type Client struct {
	config *Config

	// gRPC connections
	compressionConn *grpc.ClientConn
	analyticsConn   *grpc.ClientConn

	// Service clients
	Compression CompressionClient
	Analytics   AnalyticsClient
}

// CompressionClient provides access to compression service.
type CompressionClient interface {
	Compress(ctx context.Context, input string, mode string, budget int) (*CompressResult, error)
	GetLayers(ctx context.Context) ([]LayerInfo, error)
}

// AnalyticsClient provides access to analytics service.
type AnalyticsClient interface {
	GetMetrics(ctx context.Context) (*MetricsResult, error)
	GetEconomics(ctx context.Context) (*EconomicsResult, error)
}

// CompressResult holds compression results.
type CompressResult struct {
	Output           string
	OriginalTokens   int
	CompressedTokens int
	SavingsPercent   float64
	LayersApplied    []string
}

// LayerInfo describes a compression layer.
type LayerInfo struct {
	Number      int
	Name        string
	Research    string
	Compression string
}

// MetricsResult holds analytics metrics.
type MetricsResult struct {
	TotalCommands    int64
	TotalTokensSaved int64
	AverageSavings   float64
	P99LatencyMs     float64
}

// EconomicsResult holds economics data.
type EconomicsResult struct {
	TokensSaved        int64
	EstimatedCostSaved float64
	ModelUsed          string
}

// New creates a new TokMan client.
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	c := &Client{config: cfg}
	baseDialOptions := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, cfg.DialOptions...)

	// Connect to compression service
	if cfg.CompressionAddr != "" {
		conn, err := grpc.NewClient(cfg.CompressionAddr, baseDialOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to compression service: %w", err)
		}
		c.compressionConn = conn
		c.Compression = &compressionClient{pb: compressionpb.NewCompressionServiceClient(conn)}
	}

	// Connect to analytics service
	if cfg.AnalyticsAddr != "" {
		conn, err := grpc.NewClient(cfg.AnalyticsAddr, baseDialOptions...)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("failed to connect to analytics service: %w", err)
		}
		c.analyticsConn = conn
		c.Analytics = &analyticsClient{pb: analyticspb.NewAnalyticsServiceClient(conn)}
	}

	return c, nil
}

// Close closes all gRPC connections.
func (c *Client) Close() error {
	var errs []error
	if c.compressionConn != nil {
		if err := c.compressionConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.analyticsConn != nil {
		if err := c.analyticsConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

// compressionClient implements CompressionClient.
type compressionClient struct {
	pb compressionpb.CompressionServiceClient
}

// Compress calls the compression service.
func (c *compressionClient) Compress(ctx context.Context, input string, mode string, budget int) (*CompressResult, error) {
	resp, err := c.pb.Compress(ctx, &compressionpb.CompressRequest{
		Input:  input,
		Mode:   mode,
		Budget: int32(budget),
	})
	if err != nil {
		return nil, err
	}

	return &CompressResult{
		Output:           resp.Output,
		OriginalTokens:   int(resp.OriginalTokens),
		CompressedTokens: int(resp.CompressedTokens),
		SavingsPercent:   resp.SavingsPercent,
		LayersApplied:    resp.LayersApplied,
	}, nil
}

// GetLayers returns layer information.
func (c *compressionClient) GetLayers(ctx context.Context) ([]LayerInfo, error) {
	resp, err := c.pb.GetLayers(ctx, &compressionpb.GetLayersRequest{})
	if err != nil {
		return nil, err
	}

	layers := make([]LayerInfo, len(resp.Layers))
	for i, l := range resp.Layers {
		layers[i] = LayerInfo{
			Number:      int(l.Number),
			Name:        l.Name,
			Research:    l.Research,
			Compression: l.Compression,
		}
	}
	return layers, nil
}

// analyticsClient implements AnalyticsClient.
type analyticsClient struct {
	pb analyticspb.AnalyticsServiceClient
}

// GetMetrics returns analytics metrics.
func (c *analyticsClient) GetMetrics(ctx context.Context) (*MetricsResult, error) {
	resp, err := c.pb.GetMetrics(ctx, &analyticspb.GetMetricsRequest{})
	if err != nil {
		return nil, err
	}

	return &MetricsResult{
		TotalCommands:    resp.TotalCommands,
		TotalTokensSaved: resp.TotalTokensSaved,
		AverageSavings:   resp.AverageSavings,
		P99LatencyMs:     resp.P99LatencyMs,
	}, nil
}

// GetEconomics returns economics data.
func (c *analyticsClient) GetEconomics(ctx context.Context) (*EconomicsResult, error) {
	resp, err := c.pb.GetEconomics(ctx, &analyticspb.GetEconomicsRequest{})
	if err != nil {
		return nil, err
	}

	return &EconomicsResult{
		TokensSaved:        resp.TokensSaved,
		EstimatedCostSaved: resp.EstimatedCostSaved,
		ModelUsed:          resp.ModelUsed,
	}, nil
}
