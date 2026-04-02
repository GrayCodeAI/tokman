// Package stress provides custom stress test scenarios
package stress

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// ScenarioBuilder builds custom stress test scenarios
type ScenarioBuilder struct {
	name        string
	description string
	scenType    TestType
	fn          func(ctx context.Context) error
	weight      int
	parameters  map[string]interface{}
}

// NewScenarioBuilder creates a new scenario builder
func NewScenarioBuilder(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		name:       name,
		scenType:   TypeLoad,
		weight:     1,
		parameters: make(map[string]interface{}),
	}
}

// WithDescription sets the description
func (sb *ScenarioBuilder) WithDescription(desc string) *ScenarioBuilder {
	sb.description = desc
	return sb
}

// WithType sets the scenario type
func (sb *ScenarioBuilder) WithType(t TestType) *ScenarioBuilder {
	sb.scenType = t
	return sb
}

// WithFunction sets the test function
func (sb *ScenarioBuilder) WithFunction(fn func(ctx context.Context) error) *ScenarioBuilder {
	sb.fn = fn
	return sb
}

// WithWeight sets the weight
func (sb *ScenarioBuilder) WithWeight(w int) *ScenarioBuilder {
	sb.weight = w
	return sb
}

// WithParameter adds a parameter
func (sb *ScenarioBuilder) WithParameter(key string, value interface{}) *ScenarioBuilder {
	sb.parameters[key] = value
	return sb
}

// Build creates the scenario
func (sb *ScenarioBuilder) Build() *Scenario {
	return &Scenario{
		Name:        sb.name,
		Description: sb.description,
		Type:        sb.scenType,
		Fn:          sb.fn,
		Weight:      sb.weight,
	}
}

// PredefinedScenarios returns a library of predefined scenarios
func PredefinedScenarios() map[string]*Scenario {
	return map[string]*Scenario{
		"api_health_check": APIHealthCheckScenario(),
		"database_query":   DatabaseQueryScenario(),
		"cache_stress":     CacheStressScenario(),
		"memory_intensive": MemoryIntensiveScenario(),
		"cpu_intensive":    CPUIntensiveScenario(),
		"io_intensive":     IOIntensiveScenario(),
		"mixed_workload":   MixedWorkloadScenario(),
		"login_simulation": LoginSimulationScenario(),
		"search_query":     SearchQueryScenario(),
		"webhook_delivery": WebhookDeliveryScenario(),
		"file_upload":      FileUploadScenario(),
		"batch_processing": BatchProcessingScenario(),
		"websocket_stress": WebSocketStressScenario(),
		"graphql_query":    GraphQLQueryScenario(),
		"grpc_request":     GRPCRequestScenario(),
	}
}

