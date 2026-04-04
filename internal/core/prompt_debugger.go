package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type PromptRecord struct {
	Timestamp        time.Time
	Command          string
	RawPrompt        string
	CompressedPrompt string
	Model            string
	InputTokens      int
	OutputTokens     int
	Duration         time.Duration
}

type PromptDebugger struct {
	dir string
}

func NewPromptDebugger(dir string) *PromptDebugger {
	if dir == "" {
		dir = defaultPromptDir()
	}
	return &PromptDebugger{dir: expandPromptDir(dir)}
}

func (d *PromptDebugger) Save(record PromptRecord) error {
	if err := os.MkdirAll(d.dir, 0755); err != nil {
		return err
	}

	ts := record.Timestamp.Format("20060102-150405")
	filename := fmt.Sprintf("%s_%s.json", ts, sanitizeFilename(record.Command))
	path := filepath.Join(d.dir, filename)

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (d *PromptDebugger) List(limit int) ([]PromptRecord, error) {
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, err
	}

	var records []PromptRecord
	count := 0
	for i := len(entries) - 1; i >= 0 && count < limit; i-- {
		e := entries[i]
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(d.dir, e.Name()))
		if err != nil {
			continue
		}
		var r PromptRecord
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		records = append(records, r)
		count++
	}

	return records, nil
}

func (d *PromptDebugger) CompressionRatio(record PromptRecord) float64 {
	if record.InputTokens == 0 {
		return 0
	}
	return float64(record.InputTokens-record.OutputTokens) / float64(record.InputTokens) * 100
}

func expandPromptDir(dir string) string {
	if len(dir) > 0 && dir[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, dir[1:])
	}
	return dir
}

func sanitizeFilename(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

func defaultPromptDir() string {
	return filepath.Join(promptDataPath(), "prompts")
}

func promptDataPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "tokman")
	}

	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "tokman")
		}
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "tokman", "data")
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "tokman")
	}

	return filepath.Join(os.TempDir(), "tokman-data")
}
