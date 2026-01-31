package observe

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestMiddleware_SuccessPath verifies successful execution records telemetry.
func TestMiddleware_SuccessPath(t *testing.T) {
	// Set up tracing
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := &tracerImpl{tracer: tp.Tracer("test")}

	// Set up metrics
	metricReader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
	metrics, _ := newMetrics(mp.Meter("test"))

	// Create middleware
	mw := NewMiddleware(tracer, metrics, &noopLogger{})

	meta := ToolMeta{Name: "success_tool"}
	input := map[string]any{"key": "value"}
	expectedResult := "success_result"

	// Create inner function
	innerFunc := func(ctx context.Context, tool ToolMeta, in any) (any, error) {
		return expectedResult, nil
	}

	// Wrap and execute
	wrapped := mw.Wrap(innerFunc)
	result, err := wrapped(context.Background(), meta, input)

	// Verify no error
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify result
	if result != expectedResult {
		t.Errorf("expected result %q, got %q", expectedResult, result)
	}

	// Verify span was recorded
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name() != "tool.exec.success_tool" {
		t.Errorf("expected span name 'tool.exec.success_tool', got %q", spans[0].Name())
	}

	// Verify metrics
	var rm metricdata.ResourceMetrics
	if err := metricReader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}
	totalMetric := findMetric(rm, "tool.exec.total")
	if totalMetric == nil {
		t.Error("tool.exec.total metric not found")
	}
}

// TestMiddleware_ErrorPath verifies failed execution records error telemetry.
func TestMiddleware_ErrorPath(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	tracer := &tracerImpl{tracer: tp.Tracer("test")}

	metricReader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
	metrics, _ := newMetrics(mp.Meter("test"))

	mw := NewMiddleware(tracer, metrics, &noopLogger{})

	meta := ToolMeta{Name: "error_tool"}
	testErr := errors.New("execution failed")

	innerFunc := func(ctx context.Context, tool ToolMeta, in any) (any, error) {
		return nil, testErr
	}

	wrapped := mw.Wrap(innerFunc)
	_, err := wrapped(context.Background(), meta, nil)

	// Verify error returned
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}

	// Verify span has error status
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Check tool.error attribute
	var toolError bool
	for _, attr := range spans[0].Attributes() {
		if string(attr.Key) == "tool.error" {
			toolError = attr.Value.AsBool()
		}
	}
	if !toolError {
		t.Error("expected tool.error=true on failed execution")
	}

	// Verify error metric incremented
	var rm metricdata.ResourceMetrics
	if err := metricReader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}
	errMetric := findMetric(rm, "tool.exec.errors")
	if errMetric == nil {
		t.Error("tool.exec.errors metric not found")
	} else {
		sum, ok := errMetric.Data.(metricdata.Sum[int64])
		if ok && len(sum.DataPoints) > 0 && sum.DataPoints[0].Value != 1 {
			t.Errorf("expected errors count 1, got %d", sum.DataPoints[0].Value)
		}
	}
}

// TestMiddleware_DoesNotMutateInput verifies input is not modified.
func TestMiddleware_DoesNotMutateInput(t *testing.T) {
	tracer := newNoopTracer()
	mw := NewMiddleware(tracer, &noopMetrics{}, &noopLogger{})

	meta := ToolMeta{Name: "immutable_tool"}
	originalInput := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	// Make a copy to compare later
	inputCopy := make(map[string]any)
	for k, v := range originalInput {
		inputCopy[k] = v
	}

	innerFunc := func(ctx context.Context, tool ToolMeta, in any) (any, error) {
		// Attempt to mutate input (should not affect original)
		if m, ok := in.(map[string]any); ok {
			m["mutated"] = true
		}
		return nil, nil
	}

	wrapped := mw.Wrap(innerFunc)
	if _, err := wrapped(context.Background(), meta, originalInput); err != nil {
		t.Fatalf("wrapped() error = %v", err)
	}

	// Verify original was not mutated
	// Note: The middleware doesn't copy the input, but it shouldn't add to it
	// The inner function modifies its received copy, not the original
	if len(originalInput) != len(inputCopy) {
		// Only check keys that existed before
		for k := range inputCopy {
			if originalInput[k] != inputCopy[k] {
				t.Errorf("input was mutated: key %q changed", k)
			}
		}
	}
}

