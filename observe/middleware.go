package observe

import (
	"context"
	"time"
)

// ExecuteFunc is the signature for tool execution functions.
// This is the standard function signature that Middleware wraps.
type ExecuteFunc func(ctx context.Context, tool ToolMeta, input any) (any, error)

// Middleware wraps tool execution with observability (tracing, metrics, logging).
//
// Contract:
//   - Concurrency: Wrap() returns a thread-safe ExecuteFunc.
//   - Context: Propagates context through tracing spans.
//   - Errors: Errors from wrapped function are recorded and propagated unchanged.
//   - Ownership: Input/output values are passed through without modification.
type Middleware struct {
	tracer  Tracer
	metrics Metrics
	logger  Logger
}

// NewMiddleware creates a new Middleware with the given observability components.
func NewMiddleware(tracer Tracer, metrics Metrics, logger Logger) *Middleware {
	return &Middleware{
		tracer:  tracer,
		metrics: metrics,
		logger:  logger,
	}
}

// Wrap wraps an ExecuteFunc with tracing, metrics, and logging.
func (m *Middleware) Wrap(fn ExecuteFunc) ExecuteFunc {
	return func(ctx context.Context, tool ToolMeta, input any) (any, error) {
		// Start span
		ctx, span := m.tracer.StartSpan(ctx, tool)

		// Record start time
		start := time.Now()

		// Execute the function
		result, err := fn(ctx, tool, input)

		// Calculate duration
		duration := time.Since(start)

		// End span (records error status if err != nil)
		m.tracer.EndSpan(span, err)

		// Record metrics
		m.metrics.RecordExecution(ctx, tool, duration, err)

		// Log the execution
		toolLogger := m.logger.WithTool(tool)
		fields := []Field{
			{Key: "duration_ms", Value: float64(duration.Milliseconds())},
		}

		if err != nil {
			fields = append(fields, Field{Key: "error", Value: err.Error()})
			toolLogger.Error(ctx, "tool execution failed", fields...)
		} else {
			toolLogger.Info(ctx, "tool execution completed", fields...)
		}

		return result, err
	}
}

// MiddlewareFromObserver creates a Middleware from an Observer.
// This is a convenience function for common use cases.
func MiddlewareFromObserver(obs Observer) (*Middleware, error) {
	tracer := newTracer(obs.Tracer())

	metrics, err := newMetrics(obs.Meter())
	if err != nil {
		return nil, err
	}

	return NewMiddleware(tracer, metrics, obs.Logger()), nil
}
