// Package resilience provides resilience patterns for tool execution.
//
// It implements common reliability patterns that help tools handle failures
// gracefully. Patterns can be composed together using the Executor to build
// robust execution pipelines.
//
// # Ecosystem Position
//
// resilience sits between tool invocation and external service calls:
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│                      Tool Execution Flow                        │
//	├─────────────────────────────────────────────────────────────────┤
//	│                                                                 │
//	│   toolexec           resilience              External           │
//	│   ┌──────┐         ┌───────────┐           ┌─────────┐         │
//	│   │ Tool │────────▶│ Executor  │──────────▶│ Service │         │
//	│   │ Call │         │           │           │  (API)  │         │
//	│   └──────┘         │ ┌───────┐ │           └─────────┘         │
//	│                    │ │RateLim│ │                                │
//	│                    │ ├───────┤ │                                │
//	│                    │ │Bulkhd │ │                                │
//	│                    │ ├───────┤ │                                │
//	│                    │ │Circuit│ │                                │
//	│                    │ ├───────┤ │                                │
//	│                    │ │ Retry │ │                                │
//	│                    │ ├───────┤ │                                │
//	│                    │ │Timeout│ │                                │
//	│                    │ └───────┘ │                                │
//	│                    └───────────┘                                │
//	│                                                                 │
//	└─────────────────────────────────────────────────────────────────┘
//
// # Resilience Patterns
//
// The package provides five core patterns:
//
//   - [CircuitBreaker]: Prevents cascading failures by stopping requests to
//     failing services after a threshold is reached. Transitions through
//     Closed → Open → HalfOpen states.
//
//   - [Retry]: Automatically retries failed operations with configurable
//     backoff strategies (exponential, linear, constant) and jitter.
//
//   - [RateLimiter]: Token bucket rate limiting to prevent overwhelming
//     downstream services. Supports burst allowance and wait-on-limit.
//
//   - [Bulkhead]: Semaphore-based concurrency limiting to prevent resource
//     exhaustion and isolate failures.
//
//   - [Timeout]: Context-based timeout to ensure operations complete within
//     a time limit.
//
// # Quick Start
//
//	// Individual pattern usage
//	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
//	    MaxFailures:  5,
//	    ResetTimeout: time.Minute,
//	})
//
//	err := cb.Execute(ctx, func(ctx context.Context) error {
//	    return callExternalService(ctx)
//	})
//
//	// Composed patterns with Executor
//	executor := resilience.NewExecutor(
//	    resilience.WithRateLimiter(resilience.NewRateLimiter(resilience.RateLimiterConfig{
//	        Rate:  100,
//	        Burst: 10,
//	    })),
//	    resilience.WithCircuitBreaker(cb),
//	    resilience.WithRetry(resilience.NewRetry(resilience.RetryConfig{
//	        MaxAttempts:  3,
//	        InitialDelay: 100 * time.Millisecond,
//	    })),
//	    resilience.WithTimeout(5*time.Second),
//	)
//
//	err = executor.Execute(ctx, func(ctx context.Context) error {
//	    return callExternalService(ctx)
//	})
//
// # Execution Order
//
// When using the Executor, patterns are applied in this order (outermost first):
//
//  1. Rate Limiter - limits request rate
//  2. Bulkhead - limits concurrency
//  3. Circuit Breaker - prevents cascading failures
//  4. Retry - retries on failure
//  5. Timeout - limits execution time (innermost)
//
// # Thread Safety
//
// All exported types are safe for concurrent use after construction:
//
//   - [CircuitBreaker]: Execute() and State() are mutex-protected; Reset() is safe
//   - [Retry]: Execute() is stateless and safe for concurrent use
//   - [RateLimiter]: Allow(), AllowN(), Wait(), Execute() are mutex-protected
//   - [Bulkhead]: Acquire(), Release(), Execute() use channel-based semaphore
//   - [Timeout]: Execute() is stateless and safe for concurrent use
//   - [Executor]: Execute() is safe; all wrapped patterns maintain their guarantees
//
// # Error Handling
//
// Each pattern returns specific sentinel errors (use errors.Is for checking):
//
//   - [ErrCircuitOpen]: Circuit breaker is in open state, rejecting requests
//   - [ErrMaxRetriesExceeded]: All retry attempts exhausted
//   - [ErrRateLimitExceeded]: Rate limit exceeded and no wait configured
//   - [ErrBulkheadFull]: Bulkhead at maximum concurrency
//   - [ErrTimeout]: Operation exceeded configured timeout
//
// Example error handling:
//
//	err := executor.Execute(ctx, operation)
//	if errors.Is(err, resilience.ErrCircuitOpen) {
//	    // Service is unhealthy, circuit is protecting downstream
//	    log.Warn("circuit breaker open, using fallback")
//	    return fallbackResult, nil
//	}
//	if errors.Is(err, resilience.ErrRateLimitExceeded) {
//	    // Client should back off
//	    return nil, status.Error(codes.ResourceExhausted, "rate limited")
//	}
//
// # Callbacks and Observability
//
// Patterns support callbacks for observability integration:
//
//   - CircuitBreakerConfig.OnStateChange: Called on state transitions
//   - RetryConfig.OnRetry: Called before each retry attempt
//   - CircuitBreakerConfig.IsFailure: Custom failure classification
//   - RetryConfig.RetryIf: Custom retry decision logic
//
// # Integration with ApertureStack
//
// resilience integrates with other ApertureStack packages:
//
//   - toolexec: Wrap tool execution with resilience patterns
//   - observe: Connect callbacks to observability middleware
//   - health: Use CircuitBreaker.State() for health checks
package resilience
