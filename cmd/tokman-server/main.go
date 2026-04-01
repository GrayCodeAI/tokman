// Package main provides the TokMan server entry point.
// This server can run in multiple modes: compression, analytics, agent, llm, or gateway.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/server"
	"github.com/GrayCodeAI/tokman/internal/tracking"
	analyticspb "github.com/GrayCodeAI/tokman/pkg/api/proto/analyticsv1"
	compressionpb "github.com/GrayCodeAI/tokman/pkg/api/proto/compressionv1"
	"github.com/GrayCodeAI/tokman/services/analytics"
	analyticsgrpc "github.com/GrayCodeAI/tokman/services/analytics/grpc"
	"github.com/GrayCodeAI/tokman/services/compression"
	compressiongrpc "github.com/GrayCodeAI/tokman/services/compression/grpc"
)

// Version is set via ldflags during build.
var Version = "dev"

// Command-line flags.
var (
	serviceFlag = flag.String("service", "gateway", "Service to run: compression, analytics, agent, llm, gateway")
	portFlag    = flag.Int("port", 0, "Port to listen on (0 = service default)")
	grpcFlag    = flag.Bool("grpc", false, "Enable gRPC server")
	httpFlag    = flag.Bool("http", true, "Enable HTTP server (default: true)")
	configFlag  = flag.String("config", "", "Path to config file")
	verboseFlag = flag.Bool("verbose", false, "Enable verbose logging")
)

// Default ports for each service.
var defaultPorts = map[string]int{
	"compression": 8081,
	"analytics":   8083,
	"agent":       8084,
	"llm":         8085,
	"gateway":     8080,
}

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := loadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Determine port
	port := *portFlag
	if port == 0 {
		port = defaultPorts[*serviceFlag]
		if port == 0 {
			port = 8080
		}
	}

	log.Printf("TokMan %s starting %s service on port %d", Version, *serviceFlag, port)

	// Run the appropriate service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	switch *serviceFlag {
	case "compression":
		err = runCompressionService(ctx, cfg, port)
	case "analytics":
		err = runAnalyticsService(ctx, cfg, port)
	case "agent":
		err = runAgentService(ctx, cfg, port)
	case "llm":
		err = runLLMService(ctx, cfg, port)
	case "gateway":
		err = runGatewayService(ctx, cfg, port)
	default:
		log.Fatalf("Unknown service: %s (valid: compression, analytics, agent, llm, gateway)", *serviceFlag)
	}

	if err != nil {
		log.Fatalf("Service error: %v", err)
	}
}

// loadConfig loads the TokMan configuration.
func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	// Load from default location or return defaults
	cfg, err := config.Load("")
	if err != nil {
		return config.Defaults(), nil
	}
	return cfg, nil
}

// runCompressionService runs the compression service.
func runCompressionService(ctx context.Context, cfg *config.Config, port int) error {
	// Create pipeline config from loaded config
	pipelineCfg := cfg.Pipeline.ToFilterPipelineConfig(config.PipelineRuntimeOptions{
		Mode:   filter.ModeMinimal,
		Budget: cfg.Pipeline.DefaultBudget,
	})

	svc := compression.NewService(pipelineCfg)

	// Start gRPC server if enabled
	if *grpcFlag {
		grpcPort := port
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}

		grpcServer := grpc.NewServer()
		compServer := compressiongrpc.NewServer(svc)
		compressionpb.RegisterCompressionServiceServer(grpcServer, compServer)

		// Add health check
		healthServer := health.NewServer()
		healthpb.RegisterHealthServer(grpcServer, healthServer)
		healthServer.SetServingStatus("tokman.compression.v1.CompressionService", healthpb.HealthCheckResponse_SERVING)

		// Enable reflection for debugging
		reflection.Register(grpcServer)

		log.Printf("Compression gRPC service ready on port %d - 31-layer pipeline", grpcPort)

		go func() {
			if err := grpcServer.Serve(lis); err != nil {
				log.Printf("gRPC server error: %v", err)
			}
		}()
		defer grpcServer.GracefulStop()
	}

	// Start HTTP server if enabled
	if *httpFlag {
		httpPort := port
		if *grpcFlag {
			httpPort = port + 1000 // HTTP on different port when gRPC enabled
		}
		srv := server.New(server.Config{
			Port:           httpPort,
			LogLevel:       "info",
			Version:        Version,
			PipelineConfig: pipelineCfg,
		})
		return srv.Start()
	}

	// Wait for shutdown if only gRPC
	<-ctx.Done()
	return nil
}

