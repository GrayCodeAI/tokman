package core

import (
	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/dashboard"
)

func init() {
	registry.Add(func() { registry.Register(dashboard.Cmd()) })
}
