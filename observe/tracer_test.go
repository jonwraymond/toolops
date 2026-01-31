package observe

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestToolMeta_SpanNameWithNamespace verifies span name includes namespace.
func TestToolMeta_SpanNameWithNamespace(t *testing.T) {
	meta := ToolMeta{
		Namespace: "gh",
		Name:      "issue",
	}

	expected := "tool.exec.gh.issue"
	if got := meta.SpanName(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestToolMeta_SpanNameWithoutNamespace verifies span name without namespace.
func TestToolMeta_SpanNameWithoutNamespace(t *testing.T) {
	meta := ToolMeta{
		Namespace: "",
		Name:      "read",
	}

	expected := "tool.exec.read"
	if got := meta.SpanName(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestToolMeta_ID verifies ID generation with and without namespace.
func TestToolMeta_ID(t *testing.T) {
	tests := []struct {
		name     string
		meta     ToolMeta
		expected string
	}{
		{
			name:     "with namespace",
			meta:     ToolMeta{Namespace: "github", Name: "create_issue"},
			expected: "github.create_issue",
		},
		{
			name:     "without namespace",
			meta:     ToolMeta{Namespace: "", Name: "read_file"},
			expected: "read_file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.meta.ToolID(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestTracer_SpanAttributes verifies all attributes are present on span.
func TestTracer_SpanAttributes(t *testing.T) {
	// Set up in-memory span recorder
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := tp.Tracer("test")

	tr := &tracerImpl{tracer: tracer}
	meta := ToolMeta{
		ID:        "github.create_issue",
		Namespace: "github",
		Name:      "create_issue",
		Version:   "1.0.0",
		Tags:      []string{"api", "github"},
		Category:  "integration",
	}

	ctx, span := tr.StartSpan(context.Background(), meta)
	tr.EndSpan(span, nil)
	_ = ctx // Suppress unused warning

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	s := spans[0]

	// Verify span name
	if s.Name() != "tool.exec.github.create_issue" {
		t.Errorf("expected span name 'tool.exec.github.create_issue', got %q", s.Name())
	}

	// Verify attributes
	attrs := s.Attributes()
	attrMap := make(map[string]attribute.Value)
	for _, a := range attrs {
		attrMap[string(a.Key)] = a.Value
	}

	// Required attributes
	if v, ok := attrMap["tool.id"]; !ok || v.AsString() != "github.create_issue" {
		t.Errorf("expected tool.id='github.create_issue', got %v", v)
	}
	if v, ok := attrMap["tool.namespace"]; !ok || v.AsString() != "github" {
		t.Errorf("expected tool.namespace='github', got %v", v)
	}
	if v, ok := attrMap["tool.name"]; !ok || v.AsString() != "create_issue" {
		t.Errorf("expected tool.name='create_issue', got %v", v)
	}
	if v, ok := attrMap["tool.error"]; !ok || v.AsBool() != false {
		t.Errorf("expected tool.error=false, got %v", v)
	}

	// Optional attributes
	if v, ok := attrMap["tool.version"]; !ok || v.AsString() != "1.0.0" {
		t.Errorf("expected tool.version='1.0.0', got %v", v)
	}
	if v, ok := attrMap["tool.category"]; !ok || v.AsString() != "integration" {
		t.Errorf("expected tool.category='integration', got %v", v)
	}
}

// TestTracer_SpanAttributesMinimal verifies only required attributes when minimal meta.
func TestTracer_SpanAttributesMinimal(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := tp.Tracer("test")

	tr := &tracerImpl{tracer: tracer}
	meta := ToolMeta{
		Name: "read_file",
	}

	ctx, span := tr.StartSpan(context.Background(), meta)
	tr.EndSpan(span, nil)
	_ = ctx

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	s := spans[0]
	attrs := s.Attributes()
	attrMap := make(map[string]attribute.Value)
	for _, a := range attrs {
		attrMap[string(a.Key)] = a.Value
	}

	// Required attributes should be present
	if _, ok := attrMap["tool.id"]; !ok {
		t.Error("expected tool.id attribute")
	}
	if _, ok := attrMap["tool.name"]; !ok {
		t.Error("expected tool.name attribute")
	}
	if _, ok := attrMap["tool.error"]; !ok {
		t.Error("expected tool.error attribute")
	}

	// Optional attributes should NOT be present when empty
	if v, ok := attrMap["tool.version"]; ok && v.AsString() != "" {
		t.Errorf("expected no tool.version, got %v", v)
	}
	if v, ok := attrMap["tool.category"]; ok && v.AsString() != "" {
		t.Errorf("expected no tool.category, got %v", v)
	}
}

// TestTracer_ContextPropagation verifies parent span is propagated.
func TestTracer_ContextPropagation(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := tp.Tracer("test")

	tr := &tracerImpl{tracer: tracer}
	meta := ToolMeta{Name: "child_tool"}

	// Create parent span
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent")

	// Create child span through our tracer
	childCtx, childSpan := tr.StartSpan(parentCtx, meta)
	tr.EndSpan(childSpan, nil)
	parentSpan.End()
	_ = childCtx

	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Find the child span (the one with tool.exec prefix)
	var child sdktrace.ReadOnlySpan
	for _, s := range spans {
		if s.Name() == "tool.exec.child_tool" {
			child = s
			break
		}
	}
	if child == nil {
		t.Fatal("child span not found")
	}

	// Verify parent-child relationship
	if child.Parent().TraceID() != parentSpan.SpanContext().TraceID() {
		t.Error("child span should have same trace ID as parent")
	}
	if !child.Parent().SpanID().IsValid() {
		t.Error("child span should have valid parent span ID")
	}
}

// TestTracer_ErrorRecording verifies error sets span status and attribute.
func TestTracer_ErrorRecording(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := tp.Tracer("test")

	tr := &tracerImpl{tracer: tracer}
	meta := ToolMeta{Name: "failing_tool"}

	ctx, span := tr.StartSpan(context.Background(), meta)
	testErr := errors.New("execution failed")
	tr.EndSpan(span, testErr)
	_ = ctx

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	s := spans[0]

	// Verify error status
	if s.Status().Code != codes.Error {
		t.Errorf("expected error status, got %v", s.Status().Code)
	}

	// Verify tool.error attribute
	attrs := s.Attributes()
	var toolError bool
	for _, a := range attrs {
		if string(a.Key) == "tool.error" {
			toolError = a.Value.AsBool()
			break
		}
	}
	if !toolError {
		t.Error("expected tool.error=true")
	}
}
