package configcmd

import (
	"path/filepath"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/config"
)

func effectiveConfigPath() string {
	if shared.CfgFile != "" {
		return shared.CfgFile
	}
	return config.ConfigPath()
}

func effectiveConfigDir() string {
	return filepath.Dir(effectiveConfigPath())
}
