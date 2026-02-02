package observe_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jonwraymond/toolops/observe"
)

func ExampleNewObserver() {
	cfg := observe.Config{
		ServiceName: "example-service",
		Version:     "1.0.0",
		Tracing:     observe.TracingConfig{Enabled: true, Exporter: "none"},
		Metrics:     observe.MetricsConfig{Enabled: false},
		Logging:     observe.LoggingConfig{Enabled: true, Level: "info"},
	}

	ctx := context.Background()
	obs, err := observe.NewObserver(ctx, cfg)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer func() {
		_ = obs.Shutdown(ctx)
	}()

	fmt.Println("Observer created successfully")
	// Output:
	// Observer created successfully
}

func ExampleNewObserver_validation() {
	// Missing service name triggers validation error
	cfg := observe.Config{
		ServiceName: "", // Empty - will fail validation
	}

	ctx := context.Background()
	_, err := observe.NewObserver(ctx, cfg)
	if errors.Is(err, observe.ErrMissingServiceName) {
		fmt.Println("Caught: missing service name")
	}
	// Output:
	// Caught: missing service name
}

func ExampleConfig_Validate() {
	// Valid configuration
	cfg := observe.Config{
		ServiceName: "my-service",
		Version:     "1.0.0",
		Tracing: observe.TracingConfig{
			Enabled:   true,
			Exporter:  "stdout",
			SamplePct: 0.5, // 50% sampling
		},
		Metrics: observe.MetricsConfig{
			Enabled:  true,
			Exporter: "prometheus",
		},
		Logging: observe.LoggingConfig{
			Enabled: true,
			Level:   "info",
		},
	}

	if err := cfg.Validate(); err != nil {
		fmt.Println("Invalid:", err)
	} else {
		fmt.Println("Configuration is valid")
	}
	// Output:
	// Configuration is valid
}

func ExampleToolMeta_SpanName() {
	// With namespace
	meta := observe.ToolMeta{
		Name:      "create_issue",
		Namespace: "github",
	}
	fmt.Println(meta.SpanName())

	// Without namespace
	meta2 := observe.ToolMeta{
		Name: "read_file",
	}
	fmt.Println(meta2.SpanName())
	// Output:
	// tool.exec.github.create_issue
	// tool.exec.read_file
}

func ExampleToolMeta_ToolID() {
	// With explicit ID
	meta := observe.ToolMeta{
		ID:        "custom:tool:id",
		Name:      "ignored",
		Namespace: "ignored",
	}
	fmt.Println(meta.ToolID())

	// With namespace (ID constructed)
	meta2 := observe.ToolMeta{
		Name:      "search",
		Namespace: "github",
	}
	fmt.Println(meta2.ToolID())

	// Without namespace
	meta3 := observe.ToolMeta{
		Name: "read_file",
	}
	fmt.Println(meta3.ToolID())
	// Output:
	// custom:tool:id
	// github.search
	// read_file
}

func ExampleToolMeta_Validate() {
	// Valid metadata
	meta := observe.ToolMeta{
		Name:      "create_issue",
		Namespace: "github",
		Version:   "1.0.0",
	}
	if err := meta.Validate(); err != nil {
		fmt.Println("Invalid:", err)
	} else {
		fmt.Println("Valid tool metadata")
	}

	// Invalid - missing name
	meta2 := observe.ToolMeta{
		Namespace: "github",
	}
	if errors.Is(meta2.Validate(), observe.ErrMissingToolName) {
		fmt.Println("Caught: missing tool name")
	}
	// Output:
	// Valid tool metadata
	// Caught: missing tool name
}

func ExampleNewLoggerWithWriter() {
	var buf bytes.Buffer
	logger := observe.NewLoggerWithWriter("info", &buf)

	ctx := context.Background()
	logger.Info(ctx, "application started", observe.Field{Key: "version", Value: "1.0.0"})

	// Output contains JSON with timestamp, level, msg, and version field
	fmt.Println("Logged message contains 'application started':", bytes.Contains(buf.Bytes(), []byte("application started")))
	// Output:
	// Logged message contains 'application started': true
}

func ExampleLogger_WithTool() {
	var buf bytes.Buffer
	logger := observe.NewLoggerWithWriter("info", &buf)

	meta := observe.ToolMeta{
		Name:      "search",
		Namespace: "github",
		Version:   "2.0.0",
	}

	// Create tool-scoped logger
	toolLogger := logger.WithTool(meta)

	ctx := context.Background()
	toolLogger.Info(ctx, "tool execution started")

	// Output contains tool context
	output := buf.String()
	fmt.Println("Contains tool.name:", bytes.Contains([]byte(output), []byte("tool.name")))
	fmt.Println("Contains tool.namespace:", bytes.Contains([]byte(output), []byte("tool.namespace")))
	// Output:
	// Contains tool.name: true
	// Contains tool.namespace: true
}

func ExampleMiddleware_Wrap() {
	ctx := context.Background()

	// Create observer with disabled exporters for example
	cfg := observe.Config{
		ServiceName: "example",
		Tracing:     observe.TracingConfig{Enabled: true, Exporter: "none"},
		Metrics:     observe.MetricsConfig{Enabled: true, Exporter: "none"},
		Logging:     observe.LoggingConfig{Enabled: false},
	}
	obs, _ := observe.NewObserver(ctx, cfg)
	defer func() {
		_ = obs.Shutdown(ctx)
	}()

	// Create middleware
	mw, _ := observe.MiddlewareFromObserver(obs)

	// Define execution function
	execFn := func(ctx context.Context, tool observe.ToolMeta, input any) (any, error) {
		return map[string]string{"status": "success"}, nil
	}

	// Wrap with observability
	wrapped := mw.Wrap(execFn)

	// Execute - automatically traced, metered, and logged
	result, err := wrapped(ctx, observe.ToolMeta{
		Name:      "example_tool",
		Namespace: "demo",
	}, nil)

	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
	// Output:
	// Result: map[status:success]
}

func ExampleParseLogLevel() {
	levels := []string{"debug", "info", "warn", "error", "unknown"}
	for _, s := range levels {
		level := observe.ParseLogLevel(s)
		fmt.Printf("%s -> %s\n", s, level)
	}
	// Output:
	// debug -> debug
	// info -> info
	// warn -> warn
	// error -> error
	// unknown -> info
}
