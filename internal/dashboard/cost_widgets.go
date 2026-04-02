// Package dashboard provides cost intelligence dashboard components
package dashboard

import (
	"fmt"
	"time"
)

// CostPerRequestWidget displays real-time cost per request
type CostPerRequestWidget struct {
	CurrentCost   float64
	AverageCost   float64
	P95Cost       float64
	P99Cost       float64
	TotalRequests int64
	TotalCost     float64
	TimeRange     string
	LastUpdated   time.Time
	RefreshRate   time.Duration
}

// NewCostPerRequestWidget creates a new widget
func NewCostPerRequestWidget() *CostPerRequestWidget {
	return &CostPerRequestWidget{
		RefreshRate: 30 * time.Second,
		TimeRange:   "24h",
	}
}

// Update updates widget data
func (w *CostPerRequestWidget) Update(cost float64) {
	w.CurrentCost = cost
	w.LastUpdated = time.Now()

	if w.TotalRequests > 0 {
		w.AverageCost = w.TotalCost / float64(w.TotalRequests)
	}
}

// SpendForecastWidget displays spend forecasting
type SpendForecastWidget struct {
	CurrentSpend     float64
	ProjectedMonthly float64
	ProjectedYearly  float64
	ConfidenceLow    float64
	ConfidenceHigh   float64
	GrowthRate       float64
	BudgetLimit      float64
	DaysRemaining    int
	ForecastDate     time.Time
}

// NewSpendForecastWidget creates a new widget
func NewSpendForecastWidget() *SpendForecastWidget {
	return &SpendForecastWidget{
		ForecastDate: time.Now().Add(30 * 24 * time.Hour),
	}
}

// UpdateForecast updates forecast data
func (w *SpendForecastWidget) UpdateForecast(current, projected, growth float64) {
	w.CurrentSpend = current
	w.ProjectedMonthly = projected
	w.GrowthRate = growth

	// Calculate confidence intervals
	w.ConfidenceLow = projected * 0.85
	w.ConfidenceHigh = projected * 1.15

	// Calculate days remaining in month
	now := time.Now()
	endOfMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Add(-24 * time.Hour)
	w.DaysRemaining = int(endOfMonth.Sub(now).Hours() / 24)
}

// TokenBreakdownWidget displays token cost breakdown
type TokenBreakdownWidget struct {
	InputTokens  int64
	OutputTokens int64
	CacheTokens  int64
	InputCost    float64
	OutputCost   float64
	CacheCost    float64
	TotalTokens  int64
	TotalCost    float64
	CostPerToken float64
	TimeRange    string
	Breakdown    map[string]TokenCost
}

// TokenCost represents token cost breakdown
type TokenCost struct {
	Type       string
	Tokens     int64
	Cost       float64
	Percentage float64
}

// NewTokenBreakdownWidget creates a new widget
func NewTokenBreakdownWidget() *TokenBreakdownWidget {
	return &TokenBreakdownWidget{
		TimeRange: "7d",
		Breakdown: make(map[string]TokenCost),
	}
}

// UpdateBreakdown updates token breakdown
func (w *TokenBreakdownWidget) UpdateBreakdown(inputTokens, outputTokens, cacheTokens int64, inputCost, outputCost, cacheCost float64) {
	w.InputTokens = inputTokens
	w.OutputTokens = outputTokens
	w.CacheTokens = cacheTokens
	w.InputCost = inputCost
	w.OutputCost = outputCost
	w.CacheCost = cacheCost

	w.TotalTokens = inputTokens + outputTokens + cacheTokens
	w.TotalCost = inputCost + outputCost + cacheCost

	if w.TotalTokens > 0 {
		w.CostPerToken = w.TotalCost / float64(w.TotalTokens)
	}

	// Update breakdown
	w.Breakdown["input"] = TokenCost{
		Type:       "input",
		Tokens:     inputTokens,
		Cost:       inputCost,
		Percentage: (inputCost / w.TotalCost) * 100,
	}
	w.Breakdown["output"] = TokenCost{
		Type:       "output",
		Tokens:     outputTokens,
		Cost:       outputCost,
		Percentage: (outputCost / w.TotalCost) * 100,
	}
	w.Breakdown["cache"] = TokenCost{
		Type:       "cache",
		Tokens:     cacheTokens,
		Cost:       cacheCost,
		Percentage: (cacheCost / w.TotalCost) * 100,
	}
}

