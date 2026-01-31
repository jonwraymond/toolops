package observe

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestLogger_IncludesToolFields verifies tool fields are present in log output.
func TestLogger_IncludesToolFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	meta := ToolMeta{
		Namespace: "github",
		Name:      "create_issue",
	}

	toolLogger := logger.WithTool(meta)
	toolLogger.Info(context.Background(), "test message")

	output := buf.String()

	// Parse JSON output
	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v\nOutput: %s", err, output)
	}

	// Verify tool fields
	if v, ok := logEntry["tool.id"].(string); !ok || v != "github.create_issue" {
		t.Errorf("expected tool.id='github.create_issue', got %v", logEntry["tool.id"])
	}
	if v, ok := logEntry["tool.namespace"].(string); !ok || v != "github" {
		t.Errorf("expected tool.namespace='github', got %v", logEntry["tool.namespace"])
	}
	if v, ok := logEntry["tool.name"].(string); !ok || v != "create_issue" {
		t.Errorf("expected tool.name='create_issue', got %v", logEntry["tool.name"])
	}
}

// TestLogger_IncludesDuration verifies duration_ms field is present.
func TestLogger_IncludesDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	meta := ToolMeta{Name: "test_tool"}
	toolLogger := logger.WithTool(meta)

	toolLogger.Info(context.Background(), "test message",
		Field{Key: "duration_ms", Value: 50.5},
	)

	output := buf.String()

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	if v, ok := logEntry["duration_ms"].(float64); !ok || v != 50.5 {
		t.Errorf("expected duration_ms=50.5, got %v", logEntry["duration_ms"])
	}
}

// TestLogger_ErrorLevel verifies error log level and error field.
func TestLogger_ErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	meta := ToolMeta{Name: "error_tool"}
	toolLogger := logger.WithTool(meta)

	toolLogger.Error(context.Background(), "execution failed",
		Field{Key: "error", Value: "connection timeout"},
	)

	output := buf.String()

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	// Verify level
	if v, ok := logEntry["level"].(string); !ok || v != "error" {
		t.Errorf("expected level='error', got %v", logEntry["level"])
	}

	// Verify error field
	if v, ok := logEntry["error"].(string); !ok || v != "connection timeout" {
		t.Errorf("expected error='connection timeout', got %v", logEntry["error"])
	}
}

// TestLogger_InfoLevel verifies info log level.
func TestLogger_InfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	meta := ToolMeta{Name: "info_tool"}
	toolLogger := logger.WithTool(meta)

	toolLogger.Info(context.Background(), "operation complete")

	output := buf.String()

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	if v, ok := logEntry["level"].(string); !ok || v != "info" {
		t.Errorf("expected level='info', got %v", logEntry["level"])
	}
}

// TestLogger_InputsRedactedByDefault verifies inputs are not logged.
func TestLogger_InputsRedactedByDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	meta := ToolMeta{Name: "sensitive_tool"}
	toolLogger := logger.WithTool(meta)

	// Simulate logging with an "input" field that should be redacted
	toolLogger.Info(context.Background(), "tool executed",
		Field{Key: "input", Value: "secret_password_123"},
	)

	output := buf.String()

	// The raw input value should NOT appear
	if strings.Contains(output, "secret_password_123") {
		t.Error("raw input should be redacted, but found in output")
	}

	// Should contain redacted marker
	if !strings.Contains(output, "[REDACTED]") && !strings.Contains(output, "[redacted]") {
		// If no redacted marker, verify input field is simply not present
		var logEntry map[string]any
		if err := json.Unmarshal([]byte(output), &logEntry); err == nil {
			if _, ok := logEntry["input"]; ok {
				if v, ok := logEntry["input"].(string); ok && v == "secret_password_123" {
					t.Error("raw input should be redacted")
				}
			}
		}
	}
}

// TestLogger_LevelFiltering verifies log level filtering.
func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("warn", &buf)

	meta := ToolMeta{Name: "filtered_tool"}
	toolLogger := logger.WithTool(meta)

	// Info should be filtered out
	toolLogger.Info(context.Background(), "info message")

	output := buf.String()
	if strings.Contains(output, "info message") {
		t.Error("info message should be filtered when level is warn")
	}

	// Warn should pass through
	toolLogger.Warn(context.Background(), "warn message")

	output = buf.String()
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should pass through when level is warn")
	}
}

// TestLogger_DebugLevel verifies debug level filtering.
func TestLogger_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("debug", &buf)

	meta := ToolMeta{Name: "debug_tool"}
	toolLogger := logger.WithTool(meta)

	toolLogger.Debug(context.Background(), "debug message")

	output := buf.String()

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	if v, ok := logEntry["level"].(string); !ok || v != "debug" {
		t.Errorf("expected level='debug', got %v", logEntry["level"])
	}
}

// TestLogger_WarnLevel verifies warn level.
func TestLogger_WarnLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	meta := ToolMeta{Name: "warn_tool"}
	toolLogger := logger.WithTool(meta)

	toolLogger.Warn(context.Background(), "warning message")

	output := buf.String()

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	if v, ok := logEntry["level"].(string); !ok || v != "warn" {
		t.Errorf("expected level='warn', got %v", logEntry["level"])
	}
}

// TestLogger_VersionIncluded verifies version is included when set.
func TestLogger_VersionIncluded(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	meta := ToolMeta{
		Name:    "versioned_tool",
		Version: "2.0.0",
	}
	toolLogger := logger.WithTool(meta)

	toolLogger.Info(context.Background(), "test")

	output := buf.String()

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	if v, ok := logEntry["tool.version"].(string); !ok || v != "2.0.0" {
		t.Errorf("expected tool.version='2.0.0', got %v", logEntry["tool.version"])
	}
}
