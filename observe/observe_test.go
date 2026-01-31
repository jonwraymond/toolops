package observe

import (
	"context"
	"strings"
	"testing"
)

// TestConfigValidate_Valid verifies that a fully valid config passes validation.
func TestConfigValidate_Valid(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
		Tracing: TracingConfig{
			Enabled:   true,
			Exporter:  "stdout",
			SamplePct: 1.0,
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Exporter: "stdout",
		},
		Logging: LoggingConfig{
			Enabled: true,
			Level:   "info",
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

// TestConfigValidate_MissingServiceName verifies that empty ServiceName fails validation.
func TestConfigValidate_MissingServiceName(t *testing.T) {
	cfg := Config{
		ServiceName: "",
		Version:     "1.0.0",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing service name, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "service name") {
		t.Errorf("expected error to contain 'service name', got: %v", err)
	}
}

// TestConfigValidate_UnknownTracingExporter verifies that unknown tracing exporter fails validation.
func TestConfigValidate_UnknownTracingExporter(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Tracing: TracingConfig{
			Enabled:  true,
			Exporter: "unknown",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unknown tracing exporter, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unknown tracing exporter") {
		t.Errorf("expected error to contain 'unknown tracing exporter', got: %v", err)
	}
}

// TestConfigValidate_UnknownMetricsExporter verifies that unknown metrics exporter fails validation.
func TestConfigValidate_UnknownMetricsExporter(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Metrics: MetricsConfig{
			Enabled:  true,
			Exporter: "badvalue",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unknown metrics exporter, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unknown metrics exporter") {
		t.Errorf("expected error to contain 'unknown metrics exporter', got: %v", err)
	}
}

// TestConfigValidate_SamplePctOutOfRange verifies that SamplePct > 1.0 fails validation.
func TestConfigValidate_SamplePctOutOfRange(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Tracing: TracingConfig{
			Enabled:   true,
			Exporter:  "stdout",
			SamplePct: 1.5,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for sample percentage out of range, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "sample percentage") {
		t.Errorf("expected error to contain 'sample percentage', got: %v", err)
	}
}

// TestConfigValidate_SamplePctNegative verifies that SamplePct < 0 fails validation.
func TestConfigValidate_SamplePctNegative(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Tracing: TracingConfig{
			Enabled:   true,
			Exporter:  "stdout",
			SamplePct: -0.1,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative sample percentage, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "sample percentage") {
		t.Errorf("expected error to contain 'sample percentage', got: %v", err)
	}
}

// TestConfigValidate_UnknownLogLevel verifies that unknown log level fails validation.
func TestConfigValidate_UnknownLogLevel(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Logging: LoggingConfig{
			Enabled: true,
			Level:   "badlevel",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unknown log level, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unknown log level") {
		t.Errorf("expected error to contain 'unknown log level', got: %v", err)
	}
}

// TestNewObserver_DisabledNoop verifies that all-disabled config returns no-op observer.
func TestNewObserver_DisabledNoop(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Tracing:     TracingConfig{Enabled: false},
		Metrics:     MetricsConfig{Enabled: false},
		Logging:     LoggingConfig{Enabled: false},
	}

	obs, err := NewObserver(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if obs == nil {
		t.Fatal("expected non-nil observer")
	}
	// No-op observer should still be usable
	if obs.Tracer() == nil {
		t.Error("expected non-nil tracer (noop)")
	}
	if obs.Meter() == nil {
		t.Error("expected non-nil meter (noop)")
	}
}

// TestNewObserver_ReturnsTracerAndMeter verifies enabled config returns functional tracer/meter.
func TestNewObserver_ReturnsTracerAndMeter(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
		Tracing: TracingConfig{
			Enabled:   true,
			Exporter:  "stdout",
			SamplePct: 1.0,
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Exporter: "stdout",
		},
	}

	obs, err := NewObserver(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if obs.Tracer() == nil {
		t.Error("expected non-nil tracer")
	}
	if obs.Meter() == nil {
		t.Error("expected non-nil meter")
	}
}

// TestNewObserver_LoggerReturnsNonNil verifies logging enabled returns non-nil logger.
func TestNewObserver_LoggerReturnsNonNil(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Logging: LoggingConfig{
			Enabled: true,
			Level:   "info",
		},
	}

	obs, err := NewObserver(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if obs.Logger() == nil {
		t.Error("expected non-nil logger")
	}
}

// TestNewObserver_InvalidConfigReturnsError verifies that invalid config returns error.
func TestNewObserver_InvalidConfigReturnsError(t *testing.T) {
	cfg := Config{
		ServiceName: "", // Invalid
	}

	_, err := NewObserver(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid config, got nil")
	}
}

// TestObserver_ShutdownGracefully verifies shutdown doesn't panic.
func TestObserver_ShutdownGracefully(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Tracing: TracingConfig{
			Enabled:   true,
			Exporter:  "stdout",
			SamplePct: 1.0,
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Exporter: "stdout",
		},
	}

	obs, err := NewObserver(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Shutdown should not panic
	err = obs.Shutdown(context.Background())
	if err != nil {
		t.Errorf("expected no shutdown error, got: %v", err)
	}
}
