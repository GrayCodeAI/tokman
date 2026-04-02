// Package stress provides report generation capabilities
package stress

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// ReportGenerator generates stress test reports
type ReportGenerator struct {
	format string
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(format string) *ReportGenerator {
	return &ReportGenerator{format: format}
}

// Generate generates a report from results
func (rg *ReportGenerator) Generate(result *Result) ([]byte, error) {
	switch rg.format {
	case "html":
		return rg.generateHTML(result)
	case "pdf":
		return rg.generatePDF(result)
	case "json":
		return rg.generateJSON(result)
	case "csv":
		return rg.generateCSV(result)
	case "markdown":
		return rg.generateMarkdown(result)
	default:
		return nil, fmt.Errorf("unsupported format: %s", rg.format)
	}
}

func (rg *ReportGenerator) generateHTML(result *Result) ([]byte, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Stress Test Report - {{.Scenario}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin: 20px 0; }
        .metric-card { background: #f8f9fa; padding: 20px; border-radius: 8px; border-left: 4px solid #4CAF50; }
        .metric-value { font-size: 2em; font-weight: bold; color: #333; }
        .metric-label { color: #666; margin-top: 5px; }
        .success { color: #4CAF50; }
        .warning { color: #FF9800; }
        .error { color: #f44336; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #4CAF50; color: white; }
        tr:hover { background: #f5f5f5; }
        .chart { background: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #999; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Stress Test Report</h1>
        
        <div class="summary">
            <h2>Test Summary</h2>
            <p><strong>Scenario:</strong> {{.Scenario}}</p>
            <p><strong>Type:</strong> {{.Type}}</p>
            <p><strong>Duration:</strong> {{.Duration}}</p>
            <p><strong>Generated:</strong> {{.EndTime.Format "2006-01-02 15:04:05"}}</p>
        </div>

        <h2>📊 Key Metrics</h2>
        <div class="metric-grid">
            <div class="metric-card">
                <div class="metric-value {{if ge .SuccessRate 95.0}}success{{else if ge .SuccessRate 80.0}}warning{{else}}error{{end}}">
                    {{printf "%.2f" (div .SuccessCount .TotalRequests 100)}}%
                </div>
                <div class="metric-label">Success Rate</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.TotalRequests}}</div>
                <div class="metric-label">Total Requests</div>
            </div>
            <div class="metric-card">
                <div class="metric-value {{if le .ErrorCount 10}}success{{else if le .ErrorCount 100}}warning{{else}}error{{end}}">
                    {{.ErrorCount}}
                </div>
                <div class="metric-label">Errors</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{printf "%.2f" .ThroughputRPS}}</div>
                <div class="metric-label">Throughput (RPS)</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.LatencyP50}}</div>
                <div class="metric-label">P50 Latency</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.LatencyP95}}</div>
                <div class="metric-label">P95 Latency</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.LatencyP99}}</div>
                <div class="metric-label">P99 Latency</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.MinLatency}}</div>
                <div class="metric-label">Min Latency</div>
            </div>
        </div>

        <h2>📈 Latency Distribution</h2>
        <div class="chart">
            <table>
                <tr>
                    <th>Percentile</th>
                    <th>Latency</th>
                </tr>
                <tr>
                    <td>P50 (Median)</td>
                    <td>{{.LatencyP50}}</td>
                </tr>
                <tr>
                    <td>P95</td>
                    <td>{{.LatencyP95}}</td>
                </tr>
                <tr>
                    <td>P99</td>
                    <td>{{.LatencyP99}}</td>
                </tr>
                <tr>
                    <td>Maximum</td>
                    <td>{{.MaxLatency}}</td>
                </tr>
            </table>
        </div>

        <h2>🔍 Error Details</h2>
        <div class="chart">
            <p><strong>Total Errors:</strong> {{.ErrorCount}}</p>
            <p><strong>Timeouts:</strong> {{.TimeoutCount}}</p>
            <p><strong>Error Rate:</strong> {{printf "%.4f" .ErrorRate}}%</p>
        </div>

        <div class="footer">
            <p>Generated by TokMan Stress Testing Framework</p>
        </div>
    </div>
