package commands

import (
	"os"

	"github.com/GrayCodeAI/tokman/internal/integrity"
)

// ensureHookInstalled writes the hook file if missing or outdated
func ensureHookInstalled(hookPath string) (bool, error) {
	changed := false

	if content, err := os.ReadFile(hookPath); err == nil {
		if string(content) == rewriteHook {
			return false, nil
		}
	}

	// Write hook
	if err := os.WriteFile(hookPath, []byte(rewriteHook), 0755); err != nil {
		return false, err
	}
	changed = true

	// Store integrity hash
	if err := integrity.StoreHash(hookPath); err != nil {
		return changed, err
	}

	return changed, nil
}

// writeIfChanged writes content to file if it differs from existing content
func writeIfChanged(path, content, name string) error {
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == content {
			return nil
		}
	}
	return os.WriteFile(path, []byte(content), 0644)
}
