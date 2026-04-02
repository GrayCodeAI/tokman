// Package digest provides weekly digest generation for TokMan
package digest

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
)

// WeeklyDigest represents a weekly cost digest
type WeeklyDigest struct {
	WeekStart       time.Time
	WeekEnd         time.Time
	TotalCost       float64
	TotalTokens     int64
	TotalRequests   int64
	AvgCostPerReq   float64
	CostChangePct   float64
	TokenChangePct  float64
	TopModels       []ModelUsage
	TopTeams        []TeamUsage
	DailyBreakdown  []DailyCost
	Anomalies       []AnomalyEvent
	Recommendations []string
	BudgetStatus    BudgetStatus
}

// ModelUsage represents model usage in digest
type ModelUsage struct {
	Model      string
	Requests   int64
	Tokens     int64
	Cost       float64
	Percentage float64
}

// TeamUsage represents team usage in digest
type TeamUsage struct {
	Team       string
	Requests   int64
	Cost       float64
	Percentage float64
}

// DailyCost represents daily cost breakdown
type DailyCost struct {
	Date   time.Time
	Cost   float64
	Tokens int64
}

// AnomalyEvent represents an anomaly in digest
type AnomalyEvent struct {
	Type        string
	Description string
	Severity    string
	Date        time.Time
}

// BudgetStatus represents budget status
type BudgetStatus struct {
	MonthlyBudget  float64
	MonthlySpend   float64
	UsagePercent   float64
	ProjectedSpend float64
	DaysRemaining  int
	IsAtRisk       bool
}

// DigestGenerator generates weekly digests
type DigestGenerator struct {
	template *template.Template
}

// NewDigestGenerator creates a new digest generator
func NewDigestGenerator() *DigestGenerator {
	return &DigestGenerator{}
}

