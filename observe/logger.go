package observe

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// Logger is a minimal structured logging interface.
// Defined in observe.go, re-exported here for documentation.
// type Logger interface {
// 	Info(ctx context.Context, msg string, fields ...Field)
// 	Warn(ctx context.Context, msg string, fields ...Field)
// 	Error(ctx context.Context, msg string, fields ...Field)
// 	Debug(ctx context.Context, msg string, fields ...Field)
// }

// LogLevel represents a logging level.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// ParseLogLevel parses a string log level.
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}

// structuredLogger is a JSON structured logger implementation.
type structuredLogger struct {
	level     LogLevel
	writer    io.Writer
	mu        sync.Mutex
	toolMeta  *ToolMeta
	baseAttrs map[string]any
}

// NewLogger creates a new structured logger with the given level.
func NewLogger(level string) Logger {
	return NewLoggerWithWriter(level, os.Stderr)
}

// NewLoggerWithWriter creates a new structured logger with a custom writer.
func NewLoggerWithWriter(level string, w io.Writer) Logger {
	return &structuredLogger{
		level:     ParseLogLevel(level),
		writer:    w,
		baseAttrs: make(map[string]any),
	}
}

// WithTool returns a logger with tool context attached.
func (l *structuredLogger) WithTool(meta ToolMeta) Logger {
	attrs := make(map[string]any, len(l.baseAttrs)+4)
	for k, v := range l.baseAttrs {
		attrs[k] = v
	}

	attrs["tool.id"] = meta.ToolID()
	attrs["tool.name"] = meta.Name
	if meta.Namespace != "" {
		attrs["tool.namespace"] = meta.Namespace
	}
	if meta.Version != "" {
		attrs["tool.version"] = meta.Version
	}

	return &structuredLogger{
		level:     l.level,
		writer:    l.writer,
		toolMeta:  &meta,
		baseAttrs: attrs,
	}
}

func (l *structuredLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelInfo, msg, fields)
}

func (l *structuredLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelWarn, msg, fields)
}

func (l *structuredLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelError, msg, fields)
}

func (l *structuredLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelDebug, msg, fields)
}

func (l *structuredLogger) log(ctx context.Context, level LogLevel, msg string, fields []Field) {
	// Filter by level
	if level < l.level {
		return
	}

	// Build log entry
	entry := make(map[string]any, len(l.baseAttrs)+len(fields)+3)

	// Add timestamp and level
	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level.String()
	entry["msg"] = msg

	// Add base attributes (tool context)
	for k, v := range l.baseAttrs {
		entry[k] = v
	}

	// Add fields (with input redaction)
	for _, f := range fields {
		if isRedactedField(f.Key) {
			entry[f.Key] = "[REDACTED]"
		} else {
			entry[f.Key] = f.Value
		}
	}

	// Serialize and write
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return // Silently drop malformed log entries
	}

	l.writer.Write(data)
	l.writer.Write([]byte("\n"))
}

// isRedactedField returns true if the field should be redacted.
func isRedactedField(key string) bool {
	redactedKeys := map[string]bool{
		"input":      true,
		"inputs":     true,
		"password":   true,
		"secret":     true,
		"token":      true,
		"api_key":    true,
		"apiKey":     true,
		"credential": true,
	}
	return redactedKeys[key]
}

// ExtendedLogger extends Logger with WithTool for creating tool-scoped loggers.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Ownership: WithTool returns a logger bound to ToolMeta; returned logger may share state.
type ExtendedLogger interface {
	Logger
	WithTool(meta ToolMeta) Logger
}

// Ensure structuredLogger implements ExtendedLogger
var _ ExtendedLogger = (*structuredLogger)(nil)
