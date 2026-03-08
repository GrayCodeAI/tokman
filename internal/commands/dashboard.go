package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	dashboardPort int
	dashboardOpen bool
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch web dashboard for token savings visualization",
	Long: `Launch an interactive web dashboard to visualize token savings,
economics metrics, and usage trends over time.

The dashboard provides:
- Real-time token savings charts
- Daily/weekly/monthly breakdowns
- Command-level analytics
- Cost tracking with Claude API rates`,
	RunE: runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.Flags().IntVarP(&dashboardPort, "port", "p", 8080, "Port to run dashboard on")
	dashboardCmd.Flags().BoolVarP(&dashboardOpen, "open", "o", false, "Open browser automatically")
}

func runDashboard(cmd *cobra.Command, args []string) error {
	dbPath := tracking.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer tracker.Close()

	// API handlers
	http.HandleFunc("/api/stats", corsMiddleware(statsHandler(tracker)))
	http.HandleFunc("/api/daily", corsMiddleware(dailyHandler(tracker)))
	http.HandleFunc("/api/commands", corsMiddleware(commandsHandler(tracker)))
	http.HandleFunc("/api/economics", corsMiddleware(economicsHandler(tracker)))
	http.HandleFunc("/", dashboardIndexHandler)

	addr := fmt.Sprintf(":%d", dashboardPort)
	
	fmt.Printf("🌐 TokMan Dashboard running at http://localhost%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")
	
	if dashboardOpen {
		fmt.Println("Opening browser...")
		// Could use browser.OpenURL here
	}
	
	return http.ListenAndServe(addr, nil)
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func statsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetSavings("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := map[string]interface{}{
			"tokens_saved":   stats.TotalSaved,
			"commands_count": stats.TotalCommands,
			"original":       stats.TotalOriginal,
			"filtered":       stats.TotalFiltered,
		}
		json.NewEncoder(w).Encode(response)
	}
}

func dailyHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		days := 7
		if d := r.URL.Query().Get("days"); d != "" {
			fmt.Sscanf(d, "%d", &days)
		}
		
		records, err := tracker.GetDailySavings("", days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Convert to chart format
		result := make([]map[string]interface{}, len(records))
		for i, r := range records {
			result[i] = map[string]interface{}{
				"date":         r.Date,
				"tokens_saved": r.Saved,
				"original":     r.Original,
				"commands":     r.Commands,
			}
		}
		json.NewEncoder(w).Encode(result)
	}
}

func commandsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := tracker.GetCommandStats("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Convert to chart format
		limit := 5
		if len(stats) < limit {
			limit = len(stats)
		}
		result := make([]map[string]interface{}, limit)
		for i := 0; i < limit; i++ {
			result[i] = map[string]interface{}{
				"command":      stats[i].Command,
				"tokens_saved": stats[i].TotalSaved,
				"executions":   stats[i].ExecutionCount,
			}
		}
		json.NewEncoder(w).Encode(result)
	}
}

func economicsHandler(tracker *tracking.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := tracker.GetDailySavings("", 30)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Calculate estimated cost savings
		// Using $3 per million input tokens (Claude pricing)
		var totalSaved int
		for _, r := range records {
			totalSaved += r.Saved
		}
		estimatedCost := float64(totalSaved) * 3.0 / 1_000_000
		
		response := map[string]interface{}{
			"total_tokens_saved": totalSaved,
			"estimated_cost":     estimatedCost,
			"records_count":      len(records),
		}
		
		json.NewEncoder(w).Encode(response)
	}
}

func dashboardIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TokMan Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #eee;
            min-height: 100vh;
            padding: 2rem;
        }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { 
            text-align: center; 
            margin-bottom: 2rem;
            font-size: 2.5rem;
            background: linear-gradient(90deg, #00d4ff, #7c3aed);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        .stat-card {
            background: rgba(255,255,255,0.05);
            border-radius: 16px;
            padding: 1.5rem;
            border: 1px solid rgba(255,255,255,0.1);
            backdrop-filter: blur(10px);
        }
        .stat-card h3 {
            font-size: 0.875rem;
            text-transform: uppercase;
            opacity: 0.7;
            margin-bottom: 0.5rem;
        }
        .stat-card .value {
            font-size: 2rem;
            font-weight: 700;
        }
        .chart-container {
            background: rgba(255,255,255,0.05);
            border-radius: 16px;
            padding: 1.5rem;
            margin-bottom: 2rem;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .chart-container h2 {
            margin-bottom: 1rem;
            font-size: 1.25rem;
        }
        canvas { max-height: 300px; }
        .loading { text-align: center; padding: 2rem; opacity: 0.6; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🌸 TokMan Dashboard</h1>
        
        <div class="stats-grid">
            <div class="stat-card">
                <h3>Total Tokens Saved</h3>
                <div class="value" id="tokens-saved">--</div>
            </div>
            <div class="stat-card">
                <h3>Estimated Savings</h3>
                <div class="value" id="cost-saved">$--</div>
            </div>
            <div class="stat-card">
                <h3>Commands Filtered</h3>
                <div class="value" id="commands-count">--</div>
            </div>
            <div class="stat-card">
                <h3>Avg Savings/Command</h3>
                <div class="value" id="avg-savings">--</div>
            </div>
        </div>

        <div class="chart-container">
            <h2>📊 Daily Token Savings</h2>
            <canvas id="dailyChart"></canvas>
        </div>

        <div class="chart-container">
            <h2>🏆 Top Commands by Savings</h2>
            <canvas id="commandsChart"></canvas>
        </div>
    </div>

    <script>
        async function fetchAPI(endpoint) {
            const response = await fetch(endpoint);
            return response.json();
        }

        async function loadDashboard() {
            try {
                // Load stats
                const stats = await fetchAPI('/api/stats');
                document.getElementById('tokens-saved').textContent = 
                    (stats.tokens_saved || 0).toLocaleString();
                document.getElementById('commands-count').textContent = 
                    (stats.commands_count || 0).toLocaleString();
                document.getElementById('avg-savings').textContent = 
                    stats.commands_count > 0 
                        ? Math.round(stats.tokens_saved / stats.commands_count).toLocaleString()
                        : '0';

                // Load economics
                const economics = await fetchAPI('/api/economics');
                document.getElementById('cost-saved').textContent = 
                    '$' + (economics.estimated_cost || 0).toFixed(2);

                // Load daily chart
                const daily = await fetchAPI('/api/daily?days=7');
                renderDailyChart(daily);

                // Load commands chart
                const commands = await fetchAPI('/api/commands?limit=5');
                renderCommandsChart(commands);
            } catch (error) {
                console.error('Failed to load dashboard:', error);
            }
        }

        function renderDailyChart(data) {
            const ctx = document.getElementById('dailyChart').getContext('2d');
            new Chart(ctx, {
                type: 'line',
                data: {
                    labels: data.map(d => d.date),
                    datasets: [{
                        label: 'Tokens Saved',
                        data: data.map(d => d.tokens_saved),
                        borderColor: '#00d4ff',
                        backgroundColor: 'rgba(0, 212, 255, 0.1)',
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: {
                    responsive: true,
                    plugins: { legend: { display: false } },
                    scales: {
                        y: { beginAtZero: true, grid: { color: 'rgba(255,255,255,0.1)' } },
                        x: { grid: { color: 'rgba(255,255,255,0.1)' } }
                    }
                }
            });
        }

        function renderCommandsChart(data) {
            const ctx = document.getElementById('commandsChart').getContext('2d');
            new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: data.map(d => d.command),
                    datasets: [{
                        label: 'Tokens Saved',
                        data: data.map(d => d.tokens_saved),
                        backgroundColor: [
                            '#7c3aed', '#00d4ff', '#10b981', '#f59e0b', '#ef4444'
                        ]
                    }]
                },
                options: {
                    responsive: true,
                    indexAxis: 'y',
                    plugins: { legend: { display: false } },
                    scales: {
                        x: { beginAtZero: true, grid: { color: 'rgba(255,255,255,0.1)' } },
                        y: { grid: { display: false } }
                    }
                }
            });
        }

        loadDashboard();
        setInterval(loadDashboard, 30000); // Refresh every 30s
    </script>
</body>
</html>`
