package discover

import (
	"strings"
)

// CommandMapping defines how a command should be rewritten
type CommandMapping struct {
	Original   string
	TokManCmd  string
	Enabled    bool
	PassArgs   bool // Whether to pass additional args through
}

// Registry maps original commands to TokMan wrappers
// Based on RTK discover/registry.rs
var Registry = map[string]CommandMapping{
	// Git commands
	"git status": {
		Original:   "git status",
		TokManCmd:  "tokman git status",
		Enabled:    true,
		PassArgs:   true,
	},
	"git diff": {
		Original:   "git diff",
		TokManCmd:  "tokman git diff",
		Enabled:    true,
		PassArgs:   true,
	},
	"git log": {
		Original:   "git log",
		TokManCmd:  "tokman git log",
		Enabled:    true,
		PassArgs:   true,
	},

	// LS commands
	"ls": {
		Original:   "ls",
		TokManCmd:  "tokman ls",
		Enabled:    true,
		PassArgs:   true,
	},
	"ls -la": {
		Original:   "ls -la",
		TokManCmd:  "tokman ls",
		Enabled:    true,
		PassArgs:   true,
	},
	"ls -l": {
		Original:   "ls -l",
		TokManCmd:  "tokman ls",
		Enabled:    true,
		PassArgs:   true,
	},

	// Go test commands
	"go test": {
		Original:   "go test",
		TokManCmd:  "tokman test",
		Enabled:    true,
		PassArgs:   true,
	},

	// Go build commands
	"go build": {
		Original:   "go build",
		TokManCmd:  "tokman build",
		Enabled:    true,
		PassArgs:   true,
	},
}

// Rewrite checks if a command should be rewritten and returns the TokMan version
func Rewrite(command string) string {
	// Check for exact match first
	if mapping, ok := Registry[command]; ok && mapping.Enabled {
		return mapping.TokManCmd
	}

	// Check for prefix match (command with args)
	for original, mapping := range Registry {
		if strings.HasPrefix(command, original+" ") && mapping.Enabled {
			// Replace the original command with TokMan version, keep the rest
			return mapping.TokManCmd + strings.TrimPrefix(command, original)
		}
		if strings.HasPrefix(command, original) && len(command) == len(original) && mapping.Enabled {
			return mapping.TokManCmd
		}
	}

	// No rewrite found, return original
	return command
}

// ShouldRewrite returns true if a command should be rewritten
func ShouldRewrite(command string) bool {
	// Check for exact match
	if mapping, ok := Registry[command]; ok {
		return mapping.Enabled
	}

	// Check for prefix match
	for original, mapping := range Registry {
		if strings.HasPrefix(command, original+" ") && mapping.Enabled {
			return true
		}
		if strings.HasPrefix(command, original) && len(command) == len(original) && mapping.Enabled {
			return true
		}
	}

	return false
}

// GetMapping returns the mapping for a command if one exists
func GetMapping(command string) (CommandMapping, bool) {
	// Check exact match first
	if mapping, ok := Registry[command]; ok {
		return mapping, true
	}

	// Check prefix match
	for original, mapping := range Registry {
		if strings.HasPrefix(command, original+" ") || 
		   (strings.HasPrefix(command, original) && len(command) == len(original)) {
			return mapping, true
		}
	}

	return CommandMapping{}, false
}

// ListRewrites returns all enabled rewrites
func ListRewrites() []CommandMapping {
	var rewrites []CommandMapping
	for _, mapping := range Registry {
		if mapping.Enabled {
			rewrites = append(rewrites, mapping)
		}
	}
	return rewrites
}