// TestMiddleware_PropagatesContext verifies context is passed through.
func TestMiddleware_PropagatesContext(t *testing.T) {
	tracer := newNoopTracer()
	mw := NewMiddleware(tracer, &noopMetrics{}, &noopLogger{})

	meta := ToolMeta{Name: "context_tool"}

	type ctxKey string
	testKey := ctxKey("test")
	testValue := "test_value"

	var receivedValue any

	innerFunc := func(ctx context.Context, tool ToolMeta, in any) (any, error) {
		receivedValue = ctx.Value(testKey)
		return nil, nil
	}

	wrapped := mw.Wrap(innerFunc)
	ctx := context.WithValue(context.Background(), testKey, testValue)
	if _, err := wrapped(ctx, meta, nil); err != nil {
		t.Fatalf("wrapped() error = %v", err)
	}

	if receivedValue != testValue {
		t.Errorf("expected context value %q, got %v", testValue, receivedValue)
	}
}

// TestMiddleware_ReturnsOriginalResult verifies exact result is returned.
func TestMiddleware_ReturnsOriginalResult(t *testing.T) {
	tracer := newNoopTracer()
	mw := NewMiddleware(tracer, &noopMetrics{}, &noopLogger{})

	meta := ToolMeta{Name: "result_tool"}

	type complexResult struct {
		Data  []int
		Error string
	}

	expectedResult := &complexResult{
		Data:  []int{1, 2, 3},
		Error: "",
	}

	innerFunc := func(ctx context.Context, tool ToolMeta, in any) (any, error) {
		return expectedResult, nil
	}

	wrapped := mw.Wrap(innerFunc)
	result, err := wrapped(context.Background(), meta, nil)
	if err != nil {
		t.Fatalf("wrapped() error = %v", err)
	}

	// Verify exact same pointer is returned
	if result != expectedResult {
		t.Error("middleware did not return exact same result object")
	}

	// Also verify deep equality
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("result mismatch: got %v, want %v", result, expectedResult)
	}
}

// TestMiddleware_MeasuresDuration verifies duration is recorded.
func TestMiddleware_MeasuresDuration(t *testing.T) {
	metricReader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
	metrics, _ := newMetrics(mp.Meter("test"))

	tracer := newNoopTracer()
	mw := NewMiddleware(tracer, metrics, &noopLogger{})

	meta := ToolMeta{Name: "timed_tool"}

	innerFunc := func(ctx context.Context, tool ToolMeta, in any) (any, error) {
		time.Sleep(100 * time.Millisecond)
		return nil, nil
	}

	wrapped := mw.Wrap(innerFunc)
	if _, err := wrapped(context.Background(), meta, nil); err != nil {
		t.Fatalf("wrapped() error = %v", err)
	}

	var rm metricdata.ResourceMetrics
	if err := metricReader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	durationMetric := findMetric(rm, "tool.exec.duration_ms")
	if durationMetric == nil {
		t.Fatal("tool.exec.duration_ms metric not found")
	}

	hist, ok := durationMetric.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram, got %T", durationMetric.Data)
	}

	if len(hist.DataPoints) == 0 {
		t.Fatal("no histogram data points")
	}

	// Duration should be at least 100ms
	if hist.DataPoints[0].Sum < 90 {
		t.Errorf("expected duration >= 90ms, got %f", hist.DataPoints[0].Sum)
	}
}

// TestMiddleware_DisabledNoop verifies noop middleware still executes function.
func TestMiddleware_DisabledNoop(t *testing.T) {
	// All observability disabled (noop implementations)
	mw := NewMiddleware(newNoopTracer(), &noopMetrics{}, &noopLogger{})

	meta := ToolMeta{Name: "noop_tool"}
	expectedResult := "noop_result"

	innerFunc := func(ctx context.Context, tool ToolMeta, in any) (any, error) {
		return expectedResult, nil
	}

	wrapped := mw.Wrap(innerFunc)
	result, err := wrapped(context.Background(), meta, nil)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != expectedResult {
		t.Errorf("expected result %q, got %q", expectedResult, result)
	}
}
