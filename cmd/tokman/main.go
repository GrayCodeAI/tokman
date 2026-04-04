package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/GrayCodeAI/tokman/internal/commands"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var version = "dev"

func main() {
	shared.Version = version

	// Preload BPE tokenizer asynchronously to avoid blocking first token count
	core.WarmupBPETokenizer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var closeTrackerOnce sync.Once
	closeTracker := func() {
		closeTrackerOnce.Do(func() {
			_ = tracking.CloseGlobalTracker()
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		cancel()
		closeTracker()
	}()

	// Run commands in a context-aware way
	exitCode := commands.ExecuteContext(ctx)

	closeTracker()
	os.Exit(exitCode)
}