</body>
</html>`

	t := template.Must(template.New("report").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, result); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (rg *ReportGenerator) generatePDF(result *Result) ([]byte, error) {
	// For now, return HTML that can be converted to PDF externally
	// In production, would use a PDF library like gofpdf or unidoc
	return rg.generateHTML(result)
}

func (rg *ReportGenerator) generateJSON(result *Result) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

func (rg *ReportGenerator) generateCSV(result *Result) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write headers
	writer.Write([]string{"Metric", "Value"})

	// Write data
	writer.Write([]string{"Scenario", result.Scenario})
	writer.Write([]string{"Type", string(result.Type)})
	writer.Write([]string{"Duration", result.Duration.String()})
	writer.Write([]string{"Total Requests", fmt.Sprintf("%d", result.TotalRequests)})
	writer.Write([]string{"Success Count", fmt.Sprintf("%d", result.SuccessCount)})
	writer.Write([]string{"Error Count", fmt.Sprintf("%d", result.ErrorCount)})
	writer.Write([]string{"Timeout Count", fmt.Sprintf("%d", result.TimeoutCount)})
	writer.Write([]string{"Success Rate", fmt.Sprintf("%.2f", result.SuccessRate)})
	writer.Write([]string{"Throughput RPS", fmt.Sprintf("%.2f", result.ThroughputRPS)})
	writer.Write([]string{"P50 Latency", result.LatencyP50.String()})
	writer.Write([]string{"P95 Latency", result.LatencyP95.String()})
	writer.Write([]string{"P99 Latency", result.LatencyP99.String()})
	writer.Write([]string{"Min Latency", result.MinLatency.String()})
	writer.Write([]string{"Max Latency", result.MaxLatency.String()})

	writer.Flush()
	return buf.Bytes(), nil
}

func (rg *ReportGenerator) generateMarkdown(result *Result) ([]byte, error) {
	var md strings.Builder

	md.WriteString("# Stress Test Report\n\n")
	md.WriteString(fmt.Sprintf("**Scenario:** %s\n\n", result.Scenario))
	md.WriteString(fmt.Sprintf("**Type:** %s\n\n", result.Type))
	md.WriteString(fmt.Sprintf("**Duration:** %s\n\n", result.Duration))
	md.WriteString(fmt.Sprintf("**Generated:** %s\n\n", result.EndTime.Format(time.RFC3339)))

	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	md.WriteString(fmt.Sprintf("|--------|-------|\n"))
	md.WriteString(fmt.Sprintf("| Total Requests | %d |\n", result.TotalRequests))
	md.WriteString(fmt.Sprintf("| Success Rate | %.2f%% |\n", result.SuccessRate))
	md.WriteString(fmt.Sprintf("| Errors | %d |\n", result.ErrorCount))
	md.WriteString(fmt.Sprintf("| Throughput | %.2f RPS |\n", result.ThroughputRPS))
	md.WriteString(fmt.Sprintf("| P50 Latency | %s |\n", result.LatencyP50))
	md.WriteString(fmt.Sprintf("| P95 Latency | %s |\n", result.LatencyP95))
	md.WriteString(fmt.Sprintf("| P99 Latency | %s |\n\n", result.LatencyP99))

	md.WriteString("## Latency Distribution\n\n")
	md.WriteString(fmt.Sprintf("- **Min:** %s\n", result.MinLatency))
	md.WriteString(fmt.Sprintf("- **P50:** %s\n", result.LatencyP50))
	md.WriteString(fmt.Sprintf("- **P95:** %s\n", result.LatencyP95))
	md.WriteString(fmt.Sprintf("- **P99:** %s\n", result.LatencyP99))
	md.WriteString(fmt.Sprintf("- **Max:** %s\n\n", result.MaxLatency))

	if len(result.Errors) > 0 {
		md.WriteString("## Errors\n\n")
		for errType, count := range result.Errors {
			md.WriteString(fmt.Sprintf("- %s: %d\n", errType, count))
		}
		md.WriteString("\n")
	}

	md.WriteString("---\n\n")
	md.WriteString("*Generated by TokMan Stress Testing*\n")

	return []byte(md.String()), nil
}

// ReportScheduler schedules automatic report generation
type ReportScheduler struct {
	interval time.Duration
	formats  []string
	dir      string
}

// NewReportScheduler creates a new scheduler
func NewReportScheduler(interval time.Duration, formats []string, dir string) *ReportScheduler {
	return &ReportScheduler{
		interval: interval,
		formats:  formats,
		dir:      dir,
	}
}

// Schedule schedules report generation for a test
func (rs *ReportScheduler) Schedule(result *Result) error {
	for _, format := range rs.formats {
		generator := NewReportGenerator(format)
		data, err := generator.Generate(result)
		if err != nil {
			continue
		}

		// Save to file
		filename := fmt.Sprintf("%s/stress-report-%s-%s.%s",
			rs.dir,
			result.Scenario,
			time.Now().Format("20060102-150405"),
			format,
		)

		// Would write to file in production
		_ = data
		_ = filename
	}

	return nil
}

// ResultComparison compares two stress test results
type ResultComparison struct {
	Baseline *Result
	Current  *Result
	Changes  map[string]MetricChange
}

// MetricChange represents a change in a metric
type MetricChange struct {
	Name      string
	Baseline  float64
	Current   float64
	Change    float64
	ChangePct float64
}

// CompareResults compares two results
func CompareResults(baseline, current *Result) *ResultComparison {
	comparison := &ResultComparison{
		Baseline: baseline,
		Current:  current,
		Changes:  make(map[string]MetricChange),
	}

	// Compare metrics
	compareMetric(comparison, "success_rate", baseline.SuccessRate, current.SuccessRate)
	compareMetric(comparison, "throughput", baseline.ThroughputRPS, current.ThroughputRPS)
	compareMetric(comparison, "latency_p50", float64(baseline.LatencyP50), float64(current.LatencyP50))
	compareMetric(comparison, "latency_p95", float64(baseline.LatencyP95), float64(current.LatencyP95))
	compareMetric(comparison, "errors", float64(baseline.ErrorCount), float64(current.ErrorCount))

	return comparison
}

func compareMetric(c *ResultComparison, name string, baseline, current float64) {
	var change, changePct float64
	if baseline != 0 {
		change = current - baseline
		changePct = (change / baseline) * 100
	}

	c.Changes[name] = MetricChange{
		Name:      name,
		Baseline:  baseline,
		Current:   current,
		Change:    change,
		ChangePct: changePct,
	}
}

// FormatComparison formats a comparison as a string
func FormatComparison(c *ResultComparison) string {
	var output strings.Builder

	output.WriteString("Stress Test Comparison\n")
	output.WriteString("======================\n\n")

	output.WriteString(fmt.Sprintf("Baseline: %s\n", c.Baseline.Scenario))
	output.WriteString(fmt.Sprintf("Current:  %s\n\n", c.Current.Scenario))

	output.WriteString("Metric Changes:\n")
	output.WriteString(fmt.Sprintf("%-20s %12s %12s %12s %12s\n", "Metric", "Baseline", "Current", "Change", "Change %"))
	output.WriteString(strings.Repeat("-", 70) + "\n")

	for _, change := range c.Changes {
		symbol := "="
		if change.Change > 0 {
			symbol = "▲"
		} else if change.Change < 0 {
			symbol = "▼"
		}

		output.WriteString(fmt.Sprintf("%-20s %12.2f %12.2f %12.2f %11.2f%% %s\n",
			change.Name,
			change.Baseline,
			change.Current,
			change.Change,
			change.ChangePct,
			symbol,
		))
	}

	return output.String()
}

// ExportReport exports results to various formats
func ExportReport(result *Result, format string) ([]byte, error) {
	generator := NewReportGenerator(format)
	return generator.Generate(result)
}

// BatchReport generates reports for multiple results
func BatchReport(results []*Result, format string) (map[string][]byte, error) {
	reports := make(map[string][]byte)

	for _, result := range results {
		generator := NewReportGenerator(format)
		data, err := generator.Generate(result)
		if err != nil {
			continue
		}

		key := fmt.Sprintf("%s-%s", result.Scenario, result.EndTime.Format("20060102"))
		reports[key] = data
	}

	return reports, nil
}
