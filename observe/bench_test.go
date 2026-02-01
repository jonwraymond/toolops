package observe

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"
)

// BenchmarkLogger_Info measures logging throughput.
func BenchmarkLogger_Info(b *testing.B) {
	logger := NewLoggerWithWriter("info", io.Discard)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(ctx, "benchmark message", Field{Key: "iteration", Value: i})
	}
}

// BenchmarkLogger_Info_MultipleFields measures logging with multiple fields.
func BenchmarkLogger_Info_MultipleFields(b *testing.B) {
	logger := NewLoggerWithWriter("info", io.Discard)
	ctx := context.Background()
	fields := []Field{
		{Key: "field1", Value: "value1"},
		{Key: "field2", Value: 42},
		{Key: "field3", Value: true},
		{Key: "field4", Value: 3.14},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(ctx, "benchmark message", fields...)
	}
}

// BenchmarkLogger_WithTool measures creating tool-scoped loggers.
func BenchmarkLogger_WithTool(b *testing.B) {
	logger := NewLoggerWithWriter("info", io.Discard)
	meta := ToolMeta{
		Name:      "bench_tool",
		Namespace: "ns",
		Version:   "1.0.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.WithTool(meta)
	}
}

// BenchmarkLogger_WithTool_ThenLog measures the full pattern of creating
// a tool logger and logging.
func BenchmarkLogger_WithTool_ThenLog(b *testing.B) {
	logger := NewLoggerWithWriter("info", io.Discard)
	ctx := context.Background()
	meta := ToolMeta{
		Name:      "bench_tool",
		Namespace: "ns",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolLogger := logger.WithTool(meta)
		toolLogger.Info(ctx, "tool execution", Field{Key: "iteration", Value: i})
	}
}

// BenchmarkLogger_LevelFiltering measures overhead of level filtering.
func BenchmarkLogger_LevelFiltering(b *testing.B) {
	logger := NewLoggerWithWriter("error", io.Discard) // Only error level
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// These should be filtered out (no actual logging)
		logger.Debug(ctx, "filtered debug")
		logger.Info(ctx, "filtered info")
		logger.Warn(ctx, "filtered warn")
	}
}

// BenchmarkToolMeta_SpanName measures span name generation.
func BenchmarkToolMeta_SpanName(b *testing.B) {
	meta := ToolMeta{
		Name:      "create_issue",
		Namespace: "github",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = meta.SpanName()
	}
}

// BenchmarkToolMeta_SpanName_NoNamespace measures span name without namespace.
func BenchmarkToolMeta_SpanName_NoNamespace(b *testing.B) {
	meta := ToolMeta{
		Name: "read_file",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = meta.SpanName()
	}
}

// BenchmarkToolMeta_ToolID measures tool ID generation.
func BenchmarkToolMeta_ToolID(b *testing.B) {
	meta := ToolMeta{
		Name:      "search",
		Namespace: "github",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = meta.ToolID()
	}
}

// BenchmarkTracer_StartEndSpan measures tracer span lifecycle (noop).
func BenchmarkTracer_StartEndSpan(b *testing.B) {
	tracer := newNoopTracer()
	ctx := context.Background()
	meta := ToolMeta{
		Name:      "bench_tool",
		Namespace: "ns",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, span := tracer.StartSpan(ctx, meta)
		tracer.EndSpan(span, nil)
		_ = ctx
	}
}

// BenchmarkMetrics_RecordExecution measures metrics recording.
func BenchmarkMetrics_RecordExecution(b *testing.B) {
	ctx := context.Background()
	obs, err := NewObserver(ctx, Config{
		ServiceName: "bench",
		Metrics:     MetricsConfig{Enabled: true, Exporter: "none"},
	})
	if err != nil {
		b.Fatalf("failed to create observer: %v", err)
	}
	defer obs.Shutdown(ctx)

	metrics, err := newMetrics(obs.Meter())
	if err != nil {
		b.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{Name: "bench_tool", Namespace: "ns"}
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordExecution(ctx, meta, duration, nil)
	}
}

