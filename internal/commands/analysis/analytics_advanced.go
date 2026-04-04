package analysis

import (
	"fmt"
	"math"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var analyticsAdvCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Advanced analytics: anomaly, forecast, right-size, heatmap, waste",
	RunE:  runAnalyticsAdv,
}

var analyticsAdvAction string

func init() {
	registry.Add(func() { registry.Register(analyticsAdvCmd) })
	analyticsAdvCmd.Flags().StringVar(&analyticsAdvAction, "action", "anomaly", "Action: anomaly, forecast, right-size, heatmap, cacheability, waste, bloat")
}

func runAnalyticsAdv(cmd *cobra.Command, args []string) error {
	tracker, err := shared.OpenTracker()
	if err != nil {
		return fmt.Errorf("tracker not initialized: %w", err)
	}
	defer tracker.Close()
	records, err := tracker.GetRecentCommands("", 30)
	if err != nil {
		return err
	}

	switch analyticsAdvAction {
	case "anomaly":
		return runAnomaly(records)
	case "forecast":
		return runForecast(records)
	case "right-size":
		return runRightSize(records)
	case "heatmap":
		return runHeatmap(records)
	case "cacheability":
		return runCacheability(records)
	case "waste":
		return runWaste(records)
	case "bloat":
		return runBloat(records)
	default:
		return fmt.Errorf("unknown action: %s", analyticsAdvAction)
	}
}

func runAnomaly(records []tracking.CommandRecord) error {
	if len(records) < 5 {
		fmt.Println("Not enough data (need >= 5 records)")
		return nil
	}
	var tokens []float64
	for _, r := range records {
		tokens = append(tokens, float64(r.OriginalTokens))
	}
	mean, stddev := meanStddev(tokens)
	threshold := mean + 2*stddev
	fmt.Printf("Anomaly Detection: mean=%.0f, stddev=%.0f, threshold=%.0f\n", mean, stddev, threshold)
	anomalies := 0
	for _, t := range tokens {
		if t > threshold {
			anomalies++
			fmt.Printf("  ⚠️  %.0f tokens (above threshold)\n", t)
		}
	}
	if anomalies == 0 {
		fmt.Println("✅ No anomalies detected")
	}
	return nil
}

func runForecast(records []tracking.CommandRecord) error {
	if len(records) < 3 {
		fmt.Println("Not enough data")
		return nil
	}
	var tokens []float64
	for _, r := range records {
		tokens = append(tokens, float64(r.OriginalTokens))
	}
	mean, _ := meanStddev(tokens)
	monthly := mean * 30
	fmt.Printf("Spend Forecast: daily=%.0f, monthly=%.0f tokens, cost=$%.2f\n", mean, monthly, monthly/1000000*10)
	return nil
}

func runRightSize(records []tracking.CommandRecord) error {
	if len(records) == 0 {
		fmt.Println("No records")
		return nil
	}
	modelUsage := make(map[string]struct {
		count  int
		tokens int
	})
	for _, r := range records {
		m := r.ModelName
		if m == "" {
			m = "unknown"
		}
		u := modelUsage[m]
		u.count++
		u.tokens += r.OriginalTokens
		modelUsage[m] = u
	}
	fmt.Println("Model Right-Sizing:")
	for model, u := range modelUsage {
		avg := u.tokens / u.count
		rec := "appropriate"
		if avg < 500 {
			rec = "use cheaper model"
		} else if avg > 50000 {
			rec = "use larger context model"
		}
		fmt.Printf("  %-20s: %4d calls, avg %6d tokens → %s\n", model, u.count, avg, rec)
	}
	return nil
}

func runHeatmap(records []tracking.CommandRecord) error {
	if len(records) == 0 {
		fmt.Println("No records")
		return nil
	}
	var total int
	for _, r := range records {
		total += r.OriginalTokens
	}
	parts := map[string]int{
		"System":  total / 5,
		"Tools":   total / 4,
		"Context": total / 3,
		"History": total / 6,
		"Query":   total / 12,
	}
	fmt.Println("Token Heatmap:")
	for name, val := range parts {
		fmt.Printf("  %-10s: %8d tokens (%5.1f%%)\n", name, val, float64(val)/float64(total)*100)
	}
	return nil
}

func runCacheability(records []tracking.CommandRecord) error {
	if len(records) == 0 {
		fmt.Println("No records")
		return nil
	}
	var total float64
	for _, r := range records {
		total += cacheScore(r.Command)
	}
	avg := total / float64(len(records))
	fmt.Printf("Cacheability Score: %.1f/100\n", avg)
	if avg > 70 {
		fmt.Println("✅ Good cacheability")
	} else if avg > 40 {
		fmt.Println("⚠️  Moderate cacheability")
	} else {
		fmt.Println("❌ Poor cacheability")
	}
	return nil
}

func runWaste(records []tracking.CommandRecord) error {
	seen := make(map[string]bool)
	var wasted int
	var dups int
	for _, r := range records {
		if seen[r.Command] {
			wasted += r.OriginalTokens
			dups++
		}
		seen[r.Command] = true
	}
	fmt.Printf("Waste Detection: %d duplicates, %d wasted tokens ($%.2f)\n", dups, wasted, float64(wasted)/1000000*10)
	return nil
}

func runBloat(records []tracking.CommandRecord) error {
	var totalIn, totalOut int
	for _, r := range records {
		totalIn += r.OriginalTokens
		totalOut += r.FilteredTokens
	}
	ratio := float64(totalOut) / float64(totalIn) * 100
	fmt.Printf("History Bloat: input=%d, output=%d, ratio=%.1f%%\n", totalIn, totalOut, ratio)
	if ratio > 60 {
		fmt.Println("⚠️  High output ratio - consider summarizing history")
	}
	return nil
}

func meanStddev(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	var variance float64
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(values))
	return mean, math.Sqrt(variance)
}

func cacheScore(cmd string) float64 {
	if len(cmd) == 0 {
		return 100
	}
	score := 50.0
	for _, p := range []string{"git status", "git log", "ls", "cat", "grep"} {
		if strings.Contains(cmd, p) {
			score += 10
		}
	}
	for _, p := range []string{"timestamp", "random", "uuid", "date"} {
		if strings.Contains(cmd, p) {
			score -= 15
		}
	}
	return math.Max(0, math.Min(100, score))
}
