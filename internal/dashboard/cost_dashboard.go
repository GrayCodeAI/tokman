// Package dashboard provides cost intelligence dashboard components
package dashboard

// DashboardLayout defines the layout structure for the cost intelligence dashboard
type DashboardLayout struct {
	ID          string
	Name        string
	Description string
	Panels      []Panel
	CreatedAt   string
	UpdatedAt   string
}

// Panel represents a dashboard panel
type Panel struct {
	ID       string
	Title    string
	Type     PanelType
	Position Position
	Size     Size
	Config   PanelConfig
}

// PanelType represents the type of panel
type PanelType string

const (
	PanelCostPerRequest PanelType = "cost_per_request"
	PanelSpendForecast  PanelType = "spend_forecast"
	PanelTokenBreakdown PanelType = "token_breakdown"
	PanelModelCompare   PanelType = "model_comparison"
	PanelBudgetCap      PanelType = "budget_cap"
	PanelAnomalyDetect  PanelType = "anomaly_detection"
	PanelCostAlerts     PanelType = "cost_alerts"
	PanelPricingEditor  PanelType = "pricing_editor"
	PanelAllocationTags PanelType = "allocation_tags"
	PanelCostTrend      PanelType = "cost_trend"
)

// Position represents panel position
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Size represents panel size
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// PanelConfig holds panel-specific configuration
type PanelConfig struct {
	RefreshInterval int
	TimeRange       string
	Metrics         []string
	Aggregation     string
	Filters         map[string]string
}

// StandardDashboardLayouts returns standard dashboard layouts
func StandardDashboardLayouts() []DashboardLayout {
	return []DashboardLayout{
		{
			ID:          "cost-overview",
			Name:        "Cost Overview",
			Description: "High-level cost intelligence dashboard",
			Panels: []Panel{
				{
					ID:       "cost-per-request",
					Title:    "Cost Per Request",
					Type:     PanelCostPerRequest,
					Position: Position{X: 0, Y: 0},
					Size:     Size{Width: 6, Height: 4},
					Config: PanelConfig{
						RefreshInterval: 30,
						TimeRange:       "24h",
						Metrics:         []string{"avg_cost", "p95_cost", "total_cost"},
					},
				},
				{
					ID:       "spend-forecast",
					Title:    "Spend Forecast",
					Type:     PanelSpendForecast,
					Position: Position{X: 6, Y: 0},
					Size:     Size{Width: 6, Height: 4},
					Config: PanelConfig{
						RefreshInterval: 300,
						TimeRange:       "30d",
						Metrics:         []string{"projected_spend", "confidence_interval"},
					},
				},
				{
					ID:       "token-breakdown",
					Title:    "Token Cost Breakdown",
					Type:     PanelTokenBreakdown,
					Position: Position{X: 0, Y: 4},
					Size:     Size{Width: 8, Height: 6},
					Config: PanelConfig{
						RefreshInterval: 60,
						TimeRange:       "7d",
						Metrics:         []string{"input_tokens", "output_tokens", "cache_tokens"},
						Aggregation:     "sum",
					},
				},
				{
					ID:       "model-comparison",
					Title:    "Model Cost Comparison",
					Type:     PanelModelCompare,
					Position: Position{X: 8, Y: 4},
					Size:     Size{Width: 4, Height: 6},
					Config: PanelConfig{
						RefreshInterval: 300,
						TimeRange:       "7d",
						Metrics:         []string{"cost_per_token", "latency", "quality_score"},
					},
				},
				{
					ID:       "budget-cap",
					Title:    "Budget Cap Status",
					Type:     PanelBudgetCap,
					Position: Position{X: 0, Y: 10},
					Size:     Size{Width: 4, Height: 4},
					Config: PanelConfig{
						RefreshInterval: 60,
						TimeRange:       "30d",
						Metrics:         []string{"budget_used", "budget_remaining", "burn_rate"},
					},
				},
				{
					ID:       "anomaly-detection",
					Title:    "Anomaly Detection",
					Type:     PanelAnomalyDetect,
					Position: Position{X: 4, Y: 10},
					Size:     Size{Width: 4, Height: 4},
					Config: PanelConfig{
						RefreshInterval: 300,
						TimeRange:       "7d",
						Metrics:         []string{"spike_detection", "pattern_deviation"},
					},
				},
				{
					ID:       "cost-alerts",
					Title:    "Cost Alerts",
					Type:     PanelCostAlerts,
					Position: Position{X: 8, Y: 10},
					Size:     Size{Width: 4, Height: 4},
					Config: PanelConfig{
						RefreshInterval: 30,
						TimeRange:       "24h",
						Metrics:         []string{"active_alerts", "recent_alerts"},
					},
				},
			},
		},
		{
			ID:          "team-costs",
			Name:        "Team Cost Allocation",
			Description: "Team-based cost allocation and chargeback",
			Panels: []Panel{
				{
					ID:       "allocation-tags",
					Title:    "Cost Allocation Tags",
					Type:     PanelAllocationTags,
					Position: Position{X: 0, Y: 0},
					Size:     Size{Width: 6, Height: 6},
					Config: PanelConfig{
						RefreshInterval: 300,
						TimeRange:       "30d",
						Metrics:         []string{"cost_by_team", "cost_by_project"},
						Aggregation:     "sum",
					},
				},
				{
					ID:       "pricing-editor",
					Title:    "Custom Pricing Editor",
					Type:     PanelPricingEditor,
					Position: Position{X: 6, Y: 0},
					Size:     Size{Width: 6, Height: 6},
					Config: PanelConfig{
						RefreshInterval: 0,
						Metrics:         []string{"input_price", "output_price", "cache_price"},
					},
				},
				{
					ID:       "cost-trend",
					Title:    "Cost Trend",
					Type:     PanelCostTrend,
					Position: Position{X: 0, Y: 6},
					Size:     Size{Width: 12, Height: 6},
					Config: PanelConfig{
						RefreshInterval: 300,
						TimeRange:       "90d",
						Metrics:         []string{"daily_spend", "weekly_spend", "monthly_spend"},
						Aggregation:     "sum",
					},
				},
			},
		},
	}
}

// CreateDefaultLayout creates the default dashboard layout
func CreateDefaultLayout() *DashboardLayout {
	layouts := StandardDashboardLayouts()
	return &layouts[0]
}

// GetPanel returns a panel by ID
func (dl *DashboardLayout) GetPanel(id string) *Panel {
	for i := range dl.Panels {
		if dl.Panels[i].ID == id {
			return &dl.Panels[i]
		}
	}
	return nil
}

// AddPanel adds a panel to the layout
func (dl *DashboardLayout) AddPanel(panel Panel) {
	dl.Panels = append(dl.Panels, panel)
}

// RemovePanel removes a panel from the layout
func (dl *DashboardLayout) RemovePanel(id string) {
	for i, p := range dl.Panels {
		if p.ID == id {
			dl.Panels = append(dl.Panels[:i], dl.Panels[i+1:]...)
			return
		}
	}
}

// UpdatePanel updates a panel in the layout
func (dl *DashboardLayout) UpdatePanel(id string, updates Panel) bool {
	for i := range dl.Panels {
		if dl.Panels[i].ID == id {
			dl.Panels[i] = updates
			return true
		}
	}
	return false
}
