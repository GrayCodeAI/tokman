// Package api provides cost intelligence API endpoints
package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// CostIntelligenceAPI provides REST API for cost intelligence
type CostIntelligenceAPI struct {
	mux *http.ServeMux
}

// NewCostIntelligenceAPI creates a new API
func NewCostIntelligenceAPI() *CostIntelligenceAPI {
	api := &CostIntelligenceAPI{
		mux: http.NewServeMux(),
	}

	api.registerRoutes()
	return api
}

func (api *CostIntelligenceAPI) registerRoutes() {
	api.mux.HandleFunc("/api/v1/cost/summary", api.handleCostSummary)
	api.mux.HandleFunc("/api/v1/cost/breakdown", api.handleCostBreakdown)
	api.mux.HandleFunc("/api/v1/cost/forecast", api.handleCostForecast)
	api.mux.HandleFunc("/api/v1/cost/models", api.handleModelComparison)
	api.mux.HandleFunc("/api/v1/cost/budget", api.handleBudgetStatus)
	api.mux.HandleFunc("/api/v1/cost/anomalies", api.handleAnomalies)
	api.mux.HandleFunc("/api/v1/cost/alerts", api.handleAlerts)
	api.mux.HandleFunc("/api/v1/cost/teams", api.handleTeamCosts)
	api.mux.HandleFunc("/api/v1/cost/trends", api.handleCostTrends)
	api.mux.HandleFunc("/api/v1/cost/export", api.handleExport)
}

// ServeHTTP implements http.Handler
func (api *CostIntelligenceAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mux.ServeHTTP(w, r)
}

// CostSummaryResponse represents cost summary response
type CostSummaryResponse struct {
	TotalCost     float64 `json:"total_cost"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalRequests int64   `json:"total_requests"`
	AvgCostPerReq float64 `json:"avg_cost_per_request"`
	P95Cost       float64 `json:"p95_cost"`
	P99Cost       float64 `json:"p99_cost"`
	TimeRange     string  `json:"time_range"`
	ChangePct     float64 `json:"change_percent"`
}

func (api *CostIntelligenceAPI) handleCostSummary(w http.ResponseWriter, r *http.Request) {
	// Would fetch from database in production
	resp := CostSummaryResponse{
		TotalCost:     1250.50,
		TotalTokens:   5000000,
		TotalRequests: 10000,
		AvgCostPerReq: 0.125,
		P95Cost:       0.25,
		P99Cost:       0.50,
		TimeRange:     "7d",
		ChangePct:     -5.2,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CostBreakdownResponse represents cost breakdown response
type CostBreakdownResponse struct {
	InputCost  float64            `json:"input_cost"`
	OutputCost float64            `json:"output_cost"`
	CacheCost  float64            `json:"cache_cost"`
	ByModel    map[string]float64 `json:"by_model"`
	ByTeam     map[string]float64 `json:"by_team"`
	ByDay      []DailyCostPoint   `json:"by_day"`
}

// DailyCostPoint represents a daily cost point
type DailyCostPoint struct {
	Date   string  `json:"date"`
	Cost   float64 `json:"cost"`
	Tokens int64   `json:"tokens"`
}

func (api *CostIntelligenceAPI) handleCostBreakdown(w http.ResponseWriter, r *http.Request) {
	resp := CostBreakdownResponse{
		InputCost:  750.00,
		OutputCost: 400.00,
		CacheCost:  100.50,
		ByModel: map[string]float64{
			"gpt-4":    800.00,
			"claude-3": 350.00,
			"llama-2":  100.50,
		},
		ByTeam: map[string]float64{
			"engineering": 600.00,
			"research":    400.00,
			"marketing":   250.50,
		},
		ByDay: []DailyCostPoint{
			{Date: "2024-01-01", Cost: 150.00, Tokens: 500000},
			{Date: "2024-01-02", Cost: 175.00, Tokens: 600000},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CostForecastResponse represents cost forecast response
type CostForecastResponse struct {
	CurrentSpend     float64         `json:"current_spend"`
	ProjectedMonthly float64         `json:"projected_monthly"`
	ProjectedYearly  float64         `json:"projected_yearly"`
	ConfidenceLow    float64         `json:"confidence_low"`
	ConfidenceHigh   float64         `json:"confidence_high"`
	GrowthRate       float64         `json:"growth_rate"`
	Forecast         []ForecastPoint `json:"forecast"`
}

// ForecastPoint represents a forecast data point
type ForecastPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

func (api *CostIntelligenceAPI) handleCostForecast(w http.ResponseWriter, r *http.Request) {
	resp := CostForecastResponse{
		CurrentSpend:     1250.50,
		ProjectedMonthly: 15000.00,
		ProjectedYearly:  180000.00,
		ConfidenceLow:    12750.00,
		ConfidenceHigh:   17250.00,
		GrowthRate:       5.2,
		Forecast: []ForecastPoint{
			{Date: "2024-02-01", Value: 15000.00},
			{Date: "2024-03-01", Value: 15750.00},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ModelComparisonResponse represents model comparison response
type ModelComparisonResponse struct {
	Models []ModelComparisonData `json:"models"`
}

// ModelComparisonData represents model comparison data
type ModelComparisonData struct {
	Name          string  `json:"name"`
	CostPerToken  float64 `json:"cost_per_token"`
	AvgLatency    string  `json:"avg_latency"`
	QualityScore  float64 `json:"quality_score"`
	TotalRequests int64   `json:"total_requests"`
	TotalCost     float64 `json:"total_cost"`
	ErrorRate     float64 `json:"error_rate"`
}

func (api *CostIntelligenceAPI) handleModelComparison(w http.ResponseWriter, r *http.Request) {
	resp := ModelComparisonResponse{
		Models: []ModelComparisonData{
			{
				Name:          "gpt-4",
				CostPerToken:  0.00003,
				AvgLatency:    "1.2s",
				QualityScore:  9.5,
				TotalRequests: 5000,
				TotalCost:     800.00,
				ErrorRate:     0.5,
			},
			{
				Name:          "claude-3",
				CostPerToken:  0.000025,
				AvgLatency:    "1.5s",
				QualityScore:  9.2,
				TotalRequests: 3000,
				TotalCost:     350.00,
				ErrorRate:     0.8,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BudgetStatusResponse represents budget status response
type BudgetStatusResponse struct {
	MonthlyBudget  float64 `json:"monthly_budget"`
	MonthlySpend   float64 `json:"monthly_spend"`
	UsagePercent   float64 `json:"usage_percent"`
	ProjectedSpend float64 `json:"projected_spend"`
	DaysRemaining  int     `json:"days_remaining"`
	IsAtRisk       bool    `json:"is_at_risk"`
	BurnRate       float64 `json:"burn_rate"`
	AlertThreshold float64 `json:"alert_threshold"`
}

func (api *CostIntelligenceAPI) handleBudgetStatus(w http.ResponseWriter, r *http.Request) {
	resp := BudgetStatusResponse{
		MonthlyBudget:  20000.00,
		MonthlySpend:   1250.50,
		UsagePercent:   6.25,
		ProjectedSpend: 15000.00,
		DaysRemaining:  25,
		IsAtRisk:       false,
		BurnRate:       500.00,
		AlertThreshold: 80.0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// AnomaliesResponse represents anomalies response
type AnomaliesResponse struct {
	Anomalies []AnomalyData `json:"anomalies"`
	Total     int           `json:"total"`
}

// AnomalyData represents anomaly data
type AnomalyData struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
	Value       float64   `json:"value"`
	Expected    float64   `json:"expected"`
	Deviation   float64   `json:"deviation"`
}

func (api *CostIntelligenceAPI) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	resp := AnomaliesResponse{
		Anomalies: []AnomalyData{
			{
				Type:        "spike",
				Description: "Cost spike detected - 3x normal usage",
				Severity:    "high",
				DetectedAt:  time.Now().Add(-2 * time.Hour),
				Value:       450.00,
				Expected:    150.00,
				Deviation:   200.0,
			},
		},
		Total: 1,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// AlertsResponse represents alerts response
type AlertsResponse struct {
	Active   []AlertData `json:"active"`
	Resolved []AlertData `json:"resolved"`
	Total    int         `json:"total"`
}

// AlertData represents alert data
type AlertData struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Message    string     `json:"message"`
	Severity   string     `json:"severity"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	Value      float64    `json:"value"`
	Threshold  float64    `json:"threshold"`
}

