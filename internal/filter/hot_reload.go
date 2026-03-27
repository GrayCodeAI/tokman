package filter

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// HotReloadablePipeline wraps a PipelineCoordinator and monitors a config file
// for changes, transparently rebuilding the pipeline when the config changes.
// Task #97: Pipeline stage hot-reload.
type HotReloadablePipeline struct {
	mu          sync.RWMutex
	coordinator *PipelineCoordinator
	cfgPath     string
	cfgMtime    time.Time
	buildFn     func() (*PipelineCoordinator, error)
	stopCh      chan struct{}
	OnReload    func(err error) // optional callback called on each reload attempt
}

// NewHotReloadablePipeline creates a hot-reloadable pipeline.
// buildFn is called to construct a fresh PipelineCoordinator whenever the config
// file at cfgPath changes. If cfgPath is empty, only manual Reload() is supported.
func NewHotReloadablePipeline(cfgPath string, buildFn func() (*PipelineCoordinator, error)) (*HotReloadablePipeline, error) {
	coordinator, err := buildFn()
	if err != nil {
		return nil, fmt.Errorf("hot reload: initial build: %w", err)
	}

	hr := &HotReloadablePipeline{
		coordinator: coordinator,
		cfgPath:     cfgPath,
		buildFn:     buildFn,
		stopCh:      make(chan struct{}),
	}

	if cfgPath != "" {
		if info, err := os.Stat(cfgPath); err == nil {
			hr.cfgMtime = info.ModTime()
		}
		go hr.watchLoop(5 * time.Second)
	}

	return hr, nil
}

// Process compresses input using the current pipeline (safe for concurrent use).
func (hr *HotReloadablePipeline) Process(input string) (string, *PipelineStats) {
	hr.mu.RLock()
	c := hr.coordinator
	hr.mu.RUnlock()
	return c.Process(input)
}

// Reload manually triggers a pipeline rebuild.
func (hr *HotReloadablePipeline) Reload() error {
	newCoordinator, err := hr.buildFn()
	if err != nil {
		if hr.OnReload != nil {
			hr.OnReload(err)
		}
		return fmt.Errorf("hot reload: rebuild: %w", err)
	}

	hr.mu.Lock()
	hr.coordinator = newCoordinator
	hr.mu.Unlock()

	if hr.OnReload != nil {
		hr.OnReload(nil)
	}
	return nil
}

// Stop halts the background file watcher goroutine.
func (hr *HotReloadablePipeline) Stop() {
	select {
	case <-hr.stopCh:
	default:
		close(hr.stopCh)
	}
}

// watchLoop polls the config file for mtime changes and reloads as needed.
func (hr *HotReloadablePipeline) watchLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-hr.stopCh:
			return
		case <-ticker.C:
			info, err := os.Stat(hr.cfgPath)
			if err != nil {
				continue
			}
			hr.mu.RLock()
			mtime := hr.cfgMtime
			hr.mu.RUnlock()

			if info.ModTime().After(mtime) {
				hr.mu.Lock()
				hr.cfgMtime = info.ModTime()
				hr.mu.Unlock()
				_ = hr.Reload()
			}
		}
	}
}
