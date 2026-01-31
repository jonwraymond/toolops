package health

import (
	"context"
	"time"
)

// Status represents the health status of a component.
type Status int

const (
	// StatusHealthy indicates the component is functioning normally.
	StatusHealthy Status = iota
	// StatusDegraded indicates the component is functioning but with issues.
	StatusDegraded
	// StatusUnhealthy indicates the component is not functioning properly.
	StatusUnhealthy
)

// String returns the string representation of the status.
func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusDegraded:
		return "degraded"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// Result contains the outcome of a health check.
type Result struct {
	// Status is the health status.
	Status Status

	// Message provides additional context about the status.
	Message string

	// Details contains arbitrary metadata about the check.
	Details map[string]any

	// Duration is how long the check took.
	Duration time.Duration

	// Timestamp is when the check was performed.
	Timestamp time.Time

	// Error is the error if the check failed.
	Error error
}

// Healthy creates a healthy result.
func Healthy(message string) Result {
	return Result{
		Status:    StatusHealthy,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// Degraded creates a degraded result.
func Degraded(message string) Result {
	return Result{
		Status:    StatusDegraded,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// Unhealthy creates an unhealthy result.
func Unhealthy(message string, err error) Result {
	return Result{
		Status:    StatusUnhealthy,
		Message:   message,
		Error:     err,
		Timestamp: time.Now(),
	}
}

// WithDetails adds details to a result.
func (r Result) WithDetails(details map[string]any) Result {
	r.Details = details
	return r
}

// WithDuration sets the duration on a result.
func (r Result) WithDuration(d time.Duration) Result {
	r.Duration = d
	return r
}

// Checker is the interface for health checks.
type Checker interface {
	// Name returns the name of this checker.
	Name() string

	// Check performs the health check and returns the result.
	Check(ctx context.Context) Result
}

// CheckerFunc is an adapter to allow ordinary functions to be used as Checkers.
type CheckerFunc struct {
	name string
	fn   func(context.Context) Result
}

// NewCheckerFunc creates a new CheckerFunc.
func NewCheckerFunc(name string, fn func(context.Context) Result) *CheckerFunc {
	return &CheckerFunc{name: name, fn: fn}
}

// Name returns the name of this checker.
func (f *CheckerFunc) Name() string {
	return f.name
}

// Check performs the health check.
func (f *CheckerFunc) Check(ctx context.Context) Result {
	return f.fn(ctx)
}

// PingChecker is a simple checker that can be pinged.
type PingChecker interface {
	Checker

	// Ping checks if the component is reachable.
	Ping(ctx context.Context) error
}

// InfoChecker is a checker that can provide detailed information.
type InfoChecker interface {
	Checker

	// Info returns detailed information about the component.
	Info(ctx context.Context) (map[string]any, error)
}
