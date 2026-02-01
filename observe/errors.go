package observe

import "errors"

// Configuration errors.
var (
	// ErrMissingServiceName indicates Config.ServiceName is empty.
	ErrMissingServiceName = errors.New("observe: service name is required")

	// ErrInvalidSamplePct indicates Tracing.SamplePct is not in [0.0, 1.0].
	ErrInvalidSamplePct = errors.New("observe: sample percentage must be between 0.0 and 1.0")

	// ErrInvalidTracingExporter indicates an unknown tracing exporter name.
	ErrInvalidTracingExporter = errors.New("observe: invalid tracing exporter")

	// ErrInvalidMetricsExporter indicates an unknown metrics exporter name.
	ErrInvalidMetricsExporter = errors.New("observe: invalid metrics exporter")

	// ErrInvalidLogLevel indicates an unknown log level.
	ErrInvalidLogLevel = errors.New("observe: invalid log level")
)

// Runtime errors.
var (
	// ErrNilObserver indicates a nil Observer was provided.
	ErrNilObserver = errors.New("observe: observer is nil")

	// ErrMissingToolName indicates ToolMeta.Name is empty.
	ErrMissingToolName = errors.New("observe: tool name is required")
)

// Exporter errors.
var (
	// ErrEndpointNotConfigured indicates a required endpoint environment variable is not set.
	ErrEndpointNotConfigured = errors.New("observe: endpoint not configured")
)

// Validation constants.
const (
	// MinSamplePct is the minimum valid sampling percentage.
	MinSamplePct = 0.0
	// MaxSamplePct is the maximum valid sampling percentage.
	MaxSamplePct = 1.0
)

// ValidTracingExporters lists valid tracing exporter names.
var ValidTracingExporters = []string{"otlp", "jaeger", "stdout", "none", ""}

// ValidMetricsExporters lists valid metrics exporter names.
var ValidMetricsExporters = []string{"otlp", "prometheus", "stdout", "none", ""}

// ValidLogLevels lists valid log level names.
var ValidLogLevels = []string{"debug", "info", "warn", "error", ""}

// RedactedFields lists field keys that are automatically redacted in logs.
// These fields may contain sensitive information like credentials or secrets.
var RedactedFields = []string{
	"input",
	"inputs",
	"password",
	"secret",
	"token",
	"api_key",
	"apiKey",
	"credential",
}
