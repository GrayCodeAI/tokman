package shared

import (
	"path/filepath"

	"github.com/GrayCodeAI/tokman/internal/config"
)

// GetConfigPath returns the effective config path, honoring the shared
// override when --config is provided.
func GetConfigPath() string {
	if CfgFile != "" {
		return CfgFile
	}
	return config.ConfigPath()
}

// GetConfigDir returns the directory that contains the effective config file.
func GetConfigDir() string {
	return filepath.Dir(GetConfigPath())
}

// GetDataPath returns the TokMan data directory.
func GetDataPath() string {
	return config.DataPath()
}

// GetProjectPath returns the canonical current working directory.
func GetProjectPath() string {
	return config.ProjectPath()
}

// GetHooksPath returns the TokMan hooks directory.
func GetHooksPath() string {
	return config.HooksPath()
}

// GetFiltersDir returns the effective user filters directory.
func GetFiltersDir() string {
	return config.FiltersDir()
}
