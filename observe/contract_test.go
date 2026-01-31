package observe

import (
	"context"
	"testing"
	"time"
)

func TestObserverContract_Noops(t *testing.T) {
	cfg := Config{
		ServiceName: "observe-test",
		Tracing: TracingConfig{
			Enabled:  false,
			Exporter: "none",
		},
		Metrics: MetricsConfig{
			Enabled:  false,
			Exporter: "none",
		},
		Logging: LoggingConfig{
			Enabled: false,
			Level:   "info",
		},
	}

	obs, err := NewObserver(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewObserver failed: %v", err)
	}

	if obs.Tracer() == nil {
		t.Fatalf("expected non-nil tracer")
	}
	if obs.Meter() == nil {
		t.Fatalf("expected non-nil meter")
	}
	if obs.Logger() == nil {
		t.Fatalf("expected non-nil logger")
	}
}

func TestLoggerContract_WithTool(t *testing.T) {
	logger := &noopLogger{}
	if logger.WithTool(ToolMeta{Name: "noop"}) == nil {
		t.Fatalf("WithTool should return non-nil logger")
	}
}

func TestMetricsContract_NoPanic(t *testing.T) {
	metrics := &noopMetrics{}
	metrics.RecordExecution(context.Background(), ToolMeta{Name: "noop"}, 10*time.Millisecond, nil)
}

func TestTracerContract_NoPanic(t *testing.T) {
	tracer := newNoopTracer()
	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, ToolMeta{Name: "noop"})
	tracer.EndSpan(span, nil)
}
