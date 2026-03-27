package benchmarks

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

// ============================================================================
// 1000-AGENT STRESS TEST
// Simulates 1000 concurrent AI agents to identify bottlenecks
// ============================================================================

// AgentProfile represents different agent workload patterns
type AgentProfile struct {
	Name         string
	InputType    string
	InputSize    int    // tokens
	Frequency    string // how often this agent runs
	Priority     int    // 1=high, 2=medium, 3=low
	Mode         filter.Mode
}

// RealisticAgentProfiles based on actual AI assistant usage patterns
var RealisticAgentProfiles = []AgentProfile{
	// High-frequency small queries (most common)
	{"QuickCodeReview", "code", 200, "high", 1, filter.ModeMinimal},
	{"FileRead", "code", 500, "high", 1, filter.ModeMinimal},
	{"GitStatus", "vcs", 100, "high", 1, filter.ModeMinimal},
	{"QuickSearch", "mixed", 300, "high", 1, filter.ModeMinimal},
	
	// Medium-frequency medium queries
	{"CodeRefactor", "code", 2000, "medium", 2, filter.ModeMinimal},
	{"LogAnalysis", "logs", 5000, "medium", 2, filter.ModeAggressive},
	{"Documentation", "markdown", 3000, "medium", 2, filter.ModeMinimal},
	{"TestRunner", "code", 1500, "medium", 2, filter.ModeMinimal},
	
	// Low-frequency large queries
	{"FullRepoScan", "code", 50000, "low", 3, filter.ModeAggressive},
	{"LargeDiff", "vcs", 20000, "low", 3, filter.ModeAggressive},
	{"LongConversation", "conversation", 10000, "low", 3, filter.ModeMinimal},
	{"DeepAnalysis", "mixed", 15000, "low", 3, filter.ModeAggressive},
}

// MetricCollector collects performance metrics from all agents
type MetricCollector struct {
	mu                sync.Mutex
	TotalAgents       int32
	CompletedAgents   int32
	FailedAgents      int32
	TotalLatency      int64 // microseconds
	TotalTokensIn     int64
	TotalTokensOut    int64
	TotalTokensSaved  int64
	PeakMemoryMB      int64
	Latencies         []time.Duration
	Errors            []error
	LayerStats        map[string]*LayerMetric
	Bottlenecks       map[string]int // layer -> bottleneck count
}

type LayerMetric struct {
	TotalTime  int64
	TotalCalls int64
	MaxTime    int64
	MinTime    int64
}

func NewMetricCollector() *MetricCollector {
	return &MetricCollector{
		Latencies:   make([]time.Duration, 0, 1000),
		Errors:      make([]error, 0),
		LayerStats:  make(map[string]*LayerMetric),
		Bottlenecks: make(map[string]int),
	}
}

func (mc *MetricCollector) RecordSuccess(latency time.Duration, tokensIn, tokensOut, tokensSaved int) {
	atomic.AddInt32(&mc.CompletedAgents, 1)
	atomic.AddInt64(&mc.TotalLatency, int64(latency))
	atomic.AddInt64(&mc.TotalTokensIn, int64(tokensIn))
	atomic.AddInt64(&mc.TotalTokensOut, int64(tokensOut))
	atomic.AddInt64(&mc.TotalTokensSaved, int64(tokensSaved))
	
	mc.mu.Lock()
	mc.Latencies = append(mc.Latencies, latency)
	mc.mu.Unlock()
}

func (mc *MetricCollector) RecordError(err error) {
	atomic.AddInt32(&mc.FailedAgents, 1)
	mc.mu.Lock()
	mc.Errors = append(mc.Errors, err)
	mc.mu.Unlock()
}

func (mc *MetricCollector) RecordLayerTime(layer string, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	metric, exists := mc.LayerStats[layer]
	if !exists {
		metric = &LayerMetric{MinTime: int64(time.Hour)}
		mc.LayerStats[layer] = metric
	}
	
	metric.TotalTime += int64(duration)
	metric.TotalCalls++
	if int64(duration) > metric.MaxTime {
		metric.MaxTime = int64(duration)
	}
	if int64(duration) < metric.MinTime {
		metric.MinTime = int64(duration)
	}
}

