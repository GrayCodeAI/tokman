package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var exportFormat string
var exportLimit int

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export tracking data to CSV or JSON",
	Long: `Export command tracking data for external analysis.

Supports JSON and CSV formats. Data includes command names,
token counts, savings, and timestamps.`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "output format (json or csv)")
	exportCmd.Flags().IntVarP(&exportLimit, "limit", "n", 1000, "max records to export (0 = all)")
	registry.Add(func() { registry.Register(exportCmd) })
}

func runExport(cmd *cobra.Command, args []string) error {
	tracker := tracking.GetGlobalTracker()
	if tracker == nil {
		return fmt.Errorf("tracking not available")
	}

	cwd, _ := os.Getwd()
	limit := exportLimit
	if limit <= 0 {
		limit = 10000
	}

	records, err := tracker.GetRecentCommands(cwd, limit)
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("No tracking records found.")
		return nil
	}

	switch strings.ToLower(exportFormat) {
	case "csv":
		return writeCSV(records)
	case "json":
		return writeJSON(records)
	default:
		return fmt.Errorf("unknown format: %s (use json or csv)", exportFormat)
	}
}

func writeJSON(records []tracking.CommandRecord) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

func writeCSV(records []tracking.CommandRecord) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	header := []string{
		"id", "command", "original_tokens", "filtered_tokens",
		"saved_tokens", "project_path", "exec_time_ms", "timestamp", "parse_success",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range records {
		row := []string{
			fmt.Sprintf("%d", r.ID),
			r.Command,
			fmt.Sprintf("%d", r.OriginalTokens),
			fmt.Sprintf("%d", r.FilteredTokens),
			fmt.Sprintf("%d", r.SavedTokens),
			r.ProjectPath,
			fmt.Sprintf("%d", r.ExecTimeMs),
			r.Timestamp.Format("2006-01-02T15:04:05Z"),
			fmt.Sprintf("%t", r.ParseSuccess),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}