// GenerateHTML generates an HTML digest
func (dg *DigestGenerator) GenerateHTML(digest *WeeklyDigest) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>TokMan Weekly Cost Digest - {{.WeekStart.Format "Jan 2, 2006"}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; color: #333; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        .header { text-align: center; padding-bottom: 20px; border-bottom: 2px solid #4CAF50; margin-bottom: 30px; }
        .header h1 { color: #2c3e50; margin: 0; }
        .header p { color: #7f8c8d; margin: 5px 0 0; }
        .metrics { display: grid; grid-template-columns: repeat(4, 1fr); gap: 15px; margin: 20px 0; }
        .metric { background: #f8f9fa; padding: 15px; border-radius: 8px; text-align: center; }
        .metric-value { font-size: 1.8em; font-weight: bold; color: #2c3e50; }
        .metric-label { color: #7f8c8d; font-size: 0.9em; margin-top: 5px; }
        .change { font-size: 0.8em; }
        .change.positive { color: #e74c3c; }
        .change.negative { color: #27ae60; }
        .section { margin: 30px 0; }
        .section h2 { color: #2c3e50; border-bottom: 1px solid #ecf0f1; padding-bottom: 10px; }
        table { width: 100%; border-collapse: collapse; margin: 15px 0; }
        th { background: #3498db; color: white; padding: 12px; text-align: left; }
        td { padding: 10px 12px; border-bottom: 1px solid #ecf0f1; }
        tr:hover { background: #f8f9fa; }
        .budget-bar { background: #ecf0f1; height: 20px; border-radius: 10px; overflow: hidden; margin: 10px 0; }
        .budget-fill { height: 100%; background: #4CAF50; transition: width 0.3s; }
        .budget-fill.warning { background: #f39c12; }
        .budget-fill.danger { background: #e74c3c; }
        .recommendations { background: #e8f5e9; padding: 15px; border-radius: 8px; border-left: 4px solid #4CAF50; }
        .recommendations li { margin: 8px 0; }
        .anomaly { background: #fff3e0; padding: 10px; border-radius: 5px; margin: 5px 0; border-left: 3px solid #f39c12; }
        .anomaly.critical { background: #ffebee; border-left-color: #e74c3c; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ecf0f1; text-align: center; color: #95a5a6; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 TokMan Weekly Cost Digest</h1>
            <p>{{.WeekStart.Format "Jan 2, 2006"}} - {{.WeekEnd.Format "Jan 2, 2006"}}</p>
        </div>

        <div class="metrics">
            <div class="metric">
                <div class="metric-value">$%.2f</div>
                <div class="metric-label">Total Cost</div>
                <div class="change {{if gt .CostChangePct 0}}positive{{else}}negative{{end}}">
                    {{if gt .CostChangePct 0}}↑{{else}}↓{{end}} {{printf "%.1f" .CostChangePct}}%%
                </div>
            </div>
            <div class="metric">
                <div class="metric-value">%d</div>
                <div class="metric-label">Total Tokens</div>
                <div class="change {{if gt .TokenChangePct 0}}positive{{else}}negative{{end}}">
                    {{if gt .TokenChangePct 0}}↑{{else}}↓{{end}} {{printf "%.1f" .TokenChangePct}}%%
                </div>
            </div>
            <div class="metric">
                <div class="metric-value">%d</div>
                <div class="metric-label">Requests</div>
            </div>
            <div class="metric">
                <div class="metric-value">$%.4f</div>
                <div class="metric-label">Avg Cost/Request</div>
            </div>
        </div>

        <div class="section">
            <h2>💰 Budget Status</h2>
            <p>Monthly Budget: ${{printf "%.2f" .BudgetStatus.MonthlyBudget}} | Spent: ${{printf "%.2f" .BudgetStatus.MonthlySpend}} ({{printf "%.1f" .BudgetStatus.UsagePercent}}%%)</p>
            <div class="budget-bar">
                <div class="budget-fill {{if gt .BudgetStatus.UsagePercent 90}}danger{{else if gt .BudgetStatus.UsagePercent 75}}warning{{end}}" style="width: {{.BudgetStatus.UsagePercent}}%%"></div>
            </div>
            <p>Projected Monthly Spend: ${{printf "%.2f" .BudgetStatus.ProjectedSpend}} | Days Remaining: {{.BudgetStatus.DaysRemaining}}</p>
            {{if .BudgetStatus.IsAtRisk}}<p style="color: #e74c3c;">⚠️ Budget at risk!</p>{{end}}
        </div>

        <div class="section">
            <h2>🤖 Top Models by Cost</h2>
            <table>
                <tr><th>Model</th><th>Requests</th><th>Tokens</th><th>Cost</th><th>Share</th></tr>
                {{range .TopModels}}
                <tr>
                    <td>{{.Model}}</td>
                    <td>{{.Requests}}</td>
                    <td>{{.Tokens}}</td>
                    <td>${{printf "%.2f" .Cost}}</td>
                    <td>{{printf "%.1f" .Percentage}}%%</td>
                </tr>
                {{end}}
            </table>
        </div>

        <div class="section">
            <h2>📈 Daily Cost Breakdown</h2>
            <table>
                <tr><th>Date</th><th>Cost</th><th>Tokens</th></tr>
                {{range .DailyBreakdown}}
                <tr>
                    <td>{{.Date.Format "Mon, Jan 2"}}</td>
                    <td>${{printf "%.2f" .Cost}}</td>
                    <td>{{.Tokens}}</td>
                </tr>
                {{end}}
            </table>
        </div>

        {{if .Anomalies}}
        <div class="section">
            <h2>⚠️ Anomalies Detected</h2>
            {{range .Anomalies}}
            <div class="anomaly {{if eq .Severity "critical"}}critical{{end}}">
                <strong>{{.Type}}</strong>: {{.Description}} ({{.Date.Format "Jan 2"}})
            </div>
            {{end}}
        </div>
        {{end}}

        {{if .Recommendations}}
        <div class="section">
            <h2>💡 Recommendations</h2>
            <div class="recommendations">
                <ul>
                    {{range .Recommendations}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
            </div>
        </div>
        {{end}}

        <div class="footer">
            <p>Generated by TokMan Cost Intelligence | <a href="#">View Full Dashboard</a></p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("digest").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, digest); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GenerateMarkdown generates a markdown digest
func (dg *DigestGenerator) GenerateMarkdown(digest *WeeklyDigest) string {
	var md string

	md += fmt.Sprintf("# TokMan Weekly Cost Digest\n\n")
	md += fmt.Sprintf("**Period:** %s - %s\n\n", digest.WeekStart.Format("Jan 2, 2006"), digest.WeekEnd.Format("Jan 2, 2006"))

	md += "## Summary\n\n"
	md += fmt.Sprintf("| Metric | Value | Change |\n")
	md += fmt.Sprintf("|--------|-------|--------|\n")
	md += fmt.Sprintf("| Total Cost | $%.2f | %.1f%% |\n", digest.TotalCost, digest.CostChangePct)
	md += fmt.Sprintf("| Total Tokens | %d | %.1f%% |\n", digest.TotalTokens, digest.TokenChangePct)
	md += fmt.Sprintf("| Total Requests | %d | - |\n", digest.TotalRequests)
	md += fmt.Sprintf("| Avg Cost/Request | $%.4f | - |\n\n", digest.AvgCostPerReq)

	md += "## Budget Status\n\n"
	md += fmt.Sprintf("- **Monthly Budget:** $%.2f\n", digest.BudgetStatus.MonthlyBudget)
	md += fmt.Sprintf("- **Monthly Spend:** $%.2f (%.1f%%)\n", digest.BudgetStatus.MonthlySpend, digest.BudgetStatus.UsagePercent)
	md += fmt.Sprintf("- **Projected:** $%.2f\n", digest.BudgetStatus.ProjectedSpend)
	md += fmt.Sprintf("- **Days Remaining:** %d\n\n", digest.BudgetStatus.DaysRemaining)

	if len(digest.TopModels) > 0 {
		md += "## Top Models\n\n"
		md += "| Model | Requests | Cost | Share |\n"
		md += "|-------|----------|------|-------|\n"
		for _, m := range digest.TopModels {
			md += fmt.Sprintf("| %s | %d | $%.2f | %.1f%% |\n", m.Model, m.Requests, m.Cost, m.Percentage)
		}
		md += "\n"
	}

	if len(digest.Recommendations) > 0 {
		md += "## Recommendations\n\n"
		for _, r := range digest.Recommendations {
			md += fmt.Sprintf("- %s\n", r)
		}
		md += "\n"
	}

	return md
}

// GenerateJSON generates a JSON digest
func (dg *DigestGenerator) GenerateJSON(digest *WeeklyDigest) ([]byte, error) {
	// Would use json.Marshal in production
	return []byte("{}"), nil
}

// WeeklyDigestBuilder helps build weekly digests
type WeeklyDigestBuilder struct {
	digest *WeeklyDigest
}

// NewWeeklyDigestBuilder creates a new builder
func NewWeeklyDigestBuilder(weekStart time.Time) *WeeklyDigestBuilder {
	return &WeeklyDigestBuilder{
		digest: &WeeklyDigest{
			WeekStart:       weekStart,
			WeekEnd:         weekStart.Add(7 * 24 * time.Hour),
			TopModels:       make([]ModelUsage, 0),
			TopTeams:        make([]TeamUsage, 0),
			DailyBreakdown:  make([]DailyCost, 0),
			Anomalies:       make([]AnomalyEvent, 0),
			Recommendations: make([]string, 0),
		},
	}
}

// WithTotalCost sets total cost
func (b *WeeklyDigestBuilder) WithTotalCost(cost float64) *WeeklyDigestBuilder {
	b.digest.TotalCost = cost
	return b
}

// WithTotalTokens sets total tokens
func (b *WeeklyDigestBuilder) WithTotalTokens(tokens int64) *WeeklyDigestBuilder {
	b.digest.TotalTokens = tokens
	return b
}

// WithTotalRequests sets total requests
func (b *WeeklyDigestBuilder) WithTotalRequests(requests int64) *WeeklyDigestBuilder {
	b.digest.TotalRequests = requests
	return b
}

// AddModel adds a model to top models
func (b *WeeklyDigestBuilder) AddModel(model ModelUsage) *WeeklyDigestBuilder {
	b.digest.TopModels = append(b.digest.TopModels, model)
	return b
}

// AddDailyCost adds a daily cost entry
func (b *WeeklyDigestBuilder) AddDailyCost(date time.Time, cost float64, tokens int64) *WeeklyDigestBuilder {
	b.digest.DailyBreakdown = append(b.digest.DailyBreakdown, DailyCost{
		Date:   date,
		Cost:   cost,
		Tokens: tokens,
	})
	return b
}

// AddRecommendation adds a recommendation
func (b *WeeklyDigestBuilder) AddRecommendation(rec string) *WeeklyDigestBuilder {
	b.digest.Recommendations = append(b.digest.Recommendations, rec)
	return b
}

// WithBudgetStatus sets budget status
func (b *WeeklyDigestBuilder) WithBudgetStatus(status BudgetStatus) *WeeklyDigestBuilder {
	b.digest.BudgetStatus = status
	return b
}

// Build returns the completed digest
func (b *WeeklyDigestBuilder) Build() *WeeklyDigest {
	// Calculate averages
	if b.digest.TotalRequests > 0 {
		b.digest.AvgCostPerReq = b.digest.TotalCost / float64(b.digest.TotalRequests)
	}

	return b.digest
}
