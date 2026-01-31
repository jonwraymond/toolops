package observe

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics records execution metrics for tools.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: must honor cancellation/deadlines and return quickly.
// - Errors: implementations must not panic.
type Metrics interface {
	// RecordExecution records a tool execution with duration and error status.
	RecordExecution(ctx context.Context, meta ToolMeta, duration time.Duration, err error)
}

// metricsImpl is the concrete implementation of Metrics.
type metricsImpl struct {
	meter        metric.Meter
	totalCount   metric.Int64Counter
	errorCount   metric.Int64Counter
	durationHist metric.Float64Histogram
}

// newMetrics creates a new Metrics instance with the given meter.
func newMetrics(meter metric.Meter) (*metricsImpl, error) {
	totalCount, err := meter.Int64Counter(
		"tool.exec.total",
		metric.WithDescription("Total number of tool executions"),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, err
	}

	errorCount, err := meter.Int64Counter(
		"tool.exec.errors",
		metric.WithDescription("Total number of tool execution errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	durationHist, err := meter.Float64Histogram(
		"tool.exec.duration_ms",
		metric.WithDescription("Tool execution duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &metricsImpl{
		meter:        meter,
		totalCount:   totalCount,
		errorCount:   errorCount,
		durationHist: durationHist,
	}, nil
}

// RecordExecution records metrics for a tool execution.
func (m *metricsImpl) RecordExecution(ctx context.Context, meta ToolMeta, duration time.Duration, err error) {
	// Build common attributes
	attrs := []attribute.KeyValue{
		attribute.String("tool.id", meta.ToolID()),
		attribute.String("tool.name", meta.Name),
	}

	// Add namespace if present
	if meta.Namespace != "" {
		attrs = append(attrs, attribute.String("tool.namespace", meta.Namespace))
	}

	opt := metric.WithAttributes(attrs...)

	// Always increment total counter
	m.totalCount.Add(ctx, 1, opt)

	// Increment error counter on failure
	if err != nil {
		m.errorCount.Add(ctx, 1, opt)
	}

	// Record duration in milliseconds
	durationMs := float64(duration.Milliseconds())
	m.durationHist.Record(ctx, durationMs, opt)
}

// noopMetrics is a metrics implementation that does nothing.
type noopMetrics struct{}

func (m *noopMetrics) RecordExecution(ctx context.Context, meta ToolMeta, duration time.Duration, err error) {
}