// BenchmarkMetrics_RecordExecution_WithError measures metrics with error.
func BenchmarkMetrics_RecordExecution_WithError(b *testing.B) {
	ctx := context.Background()
	obs, err := NewObserver(ctx, Config{
		ServiceName: "bench",
		Metrics:     MetricsConfig{Enabled: true, Exporter: "none"},
	})
	if err != nil {
		b.Fatalf("failed to create observer: %v", err)
	}
	defer obs.Shutdown(ctx)

	metrics, err := newMetrics(obs.Meter())
	if err != nil {
		b.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{Name: "bench_tool", Namespace: "ns"}
	duration := 100 * time.Millisecond
	execErr := fmt.Errorf("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordExecution(ctx, meta, duration, execErr)
	}
}

// BenchmarkMiddleware_Wrap measures full middleware wrapping.
func BenchmarkMiddleware_Wrap(b *testing.B) {
	ctx := context.Background()
	obs, err := NewObserver(ctx, Config{
		ServiceName: "bench",
		Tracing:     TracingConfig{Enabled: true, Exporter: "none"},
		Metrics:     MetricsConfig{Enabled: true, Exporter: "none"},
		Logging:     LoggingConfig{Enabled: false},
	})
	if err != nil {
		b.Fatalf("failed to create observer: %v", err)
	}
	defer obs.Shutdown(ctx)

	mw, err := MiddlewareFromObserver(obs)
	if err != nil {
		b.Fatalf("failed to create middleware: %v", err)
	}

	execFn := func(ctx context.Context, tool ToolMeta, input any) (any, error) {
		return "result", nil
	}
	wrapped := mw.Wrap(execFn)
	meta := ToolMeta{Name: "bench_tool", Namespace: "ns"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wrapped(ctx, meta, nil)
	}
}

// BenchmarkMiddleware_Wrap_WithLogging measures middleware with logging enabled.
func BenchmarkMiddleware_Wrap_WithLogging(b *testing.B) {
	ctx := context.Background()
	obs, err := NewObserver(ctx, Config{
		ServiceName: "bench",
		Tracing:     TracingConfig{Enabled: true, Exporter: "none"},
		Metrics:     MetricsConfig{Enabled: true, Exporter: "none"},
		Logging:     LoggingConfig{Enabled: true, Level: "info"},
	})
	if err != nil {
		b.Fatalf("failed to create observer: %v", err)
	}
	defer obs.Shutdown(ctx)

	// Replace logger with discard writer
	obsImpl := obs.(*observer)
	obsImpl.logger = NewLoggerWithWriter("info", io.Discard)

	mw, err := MiddlewareFromObserver(obs)
	if err != nil {
		b.Fatalf("failed to create middleware: %v", err)
	}

	execFn := func(ctx context.Context, tool ToolMeta, input any) (any, error) {
		return "result", nil
	}
	wrapped := mw.Wrap(execFn)
	meta := ToolMeta{Name: "bench_tool", Namespace: "ns"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wrapped(ctx, meta, nil)
	}
}

// BenchmarkConcurrent_Logger measures concurrent logging.
func BenchmarkConcurrent_Logger(b *testing.B) {
	logger := NewLoggerWithWriter("info", io.Discard)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Info(ctx, "concurrent message", Field{Key: "iteration", Value: i})
			i++
		}
	})
}

// BenchmarkConcurrent_Middleware measures concurrent middleware execution.
func BenchmarkConcurrent_Middleware(b *testing.B) {
	ctx := context.Background()
	obs, err := NewObserver(ctx, Config{
		ServiceName: "bench",
		Tracing:     TracingConfig{Enabled: true, Exporter: "none"},
		Metrics:     MetricsConfig{Enabled: true, Exporter: "none"},
		Logging:     LoggingConfig{Enabled: false},
	})
	if err != nil {
		b.Fatalf("failed to create observer: %v", err)
	}
	defer obs.Shutdown(ctx)

	mw, err := MiddlewareFromObserver(obs)
	if err != nil {
		b.Fatalf("failed to create middleware: %v", err)
	}

	execFn := func(ctx context.Context, tool ToolMeta, input any) (any, error) {
		return "result", nil
	}
	wrapped := mw.Wrap(execFn)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			meta := ToolMeta{
				Name:      fmt.Sprintf("tool_%d", i%100),
				Namespace: fmt.Sprintf("ns_%d", i%10),
			}
			_, _ = wrapped(ctx, meta, nil)
			i++
		}
	})
}

// BenchmarkConfig_Validate measures configuration validation.
func BenchmarkConfig_Validate(b *testing.B) {
	cfg := Config{
		ServiceName: "bench-service",
		Version:     "1.0.0",
		Tracing:     TracingConfig{Enabled: true, Exporter: "otlp", SamplePct: 0.5},
		Metrics:     MetricsConfig{Enabled: true, Exporter: "prometheus"},
		Logging:     LoggingConfig{Enabled: true, Level: "info"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}
