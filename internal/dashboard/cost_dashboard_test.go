package dashboard

import (
	"testing"
	"time"
)

func TestDashboardLayout(t *testing.T) {
	layouts := StandardDashboardLayouts()
	if len(layouts) < 1 {
		t.Fatal("expected at least one layout")
	}

	layout := layouts[0]
	if layout.Name != "Cost Overview" {
		t.Errorf("expected 'Cost Overview', got %s", layout.Name)
	}

	if len(layout.Panels) == 0 {
		t.Error("expected panels")
	}
}

func TestCreateDefaultLayout(t *testing.T) {
	layout := CreateDefaultLayout()
	if layout == nil {
		t.Fatal("expected layout")
	}
	if layout.Name != "Cost Overview" {
		t.Errorf("expected 'Cost Overview', got %s", layout.Name)
	}
}

func TestGetPanel(t *testing.T) {
	layout := CreateDefaultLayout()

	panel := layout.GetPanel("cost-per-request")
	if panel == nil {
		t.Fatal("expected panel")
	}
	if panel.Title != "Cost Per Request" {
		t.Errorf("expected 'Cost Per Request', got %s", panel.Title)
	}

	panel = layout.GetPanel("non-existent")
	if panel != nil {
		t.Error("expected nil for non-existent panel")
	}
}

func TestAddPanel(t *testing.T) {
	layout := CreateDefaultLayout()
	initialCount := len(layout.Panels)

	layout.AddPanel(Panel{
		ID:    "new-panel",
		Title: "New Panel",
		Type:  PanelCostPerRequest,
	})

	if len(layout.Panels) != initialCount+1 {
		t.Errorf("expected %d panels, got %d", initialCount+1, len(layout.Panels))
	}
}

func TestRemovePanel(t *testing.T) {
	layout := CreateDefaultLayout()
	initialCount := len(layout.Panels)

	layout.RemovePanel("cost-per-request")

	if len(layout.Panels) != initialCount-1 {
		t.Errorf("expected %d panels, got %d", initialCount-1, len(layout.Panels))
	}
}

func TestUpdatePanel(t *testing.T) {
	layout := CreateDefaultLayout()

	updates := Panel{
		ID:    "cost-per-request",
		Title: "Updated Title",
		Type:  PanelCostPerRequest,
	}

	updated := layout.UpdatePanel("cost-per-request", updates)
	if !updated {
		t.Error("expected update to succeed")
	}

	panel := layout.GetPanel("cost-per-request")
	if panel.Title != "Updated Title" {
		t.Errorf("expected 'Updated Title', got %s", panel.Title)
	}
}

func TestCostPerRequestWidget(t *testing.T) {
	widget := NewCostPerRequestWidget()

	widget.TotalRequests = 100
	widget.TotalCost = 50.0
	widget.Update(0.5)

	if widget.AverageCost != 0.5 {
		t.Errorf("expected avg cost 0.5, got %.2f", widget.AverageCost)
	}
}

func TestSpendForecastWidget(t *testing.T) {
	widget := NewSpendForecastWidget()
	widget.UpdateForecast(500.0, 15000.0, 5.0)

	if widget.ProjectedMonthly != 15000.0 {
		t.Errorf("expected projected 15000, got %.2f", widget.ProjectedMonthly)
	}
}

func TestTokenBreakdownWidget(t *testing.T) {
	widget := NewTokenBreakdownWidget()
	widget.UpdateBreakdown(5000000, 3000000, 1000000, 250.0, 150.0, 50.0)

	if widget.TotalTokens != 9000000 {
		t.Errorf("expected 9M tokens, got %d", widget.TotalTokens)
	}
	if widget.TotalCost != 450.0 {
		t.Errorf("expected $450, got %.2f", widget.TotalCost)
	}
}

