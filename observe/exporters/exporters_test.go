package exporters

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestExporter_InvalidName verifies unknown exporter name returns error.
func TestExporter_InvalidName(t *testing.T) {
	_, err := NewTracingExporter(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error for invalid exporter name")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unknown exporter") {
		t.Errorf("expected error to contain 'unknown exporter', got: %v", err)
	}
}

// TestExporter_StdoutTracing verifies stdout tracing exporter.
func TestExporter_StdoutTracing(t *testing.T) {
	exp, err := NewTracingExporter(context.Background(), "stdout")
	if err != nil {
		t.Fatalf("failed to create stdout tracing exporter: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
}

// TestExporter_StdoutMetrics verifies stdout metrics reader.
func TestExporter_StdoutMetrics(t *testing.T) {
	reader, err := NewMetricsReader(context.Background(), "stdout")
	if err != nil {
		t.Fatalf("failed to create stdout metrics reader: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
}

// TestExporter_OtlpMissingEndpoint verifies OTLP without endpoint env fails.
func TestExporter_OtlpMissingEndpoint(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")

	_, err := NewTracingExporter(context.Background(), "otlp")
	if err == nil {
		t.Fatal("expected error when OTLP endpoint not configured")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "endpoint") {
		t.Errorf("expected error to contain 'endpoint', got: %v", err)
	}
}

// TestExporter_OtlpWithEndpoint verifies OTLP with endpoint env succeeds.
func TestExporter_OtlpWithEndpoint(t *testing.T) {
	// Set endpoint env var
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	exp, err := NewTracingExporter(context.Background(), "otlp")
	if err != nil {
		t.Fatalf("failed to create OTLP exporter with endpoint: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
}

// TestExporter_JaegerMissingEndpoint verifies Jaeger without endpoint fails.
func TestExporter_JaegerMissingEndpoint(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_JAEGER_ENDPOINT")

	_, err := NewTracingExporter(context.Background(), "jaeger")
	if err == nil {
		t.Fatal("expected error when Jaeger endpoint not configured")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "endpoint") {
		t.Errorf("expected error to contain 'endpoint', got: %v", err)
	}
}

// TestExporter_PrometheusReturnsReader verifies Prometheus metrics reader.
func TestExporter_PrometheusReturnsReader(t *testing.T) {
	reader, err := NewMetricsReader(context.Background(), "prometheus")
	if err != nil {
		t.Fatalf("failed to create Prometheus reader: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
}

// TestExporter_NoneReturnsNoop verifies 'none' returns no-op exporter.
func TestExporter_NoneReturnsNoop(t *testing.T) {
	exp, err := NewTracingExporter(context.Background(), "none")
	if err != nil {
		t.Fatalf("failed to create none exporter: %v", err)
	}
	// 'none' can return nil (no exporter) or a no-op
	// Both are acceptable
	_ = exp
}

// TestExporter_NoneMetricsReturnsNoop verifies 'none' returns no-op reader.
func TestExporter_NoneMetricsReturnsNoop(t *testing.T) {
	reader, err := NewMetricsReader(context.Background(), "none")
	if err != nil {
		t.Fatalf("failed to create none metrics reader: %v", err)
	}
	// 'none' can return nil (no reader) or a no-op
	_ = reader
}

// TestExporter_MetricsInvalidName verifies unknown metrics exporter returns error.
func TestExporter_MetricsInvalidName(t *testing.T) {
	_, err := NewMetricsReader(context.Background(), "badvalue")
	if err == nil {
		t.Fatal("expected error for invalid metrics exporter name")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unknown") {
		t.Errorf("expected error to contain 'unknown', got: %v", err)
	}
}
