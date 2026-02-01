// Package exporters provides factory functions for creating OpenTelemetry exporters.
package exporters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Errors for exporter configuration.
var (
	// ErrEndpointNotConfigured indicates a required endpoint environment variable is not set.
	ErrEndpointNotConfigured = errors.New("exporters: endpoint not configured")

	// ErrInvalidExporter indicates an unknown exporter name.
	ErrInvalidExporter = errors.New("exporters: invalid exporter")
)

// NewTracingExporter creates a trace span exporter based on the exporter name.
//
// Supported exporters:
//   - "stdout": Writes traces to stdout (for development)
//   - "otlp": OTLP gRPC exporter (requires OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT)
//   - "jaeger": Jaeger via OTLP (requires OTEL_EXPORTER_JAEGER_ENDPOINT)
//   - "none" or "": No-op exporter (discards traces)
//
// Returns ErrEndpointNotConfigured if OTLP/Jaeger endpoint is not set.
// Returns ErrInvalidExporter for unknown exporter names.
func NewTracingExporter(ctx context.Context, name string) (sdktrace.SpanExporter, error) {
	switch name {
	case "stdout":
		return stdouttrace.New(stdouttrace.WithWriter(os.Stdout))

	case "otlp":
		// Check for OTLP endpoint
		endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			endpoint = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		}
		if endpoint == "" {
			return nil, fmt.Errorf("%w: set OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ErrEndpointNotConfigured)
		}
		return otlptracegrpc.New(ctx)

	case "jaeger":
		// Jaeger now uses OTLP, check for endpoint
		endpoint := os.Getenv("OTEL_EXPORTER_JAEGER_ENDPOINT")
		if endpoint == "" {
			return nil, fmt.Errorf("%w: set OTEL_EXPORTER_JAEGER_ENDPOINT", ErrEndpointNotConfigured)
		}
		// Use OTLP exporter for Jaeger (Jaeger supports OTLP natively)
		return otlptracegrpc.New(ctx)

	case "none", "":
		// Return a no-op exporter that discards everything
		return stdouttrace.New(stdouttrace.WithWriter(io.Discard))

	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidExporter, name)
	}
}

// NewMetricsReader creates a metrics reader based on the exporter name.
//
// Supported exporters:
//   - "stdout": Writes metrics to stdout (for development)
//   - "otlp": OTLP gRPC exporter (requires OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_METRICS_ENDPOINT)
//   - "prometheus": Prometheus scrape endpoint
//   - "none" or "": No-op reader (discards metrics)
//
// Returns ErrEndpointNotConfigured if OTLP endpoint is not set.
// Returns ErrInvalidExporter for unknown exporter names.
func NewMetricsReader(ctx context.Context, name string) (sdkmetric.Reader, error) {
	switch name {
	case "stdout":
		exp, err := stdoutmetric.New(stdoutmetric.WithWriter(os.Stdout))
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout metrics exporter: %w", err)
		}
		return sdkmetric.NewPeriodicReader(exp), nil

	case "otlp":
		endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			endpoint = os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
		}
		if endpoint == "" {
			return nil, fmt.Errorf("%w: set OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", ErrEndpointNotConfigured)
		}
		exp, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
		}
		return sdkmetric.NewPeriodicReader(exp), nil

	case "prometheus":
		exp, err := prometheus.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
		}
		return exp, nil

	case "none", "":
		// Return a no-op reader
		exp, err := stdoutmetric.New(stdoutmetric.WithWriter(io.Discard))
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp), nil

	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidExporter, name)
	}
}
