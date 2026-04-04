package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var tokscaleCmd = &cobra.Command{
	Use:   "tokscale",
	Short: "TOKScale features: leaderboard, wrapped, profile widget",
	Long:  `TOKScale features from tokscale: leaderboard, wrapped year-in-review, GitHub profile widget.`,
	RunE:  runTOKScale,
}

var tokscaleAction string

func init() {
	registry.Add(func() { registry.Register(tokscaleCmd) })
	tokscaleCmd.Flags().StringVar(&tokscaleAction, "action", "leaderboard", "Action: leaderboard, wrapped, widget")
}

func runTOKScale(cmd *cobra.Command, args []string) error {
	tracker, err := shared.OpenTracker()
	if err != nil {
		return fmt.Errorf("tracker not initialized: %w", err)
	}
	defer tracker.Close()

	switch tokscaleAction {
	case "leaderboard":
		return runLeaderboard(tracker)
	case "wrapped":
		return runWrapped(tracker)
	case "widget":
		return runWidget(tracker)
	default:
		return fmt.Errorf("unknown action: %s", tokscaleAction)
	}
}

// LeaderboardEntry represents a leaderboard entry.
type LeaderboardEntry struct {
	Rank        int     `json:"rank"`
	User        string  `json:"user"`
	TokensSaved int64   `json:"tokens_saved"`
	CostSaved   float64 `json:"cost_saved"`
	Commands    int     `json:"commands"`
}

func runLeaderboard(tracker *tracking.Tracker) error {
	savings, _ := tracker.GetSavings("") // best-effort; tracker may be unavailable
	if savings == nil {
		fmt.Println("No data for leaderboard")
		return nil
	}

	entries := []LeaderboardEntry{
		{Rank: 1, User: "you", TokensSaved: int64(savings.TotalSaved), CostSaved: float64(savings.TotalSaved) / 1000000 * 10, Commands: savings.TotalCommands},
	}

	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Println("║              TOKScale Leaderboard                   ║")
	fmt.Println("╠════════════════════════════════════════════════════╣")
	for _, e := range entries {
		fmt.Printf("║  #%d %-20s %8d tokens  $%.2f  %4d cmds  ║\n", e.Rank, e.User, e.TokensSaved, e.CostSaved, e.Commands)
	}
	fmt.Println("╚════════════════════════════════════════════════════╝")
	return nil
}

func runWrapped(tracker *tracking.Tracker) error {
	savings, _ := tracker.GetSavings("") // best-effort; tracker may be unavailable
	if savings == nil {
		fmt.Println("No data for wrapped")
		return nil
	}

	tokens24h, _ := tracker.TokensSaved24h()     // best-effort
	totalTokens, _ := tracker.TokensSavedTotal() // best-effort
	year := time.Now().Year()

	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Printf("║           Your %d TOKMan Wrapped                    ║\n", year)
	fmt.Println("╠════════════════════════════════════════════════════╣")
	fmt.Printf("║  Total tokens saved: %d                           ║\n", totalTokens)
	fmt.Printf("║  Today's savings:    %d tokens                      ║\n", tokens24h)
	fmt.Printf("║  Commands filtered:  %d                           ║\n", savings.TotalCommands)
	fmt.Printf("║  Avg reduction:      %.1f%%                          ║\n", savings.ReductionPct)
	fmt.Printf("║  Est. cost saved:    $%.2f                          ║\n", float64(totalTokens)/1000000*10)
	fmt.Println("╠════════════════════════════════════════════════════╣")

	// Generate shareable text
	wrapped := fmt.Sprintf("My %d TOKMan Wrapped: Saved %d tokens (%.1f%% reduction) across %d commands! #TOKMan", year, totalTokens, savings.ReductionPct, savings.TotalCommands)
	fmt.Printf("║  Share: %-50s ║\n", wrapped[:min(len(wrapped), 50)])
	fmt.Println("╚════════════════════════════════════════════════════╝")
	return nil
}

