package filter

import (
	"fmt"
	"sync"
)

// TenantConfig holds per-tenant compression configuration.
// Task #196: Multi-tenant compression with per-tenant config.
type TenantConfig struct {
	TenantID       string
	Mode           string
	Budget         int
	EnabledFilters []string
	AuditLog       bool
}

// TenantRegistry stores per-tenant configs and provides thread-safe access.
type TenantRegistry struct {
	mu      sync.RWMutex
	tenants map[string]TenantConfig
}

// NewTenantRegistry creates a new, empty TenantRegistry.
func NewTenantRegistry() *TenantRegistry {
	return &TenantRegistry{
		tenants: make(map[string]TenantConfig),
	}
}

// Register adds or replaces a tenant config in the registry.
func (r *TenantRegistry) Register(cfg TenantConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[cfg.TenantID] = cfg
}

// Get retrieves the config for a tenant. Returns false if not registered.
func (r *TenantRegistry) Get(tenantID string) (TenantConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.tenants[tenantID]
	return cfg, ok
}

// Remove deletes a tenant from the registry.
func (r *TenantRegistry) Remove(tenantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tenants, tenantID)
}

// TenantPipeline creates and runs a PipelineCoordinator for a specific tenant.
type TenantPipeline struct {
	registry *TenantRegistry
}

// NewTenantPipeline creates a TenantPipeline backed by the given registry.
func NewTenantPipeline(registry *TenantRegistry) *TenantPipeline {
	return &TenantPipeline{registry: registry}
}

// buildConfig converts a TenantConfig into a PipelineConfig.
func buildConfig(cfg TenantConfig) PipelineConfig {
	mode := ModeMinimal
	switch cfg.Mode {
	case "none":
		mode = ModeNone
	case "minimal":
		mode = ModeMinimal
	case "aggressive":
		mode = ModeAggressive
	}

	pcfg := PipelineConfig{
		Mode:   mode,
		Budget: cfg.Budget,
	}

	// Enable filters by name from EnabledFilters list.
	enabled := make(map[string]bool, len(cfg.EnabledFilters))
	for _, f := range cfg.EnabledFilters {
		enabled[f] = true
	}

	// If no explicit filter list, fall back to a minimal default.
	if len(enabled) == 0 {
		pcfg.EnableEntropy = true
		return pcfg
	}

	pcfg.EnableEntropy = enabled["entropy"]
	pcfg.EnablePerplexity = enabled["perplexity"]
	pcfg.EnableGoalDriven = enabled["goal_driven"]
	pcfg.EnableAST = enabled["ast"]
	pcfg.EnableContrastive = enabled["contrastive"]
	pcfg.NgramEnabled = enabled["ngram"]
	pcfg.EnableEvaluator = enabled["evaluator"]
	pcfg.EnableGist = enabled["gist"]
	pcfg.EnableHierarchical = enabled["hierarchical"]
	pcfg.EnableCompaction = enabled["compaction"]
	pcfg.EnableAttribution = enabled["attribution"]
	pcfg.EnableH2O = enabled["h2o"]
	pcfg.EnableAttentionSink = enabled["attention_sink"]
	pcfg.EnableTFIDF = enabled["tfidf"]
	pcfg.EnableReasoningTrace = enabled["reasoning_trace"]
	pcfg.EnableSymbolicCompress = enabled["symbolic_compress"]
	pcfg.EnablePhraseGrouping = enabled["phrase_grouping"]
	pcfg.EnableNumericalQuant = enabled["numerical_quant"]
	pcfg.EnableDynamicRatio = enabled["dynamic_ratio"]
	pcfg.EnableMetaToken = enabled["meta_token"]
	pcfg.EnableSemanticChunk = enabled["semantic_chunk"]
	pcfg.EnableSketchStore = enabled["sketch_store"]
	pcfg.EnableLazyPruner = enabled["lazy_pruner"]
	pcfg.EnableSemanticAnchor = enabled["semantic_anchor"]
	pcfg.EnableAgentMemory = enabled["agent_memory"]
	pcfg.EnableQuestionAware = enabled["question_aware"]
	pcfg.EnableDensityAdaptive = enabled["density_adaptive"]

	return pcfg
}

// Process compresses input for the given tenant. Returns an error if the
// tenant is not registered.
func (tp *TenantPipeline) Process(tenantID, input string) (string, *PipelineStats, error) {
	cfg, ok := tp.registry.Get(tenantID)
	if !ok {
		return "", nil, fmt.Errorf("tenant %q is not registered", tenantID)
	}

	pcfg := buildConfig(cfg)
	p := NewPipelineCoordinator(pcfg)
	output, stats := p.Process(input)
	return output, stats, nil
}
