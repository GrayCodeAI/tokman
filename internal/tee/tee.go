package tee

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/config"
)

const maxFiles = 20

type Mode string

const (
	ModeFailures Mode = "failures"
	ModeAlways   Mode = "always"
	ModeNever    Mode = "never"
)

type Config struct {
	Enabled  bool
	Mode     Mode
	MaxFiles int
	Dir      string
}

func DefaultConfig() Config {
	return Config{
		Enabled:  true,
		Mode:     ModeFailures,
		MaxFiles: maxFiles,
		Dir:      defaultDir(),
	}
}

type TeeEntry struct {
	Timestamp time.Time
	Command   string
	Filename  string
}

func Save(command string, output string, exitCode int, cfg Config) (string, error) {
	if !cfg.Enabled || cfg.Mode == ModeNever {
		return "", nil
	}
	if cfg.Mode == ModeFailures && exitCode == 0 {
		return "", nil
	}

	dir := expandTilde(cfg.Dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	ts := time.Now().Unix()
	hash := sha256.Sum256([]byte(command))
	shortHash := fmt.Sprintf("%x", hash[:4])
	filename := fmt.Sprintf("%d_%s_%s.log", ts, shortHash, strings.ReplaceAll(command, " ", "_"))
	if len(filename) > 120 {
		filename = filename[:120] + ".log"
	}
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return "", err
	}

	rotate(dir, cfg.MaxFiles)

	return path, nil
}

func List(cfg Config) ([]TeeEntry, error) {
	dir := expandTilde(cfg.Dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var result []TeeEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".log")
		parts := strings.SplitN(name, "_", 3)
		if len(parts) < 2 {
			continue
		}
		var ts int64
		fmt.Sscanf(parts[0], "%d", &ts)
		cmd := ""
		if len(parts) >= 3 {
			cmd = strings.ReplaceAll(parts[2], "_", " ")
		}
		result = append(result, TeeEntry{
			Timestamp: time.Unix(ts, 0),
			Command:   cmd,
			Filename:  e.Name(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

func Read(filename string, cfg Config) (string, error) {
	dir := expandTilde(cfg.Dir)
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func rotate(dir string, max int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			files = append(files, e)
		}
	}

	if len(files) <= max {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for i := 0; i < len(files)-max; i++ {
		os.Remove(filepath.Join(dir, files[i].Name()))
	}
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// WriteAndHint saves output and returns a hint string for the LLM.
func WriteAndHint(output string, command string, exitCode int) string {
	cfg := DefaultConfig()
	path, err := Save(command, output, exitCode, cfg)
	if err != nil || path == "" {
		return ""
	}
	return fmt.Sprintf("[full output saved: %s]", path)
}

func defaultDir() string {
	return filepath.Join(config.DataPath(), "tee")
}
