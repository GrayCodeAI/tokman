package benchmarking

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CICDIntegration provides CI/CD integration for benchmarks
type CICDIntegration struct {
	provider string
	config   CICDConfig
}

// CICDConfig holds CI/CD configuration
type CICDConfig struct {
	Provider         string
	Repository       string
	Branch           string
	CommitSHA        string
	PullRequestID    string
	BuildID          string
	Token            string
	ResultsPath      string
	BaselineBranch   string
	FailOnRegression bool
	Thresholds       RegressionThresholds
}

// NewCICDIntegration creates a new CI/CD integration
func NewCICDIntegration(config CICDConfig) *CICDIntegration {
	if config.Provider == "" {
		config.Provider = detectCIProvider()
	}

	return &CICDIntegration{
		provider: config.Provider,
		config:   config,
	}
}

// detectCIProvider detects the CI provider from environment
func detectCIProvider() string {
	envVars := map[string]string{
		"GITHUB_ACTIONS":  "github",
		"GITLAB_CI":       "gitlab",
		"CIRCLECI":        "circleci",
		"TRAVIS":          "travis",
		"JENKINS_URL":     "jenkins",
		"BUILDKITE":       "buildkite",
		"DRONE":           "drone",
		"AZURE_PIPELINES": "azure",
	}

	for env, provider := range envVars {
		if os.Getenv(env) != "" {
			return provider
		}
	}

	return "unknown"
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() CICDConfig {
	return CICDConfig{
		Provider:         detectCIProvider(),
		Repository:       getEnvOrDefault("REPOSITORY", ""),
		Branch:           getEnvOrDefault("BRANCH", "main"),
		CommitSHA:        getEnvOrDefault("COMMIT_SHA", ""),
		PullRequestID:    getEnvOrDefault("PR_ID", ""),
		BuildID:          getEnvOrDefault("BUILD_ID", ""),
		Token:            getEnvOrDefault("TOKEN", ""),
		ResultsPath:      getEnvOrDefault("RESULTS_PATH", "benchmark-results"),
		BaselineBranch:   getEnvOrDefault("BASELINE_BRANCH", "main"),
		FailOnRegression: getEnvOrDefault("FAIL_ON_REGRESSION", "true") == "true",
		Thresholds:       DefaultRegressionThresholds(),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RunAndReport runs benchmarks and reports results to CI/CD
func (ci *CICDIntegration) RunAndReport(suite *Suite) error {
	// Run benchmarks
	runner := NewRunner()
	runner.RegisterSuite(suite)

	ctx := createTimeoutContext(suite.duration)
	report, err := runner.RunSuite(ctx, suite.name)
	if err != nil {
		return fmt.Errorf("benchmark run failed: %w", err)
	}

	// Save results
	if err := ci.saveResults(report); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	// Compare with baseline if available
	if ci.config.BaselineBranch != "" {
		baseline, err := ci.loadBaseline()
		if err == nil && baseline != nil {
			detector := NewRegressionDetector(ci.config.Thresholds)
			regressions := detector.DetectRegressions(baseline.Results, report.Results)

			// Report regressions
			if err := ci.reportRegressions(regressions); err != nil {
				return err
			}

			// Fail build if configured
			if ci.config.FailOnRegression && len(regressions) > 0 {
				criticalCount := 0
				for _, r := range regressions {
					if r.Severity == SeverityCritical {
						criticalCount++
					}
				}
				if criticalCount > 0 {
					return fmt.Errorf("%d critical regressions detected", criticalCount)
				}
			}
		}
	}

	// Post results to CI
	return ci.postResults(report)
}

func (ci *CICDIntegration) saveResults(report *SuiteReport) error {
	// Ensure directory exists
	if err := os.MkdirAll(ci.config.ResultsPath, 0755); err != nil {
		return err
	}

	// Generate filename
	filename := fmt.Sprintf("benchmark-%s-%s.json",
		report.Name,
		report.StartTime.Format("20060102-150405"),
	)
	filepath := filepath.Join(ci.config.ResultsPath, filename)

	// Export as JSON
	formatter := NewJSONFormatter(true)
	data, err := formatter.FormatReport(report)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

func (ci *CICDIntegration) loadBaseline() (*SuiteReport, error) {
	// Look for baseline file
	pattern := filepath.Join(ci.config.ResultsPath, "benchmark-*-main-*.json")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no baseline found")
	}

	// Load most recent baseline
	latestFile := files[len(files)-1]
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var report SuiteReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	return &report, nil
}

func (ci *CICDIntegration) reportRegressions(regressions []Regression) error {
	if len(regressions) == 0 {
		return nil
	}

	// Generate report
	detector := NewRegressionDetector(ci.config.Thresholds)
	report := detector.GenerateReport([]BenchmarkResult{}, []BenchmarkResult{})
	report.Regressions = regressions

	// Format report
	output := FormatRegressionReport(report)

	// Post to CI based on provider
	switch ci.provider {
	case "github":
		return ci.postGitHubComment(output)
	case "gitlab":
		return ci.postGitLabComment(output)
	default:
		// Print to stdout
		fmt.Println(output)
		return nil
	}
}

func (ci *CICDIntegration) postGitHubComment(comment string) error {
	if ci.config.PullRequestID == "" || ci.config.Token == "" {
		fmt.Println(comment)
		return nil
	}

	// Use GitHub CLI if available
	if _, err := exec.LookPath("gh"); err == nil {
		cmd := exec.Command("gh", "pr", "comment", ci.config.PullRequestID, "--body", comment)
		return cmd.Run()
	}

	// Fallback to API
	fmt.Println("GitHub comment would be posted:")
	fmt.Println(comment)
	return nil
}

func (ci *CICDIntegration) postGitLabComment(comment string) error {
	fmt.Println("GitLab comment would be posted:")
	fmt.Println(comment)
	return nil
}

func (ci *CICDIntegration) postResults(report *SuiteReport) error {
	// Generate summary
	summary := report.Summary()

	output := fmt.Sprintf("\n## Benchmark Results\n\n")
	output += fmt.Sprintf("**Suite:** %s\n\n", report.Name)
	output += fmt.Sprintf("**Duration:** %v\n\n", report.Duration)
	output += fmt.Sprintf("**Benchmarks:** %d total\n\n", summary.TotalBenchmarks)
	output += fmt.Sprintf("**Success Rate:** %.2f%%\n\n", summary.SuccessRate)
	output += fmt.Sprintf("**Avg Throughput:** %.2f\n\n", summary.AvgThroughput)

	// Print to stdout for CI logs
	fmt.Println(output)

	// Set output variables if available
	ci.setOutput("benchmark_count", fmt.Sprintf("%d", summary.TotalBenchmarks))
	ci.setOutput("benchmark_success_rate", fmt.Sprintf("%.2f", summary.SuccessRate))

	return nil
}

func (ci *CICDIntegration) setOutput(name, value string) {
	// GitHub Actions
	if os.Getenv("GITHUB_OUTPUT") != "" {
		f, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintf(f, "%s=%s\n", name, value)
			f.Close()
		}
	}

	// GitLab CI
	if os.Getenv("CI") != "" {
		fmt.Printf("%s=%s\n", name, value)
	}
}

// GenerateGitHubActionsWorkflow generates GitHub Actions workflow YAML
func GenerateGitHubActionsWorkflow() string {
	return `name: Benchmark

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'

    - name: Run benchmarks
      run: |
        go test -bench=. -benchmem ./... | tee benchmark.txt

    - name: Compare benchmarks
      uses: benchmark-action/github-action-benchmark@v1
      with:
        tool: 'go'
        output-file-path: benchmark.txt
        github-token: ${{ secrets.GITHUB_TOKEN }}
        auto-push: true
        alert-threshold: '150%'
        comment-on-alert: true
        fail-on-alert: true
`
}

// GenerateGitLabCIConfig generates GitLab CI configuration
func GenerateGitLabCIConfig() string {
	return `benchmark:
  stage: test
  image: golang:1.21
  script:
    - go test -bench=. -benchmem ./... | tee benchmark.txt
    - go run ./cmd/benchmark-compare -baseline main -current $CI_COMMIT_SHA
  artifacts:
    paths:
      - benchmark.txt
    expire_in: 1 week
  only:
    - merge_requests
    - main
`
}

// CreateComment creates a formatted comment for posting to CI
func CreateComment(report *SuiteReport, regressions []Regression) string {
	var output strings.Builder

	output.WriteString("## 📊 Benchmark Results\n\n")

	// Summary
	summary := report.Summary()
	output.WriteString("### Summary\n\n")
	output.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	output.WriteString(fmt.Sprintf("|--------|-------|\n"))
	output.WriteString(fmt.Sprintf("| Total Benchmarks | %d |\n", summary.TotalBenchmarks))
	output.WriteString(fmt.Sprintf("| Success Rate | %.2f%% |\n", summary.SuccessRate))
	output.WriteString(fmt.Sprintf("| Avg Throughput | %.2f |\n", summary.AvgThroughput))
	output.WriteString(fmt.Sprintf("| Duration | %v |\n\n", report.Duration))

	// Regressions
	if len(regressions) > 0 {
		output.WriteString("### ⚠️ Regressions Detected\n\n")
		output.WriteString("| Benchmark | Metric | Change | Severity |\n")
		output.WriteString("|-----------|--------|--------|----------|\n")

		for _, r := range regressions {
			emoji := "⚠️"
			if r.Severity == SeverityCritical {
				emoji = "🚨"
			} else if r.Severity == SeverityHigh {
				emoji = "❌"
			}

			output.WriteString(fmt.Sprintf("| %s | %s %s | %.2f%% | %s |\n",
				emoji, r.BenchmarkID, r.Metric, r.ChangePct, r.Severity))
		}

		output.WriteString("\n")
	}

	// Details
	output.WriteString("### Benchmark Details\n\n")
	output.WriteString("<details>\n")
	output.WriteString("<summary>Click to expand</summary>\n\n")

	for _, r := range report.Results {
		output.WriteString(fmt.Sprintf("**%s**\n", r.Name))
		output.WriteString(fmt.Sprintf("- Duration: %v\n", r.Duration))
		output.WriteString(fmt.Sprintf("- Throughput: %.2f\n", r.Throughput))
		output.WriteString(fmt.Sprintf("- Memory: %.2f MB\n\n", r.MemoryUsedMB))
	}

	output.WriteString("</details>\n")

	return output.String()
}

// CommentPoster posts comments to CI systems
type CommentPoster interface {
	Post(comment string) error
}

// GitHubCommentPoster posts to GitHub
type GitHubCommentPoster struct {
	Token string
	Repo  string
	PR    string
}

// Post posts a comment to GitHub
func (g *GitHubCommentPoster) Post(comment string) error {
	// Implementation would use GitHub API
	fmt.Printf("Posting to GitHub PR %s:\n%s\n", g.PR, comment)
	return nil
}

// StatusUpdater updates commit status
type StatusUpdater interface {
	Update(status CommitStatus, description string) error
}

// CommitStatus represents commit status
type CommitStatus string

const (
	StatusSuccess CommitStatus = "success"
	StatusFailure CommitStatus = "failure"
	StatusPending CommitStatus = "pending"
	StatusError   CommitStatus = "error"
)

// createTimeoutContext creates a context with timeout
func createTimeoutContext(duration time.Duration) context.Context {
	if duration <= 0 {
		duration = 30 * time.Minute
	}
	ctx, _ := context.WithTimeout(context.Background(), duration)
	return ctx
}
