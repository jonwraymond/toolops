package observe

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// ToolMeta contains metadata about a tool for telemetry purposes.
type ToolMeta struct {
	ID        string   // Fully qualified tool ID (namespace.name or just name)
	Namespace string   // Tool namespace (may be empty)
	Name      string   // Tool name (required)
	Version   string   // Tool version (optional)
	Tags      []string // Tool tags for discovery (optional)
	Category  string   // Tool category (optional)
}

// SpanName returns the deterministic span name for this tool.
// Format: tool.exec.<namespace>.<name> or tool.exec.<name>
func (m ToolMeta) SpanName() string {
	if m.Namespace != "" {
		return "tool.exec." + m.Namespace + "." + m.Name
	}
	return "tool.exec." + m.Name
}

// ToolID returns the fully qualified tool identifier.
// If ID field is set, returns it. Otherwise constructs from namespace and name.
func (m ToolMeta) ToolID() string {
	if m.ID != "" {
		return m.ID
	}
	if m.Namespace != "" {
		return m.Namespace + "." + m.Name
	}
	return m.Name
}

// Tracer wraps OpenTelemetry tracing with tool-specific span management.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: StartSpan must honor cancellation/deadlines and return ctx.Err() when canceled.
// - Errors: EndSpan must be best-effort and must not panic.
type Tracer interface {
	// StartSpan starts a new span for tool execution.
	StartSpan(ctx context.Context, meta ToolMeta) (context.Context, trace.Span)

	// EndSpan ends the span, recording any error.
	EndSpan(span trace.Span, err error)
}

// tracerImpl is the concrete implementation of Tracer.
type tracerImpl struct {
	tracer trace.Tracer
}

// newTracer creates a new Tracer wrapping the given OpenTelemetry tracer.
func newTracer(t trace.Tracer) Tracer {
	return &tracerImpl{tracer: t}
}

// StartSpan starts a new span with tool metadata as attributes.
func (t *tracerImpl) StartSpan(ctx context.Context, meta ToolMeta) (context.Context, trace.Span) {
	spanName := meta.SpanName()

	// Build attributes
	attrs := []attribute.KeyValue{
		attribute.String("tool.id", meta.ToolID()),
		attribute.String("tool.name", meta.Name),
		attribute.Bool("tool.error", false), // Will be updated in EndSpan if error
	}

	// Add namespace if present
	if meta.Namespace != "" {
		attrs = append(attrs, attribute.String("tool.namespace", meta.Namespace))
	}

	// Add optional attributes if present
	if meta.Version != "" {
		attrs = append(attrs, attribute.String("tool.version", meta.Version))
	}
	if meta.Category != "" {
		attrs = append(attrs, attribute.String("tool.category", meta.Category))
	}
	if len(meta.Tags) > 0 {
		attrs = append(attrs, attribute.StringSlice("tool.tags", meta.Tags))
	}

	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	return ctx, span
}

// EndSpan ends the span and records the error status if present.
func (t *tracerImpl) EndSpan(span trace.Span, err error) {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("tool.error", true))
		span.RecordError(err)
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

// noopTracer is a tracer that does nothing.
type noopTracer struct {
	noop trace.Tracer
}

// newNoopTracer creates a no-op tracer.
func newNoopTracer() Tracer {
	return &noopTracer{
		noop: tracenoop.NewTracerProvider().Tracer("noop"),
	}
}

func (t *noopTracer) StartSpan(ctx context.Context, meta ToolMeta) (context.Context, trace.Span) {
	return t.noop.Start(ctx, meta.SpanName())
}

func (t *noopTracer) EndSpan(span trace.Span, err error) {
	span.End()
}
