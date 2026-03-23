package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/httpmw"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// AlertConfig holds alert threshold configuration
type AlertConfig struct {
	DailyTokenLimit     int64   `json:"daily_token_limit"`
	WeeklyTokenLimit    int64   `json:"weekly_token_limit"`
	UsageSpikeThreshold float64 `json:"usage_spike_threshold"` // multiplier for spike detection
	Enabled             bool    `json:"enabled"`
}

// Config holds dashboard configuration
type Config struct {
	Port             int         `json:"port"`
	Bind             string      `json:"bind"`
	UpdateInterval   int         `json:"update_interval"`
	Theme            string      `json:"theme"`
	Alerts           AlertConfig `json:"alerts"`
	EnableExport     bool        `json:"enable_export"`
	HistoryRetention int         `json:"history_retention"`
}

// DefaultConfig returns default dashboard configuration
var defaultConfig = Config{
	Port:             8080,
	Bind:             "localhost",
	UpdateInterval:   30000,
	Theme:            "dark",
	EnableExport:     true,
	HistoryRetention: 90,
	Alerts: AlertConfig{
		DailyTokenLimit:     1000000,
		WeeklyTokenLimit:    5000000,
		UsageSpikeThreshold: 2.0,
		Enabled:             true,
	},
}

var (
	Port   int
	Open   bool
	Bind   string
	APIKey string

	configMu sync.RWMutex
)

// Cmd returns the dashboard cobra command for registration.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch web dashboard for token savings visualization",
		Long: `Launch an interactive web dashboard to visualize token savings,
economics metrics, and usage trends over time.

The dashboard provides:
- Real-time token savings charts
- Daily/weekly/monthly breakdowns
- Command-level analytics
- Cost tracking with Claude API rates
- LLM usage integration via ccusage`,
		RunE: runDashboard,
	}

	cmd.Flags().IntVarP(&Port, "port", "p", 8080, "Port to run dashboard on")
	cmd.Flags().BoolVarP(&Open, "open", "o", false, "Open browser automatically")
	cmd.Flags().StringVar(&Bind, "bind", "localhost", "Address to bind server to (e.g., 0.0.0.0 for all interfaces)")
	cmd.Flags().StringVar(&APIKey, "api-key", "", "API key for authentication (empty = no auth)")

	return cmd
}

func runDashboard(cmd *cobra.Command, args []string) error {
	dbPath := tracking.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer tracker.Close()

	// API handlers
	http.Handle("/api/stats", corsMiddleware(statsHandler(tracker)))
	http.Handle("/api/daily", corsMiddleware(dailyHandler(tracker)))
	http.Handle("/api/weekly", corsMiddleware(weeklyHandler(tracker)))
	http.Handle("/api/monthly", corsMiddleware(monthlyHandler(tracker)))
	http.Handle("/api/commands", corsMiddleware(commandsHandler(tracker)))
	http.Handle("/api/recent", corsMiddleware(recentHandler(tracker)))
	http.Handle("/api/economics", corsMiddleware(economicsHandler(tracker)))
	http.Handle("/api/performance", corsMiddleware(performanceHandler(tracker)))
	http.Handle("/api/failures", corsMiddleware(failuresHandler(tracker)))
	http.Handle("/api/top-commands", corsMiddleware(topCommandsHandler(tracker)))
	http.Handle("/api/hourly", corsMiddleware(hourlyHandler(tracker)))
	http.Handle("/api/export/csv", corsMiddleware(exportCSVHandler(tracker)))
	// New endpoints for enhanced dashboard
	http.Handle("/api/llm-status", corsMiddleware(llmStatusHandler(tracker)))
	http.Handle("/api/daily-breakdown", corsMiddleware(dailyBreakdownHandler(tracker)))
	http.Handle("/api/project-stats", corsMiddleware(projectStatsHandler(tracker)))
	http.Handle("/api/session-stats", corsMiddleware(sessionStatsHandler(tracker)))
	http.Handle("/api/savings-trend", corsMiddleware(savingsTrendHandler(tracker)))
	// New enhanced endpoints
	http.Handle("/api/alerts", corsMiddleware(alertsHandler(tracker)))
	http.Handle("/api/export/json", corsMiddleware(exportJSONHandler(tracker)))
	http.Handle("/api/model-breakdown", corsMiddleware(modelBreakdownHandler(tracker)))
	http.Handle("/api/config", corsMiddleware(configHandler(tracker)))
	http.Handle("/api/report", corsMiddleware(reportHandler(tracker)))
	http.Handle("/api/cache-metrics", corsMiddleware(cacheMetricsHandler(tracker)))
	http.HandleFunc("/logo", logoHandler)
	http.HandleFunc("/", dashboardIndexHandler)

	rl := httpmw.NewDefault()

	var handler http.Handler = http.DefaultServeMux
	if APIKey != "" {
		handler = authMiddleware(APIKey, handler)
	}
	handler = rl.Middleware(handler)

	addr := fmt.Sprintf("%s:%d", Bind, Port)

	fmt.Printf("🌐 TokMan Dashboard running at http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")

	if Open {
		fmt.Println("Opening browser...")
		// Could use browser.OpenURL here
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if APIKey == "" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				origin := r.Header.Get("Origin")
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health and static endpoints
		if r.URL.Path == "/" || r.URL.Path == "/logo" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("Authorization")
		if strings.HasPrefix(key, "Bearer ") {
			key = key[7:]
		}

		if key != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	http.ServeFile(w, r, "docs/logo.svg")
}

func dashboardIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}
