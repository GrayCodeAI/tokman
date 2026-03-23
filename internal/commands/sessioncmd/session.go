package sessioncmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/discover"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Show TokMan adoption across Claude Code sessions",
	Long: `Analyze Claude Code session history to show TokMan adoption metrics.

Scans ~/.claude/projects/ for session JSONL files and shows what percentage
of Bash commands would be handled by TokMan wrappers.

Examples:
  tokman session`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSession(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	registry.Add(func() { registry.Register(sessionCmd) })
}

type SessionSummary struct {
	ID           string
	Date         string
	TotalCmds    int
	TokmanCmds   int
	OutputTokens int
}

func (s *SessionSummary) AdoptionPct() float64 {
	if s.TotalCmds == 0 {
		return 0.0
	}
	return float64(s.TokmanCmds) / float64(s.TotalCmds) * 100.0
}

type ExtractedCommand struct {
	Command   string
	OutputLen int
	SessionID string
	IsError   bool
}

func runSession() error {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}
	projectsDir := filepath.Join(home, ".claude", "projects")

	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		fmt.Println("No Claude Code sessions found in the last 30 days.")
		fmt.Println("Make sure Claude Code has been used at least once.")
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -30)

	var sessionFiles []string
	err = filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if strings.Contains(path, "subagents") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			return nil
		}
		sessionFiles = append(sessionFiles, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk projects directory: %w", err)
	}

	if len(sessionFiles) == 0 {
		fmt.Println("No Claude Code sessions found in the last 30 days.")
		fmt.Println("Make sure Claude Code has been used at least once.")
		return nil
	}

	sort.Slice(sessionFiles, func(i, j int) bool {
		iInfo, iErr := os.Stat(sessionFiles[i])
		jInfo, jErr := os.Stat(sessionFiles[j])
		if iErr != nil || jErr != nil {
			return iErr == nil
		}
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	if len(sessionFiles) > 10 {
		sessionFiles = sessionFiles[:10]
	}

	var summaries []SessionSummary

	for _, path := range sessionFiles {
		cmds, err := extractCommands(path)
		if err != nil {
			continue
		}

		if len(cmds) == 0 {
			continue
		}

		totalCmds, tokmanCmds, outputTokens := countTokmanCommands(cmds)

		id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if len(id) > 8 {
			id = id[:8]
		}

		info, statErr := os.Stat(path)
		var date string
		if statErr == nil && info != nil {
			elapsed := time.Since(info.ModTime())
			days := int(elapsed.Hours() / 24)
			switch {
			case days == 0:
				date = "Today"
			case days == 1:
				date = "Yesterday"
			default:
				date = fmt.Sprintf("%dd ago", days)
			}
		} else {
			date = "?"
		}

		summaries = append(summaries, SessionSummary{
			ID:           id,
			Date:         date,
			TotalCmds:    totalCmds,
			TokmanCmds:   tokmanCmds,
			OutputTokens: outputTokens,
		})
	}

	if len(summaries) == 0 {
		fmt.Println("No sessions with Bash commands found.")
		return nil
	}

	fmt.Println()
	fmt.Printf("%s\n", green("TokMan Session Overview (last 10)"))
	fmt.Println(strings.Repeat("─", 70))
	fmt.Printf("%-12s %-12s %5s %5s %9s %-7s %8s\n",
		"Session", "Date", "Cmds", "TokMan", "Adoption", "", "Output")
	fmt.Println(strings.Repeat("─", 70))

	var grandTotalCmds, grandTokmanCmds int

	for _, s := range summaries {
		pct := s.AdoptionPct()
		bar := progressBar(pct, 5)
		grandTotalCmds += s.TotalCmds
		grandTokmanCmds += s.TokmanCmds

		fmt.Printf("%-12s %-12s %5d %5d %8.0f%% %-7s %8s\n",
			s.ID,
			s.Date,
			s.TotalCmds,
			s.TokmanCmds,
			pct,
			bar,
			utils.FormatTokens(s.OutputTokens),
		)
	}

	fmt.Println(strings.Repeat("─", 70))

	avgAdoption := 0.0
	if grandTotalCmds > 0 {
		avgAdoption = float64(grandTokmanCmds) / float64(grandTotalCmds) * 100.0
	}
	fmt.Printf("Average adoption: %.0f%%\n", avgAdoption)
	fmt.Printf("Tip: Run %s to find missed TokMan opportunities\n", cyan("`tokman discover`"))
	fmt.Println()

	return nil
}

func extractCommands(path string) ([]ExtractedCommand, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")

	var commands []ExtractedCommand
	var pendingToolUses []struct {
		id      string
		command string
	}
	toolResults := make(map[string]struct {
		outputLen int
		isError   bool
	})

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.Contains(line, "\"Bash\"") && !strings.Contains(line, "\"tool_result\"") {
			continue
		}

		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		entryType, _ := entry["type"].(string)

		switch entryType {
		case "assistant":
			message, _ := entry["message"].(map[string]any)
			content, _ := message["content"].([]any)
			for _, block := range content {
				blockMap, _ := block.(map[string]any)
				if blockMap["type"] == "tool_use" && blockMap["name"] == "Bash" {
					id, _ := blockMap["id"].(string)
					input, _ := blockMap["input"].(map[string]any)
					cmd, _ := input["command"].(string)
					if id != "" && cmd != "" {
						pendingToolUses = append(pendingToolUses, struct {
							id      string
							command string
						}{id: id, command: cmd})
					}
				}
			}
		case "user":
			message, _ := entry["message"].(map[string]any)
			content, _ := message["content"].([]any)
			for _, block := range content {
				blockMap, _ := block.(map[string]any)
				if blockMap["type"] == "tool_result" {
					id, _ := blockMap["tool_use_id"].(string)
					contentStr, _ := blockMap["content"].(string)
					isError, _ := blockMap["is_error"].(bool)
					if id != "" {
						toolResults[id] = struct {
							outputLen int
							isError   bool
						}{
							outputLen: len(contentStr),
							isError:   isError,
						}
					}
				}
			}
		}
	}

	for _, tu := range pendingToolUses {
		result, ok := toolResults[tu.id]
		outputLen := 0
		isError := false
		if ok {
			outputLen = result.outputLen
			isError = result.isError
		}
		commands = append(commands, ExtractedCommand{
			Command:   tu.command,
			OutputLen: outputLen,
			SessionID: sessionID,
			IsError:   isError,
		})
	}

	return commands, nil
}

