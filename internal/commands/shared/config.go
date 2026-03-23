package shared

import (
	"sync"

	"github.com/GrayCodeAI/tokman/internal/config"
)

// Configuration loading and caching.
// This file handles config state - depends only on config package.

var (
	cachedConfig *config.Config
	configOnce   sync.Once
	cfgFile      string
)

// SetConfigFile sets the config file path.
func SetConfigFile(path string) {
	cfgFile = path
}

// GetConfig returns the cached configuration.
func GetConfig() (*config.Config, error) {
	return GetCachedConfig(), nil
}

// GetCachedConfig returns the cached config, loading it on first access.
func GetCachedConfig() *config.Config {
	configOnce.Do(func() {
		cfgFileToUse := cfgFile
		if cfgFileToUse == "" {
			cfgFileToUse = CfgFile
		}
		cachedConfig, _ = config.Load(cfgFileToUse)
	})
	if cachedConfig == nil {
		cachedConfig = config.Defaults()
	}
	return cachedConfig
}
