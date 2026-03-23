package main

import (
	"github.com/GrayCodeAI/tokman/internal/commands"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var version = "dev"

func main() {
	shared.Version = version
	commands.Execute()
}
