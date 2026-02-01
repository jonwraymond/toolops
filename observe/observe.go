package observe

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/jonwraymond/toolops/observe/exporters"
)

// Config holds all configuration for the Observer.
type Config struct {
	ServiceName string
	Version     string
	Tracing     TracingConfig
	Metrics     MetricsConfig
	Logging     LoggingConfig
}

// TracingConfig configures the tracing subsystem.
type TracingConfig struct {
	Enabled   bool
	Exporter  string  // otlp|jaeger|stdout|none
	SamplePct float64 // 0.0-1.0
}

// MetricsConfig configures the metrics subsystem.
type MetricsConfig struct {
	Enabled  bool
	Exporter string // otlp|prometheus|stdout|none
}

// LoggingConfig configures the logging subsystem.
type LoggingConfig struct {
	Enabled bool
	Level   string // debug|info|warn|error
}

// Valid tracing exporters.
var validTracingExporters = map[string]bool{
	"otlp":   true,
	"jaeger": true,
	"stdout": true,
	"none":   true,
	"":       true, // Empty is valid (disabled)
}

// Valid metrics exporters.
var validMetricsExporters = map[string]bool{
	"otlp":       true,
	"prometheus": true,
	"stdout":     true,
	"none":       true,
	"":           true, // Empty is valid (disabled)
}

// Valid log levels.
var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
	"":      true, // Empty is valid (disabled)
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return errors.New("service name is required")
	}

	if c.Tracing.Enabled {
		if !validTracingExporters[c.Tracing.Exporter] {
			return fmt.Errorf("unknown tracing exporter: %q", c.Tracing.Exporter)
		}
		if c.Tracing.SamplePct < 0 || c.Tracing.SamplePct > 1.0 {
			return fmt.Errorf("sample percentage must be between 0.0 and 1.0, got: %f", c.Tracing.SamplePct)
		}
	}

	if c.Metrics.Enabled {
		if !validMetricsExporters[c.Metrics.Exporter] {
			return fmt.Errorf("unknown metrics exporter: %q", c.Metrics.Exporter)
		}
	}

	if c.Logging.Enabled {
		if !validLogLevels[c.Logging.Level] {
			return fmt.Errorf("unknown log level: %q", c.Logging.Level)
		}
	}

	return nil
}

// Observer provides access to telemetry primitives.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: Shutdown must honor cancellation/deadlines.
// - Errors: Shutdown should be idempotent and return the first error encountered.
type Observer interface {
	// Tracer returns the configured tracer.
	Tracer() trace.Tracer

	// Meter returns the configured meter.
	Meter() metric.Meter

	// Logger returns the configured logger.
	Logger() Logger

	// Shutdown gracefully shuts down all telemetry providers.
	Shutdown(ctx context.Context) error
}

// Logger is a minimal structured logging interface.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: methods should honor cancellation/deadlines where applicable.
// - Errors: logging must be best-effort and must not panic.
type Logger interface {
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)
	WithTool(meta ToolMeta) Logger
}

// Field represents a structured log field.
type Field struct {
	Key   string
	Value any
}

// observer is the concrete implementation of Observer.
type observer struct {
	tracer         trace.Tracer
	meter          metric.Meter
	logger         Logger
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

// NewObserver creates a new Observer with the given configuration.
func NewObserver(ctx context.Context, cfg Config) (Observer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	obs := &observer{}

	// Set up resource for all providers
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Set up tracing
	if cfg.Tracing.Enabled {
		tp, tracer, err := setupTracing(ctx, cfg, res)
		if err != nil {
			return nil, fmt.Errorf("failed to setup tracing: %w", err)
		}
		obs.tracerProvider = tp
		obs.tracer = tracer
	} else {
		obs.tracer = tracenoop.NewTracerProvider().Tracer("noop")
	}

	// Set up metrics
	if cfg.Metrics.Enabled {
		mp, meter, err := setupMetrics(ctx, cfg, res)
		if err != nil {
			return nil, fmt.Errorf("failed to setup metrics: %w", err)
		}
		obs.meterProvider = mp
		obs.meter = meter
	} else {
		obs.meter = noop.NewMeterProvider().Meter("noop")
	}

	// Set up logging
	if cfg.Logging.Enabled {
		obs.logger = NewLogger(cfg.Logging.Level)
	} else {
		obs.logger = &noopLogger{}
	}

	return obs, nil
}

func setupTracing(ctx context.Context, cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, trace.Tracer, error) {
	exporter, err := exporters.NewTracingExporter(ctx, cfg.Tracing.Exporter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Configure sampler based on SamplePct
	var sampler sdktrace.Sampler
	if cfg.Tracing.SamplePct >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.Tracing.SamplePct <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.Tracing.SamplePct)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	}
	if exporter != nil {
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer(cfg.ServiceName)
	return tp, tracer, nil
}

func setupMetrics(ctx context.Context, cfg Config, res *resource.Resource) (*sdkmetric.MeterProvider, metric.Meter, error) {
	reader, err := exporters.NewMetricsReader(ctx, cfg.Metrics.Exporter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metrics reader: %w", err)
	}

	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	if reader != nil {
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	mp := sdkmetric.NewMeterProvider(opts...)
	otel.SetMeterProvider(mp)

	meter := mp.Meter(cfg.ServiceName)
	return mp, meter, nil
}

func (o *observer) Tracer() trace.Tracer {
	return o.tracer
}

func (o *observer) Meter() metric.Meter {
	return o.meter
}

func (o *observer) Logger() Logger {
	return o.logger
}

func (o *observer) Shutdown(ctx context.Context) error {
	var errs []error

	if o.tracerProvider != nil {
		if err := o.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}

	if o.meterProvider != nil {
		if err := o.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (l *noopLogger) Info(ctx context.Context, msg string, fields ...Field)  {}
func (l *noopLogger) Warn(ctx context.Context, msg string, fields ...Field)  {}
func (l *noopLogger) Error(ctx context.Context, msg string, fields ...Field) {}
func (l *noopLogger) Debug(ctx context.Context, msg string, fields ...Field) {}
func (l *noopLogger) WithTool(meta ToolMeta) Logger                          { return l }