func countTokmanCommands(cmds []ExtractedCommand) (int, int, int) {
	var total, tokman, output int

	for _, c := range cmds {
		parts := splitCommandChain(c.Command)
		for _, part := range parts {
			total++
			if strings.HasPrefix(part, "tokman ") {
				tokman++
			} else {
				class := discover.ClassifyCommand(part)
				if class.Supported {
					tokman++
				}
			}
		}
		output += c.OutputLen
	}

	return total, tokman, output
}

func splitCommandChain(cmd string) []string {
	var parts []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		b := cmd[i]
		switch {
		case b == '\'' && !inDouble:
			inSingle = !inSingle
			current.WriteByte(b)
		case b == '"' && !inSingle:
			inDouble = !inDouble
			current.WriteByte(b)
		case b == '|' && !inSingle && !inDouble:
			if i+1 < len(cmd) && cmd[i+1] == '|' {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
				i++
			} else {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
			}
		case b == '&' && !inSingle && !inDouble && i+1 < len(cmd) && cmd[i+1] == '&':
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
			i++
		case b == ';' && !inSingle && !inDouble:
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteByte(b)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}

	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}

	return result
}

func progressBar(pct float64, width int) string {
	filled := int((pct / 100.0) * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return strings.Repeat("@", filled) + strings.Repeat(".", empty)
}
