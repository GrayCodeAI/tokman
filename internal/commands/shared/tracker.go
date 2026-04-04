package shared

import (
	"fmt"

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// GetDatabasePath returns the effective tracking database path using the
// loaded config when available, falling back to default path resolution.
func GetDatabasePath() string {
	cfg, err := GetCachedConfig()
	if err == nil && cfg != nil {
		return cfg.GetDatabasePath()
	}
	return config.DatabasePath()
}

// OpenTracker opens the tracking database using the effective configured path.
func OpenTracker() (*tracking.Tracker, error) {
	dbPath := GetDatabasePath()
	if dbPath == "" {
		return nil, fmt.Errorf("cannot determine database path")
	}
	return tracking.NewTracker(dbPath)
}
