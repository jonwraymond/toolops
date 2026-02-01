// Package observe provides OpenTelemetry-based observability for tool execution.
//
// It is a pure instrumentation library: no execution, no transport, no I/O
// beyond exporter setup. Consumers wire the Observer into toolrun/toolruntime
// or server middleware.
//
// # Overview
//
// observe provides three observability pillars:
//   - Tracing: OpenTelemetry spans with tool metadata attributes
//   - Metrics: Execution counters and duration histograms
//   - Logging: Structured JSON logging with automatic field redaction
//
// # Core Components
//
//   - [Observer]: Main facade providing Tracer, Meter, and Logger access
//   - [Tracer]: Span creation with tool metadata as span attributes
//   - [Metrics]: Records execution counts, errors, and duration histograms
//   - [Logger]: Structured JSON logging with sensitive field redaction
//   - [Middleware]: Wraps ExecuteFunc with complete observability
//
// # Quick Start
//
//	cfg := observe.Config{
//	    ServiceName: "my-service",
//	    Version:     "1.0.0",
//	    Tracing:     observe.TracingConfig{Enabled: true, Exporter: "otlp", SamplePct: 1.0},
//	    Metrics:     observe.MetricsConfig{Enabled: true, Exporter: "prometheus"},
//	    Logging:     observe.LoggingConfig{Enabled: true, Level: "info"},
//	}
//
//	obs, err := observe.NewObserver(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer obs.Shutdown(ctx)
//
//	// Create middleware and wrap tool execution
//	mw, _ := observe.MiddlewareFromObserver(obs)
//	wrappedExec := mw.Wrap(originalExecuteFunc)
//
//	// Execute - automatically traced, metered, and logged
//	result, err := wrappedExec(ctx, toolMeta, input)
//
// # Telemetry Details
//
// Tracing creates spans with deterministic names:
//   - With namespace: "tool.exec.<namespace>.<name>" (e.g., "tool.exec.github.create_issue")
//   - Without namespace: "tool.exec.<name>" (e.g., "tool.exec.read_file")
//
// Span attributes include:
//   - tool.id: Fully qualified tool identifier
//   - tool.name: Tool name (required)
//   - tool.namespace: Tool namespace (if set)
//   - tool.version: Tool version (if set)
//   - tool.category: Tool category (if set)
//   - tool.tags: Discovery tags (if set)
//   - tool.error: Boolean indicating execution failure
//
// Metrics recorded:
//   - tool.exec.total (counter): Total executions by tool
//   - tool.exec.errors (counter): Total errors by tool
//   - tool.exec.duration_ms (histogram): Duration distribution in milliseconds
//
// All metrics include labels: tool.id, tool.name, tool.namespace (if set).
//
// # Sensitive Field Redaction
//
// The logger automatically redacts these fields to prevent credential leakage:
//   - input, inputs
//   - password, secret, token
//   - api_key, apiKey, credential
//
// See [RedactedFields] for the complete list.
//
// # Exporter Configuration
//
// Tracing exporters:
//   - "otlp": OTLP gRPC (requires OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT)
//   - "jaeger": Jaeger via OTLP (requires OTEL_EXPORTER_JAEGER_ENDPOINT)
//   - "stdout": Console output for development
//   - "none" or "": Disabled (no-op)
//
// Metrics exporters:
//   - "otlp": OTLP gRPC (requires OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_METRICS_ENDPOINT)
//   - "prometheus": Prometheus scrape endpoint
//   - "stdout": Console output for development
//   - "none" or "": Disabled (no-op)
//
// # Thread Safety
//
// All exported types are safe for concurrent use after construction:
//   - [Observer]: Tracer(), Meter(), Logger() are safe; Shutdown() is idempotent
//   - [Tracer]: StartSpan() and EndSpan() are safe for concurrent use
//   - [Metrics]: RecordExecution() is safe for concurrent use
//   - [Logger]: All logging methods are mutex-protected
//   - [Middleware]: Wrap() returns a thread-safe ExecuteFunc
//
// # Error Handling
//
// Configuration errors (use errors.Is for checking):
//   - [ErrMissingServiceName]: Config.ServiceName is empty
//   - [ErrInvalidSamplePct]: Tracing.SamplePct not in [0.0, 1.0]
//   - [ErrInvalidTracingExporter]: Unknown tracing exporter name
//   - [ErrInvalidMetricsExporter]: Unknown metrics exporter name
//   - [ErrInvalidLogLevel]: Unknown log level
//
// Exporter errors:
//   - [ErrEndpointNotConfigured]: Required endpoint env var not set
//
// Runtime errors:
//   - [ErrNilObserver]: Nil Observer passed to function
//   - [ErrMissingToolName]: ToolMeta.Name is empty
//
// Example error handling:
//
//	obs, err := observe.NewObserver(ctx, cfg)
//	if errors.Is(err, observe.ErrMissingServiceName) {
//	    // Handle missing service name
//	}
//	if errors.Is(err, observe.ErrEndpointNotConfigured) {
//	    // Handle missing OTLP endpoint
//	}
//
// # Integration with ApertureStack
//
// observe integrates with other ApertureStack packages:
//   - toolrun/toolruntime: Wrap tool execution with Middleware
//   - MCP servers: Add observability to tool handlers
//   - HTTP middleware: Instrument API endpoints
package observe