func (mc *MetricCollector) RecordBottleneck(layer string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.Bottlenecks[layer]++
}

// generateAgentInput creates realistic input for each agent type
func generateAgentInput(profile AgentProfile) string {
	switch profile.InputType {
	case "code":
		return generateCodeInput(profile.InputSize)
	case "logs":
		return generateLogInput(profile.InputSize)
	case "vcs":
		return generateVCSInput(profile.InputSize)
	case "markdown":
		return generateMarkdownInput(profile.InputSize)
	case "conversation":
		return generateConversationInput(profile.InputSize)
	case "mixed":
		return generateMixedInput(profile.InputSize)
	default:
		return generateMixedInput(profile.InputSize)
	}
}

func generateCodeInput(targetTokens int) string {
	// Approximate 4 chars per token
	targetChars := targetTokens * 4
	
	var sb strings.Builder
	sb.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n\t\"context\"\n\t\"time\"\n)\n\n")
	
	for sb.Len() < targetChars {
		sb.WriteString("func processItem(data string) error {\n")
		sb.WriteString("\tif data == \"\" {\n\t\treturn fmt.Errorf(\"empty data\")\n\t}\n")
		sb.WriteString("\tresult := strings.TrimSpace(data)\n")
		sb.WriteString("\tfmt.Println(result)\n")
		sb.WriteString("\treturn nil\n}\n\n")
		sb.WriteString("type Service struct {\n")
		sb.WriteString("\tName    string\n")
		sb.WriteString("\tEnabled bool\n")
		sb.WriteString("\tCount   int\n")
		sb.WriteString("}\n\n")
		sb.WriteString("func (s *Service) Run(ctx context.Context) error {\n")
		sb.WriteString("\tfor i := 0; i < s.Count; i++ {\n")
		sb.WriteString("\t\tselect {\n")
		sb.WriteString("\t\tcase <-ctx.Done():\n")
		sb.WriteString("\t\t\treturn ctx.Err()\n")
		sb.WriteString("\t\tdefault:\n")
		sb.WriteString("\t\t\tif err := processItem(fmt.Sprintf(\"item-%d\", i)); err != nil {\n")
		sb.WriteString("\t\t\t\treturn err\n")
		sb.WriteString("\t\t\t}\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\treturn nil\n}\n\n")
	}
	
	return sb.String()
}

func generateLogInput(targetTokens int) string {
	targetChars := targetTokens * 4
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "TRACE"}
	services := []string{"api", "auth", "db", "cache", "worker", "scheduler"}
	
	var sb strings.Builder
	now := time.Now()
	
	for i := 0; sb.Len() < targetChars; i++ {
		t := now.Add(time.Duration(i) * time.Second)
		level := levels[i%len(levels)]
		service := services[i%len(services)]
		
		sb.WriteString(fmt.Sprintf("%s [%s] [%s] Request processed id=%d duration=%dms status=%d\n",
			t.Format("2006-01-02T15:04:05.000Z07:00"),
			level, service, i%10000, (i%500)+10, 200+(i%5)))
		
		if i%10 == 0 {
			sb.WriteString(fmt.Sprintf("%s [%s] [%s] Memory usage: alloc=%dMB sys=%dMB gc_count=%d\n",
				t.Format("2006-01-02T15:04:05.000Z07:00"),
				"DEBUG", service, 100+i%50, 200+i%100, i/100))
		}
	}
	
	return sb.String()
}

func generateVCSInput(targetTokens int) string {
	targetChars := targetTokens * 4
	
	var sb strings.Builder
	sb.WriteString("diff --git a/src/main.go b/src/main.go\n")
	sb.WriteString("index abc1234..def5678 100644\n")
	sb.WriteString("--- a/src/main.go\n")
	sb.WriteString("+++ b/src/main.go\n")
	
	for chunk := 0; sb.Len() < targetChars; chunk++ {
		sb.WriteString(fmt.Sprintf("@@ -%d,10 +%d,15 @@ func main()\n", chunk*20, chunk*20))
		sb.WriteString(" import (\n")
		sb.WriteString(" \t\"fmt\"\n")
		sb.WriteString("+\t\"context\"\n")
		sb.WriteString("+\t\"sync\"\n")
		sb.WriteString(" )\n")
		sb.WriteString(" \n")
		sb.WriteString(" func process() {\n")
		sb.WriteString("-\tfmt.Println(\"old\")\n")
		sb.WriteString("+\tctx := context.Background()\n")
		sb.WriteString("+\tvar wg sync.WaitGroup\n")
		sb.WriteString("+\tfmt.Println(\"new\")\n")
		sb.WriteString(" }\n")
	}
	
	return sb.String()
}