// ModelComparisonWidget displays model cost comparison
type ModelComparisonWidget struct {
	Models        []ModelCostData
	SelectedModel string
	TimeRange     string
}

// ModelCostData represents model cost data
type ModelCostData struct {
	Name          string
	CostPerToken  float64
	AvgLatency    time.Duration
	QualityScore  float64
	TotalRequests int64
	TotalCost     float64
	ErrorRate     float64
}

// NewModelComparisonWidget creates a new widget
func NewModelComparisonWidget() *ModelComparisonWidget {
	return &ModelComparisonWidget{
		TimeRange: "7d",
		Models:    make([]ModelCostData, 0),
	}
}

// AddModel adds a model to comparison
func (w *ModelComparisonWidget) AddModel(model ModelCostData) {
	w.Models = append(w.Models, model)
}

// GetBestModel returns the best model by cost efficiency
func (w *ModelComparisonWidget) GetBestModel() *ModelCostData {
	if len(w.Models) == 0 {
		return nil
	}

	best := &w.Models[0]
	bestScore := best.QualityScore / (best.CostPerToken + 0.0001)

	for i := 1; i < len(w.Models); i++ {
		score := w.Models[i].QualityScore / (w.Models[i].CostPerToken + 0.0001)
		if score > bestScore {
			best = &w.Models[i]
			bestScore = score
		}
	}

	return best
}

// BudgetCapWidget displays budget cap status
type BudgetCapWidget struct {
	BudgetLimit     float64
	BudgetUsed      float64
	BudgetRemaining float64
	UsagePercent    float64
	BurnRate        float64
	DaysUntilCap    int
	AlertThreshold  float64
	IsAtRisk        bool
}

// NewBudgetCapWidget creates a new widget
func NewBudgetCapWidget() *BudgetCapWidget {
	return &BudgetCapWidget{
		AlertThreshold: 80.0,
	}
}

// UpdateBudget updates budget data
func (w *BudgetCapWidget) UpdateBudget(limit, used float64) {
	w.BudgetLimit = limit
	w.BudgetUsed = used
	w.BudgetRemaining = limit - used
	w.UsagePercent = (used / limit) * 100

	// Calculate burn rate (daily)
	w.BurnRate = used / 30.0

	// Calculate days until cap
	if w.BurnRate > 0 {
		w.DaysUntilCap = int(w.BudgetRemaining / w.BurnRate)
	}

	// Check if at risk
	w.IsAtRisk = w.UsagePercent >= w.AlertThreshold
}

// AnomalyDetectionWidget displays anomaly detection
type AnomalyDetectionWidget struct {
	Anomalies    []Anomaly
	LastScan     time.Time
	ScanInterval time.Duration
	Sensitivity  float64
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type        string
	Description string
	Severity    string
	DetectedAt  time.Time
	Value       float64
	Expected    float64
	Deviation   float64
}

// NewAnomalyDetectionWidget creates a new widget
func NewAnomalyDetectionWidget() *AnomalyDetectionWidget {
	return &AnomalyDetectionWidget{
		ScanInterval: 5 * time.Minute,
		Sensitivity:  2.0,
		Anomalies:    make([]Anomaly, 0),
	}
}

// AddAnomaly adds a detected anomaly
func (w *AnomalyDetectionWidget) AddAnomaly(anomaly Anomaly) {
	w.Anomalies = append(w.Anomalies, anomaly)
	w.LastScan = time.Now()
}

// GetHighSeverityAnomalies returns high severity anomalies
func (w *AnomalyDetectionWidget) GetHighSeverityAnomalies() []Anomaly {
	result := make([]Anomaly, 0)
	for _, a := range w.Anomalies {
		if a.Severity == "high" || a.Severity == "critical" {
			result = append(result, a)
		}
	}
	return result
}

// CostAlertsWidget displays cost alerts
type CostAlertsWidget struct {
	ActiveAlerts   []CostAlert
	ResolvedAlerts []CostAlert
	AlertCount     int
}

