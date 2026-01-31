// Package resilience provides resilience patterns for tool execution.
//
// This package implements common resilience patterns that help tools handle
// failures gracefully. The patterns can be composed together to build robust
// execution pipelines.
//
// # Patterns
//
// The package provides the following resilience patterns:
//
//   - Circuit Breaker: Prevents cascading failures by stopping requests to
//     failing services after a threshold is reached.
//
//   - Retry: Automatically retries failed operations with configurable
//     backoff strategies (exponential, linear, constant).
//
//   - Rate Limiter: Controls the rate of operations to prevent overwhelming
//     downstream services.
//
//   - Bulkhead: Limits concurrent operations to prevent resource exhaustion.
//
//   - Timeout: Ensures operations complete within a time limit.
//
// # Usage
//
// Each pattern can be used independently or composed together:
//
//	// Create a circuit breaker
//	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
//	    MaxFailures:  5,
//	    ResetTimeout: time.Minute,
//	})
//
//	// Create a retry policy
//	retry := resilience.NewRetry(resilience.RetryConfig{
//	    MaxAttempts:  3,
//	    InitialDelay: 100 * time.Millisecond,
//	    MaxDelay:     5 * time.Second,
//	    Multiplier:   2.0,
//	})
//
//	// Create a rate limiter
//	rl := resilience.NewRateLimiter(resilience.RateLimiterConfig{
//	    Rate:  100, // requests per second
//	    Burst: 10,
//	})
//
//	// Compose patterns
//	executor := resilience.NewExecutor(
//	    resilience.WithCircuitBreaker(cb),
//	    resilience.WithRetry(retry),
//	    resilience.WithRateLimiter(rl),
//	    resilience.WithTimeout(5*time.Second),
//	)
//
//	err := executor.Execute(ctx, func(ctx context.Context) error {
//	    return callExternalService(ctx)
//	})
package resilience
