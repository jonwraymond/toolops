// Package health provides health checking primitives for tool operations.
//
// It implements a generic health checking framework for monitoring component
// health in tool execution systems. The package provides interfaces for defining
// health checks, aggregating results from multiple checkers, and exposing
// health status via HTTP endpoints compatible with Kubernetes probes.
//
// # Ecosystem Position
//
// health integrates with service mesh and orchestration systems:
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│                     Health Check Architecture                   │
//	├─────────────────────────────────────────────────────────────────┤
//	│                                                                 │
//	│   Kubernetes          health              Components            │
//	│   ┌─────────┐      ┌───────────┐        ┌───────────┐          │
//	│   │Liveness │─────▶│  HTTP     │        │  Memory   │          │
//	│   │ Probe   │      │ Handlers  │        │  Checker  │          │
//	│   ├─────────┤      │           │        ├───────────┤          │
//	│   │Readiness│─────▶│ /healthz  │◀───────│  Database │          │
//	│   │ Probe   │      │ /readyz   │        │  Checker  │          │
//	│   └─────────┘      │ /health   │        ├───────────┤          │
//	│                    │           │        │   Cache   │          │
//	│   Load Balancer    │ ┌───────┐ │        │  Checker  │          │
//	│   ┌─────────┐      │ │Aggreg-│◀┼────────┴───────────┘          │
//	│   │ Health  │─────▶│ │ ator  │ │                                │
//	│   │ Checks  │      │ └───────┘ │                                │
//	│   └─────────┘      └───────────┘                                │
//	│                                                                 │
//	└─────────────────────────────────────────────────────────────────┘
//
// # Status Types
//
// The [Status] type represents component health:
//
//   - [StatusHealthy]: Component is functioning normally
//   - [StatusDegraded]: Component is functioning but with issues
//   - [StatusUnhealthy]: Component is not functioning properly
//
// # Core Components
//
//   - [Checker]: Interface for health checks (Name() + Check())
//   - [CheckerFunc]: Adapter for function-based checkers
//   - [Result]: Health check outcome with status, message, details, duration
//   - [Aggregator]: Combines multiple checkers into composite health
//   - [MemoryChecker]: Built-in checker for memory usage thresholds
//
// # Quick Start
//
//	// Create checkers
//	memCheck := health.NewMemoryChecker(health.MemoryCheckerConfig{
//	    WarningThreshold:  0.80,
//	    CriticalThreshold: 0.95,
//	})
//
//	dbCheck := health.NewCheckerFunc("database", func(ctx context.Context) health.Result {
//	    if err := db.PingContext(ctx); err != nil {
//	        return health.Unhealthy("database unreachable", err)
//	    }
//	    return health.Healthy("database connected")
//	})
//
//	// Create aggregator
//	agg := health.NewAggregator()
//	agg.Register("memory", memCheck)
//	agg.Register("database", dbCheck)
//
//	// Check all components
//	results := agg.CheckAll(ctx)
//	overall := agg.OverallStatus(results)
//
// # HTTP Endpoints
//
// The package provides Kubernetes-compatible HTTP handlers:
//
//   - [LivenessHandler]: Simple /healthz endpoint - always returns 200 if running
//   - [ReadinessHandler]: Runs all checks, returns 503 if any unhealthy
//   - [DetailedHandler]: Returns JSON with full check details
//   - [SingleCheckHandler]: Check a specific component by name
//   - [RegisterHandlers]: Convenience function to register all handlers
//
// Example registration:
//
//	mux := http.NewServeMux()
//	health.RegisterHandlers(mux, aggregator)
//	// Registers: /healthz, /readyz, /health
//
// # Aggregation Behavior
//
// The [Aggregator] computes overall status using worst-case logic:
//
//   - If ANY check is Unhealthy → overall Unhealthy
//   - If ANY check is Degraded (and none Unhealthy) → overall Degraded
//   - If ALL checks are Healthy → overall Healthy
//
// Checks can run in parallel (default) or sequentially via [AggregatorConfig].
//
// # Thread Safety
//
// All exported types are safe for concurrent use:
//
//   - [Aggregator]: sync.RWMutex protects registration and check execution
//   - [MemoryChecker]: Stateless, concurrent-safe
//   - [CheckerFunc]: Delegates to user function, ensure your function is safe
//   - [Result]: Immutable after creation
//
// # Error Handling
//
// Sentinel errors (use errors.Is for checking):
//
//   - [ErrCheckFailed]: Generic health check failure
//   - [ErrCheckTimeout]: Check exceeded timeout
//   - [ErrCheckerNotFound]: Named checker not registered
//   - [ErrNoCheckers]: No checkers registered in aggregator
//
// # Integration with ApertureStack
//
// health integrates with other ApertureStack packages:
//
//   - resilience: Use CircuitBreaker.State() for health status
//   - observe: Log health check results via observability middleware
//   - MCP servers: Expose health endpoints for tool server monitoring
package health