func (api *CostIntelligenceAPI) handleAlerts(w http.ResponseWriter, r *http.Request) {
	resp := AlertsResponse{
		Active: []AlertData{
			{
				ID:        "alert-1",
				Type:      "budget_threshold",
				Message:   "Monthly budget usage exceeded 80%",
				Severity:  "warning",
				CreatedAt: time.Now().Add(-1 * time.Hour),
				Value:     85.0,
				Threshold: 80.0,
			},
		},
		Resolved: []AlertData{},
		Total:    1,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TeamCostsResponse represents team costs response
type TeamCostsResponse struct {
	Teams []TeamCostData `json:"teams"`
	Total float64        `json:"total"`
}

// TeamCostData represents team cost data
type TeamCostData struct {
	Name     string  `json:"name"`
	Cost     float64 `json:"cost"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Change   float64 `json:"change"`
}

func (api *CostIntelligenceAPI) handleTeamCosts(w http.ResponseWriter, r *http.Request) {
	resp := TeamCostsResponse{
		Teams: []TeamCostData{
			{Name: "Engineering", Cost: 600.00, Requests: 5000, Tokens: 2500000, Change: -2.5},
			{Name: "Research", Cost: 400.00, Requests: 3000, Tokens: 1500000, Change: 5.0},
			{Name: "Marketing", Cost: 250.50, Requests: 2000, Tokens: 1000000, Change: 10.0},
		},
		Total: 1250.50,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CostTrendsResponse represents cost trends response
type CostTrendsResponse struct {
	DataPoints []CostTrendPoint `json:"data_points"`
	Trend      string           `json:"trend"`
	ChangePct  float64          `json:"change_percent"`
}

// CostTrendPoint represents a cost trend data point
type CostTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Cost      float64   `json:"cost"`
	Tokens    int64     `json:"tokens"`
}

func (api *CostIntelligenceAPI) handleCostTrends(w http.ResponseWriter, r *http.Request) {
	resp := CostTrendsResponse{
		DataPoints: []CostTrendPoint{
			{Timestamp: time.Now().Add(-7 * 24 * time.Hour), Cost: 150.00, Tokens: 500000},
			{Timestamp: time.Now().Add(-6 * 24 * time.Hour), Cost: 175.00, Tokens: 600000},
		},
		Trend:     "increasing",
		ChangePct: 5.2,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ExportResponse represents export response
type ExportResponse struct {
	URL      string    `json:"url"`
	Format   string    `json:"format"`
	Expires  time.Time `json:"expires"`
	FileSize int64     `json:"file_size"`
}

func (api *CostIntelligenceAPI) handleExport(w http.ResponseWriter, r *http.Request) {
	resp := ExportResponse{
		URL:      "/api/v1/cost/export/download/abc123",
		Format:   "csv",
		Expires:  time.Now().Add(24 * time.Hour),
		FileSize: 102400,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