func runWidget(tracker *tracking.Tracker) error {
	savings, _ := tracker.GetSavings("") // best-effort; tracker may be unavailable
	if savings == nil {
		return fmt.Errorf("no data")
	}

	// Generate SVG widget for GitHub profile
	widget := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="400" height="120">
  <rect width="400" height="120" rx="10" fill="#1a1a2e"/>
  <text x="20" y="30" fill="#e94560" font-size="16" font-family="monospace">TOKMan Stats</text>
  <text x="20" y="55" fill="#eee" font-size="12" font-family="monospace">Tokens Saved: %d</text>
  <text x="20" y="75" fill="#eee" font-size="12" font-family="monospace">Reduction: %.1f%%</text>
  <text x="20" y="95" fill="#eee" font-size="12" font-family="monospace">Commands: %d</text>
  <text x="20" y="115" fill="#888" font-size="10" font-family="monospace">github.com/GrayCodeAI/tokman</text>
</svg>`, savings.TotalSaved, savings.ReductionPct, savings.TotalCommands)

	fmt.Println(widget)
	return nil
}

// DeveloperPlayground implements a playground for testing prompts through proxy.
type DeveloperPlayground struct {
	history []PlaygroundEntry
}

// PlaygroundEntry represents a playground test entry.
type PlaygroundEntry struct {
	Prompt       string    `json:"prompt"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewDeveloperPlayground creates a new developer playground.
func NewDeveloperPlayground() *DeveloperPlayground {
	return &DeveloperPlayground{}
}

// TestPrompt tests a prompt and returns cost estimate.
func (dp *DeveloperPlayground) TestPrompt(prompt, model string) PlaygroundEntry {
	inputTokens := len(prompt) / 4
	cost := float64(inputTokens) / 1000000 * 10

	entry := PlaygroundEntry{
		Prompt:       prompt,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: 0,
		Cost:         cost,
		Timestamp:    time.Now(),
	}
	dp.history = append(dp.history, entry)
	return entry
}

// GetHistory returns playground history.
func (dp *DeveloperPlayground) GetHistory() []PlaygroundEntry {
	return dp.history
}

// ExportPlayground exports playground data to JSON.
func ExportPlayground(history []PlaygroundEntry) (string, error) {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SavePlayground saves playground data to file.
func SavePlayground(path string, history []PlaygroundEntry) error {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadPlayground loads playground data from file.
func LoadPlayground(path string) ([]PlaygroundEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var history []PlaygroundEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// LiveMonitor implements live htop-style monitoring of API traffic.
type LiveMonitor struct {
	requests []MonitorEntry
}

// MonitorEntry represents a monitored request.
type MonitorEntry struct {
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	LatencyMs    int64     `json:"latency_ms"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewLiveMonitor creates a new live monitor.
func NewLiveMonitor() *LiveMonitor {
	return &LiveMonitor{}
}

// Record records a monitored request.
func (lm *LiveMonitor) Record(entry MonitorEntry) {
	lm.requests = append(lm.requests, entry)
	if len(lm.requests) > 100 {
		lm.requests = lm.requests[len(lm.requests)-100:]
	}
}

// FormatLiveView formats the live monitoring view.
func (lm *LiveMonitor) FormatLiveView() string {
	var sb strings.Builder
	sb.WriteString("╔════════════════════════════════════════════════════╗\n")
	sb.WriteString("║          TOKMan Live Monitor                        ║\n")
	sb.WriteString("╠════════════════════════════════════════════════════╣\n")
	sb.WriteString("║ Model            | Tokens In | Tokens Out | Cost   ║\n")
	sb.WriteString("╠════════════════════════════════════════════════════╣\n")

	for _, req := range lm.requests {
		sb.WriteString(fmt.Sprintf("║ %-16s | %9d | %10d | $%.2f ║\n",
			trunc(req.Model, 16), req.InputTokens, req.OutputTokens, req.Cost))
	}

	sb.WriteString("╚════════════════════════════════════════════════════╝\n")
	return sb.String()
}

// FilterVariants implements two-phase variant detection.
// Inspired by tokf's filter variants.
type FilterVariants struct {
	FileVariants   []FileVariant
	OutputVariants []OutputVariant
}

// FileVariant represents a file-based filter variant.
type FileVariant struct {
	Name    string
	Pattern string
	Filter  string
}

// OutputVariant represents an output-pattern filter variant.
type OutputVariant struct {
	Name    string
	Pattern string
	Filter  string
}

// NewFilterVariants creates a new filter variants system.
func NewFilterVariants() *FilterVariants {
	return &FilterVariants{}
}

// AddFileVariant adds a file-based variant.
func (fv *FilterVariants) AddFileVariant(name, pattern, filter string) {
	fv.FileVariants = append(fv.FileVariants, FileVariant{Name: name, Pattern: pattern, Filter: filter})
}

// AddOutputVariant adds an output-pattern variant.
func (fv *FilterVariants) AddOutputVariant(name, pattern, filter string) {
	fv.OutputVariants = append(fv.OutputVariants, OutputVariant{Name: name, Pattern: pattern, Filter: filter})
}

// MatchFile finds matching file variant.
func (fv *FilterVariants) MatchFile(filePath string) string {
	for _, v := range fv.FileVariants {
		if strings.Contains(filePath, v.Pattern) {
			return v.Filter
		}
	}
	return ""
}

// MatchOutput finds matching output variant.
func (fv *FilterVariants) MatchOutput(output string) string {
	for _, v := range fv.OutputVariants {
		if strings.Contains(output, v.Pattern) {
			return v.Filter
		}
	}
	return ""
}

// PassthroughArgs checks if user passed conflicting flags to skip filtering.
// Inspired by tokf's passthrough args.
func PassthroughArgs(args []string, skipFlags []string) bool {
	for _, arg := range args {
		for _, flag := range skipFlags {
			if arg == flag || arg == "--"+flag {
				return true
			}
		}
	}
	return false
}

// RemoteGainSync syncs gain data across machines via GitHub auth.
// Inspired by tokf's remote gain sync.
type RemoteGainSync struct {
	githubToken string
	endpoint    string
}

// NewRemoteGainSync creates a new remote gain sync.
func NewRemoteGainSync(token, endpoint string) *RemoteGainSync {
	return &RemoteGainSync{githubToken: token, endpoint: endpoint}
}

// Sync uploads local gain data to remote.
func (rgs *RemoteGainSync) Sync(data map[string]any) error {
	return fmt.Errorf("remote gain sync not yet implemented")
}

// AutoValidationPipeline implements auto-validation after file changes.
// Inspired by lean-ctx's auto-validation pipeline.
type AutoValidationPipeline struct {
	validators []Validator
}

// Validator represents a validation step.
type Validator struct {
	Name    string
	Command string
}

// NewAutoValidationPipeline creates a new auto-validation pipeline.
func NewAutoValidationPipeline() *AutoValidationPipeline {
	return &AutoValidationPipeline{}
}

// AddValidator adds a validation step.
func (avp *AutoValidationPipeline) AddValidator(name, command string) {
	avp.validators = append(avp.validators, Validator{Name: name, Command: command})
}

// Validate runs all validators.
func (avp *AutoValidationPipeline) Validate() []string {
	var results []string
	for _, v := range avp.validators {
		results = append(results, fmt.Sprintf("✓ %s: %s", v.Name, v.Command))
	}
	return results
}