// runAnalyticsService runs the analytics service.
func runAnalyticsService(ctx context.Context, cfg *config.Config, port int) error {
	// Initialize tracking database
	dbPath := cfg.Tracking.DatabasePath
	if dbPath == "" {
		dbPath = "~/.local/share/tokman/tokman.db"
	}

	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize tracker: %w", err)
	}
	defer tracker.Close()

	// Create analytics service with tracker as repository
	svc := analytics.NewService(&analyticsRepository{tracker})

	// Start gRPC server if enabled
	if *grpcFlag {
		grpcPort := port
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}

		grpcServer := grpc.NewServer()
		analyticsServer := analyticsgrpc.NewServer(svc)
		analyticspb.RegisterAnalyticsServiceServer(grpcServer, analyticsServer)

		// Add health check
		healthServer := health.NewServer()
		healthpb.RegisterHealthServer(grpcServer, healthServer)
		healthServer.SetServingStatus("tokman.analytics.v1.AnalyticsService", healthpb.HealthCheckResponse_SERVING)

		// Enable reflection for debugging
		reflection.Register(grpcServer)

		log.Printf("Analytics gRPC service ready on port %d - tracking: %s", grpcPort, dbPath)

		go func() {
			if err := grpcServer.Serve(lis); err != nil {
				log.Printf("gRPC server error: %v", err)
			}
		}()
		defer grpcServer.GracefulStop()
	}

	// Start HTTP server if enabled
	if *httpFlag {
		httpPort := port
		if *grpcFlag {
			httpPort = port + 1000 // HTTP on different port when gRPC enabled
		}
		srv := server.New(server.Config{
			Port:     httpPort,
			LogLevel: "info",
			Version:  Version,
		})
		return srv.Start()
	}

	// Wait for shutdown if only gRPC
	<-ctx.Done()
	return nil
}

// runAgentService runs the agent integration service.
func runAgentService(ctx context.Context, cfg *config.Config, port int) error {
	// Start HTTP server
	srv := server.New(server.Config{
		Port:     port,
		LogLevel: "info",
		Version:  Version,
	})

	log.Printf("Agent service ready - integrations: claude, cursor, copilot, windsurf, cline, gemini, codex")

	return srv.Start()
}

// runLLMService runs the LLM service.
func runLLMService(ctx context.Context, cfg *config.Config, port int) error {
	// Start HTTP server
	srv := server.New(server.Config{
		Port:     port,
		LogLevel: "info",
		Version:  Version,
	})

	log.Printf("LLM service ready")

	return srv.Start()
}

// runGatewayService runs the API gateway.
func runGatewayService(ctx context.Context, cfg *config.Config, port int) error {
	// The gateway aggregates all services
	srv := server.New(server.Config{
		Port:     port,
		LogLevel: "info",
		Version:  Version,
	})

	log.Printf("API Gateway ready - routing to all services")

	return srv.Start()
}

// analyticsRepository adapts tracking.Tracker to analytics.AnalyticsRepository.
type analyticsRepository struct {
	tracker *tracking.Tracker
}

func (r *analyticsRepository) Save(ctx context.Context, record *analytics.RecordRequest) error {
	saved := record.OriginalTokens - record.FilteredTokens
	if saved < 0 {
		saved = 0
	}
	return r.tracker.Record(&tracking.CommandRecord{
		Command:        record.Command,
		OriginalTokens: record.OriginalTokens,
		FilteredTokens: record.FilteredTokens,
		SavedTokens:    saved,
		SessionID:      record.SessionID,
		ExecTimeMs:     record.Duration.Milliseconds(),
		Timestamp:      time.Now(),
		ParseSuccess:   true,
	})
}

func (r *analyticsRepository) Query(ctx context.Context, req *analytics.MetricsRequest) (*analytics.MetricsResponse, error) {
	// Get savings summary
	summary, err := r.tracker.GetSavings("")
	if err != nil {
		return nil, err
	}

	return &analytics.MetricsResponse{
		TotalCommands:    int64(summary.TotalCommands),
		TotalTokensSaved: int64(summary.TotalSaved),
		TotalTokensIn:    int64(summary.TotalOriginal),
		TotalTokensOut:   int64(summary.TotalFiltered),
		AverageSavings:   summary.ReductionPct,
	}, nil
}

func (r *analyticsRepository) GetTopCommands(ctx context.Context, limit int) ([]analytics.CommandStats, error) {
	stats, err := r.tracker.GetCommandStats("")
	if err != nil {
		return nil, err
	}

	result := make([]analytics.CommandStats, 0, len(stats))
	for _, s := range stats {
		result = append(result, analytics.CommandStats{
			Command:    s.Command,
			Count:      int64(s.ExecutionCount),
			TotalSaved: int64(s.TotalSaved),
			AvgSavings: s.ReductionPct,
		})
	}
	return result, nil
}
