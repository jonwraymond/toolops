// Package exporters provides factory functions for creating OpenTelemetry exporters.
package exporters

import (
	"context"
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

// NewTracingExporter creates a trace span exporter based on the exporter name.
// Supported exporters: stdout, otlp, jaeger, none
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
			return nil, fmt.Errorf("OTLP endpoint not configured: set OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		}
		return otlptracegrpc.New(ctx)

	case "jaeger":
		// Jaeger now uses OTLP, check for endpoint
		endpoint := os.Getenv("OTEL_EXPORTER_JAEGER_ENDPOINT")
		if endpoint == "" {
			return nil, fmt.Errorf("Jaeger endpoint not configured: set OTEL_EXPORTER_JAEGER_ENDPOINT")
		}
		// Use OTLP exporter for Jaeger (Jaeger supports OTLP natively)
		return otlptracegrpc.New(ctx)

	case "none", "":
		// Return a no-op exporter that discards everything
		return stdouttrace.New(stdouttrace.WithWriter(io.Discard))

	default:
		return nil, fmt.Errorf("unknown exporter: %q", name)
	}
}

// NewMetricsReader creates a metrics reader based on the exporter name.
// Supported exporters: stdout, otlp, prometheus, none
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
			return nil, fmt.Errorf("OTLP metrics endpoint not configured: set OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
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
		return nil, fmt.Errorf("unknown metrics exporter: %q", name)
	}
}