func generateMarkdownInput(targetTokens int) string {
	targetChars := targetTokens * 4
	
	var sb strings.Builder
	sb.WriteString("# Project Documentation\n\n")
	sb.WriteString("## Overview\n\nThis document describes the architecture.\n\n")
	
	for i := 0; sb.Len() < targetChars; i++ {
		sb.WriteString(fmt.Sprintf("### Section %d: Feature Implementation\n\n", i))
		sb.WriteString("This section describes the implementation details.\n\n")
		sb.WriteString("```go\n")
		sb.WriteString("func Example() {\n")
		sb.WriteString("\t// Implementation here\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")
		sb.WriteString("- Point 1: Important detail\n")
		sb.WriteString("- Point 2: Another detail\n")
		sb.WriteString("- Point 3: Final detail\n\n")
	}
	
	return sb.String()
}

func generateConversationInput(targetTokens int) string {
	targetChars := targetTokens * 4
	
	var sb strings.Builder
	turns := []string{"User", "Assistant"}
	
	for i := 0; sb.Len() < targetChars; i++ {
		role := turns[i%2]
		sb.WriteString(fmt.Sprintf("%s: This is message %d in the conversation. ", role, i))
		sb.WriteString("It contains some context and questions about the codebase. ")
		sb.WriteString("We're discussing implementation details and best practices.\n\n")
	}
	
	return sb.String()
}

func generateMixedInput(targetTokens int) string {
	targetChars := targetTokens * 4
	
	var sb strings.Builder
	
	// Mix of code, logs, and markdown
	sb.WriteString("## Code Review\n\n")
	sb.WriteString("### Files Changed\n\n")
	sb.WriteString(generateVCSInput(targetTokens / 3))
	sb.WriteString("\n### Build Output\n\n")
	sb.WriteString(generateLogInput(targetTokens / 3))
	sb.WriteString("\n### Implementation\n\n")
	sb.WriteString(generateCodeInput(targetTokens / 3))
	
	for sb.Len() < targetChars {
		sb.WriteString("\n\nAdditional context and notes.\n")
	}
	
	return sb.String()
}

// Test1000AgentsStress launches 1000 concurrent agents
func Test1000AgentsStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 1000-agent stress test in short mode")
	}
	
	numAgents := 1000
	collector := NewMetricCollector()
	
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    TokMan 1000-Agent Stress Test - Bottleneck Analysis                        ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Agents: %d | CPUs: %d | GOMAXPROCS: %d\n", numAgents, runtime.NumCPU(), runtime.GOMAXPROCS(0))
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════════════════════╝\n\n")
	
	start := time.Now()
	
	// Launch agents in waves based on priority
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, runtime.NumCPU()*4) // Limit concurrent goroutines
	
	// Phase 1: High-priority agents (60% of total)
	phase1Count := numAgents * 60 / 100
	fmt.Printf("Phase 1: Launching %d high-priority agents...\n", phase1Count)
	for i := 0; i < phase1Count; i++ {
		wg.Add(1)
		go launchAgent(i, "high", &wg, semaphore, collector)
	}
	
	// Phase 2: Medium-priority agents (30% of total)
	phase2Count := numAgents * 30 / 100
	fmt.Printf("Phase 2: Launching %d medium-priority agents...\n", phase2Count)
	for i := 0; i < phase2Count; i++ {
		wg.Add(1)
		go launchAgent(phase1Count+i, "medium", &wg, semaphore, collector)
	}
	
	// Phase 3: Low-priority agents (10% of total)
	phase3Count := numAgents - phase1Count - phase2Count
	fmt.Printf("Phase 3: Launching %d low-priority agents...\n", phase3Count)
	for i := 0; i < phase3Count; i++ {
		wg.Add(1)
		go launchAgent(phase1Count+phase2Count+i, "low", &wg, semaphore, collector)
	}
	
	wg.Wait()
	totalDuration := time.Since(start)
	
	// Print comprehensive analysis
	printStressTestResults(t, collector, numAgents, totalDuration)
}