// CostAlert represents a cost alert
type CostAlert struct {
	ID         string
	Type       string
	Message    string
	Severity   string
	CreatedAt  time.Time
	ResolvedAt *time.Time
	Value      float64
	Threshold  float64
}

// NewCostAlertsWidget creates a new widget
func NewCostAlertsWidget() *CostAlertsWidget {
	return &CostAlertsWidget{
		ActiveAlerts:   make([]CostAlert, 0),
		ResolvedAlerts: make([]CostAlert, 0),
	}
}

// AddAlert adds a cost alert
func (w *CostAlertsWidget) AddAlert(alert CostAlert) {
	w.ActiveAlerts = append(w.ActiveAlerts, alert)
	w.AlertCount = len(w.ActiveAlerts)
}

// ResolveAlert resolves an alert
func (w *CostAlertsWidget) ResolveAlert(id string) {
	for i, alert := range w.ActiveAlerts {
		if alert.ID == id {
			now := time.Now()
			alert.ResolvedAt = &now
			w.ResolvedAlerts = append(w.ResolvedAlerts, alert)
			w.ActiveAlerts = append(w.ActiveAlerts[:i], w.ActiveAlerts[i+1:]...)
			w.AlertCount = len(w.ActiveAlerts)
			return
		}
	}
}

// PricingEditorWidget allows custom pricing configuration
type PricingEditorWidget struct {
	Models      map[string]ModelPricing
	Currency    string
	LastUpdated time.Time
}

// ModelPricing represents pricing for a model
type ModelPricing struct {
	ModelName     string
	InputPrice    float64 // per 1K tokens
	OutputPrice   float64 // per 1K tokens
	CachePrice    float64 // per 1K tokens
	EffectiveDate time.Time
}

// NewPricingEditorWidget creates a new widget
func NewPricingEditorWidget() *PricingEditorWidget {
	return &PricingEditorWidget{
		Models:   make(map[string]ModelPricing),
		Currency: "USD",
	}
}

// SetModelPricing sets pricing for a model
func (w *PricingEditorWidget) SetModelPricing(model string, pricing ModelPricing) {
	w.Models[model] = pricing
	w.LastUpdated = time.Now()
}

// GetModelPricing returns pricing for a model
func (w *PricingEditorWidget) GetModelPricing(model string) (ModelPricing, bool) {
	pricing, ok := w.Models[model]
	return pricing, ok
}

// AllocationTagsWidget displays cost allocation by tags
type AllocationTagsWidget struct {
	Tags      map[string]TagCost
	TotalCost float64
	TimeRange string
}

// TagCost represents cost for a tag
type TagCost struct {
	Tag        string
	Cost       float64
	Percentage float64
	Trend      float64
}

// NewAllocationTagsWidget creates a new widget
func NewAllocationTagsWidget() *AllocationTagsWidget {
	return &AllocationTagsWidget{
		Tags:      make(map[string]TagCost),
		TimeRange: "30d",
	}
}

// UpdateTagCost updates cost for a tag
func (w *AllocationTagsWidget) UpdateTagCost(tag string, cost float64) {
	w.Tags[tag] = TagCost{
		Tag:   tag,
		Cost:  cost,
		Trend: 0,
	}

	// Recalculate total
	w.TotalCost = 0
	for _, tc := range w.Tags {
		w.TotalCost += tc.Cost
	}

	// Recalculate percentages
	for tag := range w.Tags {
		if w.TotalCost > 0 {
			tc := w.Tags[tag]
			tc.Percentage = (tc.Cost / w.TotalCost) * 100
			w.Tags[tag] = tc
		}
	}
}

// CostTrendWidget displays cost trends over time
type CostTrendWidget struct {
	DataPoints  []CostDataPoint
	TimeRange   string
	Aggregation string
}

// CostDataPoint represents a cost data point
type CostDataPoint struct {
	Timestamp time.Time
	Cost      float64
	Tokens    int64
	Requests  int64
}

// NewCostTrendWidget creates a new widget
func NewCostTrendWidget() *CostTrendWidget {
	return &CostTrendWidget{
		DataPoints:  make([]CostDataPoint, 0),
		TimeRange:   "90d",
		Aggregation: "daily",
	}
}