func TestBudgetCapWidget(t *testing.T) {
	widget := NewBudgetCapWidget()
	widget.UpdateBudget(20000.0, 18000.0)

	if widget.UsagePercent != 90.0 {
		t.Errorf("expected 90%%, got %.2f", widget.UsagePercent)
	}
	if !widget.IsAtRisk {
		t.Error("expected at risk")
	}
}

func TestAnomalyDetectionWidget(t *testing.T) {
	widget := NewAnomalyDetectionWidget()

	widget.AddAnomaly(Anomaly{
		Type:        "spike",
		Description: "Cost spike",
		Severity:    "high",
		DetectedAt:  time.Now(),
	})

	if len(widget.Anomalies) != 1 {
		t.Errorf("expected 1 anomaly, got %d", len(widget.Anomalies))
	}

	widget.AddAnomaly(Anomaly{
		Type:        "pattern",
		Description: "Pattern deviation",
		Severity:    "critical",
		DetectedAt:  time.Now(),
	})

	high := widget.GetHighSeverityAnomalies()
	if len(high) != 2 {
		t.Errorf("expected 2 high severity, got %d", len(high))
	}
}

func TestCostAlertsWidget(t *testing.T) {
	widget := NewCostAlertsWidget()

	widget.AddAlert(CostAlert{
		ID:       "alert-1",
		Type:     "budget",
		Message:  "Budget exceeded",
		Severity: "warning",
	})

	if widget.AlertCount != 1 {
		t.Errorf("expected 1 alert, got %d", widget.AlertCount)
	}

	widget.ResolveAlert("alert-1")
	if widget.AlertCount != 0 {
		t.Errorf("expected 0 active alerts, got %d", widget.AlertCount)
	}
}

func TestPricingEditorWidget(t *testing.T) {
	widget := NewPricingEditorWidget()

	widget.SetModelPricing("gpt-4", ModelPricing{
		ModelName:   "gpt-4",
		InputPrice:  0.03,
		OutputPrice: 0.06,
	})

	pricing, ok := widget.GetModelPricing("gpt-4")
	if !ok {
		t.Error("expected pricing")
	}
	if pricing.InputPrice != 0.03 {
		t.Errorf("expected input price 0.03, got %.4f", pricing.InputPrice)
	}
}

func TestAllocationTagsWidget(t *testing.T) {
	widget := NewAllocationTagsWidget()

	widget.UpdateTagCost("engineering", 500.0)
	widget.UpdateTagCost("research", 300.0)

	if widget.TotalCost != 800.0 {
		t.Errorf("expected total cost 800, got %.2f", widget.TotalCost)
	}
}

func TestCostTrendWidget(t *testing.T) {
	widget := NewCostTrendWidget()

	widget.AddDataPoint(CostDataPoint{
		Timestamp: time.Now().Add(-24 * time.Hour),
		Cost:      100.0,
	})
	widget.AddDataPoint(CostDataPoint{
		Timestamp: time.Now(),
		Cost:      120.0,
	})

	trend := widget.GetTrend()
	if trend != "increasing" {
		t.Errorf("expected 'increasing', got %s", trend)
	}
}

func TestDashboardState(t *testing.T) {
	state := NewDashboardState()
	if state == nil {
		t.Fatal("expected state")
	}

	state.UpdateAll()

	output := state.String()
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestTeamUsageWidget(t *testing.T) {
	widget := NewAllocationTagsWidget()

	widget.UpdateTagCost("team-a", 100.0)
	widget.UpdateTagCost("team-b", 200.0)

	if widget.TotalCost != 300.0 {
		t.Errorf("expected 300, got %.2f", widget.TotalCost)
	}

	teamA := widget.Tags["team-a"]
	if teamA.Percentage != 33.33333333333333 {
		t.Errorf("expected 33.33%%, got %.2f", teamA.Percentage)
	}
}

func BenchmarkDashboardStateUpdate(b *testing.B) {
	state := NewDashboardState()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.UpdateAll()
	}
}