func launchAgent(agentID int, priority string, wg *sync.WaitGroup, semaphore chan struct{}, collector *MetricCollector) {
	defer wg.Done()
	
	// Acquire semaphore
	semaphore <- struct{}{}
	defer func() { <-semaphore }()
	
	// Select profile based on priority
	var profile AgentProfile
	profiles := []AgentProfile{}
	for _, p := range RealisticAgentProfiles {
		if priority == "high" && p.Priority == 1 {
			profiles = append(profiles, p)
		} else if priority == "medium" && p.Priority == 2 {
			profiles = append(profiles, p)
		} else if priority == "low" && p.Priority == 3 {
			profiles = append(profiles, p)
		}
	}
	
	if len(profiles) == 0 {
		profiles = RealisticAgentProfiles
	}
	profile = profiles[agentID%len(profiles)]
	
	// Generate input
	input := generateAgentInput(profile)
	
	// Create pipeline - use ParallelPipeline for Phase 3 optimization
	cfg := filter.PipelineConfig{
		Mode:                profile.Mode,
		EnableEntropy:       true,
		EnablePerplexity:    true,
		EnableGoalDriven:    true,
		EnableAST:           true,
		EnableContrastive:   true,
		EnableEvaluator:     true,
		EnableGist:          true,
		EnableHierarchical:  true,
		EnableCompaction:    true,
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	}
	
	pipeline := filter.NewParallelPipeline(cfg)
	
	// Process and measure
	start := time.Now()
	output, stats := pipeline.Process(input)
	latency := time.Since(start)
	
	// Check for bottlenecks (layers taking > 10ms)
	bottleneckThreshold := 10 * time.Millisecond
	if latency > bottleneckThreshold {
		collector.RecordBottleneck("pipeline_total")
	}
	
	// Record metrics
	tokensIn := filter.EstimateTokens(input)
	tokensOut := filter.EstimateTokens(output)
	collector.RecordSuccess(latency, tokensIn, tokensOut, stats.TotalSaved)
}

func printStressTestResults(t *testing.T, collector *MetricCollector, numAgents int, totalDuration time.Duration) {
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                              STRESS TEST RESULTS                                              ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	
	// Overall stats
	completed := atomic.LoadInt32(&collector.CompletedAgents)
	failed := atomic.LoadInt32(&collector.FailedAgents)
	
	fmt.Printf("║ AGENT EXECUTION\n")
	fmt.Printf("║   Total agents:      %d\n", numAgents)
	fmt.Printf("║   Completed:         %d (%.1f%%)\n", completed, float64(completed)/float64(numAgents)*100)
	fmt.Printf("║   Failed:            %d (%.1f%%)\n", failed, float64(failed)/float64(numAgents)*100)
	fmt.Printf("║   Total time:        %v\n", totalDuration)
	fmt.Printf("║   Agents/sec:        %.2f\n", float64(numAgents)/totalDuration.Seconds())
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	
	// Token metrics
	tokensIn := atomic.LoadInt64(&collector.TotalTokensIn)
	tokensOut := atomic.LoadInt64(&collector.TotalTokensOut)
	tokensSaved := atomic.LoadInt64(&collector.TotalTokensSaved)
	
	fmt.Printf("║ TOKEN METRICS\n")
	fmt.Printf("║   Total input:       %d tokens\n", tokensIn)
	fmt.Printf("║   Total output:      %d tokens\n", tokensOut)
	fmt.Printf("║   Total saved:       %d tokens\n", tokensSaved)
	fmt.Printf("║   Compression:       %.2f%%\n", float64(tokensSaved)/float64(tokensIn)*100)
	fmt.Printf("║   Throughput:        %.2f MB/s\n", float64(tokensIn*4)/totalDuration.Seconds()/1024/1024)
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	
	// Latency analysis
	collector.mu.Lock()
	latencies := collector.Latencies
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	
	fmt.Printf("║ LATENCY ANALYSIS\n")
	if len(latencies) > 0 {
		p50 := latencies[len(latencies)*50/100]
		p90 := latencies[len(latencies)*90/100]
		p95 := latencies[len(latencies)*95/100]
		p99 := latencies[len(latencies)*99/100]
		avg := time.Duration(int64(atomic.LoadInt64(&collector.TotalLatency)) / int64(completed))
		
		fmt.Printf("║   Average:           %v\n", avg)
		fmt.Printf("║   P50:               %v\n", p50)
		fmt.Printf("║   P90:               %v\n", p90)
		fmt.Printf("║   P95:               %v\n", p95)
		fmt.Printf("║   P99:               %v\n", p99)
		fmt.Printf("║   Max:               %v\n", latencies[len(latencies)-1])
		fmt.Printf("║   Min:               %v\n", latencies[0])
	}
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	
	// Bottleneck analysis
	fmt.Printf("║ BOTTLENECK DETECTION\n")
	for layer, count := range collector.Bottlenecks {
		fmt.Printf("║   %-20s: %d occurrences (>10ms)\n", layer, count)
	}
	if len(collector.Bottlenecks) == 0 {
		fmt.Printf("║   No bottlenecks detected (all <10ms)\n")
	}
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	
	// Improvement recommendations
	fmt.Printf("║ IMPROVEMENT RECOMMENDATIONS\n")
	recommendations := generateRecommendations(collector, latencies, tokensIn, tokensOut)
	for i, rec := range recommendations {
		fmt.Printf("║   %d. %s\n", i+1, rec)
	}
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════════════════════╝\n\n")
	
	collector.mu.Unlock()
	
	// Assert minimum performance requirements
	require.NoError(t, collector.Errors...)
	require.GreaterOrEqual(t, int(completed), numAgents*95/100, "At least 95%% of agents should complete")
}

