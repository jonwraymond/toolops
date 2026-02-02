package resilience_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolops/resilience"
)

func ExampleNewCircuitBreaker() {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		MaxFailures:  3,
		ResetTimeout: time.Second,
	})

	ctx := context.Background()
	err := cb.Execute(ctx, func(ctx context.Context) error {
		// Simulated successful operation
		return nil
	})

	if err == nil {
		fmt.Println("Operation succeeded")
	}
	// Output:
	// Operation succeeded
}

func ExampleCircuitBreaker_State() {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: time.Minute,
	})

	ctx := context.Background()

	// Initial state is closed
	fmt.Println("Initial state:", cb.State())

	// Cause failures to open the circuit
	simulatedErr := errors.New("service unavailable")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return simulatedErr
		})
	}

	fmt.Println("After failures:", cb.State())

	// Reset the circuit
	cb.Reset()
	fmt.Println("After reset:", cb.State())
	// Output:
	// Initial state: closed
	// After failures: open
	// After reset: closed
}

func ExampleNewCircuitBreaker_withStateChange() {
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: time.Minute,
		OnStateChange: func(from, to resilience.State) {
			fmt.Printf("Circuit changed: %s -> %s\n", from, to)
		},
	})

	ctx := context.Background()
	simulatedErr := errors.New("failure")

	// Trigger circuit open
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return simulatedErr
	})
	// Output:
	// Circuit changed: closed -> open
}

func ExampleNewRetry() {
	retry := resilience.NewRetry(resilience.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Strategy:     resilience.BackoffExponential,
		Jitter:       false, // Disabled for predictable example
	})

	ctx := context.Background()
	attempts := 0

	err := retry.Execute(ctx, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil // Success on third attempt
	})

	if err == nil {
		fmt.Printf("Succeeded after %d attempts\n", attempts)
	}
	// Output:
	// Succeeded after 3 attempts
}

func ExampleNewRetry_withCallback() {
	retry := resilience.NewRetry(resilience.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		Jitter:       false,
		OnRetry: func(attempt int, err error, delay time.Duration) {
			fmt.Printf("Attempt %d failed, retrying\n", attempt)
		},
	})

	ctx := context.Background()
	attempts := 0

	_ = retry.Execute(ctx, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary")
		}
		return nil
	})

	fmt.Println("Completed")
	// Output:
	// Attempt 1 failed, retrying
	// Attempt 2 failed, retrying
	// Completed
}

func ExampleNewRateLimiter() {
	rl := resilience.NewRateLimiter(resilience.RateLimiterConfig{
		Rate:  100, // 100 requests per second
		Burst: 5,   // Allow burst of 5
	})

	// Check if request is allowed
	if rl.Allow() {
		fmt.Println("Request 1 allowed")
	}

	// AllowN for batch operations
	if rl.AllowN(3) {
		fmt.Println("Batch of 3 allowed")
	}
	// Output:
	// Request 1 allowed
	// Batch of 3 allowed
}

func ExampleRateLimiter_Execute() {
	rl := resilience.NewRateLimiter(resilience.RateLimiterConfig{
		Rate:        10,
		Burst:       2,
		WaitOnLimit: false,
	})

	ctx := context.Background()
	successCount := 0

	// Execute multiple operations
	for i := 0; i < 3; i++ {
		err := rl.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		if err == nil {
			successCount++
		}
	}

	fmt.Printf("Successful executions: %d\n", successCount)
	// Output:
	// Successful executions: 2
}

func ExampleNewBulkhead() {
	bh := resilience.NewBulkhead(resilience.BulkheadConfig{
		MaxConcurrent: 2,
		MaxWait:       0, // No waiting
	})

	ctx := context.Background()

	// Acquire slots
	err1 := bh.Acquire(ctx)
	err2 := bh.Acquire(ctx)
	err3 := bh.Acquire(ctx) // Should fail

	fmt.Println("Slot 1:", err1 == nil)
	fmt.Println("Slot 2:", err2 == nil)
	fmt.Println("Slot 3:", errors.Is(err3, resilience.ErrBulkheadFull))

	// Release a slot
	bh.Release()

	// Now we can acquire again
	err4 := bh.Acquire(ctx)
	fmt.Println("Slot 4 after release:", err4 == nil)
	// Output:
	// Slot 1: true
	// Slot 2: true
	// Slot 3: true
	// Slot 4 after release: true
}

func ExampleBulkhead_Metrics() {
	bh := resilience.NewBulkhead(resilience.BulkheadConfig{
		MaxConcurrent: 5,
	})

	ctx := context.Background()

	// Acquire some slots
	_ = bh.Acquire(ctx)
	_ = bh.Acquire(ctx)

	metrics := bh.Metrics()
	fmt.Printf("Active: %d, Available: %d, MaxConcurrent: %d\n",
		metrics.Active, metrics.Available, metrics.MaxConcurrent)
	// Output:
	// Active: 2, Available: 3, MaxConcurrent: 5
}

func ExampleNewTimeout() {
	timeout := resilience.NewTimeout(resilience.TimeoutConfig{
		Timeout: 100 * time.Millisecond,
	})

	ctx := context.Background()

	// Fast operation succeeds
	err := timeout.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	fmt.Println("Fast operation error:", err)

	// Slow operation times out
	err = timeout.Execute(ctx, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	fmt.Println("Slow operation timed out:", errors.Is(err, resilience.ErrTimeout))
	// Output:
	// Fast operation error: <nil>
	// Slow operation timed out: true
}

func ExampleExecuteWithTimeout() {
	ctx := context.Background()

	err := resilience.ExecuteWithTimeout(ctx, 50*time.Millisecond, func(ctx context.Context) error {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	fmt.Println("Completed without timeout:", err == nil)
	// Output:
	// Completed without timeout: true
}

func ExampleNewExecutor() {
	// Create individual patterns
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: time.Minute,
	})

	retry := resilience.NewRetry(resilience.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		Jitter:       false,
	})

	rl := resilience.NewRateLimiter(resilience.RateLimiterConfig{
		Rate:  100,
		Burst: 10,
	})

	// Compose into an executor
	executor := resilience.NewExecutor(
		resilience.WithRateLimiter(rl),
		resilience.WithCircuitBreaker(cb),
		resilience.WithRetry(retry),
		resilience.WithTimeout(time.Second),
	)

	ctx := context.Background()
	err := executor.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	fmt.Println("Executor succeeded:", err == nil)
	// Output:
	// Executor succeeded: true
}

func ExampleExecutor_withBulkhead() {
	bh := resilience.NewBulkhead(resilience.BulkheadConfig{
		MaxConcurrent: 10,
	})

	executor := resilience.NewExecutor(
		resilience.WithBulkhead(bh),
		resilience.WithTimeout(time.Second),
	)

	ctx := context.Background()
	err := executor.Execute(ctx, func(ctx context.Context) error {
		// Operation protected by bulkhead and timeout
		return nil
	})

	fmt.Println("Bulkhead executor succeeded:", err == nil)
	// Output:
	// Bulkhead executor succeeded: true
}