// APIHealthCheckScenario simulates API health checks
func APIHealthCheckScenario() *Scenario {
	return NewScenarioBuilder("api_health_check").
		WithDescription("Simulates API health check requests").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			// Simulate HTTP health check
			delay := time.Duration(10+rand.Intn(50)) * time.Millisecond
			select {
			case <-time.After(delay):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(1).
		Build()
}

// DatabaseQueryScenario simulates database queries
func DatabaseQueryScenario() *Scenario {
	return NewScenarioBuilder("database_query").
		WithDescription("Simulates database read/write operations").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			// Simulate DB query latency
			delay := time.Duration(20+rand.Intn(100)) * time.Millisecond
			select {
			case <-time.After(delay):
				// Simulate occasional errors (2% error rate)
				if rand.Float32() < 0.02 {
					return fmt.Errorf("database connection timeout")
				}
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(3).
		Build()
}

// CacheStressScenario simulates cache operations
func CacheStressScenario() *Scenario {
	return NewScenarioBuilder("cache_stress").
		WithDescription("Simulates cache read/write operations").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			delay := time.Duration(1+rand.Intn(5)) * time.Millisecond
			select {
			case <-time.After(delay):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(5).
		Build()
}

// MemoryIntensiveScenario simulates memory-intensive operations
func MemoryIntensiveScenario() *Scenario {
	return NewScenarioBuilder("memory_intensive").
		WithDescription("Allocates and manipulates large memory chunks").
		WithType(TypeStress).
		WithFunction(func(ctx context.Context) error {
			// Allocate memory
			size := 1024 * 1024 * (1 + rand.Intn(10)) // 1-10 MB
			data := make([]byte, size)
			_ = data

			select {
			case <-time.After(10 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(1).
		Build()
}

// CPUIntensiveScenario simulates CPU-intensive operations
func CPUIntensiveScenario() *Scenario {
	return NewScenarioBuilder("cpu_intensive").
		WithDescription("Performs CPU-intensive calculations").
		WithType(TypeStress).
		WithFunction(func(ctx context.Context) error {
			// Do some CPU work
			iterations := 1000 + rand.Intn(9000)
			result := 0
			for i := 0; i < iterations; i++ {
				result += i * i
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}
			_ = result
			return nil
		}).
		WithWeight(1).
		Build()
}

// IOIntensiveScenario simulates I/O operations
func IOIntensiveScenario() *Scenario {
	return NewScenarioBuilder("io_intensive").
		WithDescription("Simulates file I/O operations").
		WithType(TypeSoak).
		WithFunction(func(ctx context.Context) error {
			delay := time.Duration(50+rand.Intn(200)) * time.Millisecond
			select {
			case <-time.After(delay):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(2).
		Build()
}

// MixedWorkloadScenario combines multiple workloads
func MixedWorkloadScenario() *Scenario {
	return NewScenarioBuilder("mixed_workload").
		WithDescription("Randomly executes different workload types").
		WithType(TypeSpike).
		WithFunction(func(ctx context.Context) error {
			// Random workload type
			workloadType := rand.Intn(4)

			switch workloadType {
			case 0: // CPU
				iterations := 100 + rand.Intn(500)
				result := 1
				for i := 0; i < iterations; i++ {
					result *= 2
				}
				_ = result

			case 1: // Memory
				data := make([]byte, 1024*(10+rand.Intn(50)))
				_ = data

			case 2: // I/O simulation
				select {
				case <-time.After(time.Duration(20+rand.Intn(80)) * time.Millisecond):
				case <-ctx.Done():
					return ctx.Err()
				}

			case 3: // Network simulation
				select {
				case <-time.After(time.Duration(5+rand.Intn(25)) * time.Millisecond):
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}).
		WithWeight(3).
		Build()
}

// LoginSimulationScenario simulates user login
func LoginSimulationScenario() *Scenario {
	return NewScenarioBuilder("login_simulation").
		WithDescription("Simulates user authentication flow").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			// Simulate login flow with multiple steps
			steps := []time.Duration{
				time.Duration(20+rand.Intn(30)) * time.Millisecond, // Auth check
				time.Duration(10+rand.Intn(20)) * time.Millisecond, // Session create
				time.Duration(5+rand.Intn(15)) * time.Millisecond,  // Profile load
			}

			for _, delay := range steps {
				select {
				case <-time.After(delay):
					// Simulate occasional auth failures
					if rand.Float32() < 0.01 {
						return fmt.Errorf("authentication failed")
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}).
		WithWeight(2).
		Build()
}

// SearchQueryScenario simulates search operations
func SearchQueryScenario() *Scenario {
	return NewScenarioBuilder("search_query").
		WithDescription("Simulates search query execution").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			// Simulate search with varying complexity
			complexity := rand.Intn(100)
			var delay time.Duration

			if complexity < 80 {
				delay = time.Duration(10+rand.Intn(50)) * time.Millisecond // Simple query
			} else if complexity < 95 {
				delay = time.Duration(50+rand.Intn(150)) * time.Millisecond // Complex query
			} else {
				delay = time.Duration(200+rand.Intn(300)) * time.Millisecond // Very complex
			}

			select {
			case <-time.After(delay):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(4).
		Build()
}

// WebhookDeliveryScenario simulates webhook delivery
func WebhookDeliveryScenario() *Scenario {
	return NewScenarioBuilder("webhook_delivery").
		WithDescription("Simulates webhook delivery attempts").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			// Simulate webhook with retry logic
			attempts := 1 + rand.Intn(3)

			for i := 0; i < attempts; i++ {
				delay := time.Duration(100+rand.Intn(500)) * time.Millisecond

				select {
				case <-time.After(delay):
					// Simulate failures that will be retried
					if i < attempts-1 && rand.Float32() < 0.3 {
						continue // Retry
					}
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}).
		WithWeight(1).
		Build()
}

// FileUploadScenario simulates file uploads
func FileUploadScenario() *Scenario {
	return NewScenarioBuilder("file_upload").
		WithDescription("Simulates file upload operations").
		WithType(TypeStress).
		WithFunction(func(ctx context.Context) error {
			// Simulate upload with chunks
			fileSize := 1024 * 1024 * (1 + rand.Intn(100)) // 1-100 MB
			chunkSize := 64 * 1024                         // 64 KB chunks
			chunks := fileSize / chunkSize

			for i := 0; i < chunks; i++ {
				select {
				case <-time.After(time.Duration(10+rand.Intn(20)) * time.Millisecond):
					// Continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}).
		WithWeight(1).
		Build()
}

// BatchProcessingScenario simulates batch job processing
func BatchProcessingScenario() *Scenario {
	return NewScenarioBuilder("batch_processing").
		WithDescription("Simulates batch job processing").
		WithType(TypeSoak).
		WithFunction(func(ctx context.Context) error {
			// Simulate processing batch items
			batchSize := 10 + rand.Intn(90)

			for i := 0; i < batchSize; i++ {
				// Process item
				select {
				case <-time.After(time.Duration(5+rand.Intn(15)) * time.Millisecond):
					// Continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}).
		WithWeight(2).
		Build()
}

// WebSocketStressScenario simulates WebSocket connections
func WebSocketStressScenario() *Scenario {
	return NewScenarioBuilder("websocket_stress").
		WithDescription("Simulates WebSocket message exchange").
		WithType(TypeBreakdown).
		WithFunction(func(ctx context.Context) error {
			// Simulate connection lifecycle
			messages := 5 + rand.Intn(20)

			for i := 0; i < messages; i++ {
				select {
				case <-time.After(time.Duration(50+rand.Intn(200)) * time.Millisecond):
					// Continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}).
		WithWeight(1).
		Build()
}

// GraphQLQueryScenario simulates GraphQL queries
func GraphQLQueryScenario() *Scenario {
	return NewScenarioBuilder("graphql_query").
		WithDescription("Simulates GraphQL query execution").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			// Simulate query complexity
			complexity := rand.Intn(100)
			var delay time.Duration

			if complexity < 70 {
				delay = time.Duration(10+rand.Intn(30)) * time.Millisecond // Simple query
			} else if complexity < 90 {
				delay = time.Duration(30+rand.Intn(70)) * time.Millisecond // Medium query
			} else {
				delay = time.Duration(100+rand.Intn(200)) * time.Millisecond // Complex query with nesting
			}

			select {
			case <-time.After(delay):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(2).
		Build()
}

// GRPCRequestScenario simulates gRPC requests
func GRPCRequestScenario() *Scenario {
	return NewScenarioBuilder("grpc_request").
		WithDescription("Simulates gRPC unary and streaming requests").
		WithType(TypeLoad).
		WithFunction(func(ctx context.Context) error {
			// gRPC is typically faster than HTTP
			delay := time.Duration(5+rand.Intn(25)) * time.Millisecond

			select {
			case <-time.After(delay):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).
		WithWeight(3).
		Build()
}

// ScenarioComposer composes multiple scenarios
type ScenarioComposer struct {
	scenarios []*Scenario
	weights   []int
}

// NewScenarioComposer creates a new composer
func NewScenarioComposer() *ScenarioComposer {
	return &ScenarioComposer{
		scenarios: make([]*Scenario, 0),
		weights:   make([]int, 0),
	}
}

// Add adds a scenario with weight
func (sc *ScenarioComposer) Add(scenario *Scenario, weight int) *ScenarioComposer {
	sc.scenarios = append(sc.scenarios, scenario)
	sc.weights = append(sc.weights, weight)
	return sc
}

// Compose creates a composite scenario
func (sc *ScenarioComposer) Compose(name string) *Scenario {
	totalWeight := 0
	for _, w := range sc.weights {
		totalWeight += w
	}

	return &Scenario{
		Name: name,
		Type: TypeLoad,
		Fn: func(ctx context.Context) error {
			// Select scenario based on weight
			pick := rand.Intn(totalWeight)
			cumulative := 0

			for i, scenario := range sc.scenarios {
				cumulative += sc.weights[i]
				if pick < cumulative {
					return scenario.Fn(ctx)
				}
			}

			return sc.scenarios[0].Fn(ctx)
		},
	}
}

// ScenarioTemplate represents a reusable scenario template
type ScenarioTemplate struct {
	Name        string
	Description string
	Type        TestType
	BaseFn      func(ctx context.Context, params map[string]interface{}) error
	Defaults    map[string]interface{}
}

// Instantiate creates a scenario from template
func (st *ScenarioTemplate) Instantiate(params map[string]interface{}) *Scenario {
	// Merge params with defaults
	merged := make(map[string]interface{})
	for k, v := range st.Defaults {
		merged[k] = v
	}
	for k, v := range params {
		merged[k] = v
	}

	return &Scenario{
		Name:        st.Name,
		Description: st.Description,
		Type:        st.Type,
		Fn: func(ctx context.Context) error {
			return st.BaseFn(ctx, merged)
		},
	}
}

// CommonTemplates returns common scenario templates
func CommonTemplates() map[string]*ScenarioTemplate {
	return map[string]*ScenarioTemplate{
		"http_request": {
			Name:        "http_request",
			Description: "Generic HTTP request scenario",
			Type:        TypeLoad,
			Defaults: map[string]interface{}{
				"min_latency_ms": 10,
				"max_latency_ms": 100,
				"error_rate":     0.01,
			},
			BaseFn: func(ctx context.Context, params map[string]interface{}) error {
				minLatency := params["min_latency_ms"].(int)
				maxLatency := params["max_latency_ms"].(int)
				errorRate := params["error_rate"].(float64)

				delay := time.Duration(minLatency+rand.Intn(maxLatency-minLatency)) * time.Millisecond

				select {
				case <-time.After(delay):
					if rand.Float64() < errorRate {
						return fmt.Errorf("HTTP error")
					}
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		},
	}
}