func generateRecommendations(collector *MetricCollector, latencies []time.Duration, tokensIn, tokensOut int64) []string {
	var recommendations []string
	
	if len(latencies) == 0 {
		return []string{"Insufficient data for recommendations"}
	}
	
	// Check P99 latency
	p99 := latencies[len(latencies)*99/100]
	if p99 > 100*time.Millisecond {
		recommendations = append(recommendations, 
			fmt.Sprintf("HIGH P99 latency (%v) - Consider async processing or layer optimization", p99))
	}
	
	// Check compression ratio
	compressionRatio := float64(tokensIn-tokensOut) / float64(tokensIn) * 100
	if compressionRatio < 50 {
		recommendations = append(recommendations,
			fmt.Sprintf("LOW compression ratio (%.1f%%) - Enable more aggressive filtering", compressionRatio))
	}
	
	// Check bottleneck patterns
	for layer, count := range collector.Bottlenecks {
		if count > 100 {
			recommendations = append(recommendations,
				fmt.Sprintf("LAYER '%s' bottleneck (%d hits) - Optimize or add caching", layer, count))
		}
	}
	
	// Check error rate
	failed := atomic.LoadInt32(&collector.FailedAgents)
	if failed > 10 {
		recommendations = append(recommendations,
			fmt.Sprintf("HIGH failure rate (%d errors) - Add error handling and retries", failed))
	}
	
	// Memory pressure check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Alloc > 500*1024*1024 { // > 500MB
		recommendations = append(recommendations,
			"HIGH memory usage - Implement streaming for large inputs")
	}
	
	// Throughput optimization
	throughput := float64(tokensIn*4) / float64(len(latencies)) / 1024 // KB per request
	if throughput < 100 { // < 100KB per request average
		recommendations = append(recommendations,
			"LOW throughput - Consider batch processing or connection pooling")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Performance is within acceptable parameters")
		recommendations = append(recommendations, "Consider A/B testing different filter combinations")
		recommendations = append(recommendations, "Monitor production metrics for edge cases")
	}
	
	return recommendations
}

// require provides assertion utilities for the test
var require = &requireT{}

type requireT struct{}

func (r *requireT) NoError(t *testing.T, errs ...error) {
	for _, err := range errs {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

func (r *requireT) GreaterOrEqual(t *testing.T, actual, expected int, msg string) {
	if actual < expected {
		t.Fatalf("%s: expected >= %d, got %d", msg, expected, actual)
	}
}
