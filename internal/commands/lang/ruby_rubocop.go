package lang

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func runRubocopCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rubocop %s\n", strings.Join(args, " "))
	}

	// Use --format json for structured output
	jsonArgs := append([]string{"--format", "json"}, args...)
	execCmd := exec.Command("rubocop", jsonArgs...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRubocopOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("rubocop %s", strings.Join(args, " ")), "tokman ruby rubocop", originalTokens, filteredTokens)

	return err
}

// RuboCopJSON represents the JSON output from rubocop --format json
type RuboCopJSON struct {
	Metadata RuboCopMetadata `json:"metadata"`
	Files    []RuboCopFile   `json:"files"`
	Summary  RuboCopSummary  `json:"summary"`
}

type RuboCopMetadata struct {
	RuboCopVersion string `json:"rubocop_version"`
}

type RuboCopFile struct {
	Path     string           `json:"path"`
	Offenses []RuboCopOffense `json:"offenses"`
}

type RuboCopOffense struct {
	Severity string          `json:"severity"`
	Message  string          `json:"message"`
	CopName  string          `json:"cop_name"`
	Location RuboCopLocation `json:"location"`
}

type RuboCopLocation struct {
	StartLine int `json:"start_line"`
}

type RuboCopSummary struct {
	OffenseCount       int `json:"offense_count"`
	TargetFileCount    int `json:"target_file_count"`
	InspectedFileCount int `json:"inspected_file_count"`
}

func filterRubocopOutput(raw string) string {
	// Try to parse as JSON
	var rubocop RuboCopJSON
	if err := json.Unmarshal([]byte(raw), &rubocop); err != nil {
		// Fall back to text parsing
		return filterRubocopTextOutput(raw)
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		return filterRubocopOutputUltraCompact(rubocop)
	}

	var result []string
	result = append(result, "📋 RuboCop Results:")
	result = append(result, fmt.Sprintf("   📁 %d files inspected", rubocop.Summary.InspectedFileCount))

	if rubocop.Summary.OffenseCount == 0 {
		result = append(result, "   ✅ No offenses detected")
		return strings.Join(result, "\n")
	}

	// Count by severity
	severityCount := make(map[string]int)
	for _, file := range rubocop.Files {
		for _, offense := range file.Offenses {
			severityCount[offense.Severity]++
		}
	}

	result = append(result, fmt.Sprintf("   ⚠️  %d offenses:", rubocop.Summary.OffenseCount))
	for sev, count := range severityCount {
		result = append(result, fmt.Sprintf("      %s: %d", sev, count))
	}

	// Show offenses by file
	offenseCount := 0
	for _, file := range rubocop.Files {
		if len(file.Offenses) > 0 {
			shortPath := file.Path
			if idx := strings.LastIndex(file.Path, "/"); idx >= 0 {
				shortPath = file.Path[idx+1:]
			}
			result = append(result, "")
			result = append(result, fmt.Sprintf("   📄 %s (%d offenses):", shortPath, len(file.Offenses)))
			for _, offense := range file.Offenses {
				offenseCount++
				if offenseCount > 15 {
					result = append(result, fmt.Sprintf("   ... +%d more offenses", rubocop.Summary.OffenseCount-15))
					break
				}
				severityIcon := "⚠️"
				if offense.Severity == "error" || offense.Severity == "fatal" {
					severityIcon = "❌"
				}
				result = append(result, fmt.Sprintf("      %s L%d: %s",
					severityIcon,
					offense.Location.StartLine,
					shared.TruncateLine(offense.Message, 60)))
			}
			if offenseCount > 15 {
				break
			}
		}
	}

	return strings.Join(result, "\n")
}

func filterRubocopOutputUltraCompact(rubocop RuboCopJSON) string {
	if rubocop.Summary.OffenseCount == 0 {
		return "✅ Clean"
	}

	var result []string
	result = append(result, fmt.Sprintf("O:%d in %d files", rubocop.Summary.OffenseCount, rubocop.Summary.InspectedFileCount))

	// Show up to 5 files with offenses
	fileCount := 0
	for _, file := range rubocop.Files {
		if len(file.Offenses) > 0 && fileCount < 5 {
			shortPath := file.Path
			if idx := strings.LastIndex(file.Path, "/"); idx >= 0 {
				shortPath = file.Path[idx+1:]
			}
			result = append(result, fmt.Sprintf("%s: %d", shortPath, len(file.Offenses)))
			fileCount++
		}
	}

	return strings.Join(result, "\n")
}

func filterRubocopTextOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var offenses []string
	fileCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for file markers
		if strings.HasPrefix(line, "Inspecting") {
			fileCount++
		} else if strings.Contains(line, " offenses detected") || strings.Contains(line, " offense detected") {
			result = append(result, line)
		} else if strings.Contains(line, ":") && (strings.Contains(line, "error:") ||
			strings.Contains(line, "Warning") || strings.Contains(line, "Convention") ||
			strings.Contains(line, "Style")) {
			offenses = append(offenses, shared.TruncateLine(line, 100))
		}
	}

	if len(offenses) > 0 {
		result = append(result, "")
		result = append(result, "Offenses:")
		for i, o := range offenses {
			if i >= 15 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(offenses)-15))
				break
			}
			result = append(result, fmt.Sprintf("   - %s", o))
		}
	}

	if len(result) == 0 {
		return raw
	}
	return strings.Join(result, "\n")
}