// AddDataPoint adds a cost data point
func (w *CostTrendWidget) AddDataPoint(point CostDataPoint) {
	w.DataPoints = append(w.DataPoints, point)
}

// GetTrend returns the cost trend direction
func (w *CostTrendWidget) GetTrend() string {
	if len(w.DataPoints) < 2 {
		return "stable"
	}

	first := w.DataPoints[0].Cost
	last := w.DataPoints[len(w.DataPoints)-1].Cost

	change := ((last - first) / first) * 100

	switch {
	case change > 10:
		return "increasing"
	case change < -10:
		return "decreasing"
	default:
		return "stable"
	}
}

// DashboardState represents the complete dashboard state
type DashboardState struct {
	CostPerRequest   *CostPerRequestWidget
	SpendForecast    *SpendForecastWidget
	TokenBreakdown   *TokenBreakdownWidget
	ModelComparison  *ModelComparisonWidget
	BudgetCap        *BudgetCapWidget
	AnomalyDetection *AnomalyDetectionWidget
	CostAlerts       *CostAlertsWidget
	PricingEditor    *PricingEditorWidget
	AllocationTags   *AllocationTagsWidget
	CostTrend        *CostTrendWidget
}

// NewDashboardState creates a new dashboard state
func NewDashboardState() *DashboardState {
	return &DashboardState{
		CostPerRequest:   NewCostPerRequestWidget(),
		SpendForecast:    NewSpendForecastWidget(),
		TokenBreakdown:   NewTokenBreakdownWidget(),
		ModelComparison:  NewModelComparisonWidget(),
		BudgetCap:        NewBudgetCapWidget(),
		AnomalyDetection: NewAnomalyDetectionWidget(),
		CostAlerts:       NewCostAlertsWidget(),
		PricingEditor:    NewPricingEditorWidget(),
		AllocationTags:   NewAllocationTagsWidget(),
		CostTrend:        NewCostTrendWidget(),
	}
}

// UpdateAll updates all widgets with sample data
func (ds *DashboardState) UpdateAll() {
	// Update cost per request
	ds.CostPerRequest.Update(0.05)
	ds.CostPerRequest.TotalRequests = 10000
	ds.CostPerRequest.TotalCost = 500.0
	ds.CostPerRequest.AverageCost = 0.05
	ds.CostPerRequest.P95Cost = 0.08
	ds.CostPerRequest.P99Cost = 0.12

	// Update spend forecast
	ds.SpendForecast.UpdateForecast(500.0, 15000.0, 5.0)
	ds.SpendForecast.BudgetLimit = 20000.0

	// Update token breakdown
	ds.TokenBreakdown.UpdateBreakdown(5000000, 3000000, 1000000, 250.0, 150.0, 50.0)

	// Update budget cap
	ds.BudgetCap.UpdateBudget(20000.0, 500.0)

	// Format state as string for display
	_ = ds.String()
}

// String returns a string representation of the dashboard state
func (ds *DashboardState) String() string {
	output := "=== Cost Intelligence Dashboard ===\n\n"

	output += fmt.Sprintf("Cost Per Request: $%.4f (avg), $%.4f (p95)\n",
		ds.CostPerRequest.AverageCost, ds.CostPerRequest.P95Cost)

	output += fmt.Sprintf("Spend Forecast: $%.2f/month (budget: $%.2f)\n",
		ds.SpendForecast.ProjectedMonthly, ds.SpendForecast.BudgetLimit)

	output += fmt.Sprintf("Token Usage: %d total ($%.2f)\n",
		ds.TokenBreakdown.TotalTokens, ds.TokenBreakdown.TotalCost)

	output += fmt.Sprintf("Budget: %.1f%% used ($%.2f / $%.2f)\n",
		ds.BudgetCap.UsagePercent, ds.BudgetCap.BudgetUsed, ds.BudgetCap.BudgetLimit)

	output += fmt.Sprintf("Active Alerts: %d\n", ds.CostAlerts.AlertCount)

	output += fmt.Sprintf("Cost Trend: %s\n", ds.CostTrend.GetTrend())

	return output
}
