package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

// BenchmarkCircuitBreaker_Execute_Closed measures happy path execution.
func BenchmarkCircuitBreaker_Execute_Closed(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  100,
		ResetTimeout: time.Minute,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

// BenchmarkCircuitBreaker_StateCheck measures state inspection overhead.
func BenchmarkCircuitBreaker_StateCheck(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: time.Minute,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.State()
	}
}

// BenchmarkCircuitBreaker_Metrics measures metrics retrieval.
func BenchmarkCircuitBreaker_Metrics(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: time.Minute,
	})
	ctx := context.Background()

	// Generate some activity
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Metrics()
	}
}

// BenchmarkCircuitBreaker_Concurrent measures parallel execution.
func BenchmarkCircuitBreaker_Concurrent(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  1000,
		ResetTimeout: time.Minute,
	})
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
		}
	})
}

// BenchmarkRetry_NoRetries measures retry with immediate success.
func BenchmarkRetry_NoRetries(b *testing.B) {
	retry := NewRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retry.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

// BenchmarkRetry_Config measures config retrieval.
func BenchmarkRetry_Config(b *testing.B) {
	retry := NewRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retry.Config()
	}
}

// BenchmarkRateLimiter_Allow measures single token check.
func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  1000000, // Very high rate to avoid blocking
		Burst: 1000000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.Allow()
	}
}

// BenchmarkRateLimiter_AllowN measures batch token check.
func BenchmarkRateLimiter_AllowN(b *testing.B) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  1000000,
		Burst: 1000000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.AllowN(10)
	}
}

// BenchmarkRateLimiter_Tokens measures token count retrieval.
func BenchmarkRateLimiter_Tokens(b *testing.B) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  100,
		Burst: 10,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.Tokens()
	}
}

// BenchmarkRateLimiter_Concurrent measures parallel token checks.
func BenchmarkRateLimiter_Concurrent(b *testing.B) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  1000000,
		Burst: 1000000,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rl.Allow()
		}
	})
}

// BenchmarkBulkhead_Execute measures semaphore acquire/release.
func BenchmarkBulkhead_Execute(b *testing.B) {
	bh := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1000,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bh.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

// BenchmarkBulkhead_AcquireRelease measures acquire/release pair.
func BenchmarkBulkhead_AcquireRelease(b *testing.B) {
	bh := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1000,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bh.Acquire(ctx)
		bh.Release()
	}
}

// BenchmarkBulkhead_Metrics measures metrics retrieval.
func BenchmarkBulkhead_Metrics(b *testing.B) {
	bh := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 10,
	})
	ctx := context.Background()

	// Acquire some slots
	_ = bh.Acquire(ctx)
	_ = bh.Acquire(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bh.Metrics()
	}
}

// BenchmarkBulkhead_Concurrent measures parallel semaphore operations.
func BenchmarkBulkhead_Concurrent(b *testing.B) {
	bh := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 100,
	})
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bh.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
		}
	})
}

// BenchmarkTimeout_Execute_Fast measures fast execution path.
func BenchmarkTimeout_Execute_Fast(b *testing.B) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: time.Second,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timeout.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

// BenchmarkTimeout_Config measures config retrieval.
func BenchmarkTimeout_Config(b *testing.B) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: time.Second,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timeout.Config()
	}
}

// BenchmarkExecutor_SinglePattern measures executor with one pattern.
func BenchmarkExecutor_SinglePattern(b *testing.B) {
	executor := NewExecutor(
		WithTimeout(time.Second),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

// BenchmarkExecutor_AllPatterns measures executor with all patterns.
func BenchmarkExecutor_AllPatterns(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  100,
		ResetTimeout: time.Minute,
	})
	retry := NewRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
	})
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  1000000,
		Burst: 1000000,
	})
	bh := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1000,
	})

	executor := NewExecutor(
		WithRateLimiter(rl),
		WithBulkhead(bh),
		WithCircuitBreaker(cb),
		WithRetry(retry),
		WithTimeout(time.Second),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

// BenchmarkExecutor_Concurrent measures parallel executor usage.
func BenchmarkExecutor_Concurrent(b *testing.B) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  10000,
		ResetTimeout: time.Minute,
	})
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  1000000,
		Burst: 1000000,
	})

	executor := NewExecutor(
		WithRateLimiter(rl),
		WithCircuitBreaker(cb),
		WithTimeout(time.Second),
	)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = executor.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
		}
	})
}

// BenchmarkState_String measures state string conversion.
func BenchmarkState_String(b *testing.B) {
	states := []State{StateClosed, StateOpen, StateHalfOpen}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = states[i%3].String()
	}
}

// BenchmarkErrorIs measures error checking with errors.Is.
func BenchmarkErrorIs(b *testing.B) {
	err := ErrCircuitOpen

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, ErrCircuitOpen)
	}
}
