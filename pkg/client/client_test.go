package client

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	analyticspb "github.com/GrayCodeAI/tokman/pkg/api/proto/analyticsv1"
	compressionpb "github.com/GrayCodeAI/tokman/pkg/api/proto/compressionv1"
)

// Mock compression server for testing
type mockCompressionServer struct {
	compressionpb.UnimplementedCompressionServiceServer
}

func (s *mockCompressionServer) Compress(ctx context.Context, req *compressionpb.CompressRequest) (*compressionpb.CompressResponse, error) {
	return &compressionpb.CompressResponse{
		Output:           req.Input + "_compressed",
		OriginalTokens:   100,
		CompressedTokens: 50,
		SavingsPercent:   50.0,
		LayersApplied:    []string{"entropy", "perplexity"},
	}, nil
}

func (s *mockCompressionServer) GetLayers(ctx context.Context, req *compressionpb.GetLayersRequest) (*compressionpb.GetLayersResponse, error) {
	return &compressionpb.GetLayersResponse{
		Layers: []*compressionpb.LayerInfo{
			{Number: 1, Name: "Entropy Filter", Research: "Mila 2023", Compression: "30%"},
		},
	}, nil
}

// Mock analytics server for testing
type mockAnalyticsServer struct {
	analyticspb.UnimplementedAnalyticsServiceServer
}

func (s *mockAnalyticsServer) GetMetrics(ctx context.Context, req *analyticspb.GetMetricsRequest) (*analyticspb.GetMetricsResponse, error) {
	return &analyticspb.GetMetricsResponse{
		TotalCommands:    1000,
		TotalTokensSaved: 50000,
		AverageSavings:   45.5,
		P99LatencyMs:     25.0,
	}, nil
}

func (s *mockAnalyticsServer) GetEconomics(ctx context.Context, req *analyticspb.GetEconomicsRequest) (*analyticspb.GetEconomicsResponse, error) {
	return &analyticspb.GetEconomicsResponse{
		TokensSaved:        50000,
		EstimatedCostSaved: 15.50,
		ModelUsed:          "gpt-4",
	}, nil
}

func startMockServer(t *testing.T) (*bufconn.Listener, *grpc.Server) {
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	compressionpb.RegisterCompressionServiceServer(server, &mockCompressionServer{})
	analyticspb.RegisterAnalyticsServiceServer(server, &mockAnalyticsServer{})

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("server error: %v", err)
		}
	}()

	return lis, server
}

func testDialOptions(lis *bufconn.Listener) []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
	}
}

func TestClientCompress(t *testing.T) {
	lis, server := startMockServer(t)
	defer server.Stop()

	cfg := &Config{
		CompressionAddr: "passthrough:///bufnet",
		Timeout:         5 * time.Second,
		DialOptions:     testDialOptions(lis),
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Compression.Compress(ctx, "test input", "minimal", 1000)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Output != "test input_compressed" {
		t.Errorf("expected output 'test input_compressed', got '%s'", result.Output)
	}
	if result.OriginalTokens != 100 {
		t.Errorf("expected 100 original tokens, got %d", result.OriginalTokens)
	}
	if result.CompressedTokens != 50 {
		t.Errorf("expected 50 compressed tokens, got %d", result.CompressedTokens)
	}
	if result.SavingsPercent != 50.0 {
		t.Errorf("expected 50.0 savings percent, got %f", result.SavingsPercent)
	}
}

func TestClientGetLayers(t *testing.T) {
	lis, server := startMockServer(t)
	defer server.Stop()

	cfg := &Config{
		CompressionAddr: "passthrough:///bufnet",
		Timeout:         5 * time.Second,
		DialOptions:     testDialOptions(lis),
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	layers, err := client.Compression.GetLayers(ctx)
	if err != nil {
		t.Fatalf("GetLayers failed: %v", err)
	}

	if len(layers) != 1 {
		t.Errorf("expected 1 layer, got %d", len(layers))
	}
	if layers[0].Name != "Entropy Filter" {
		t.Errorf("expected layer name 'Entropy Filter', got '%s'", layers[0].Name)
	}
}

func TestClientGetMetrics(t *testing.T) {
	lis, server := startMockServer(t)
	defer server.Stop()

	cfg := &Config{
		AnalyticsAddr: "passthrough:///bufnet",
		Timeout:       5 * time.Second,
		DialOptions:   testDialOptions(lis),
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metrics, err := client.Analytics.GetMetrics(ctx)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if metrics.TotalCommands != 1000 {
		t.Errorf("expected 1000 total commands, got %d", metrics.TotalCommands)
	}
	if metrics.TotalTokensSaved != 50000 {
		t.Errorf("expected 50000 tokens saved, got %d", metrics.TotalTokensSaved)
	}
}

func TestClientGetEconomics(t *testing.T) {
	lis, server := startMockServer(t)
	defer server.Stop()

	cfg := &Config{
		AnalyticsAddr: "passthrough:///bufnet",
		Timeout:       5 * time.Second,
		DialOptions:   testDialOptions(lis),
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	econ, err := client.Analytics.GetEconomics(ctx)
	if err != nil {
		t.Fatalf("GetEconomics failed: %v", err)
	}

	if econ.TokensSaved != 50000 {
		t.Errorf("expected 50000 tokens saved, got %d", econ.TokensSaved)
	}
	if econ.ModelUsed != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", econ.ModelUsed)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.CompressionAddr != "localhost:50051" {
		t.Errorf("expected compression addr 'localhost:50051', got '%s'", cfg.CompressionAddr)
	}
	if cfg.AnalyticsAddr != "localhost:50053" {
		t.Errorf("expected analytics addr 'localhost:50053', got '%s'", cfg.AnalyticsAddr)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", cfg.Timeout)
	}
}

func TestClientClose(t *testing.T) {
	cfg := &Config{
		CompressionAddr: "localhost:50051",
		AnalyticsAddr:   "localhost:50053",
	}

	client, err := New(cfg)
	if err != nil {
		// Connection may fail if no server is running, but Close should still work
		t.Logf("expected connection error (no server): %v", err)
		return
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
