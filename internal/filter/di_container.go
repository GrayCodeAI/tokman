package filter

import (
	"fmt"
	"sync"
)

// Container is a simple dependency injection container for filters.
// It registers named factory functions and resolves them on demand,
// with optional singleton lifecycle.
// Task #133: Filter dependency injection container.
type Container struct {
	mu        sync.RWMutex
	factories map[string]FilterFactory
	singletons map[string]Filter
}

// FilterFactory is a function that creates a Filter instance.
type FilterFactory func(cfg map[string]any) (Filter, error)

// NewContainer creates an empty DI container.
func NewContainer() *Container {
	return &Container{
		factories:  make(map[string]FilterFactory),
		singletons: make(map[string]Filter),
	}
}

// Register adds a named factory to the container.
func (c *Container) Register(name string, factory FilterFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.factories[name] = factory
}

// RegisterSingleton registers a pre-built Filter as a singleton.
func (c *Container) RegisterSingleton(name string, f Filter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.singletons[name] = f
}

// Resolve returns a Filter by name, creating it from the factory if needed.
// cfg is passed to the factory function. If cfg is nil, an empty map is used.
func (c *Container) Resolve(name string, cfg map[string]any) (Filter, error) {
	c.mu.RLock()
	if f, ok := c.singletons[name]; ok {
		c.mu.RUnlock()
		return f, nil
	}
	factory, ok := c.factories[name]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("di: no filter registered for %q", name)
	}
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return factory(cfg)
}

// MustResolve resolves a filter and panics on error.
func (c *Container) MustResolve(name string, cfg map[string]any) Filter {
	f, err := c.Resolve(name, cfg)
	if err != nil {
		panic("di: " + err.Error())
	}
	return f
}

// ResolveAll resolves a list of named filters in order.
func (c *Container) ResolveAll(names []string) ([]Filter, error) {
	filters := make([]Filter, 0, len(names))
	for _, name := range names {
		f, err := c.Resolve(name, nil)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

// Registered returns all registered filter names.
func (c *Container) Registered() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.factories)+len(c.singletons))
	for n := range c.factories {
		names = append(names, n)
	}
	for n := range c.singletons {
		if _, exists := c.factories[n]; !exists {
			names = append(names, n)
		}
	}
	return names
}

// DefaultContainer returns a pre-populated container with all built-in filters.
func DefaultContainer() *Container {
	c := NewContainer()

	// Register built-in filters as singletons.
	c.RegisterSingleton("whitespace", NewWhitespaceNormalizer())
	c.RegisterSingleton("var_shorten", NewVarShortenFilter())
	c.RegisterSingleton("smart_func_prune", NewSmartFuncPruner())
	c.RegisterSingleton("ansi_strip", NewANSIStripFilter())
	c.RegisterSingleton("sliding_window", NewSlidingWindowBudget(128_000))
	c.RegisterSingleton("smart_log", NewSmartLogFilter())

	// Register factory-based filters that need configuration.
	c.Register("numerical_quant", func(cfg map[string]any) (Filter, error) {
		return NewNumericalQuantizer(), nil
	})
	c.Register("sql_compress", func(cfg map[string]any) (Filter, error) {
		return NewSQLCompressFilter(), nil
	})
	c.Register("anonymize", func(cfg map[string]any) (Filter, error) {
		return NewAnonymizeFilter(), nil
	})

	return c
}
