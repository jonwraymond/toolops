package resilience

import (
	"context"
	"time"
)

// Executor composes multiple resilience patterns.
type Executor struct {
	circuitBreaker *CircuitBreaker
	retry          *Retry
	rateLimiter    *RateLimiter
	bulkhead       *Bulkhead
	timeout        *Timeout
}

// ExecutorOption configures an Executor.
type ExecutorOption func(*Executor)

// NewExecutor creates a new resilience executor.
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithCircuitBreaker adds a circuit breaker to the executor.
func WithCircuitBreaker(cb *CircuitBreaker) ExecutorOption {
	return func(e *Executor) {
		e.circuitBreaker = cb
	}
}

// WithRetry adds retry logic to the executor.
func WithRetry(r *Retry) ExecutorOption {
	return func(e *Executor) {
		e.retry = r
	}
}

// WithRateLimiter adds rate limiting to the executor.
func WithRateLimiter(rl *RateLimiter) ExecutorOption {
	return func(e *Executor) {
		e.rateLimiter = rl
	}
}

// WithBulkhead adds bulkhead isolation to the executor.
func WithBulkhead(b *Bulkhead) ExecutorOption {
	return func(e *Executor) {
		e.bulkhead = b
	}
}

// WithTimeout adds timeout to the executor.
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = NewTimeout(TimeoutConfig{Timeout: timeout})
	}
}

// WithTimeoutConfig adds timeout with custom config to the executor.
func WithTimeoutConfig(t *Timeout) ExecutorOption {
	return func(e *Executor) {
		e.timeout = t
	}
}

// Execute runs the operation through all configured resilience patterns.
//
// The execution order is:
// 1. Rate Limiter (if configured) - limits request rate
// 2. Bulkhead (if configured) - limits concurrency
// 3. Circuit Breaker (if configured) - prevents cascading failures
// 4. Retry (if configured) - retries on failure
// 5. Timeout (if configured) - limits execution time
func (e *Executor) Execute(ctx context.Context, op func(context.Context) error) error {
	// Build the execution chain from inside out
	execute := op

	// Wrap with timeout (innermost)
	if e.timeout != nil {
		inner := execute
		execute = func(ctx context.Context) error {
			return e.timeout.Execute(ctx, inner)
		}
	}

	// Wrap with retry
	if e.retry != nil {
		inner := execute
		execute = func(ctx context.Context) error {
			return e.retry.Execute(ctx, inner)
		}
	}

	// Wrap with circuit breaker
	if e.circuitBreaker != nil {
		inner := execute
		execute = func(ctx context.Context) error {
			return e.circuitBreaker.Execute(ctx, inner)
		}
	}

	// Wrap with bulkhead
	if e.bulkhead != nil {
		inner := execute
		execute = func(ctx context.Context) error {
			return e.bulkhead.Execute(ctx, inner)
		}
	}

	// Wrap with rate limiter (outermost)
	if e.rateLimiter != nil {
		inner := execute
		execute = func(ctx context.Context) error {
			return e.rateLimiter.Execute(ctx, inner)
		}
	}

	return execute(ctx)
}
