package filter

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// tomlConfig is the expected shape of the watched TOML file.
// Only the fields that ValidateTOMLConfig inspects are declared here.
type tomlConfig struct {
	Budget int64  `toml:"budget"`
	Mode   string `toml:"mode"`
}

// ValidateTOMLConfig parses and validates a TOML config file at path.
// It returns a slice of human-readable error strings.
// An empty slice means the file is valid.
//
// Rules:
//   - File must be valid TOML.
//   - Keys "budget" and "mode" are required.
//   - mode must be one of: "none", "minimal", "aggressive".
//   - budget must be > 0 when present.
func ValidateTOMLConfig(path string) []string {
	var errs []string

	data, err := os.ReadFile(path)
	if err != nil {
		return append(errs, fmt.Sprintf("cannot read file: %s", err.Error()))
	}

	// Parse into a raw map first to detect which keys are present.
	var raw map[string]interface{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return append(errs, fmt.Sprintf("invalid TOML: %s", err.Error()))
	}

	// Check required keys.
	if _, ok := raw["budget"]; !ok {
		errs = append(errs, "missing required key: budget")
	}
	if _, ok := raw["mode"]; !ok {
		errs = append(errs, "missing required key: mode")
	}

	// Typed decode for value validation.
	var cfg tomlConfig
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		// Already caught above; skip double-reporting.
		return errs
	}

	validModes := map[string]bool{"none": true, "minimal": true, "aggressive": true}
	if modeRaw, ok := raw["mode"]; ok {
		if modeStr, ok := modeRaw.(string); ok {
			if !validModes[modeStr] {
				errs = append(errs, fmt.Sprintf("invalid mode %q (valid: none, minimal, aggressive)", modeStr))
			}
		} else {
			errs = append(errs, "mode must be a string")
		}
	}

	if budgetRaw, ok := raw["budget"]; ok {
		switch b := budgetRaw.(type) {
		case int64:
			if b <= 0 {
				errs = append(errs, fmt.Sprintf("budget must be > 0, got %d", b))
			}
		case float64:
			if b <= 0 {
				errs = append(errs, fmt.Sprintf("budget must be > 0, got %g", b))
			}
		default:
			errs = append(errs, "budget must be a number")
		}
	}

	return errs
}

// TOMLValidator watches a TOML config file and validates it whenever the file
// changes on disk.
type TOMLValidator struct {
	mu   sync.Mutex
	path string
}

// NewTOMLValidator creates a TOMLValidator for the given file path.
func NewTOMLValidator(path string) *TOMLValidator {
	return &TOMLValidator{path: path}
}

// Validate validates the currently configured file path.
func (tv *TOMLValidator) Validate() []string {
	tv.mu.Lock()
	path := tv.path
	tv.mu.Unlock()
	return ValidateTOMLConfig(path)
}

// WatchAndValidate starts a background goroutine that polls path every
// interval, calling onError when validation errors are detected. It returns
// a stop function; calling it terminates the watcher.
func WatchAndValidate(path string, interval time.Duration, onError func([]string)) func() {
	stopCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var lastMtime time.Time

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				info, err := os.Stat(path)
				if err != nil {
					// File may not exist yet; report as error.
					if onError != nil {
						onError([]string{fmt.Sprintf("cannot stat file: %s", err.Error())})
					}
					continue
				}

				mtime := info.ModTime()
				if mtime == lastMtime {
					// File unchanged since last check.
					continue
				}
				lastMtime = mtime

				if errs := ValidateTOMLConfig(path); len(errs) > 0 {
					if onError != nil {
						onError(errs)
					}
				}
			}
		}
	}()

	return func() {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}
}
