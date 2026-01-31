package resilience

import "errors"

// Sentinel errors for resilience operations.
var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("resilience: circuit breaker is open")

	// ErrMaxRetriesExceeded is returned when max retry attempts are exhausted.
	ErrMaxRetriesExceeded = errors.New("resilience: max retries exceeded")

	// ErrRateLimitExceeded is returned when the rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("resilience: rate limit exceeded")

	// ErrBulkheadFull is returned when the bulkhead is at capacity.
	ErrBulkheadFull = errors.New("resilience: bulkhead at capacity")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("resilience: operation timed out")
)
