package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewExecutor(t *testing.T) {
	e := NewExecutor()

	if e.circuitBreaker != nil {
		t.Error("Default executor should not have circuit breaker")
	}
	if e.retry != nil {
		t.Error("Default executor should not have retry")
	}
	if e.rateLimiter != nil {
		t.Error("Default executor should not have rate limiter")
	}
	if e.bulkhead != nil {
		t.Error("Default executor should not have bulkhead")
	}
	if e.timeout != nil {
		t.Error("Default executor should not have timeout")
	}
}

func TestExecutor_WithOptions(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})
	retry := NewRetry(RetryConfig{})
	rl := NewRateLimiter(RateLimiterConfig{})
	b := NewBulkhead(BulkheadConfig{})

	e := NewExecutor(
		WithCircuitBreaker(cb),
		WithRetry(retry),
		WithRateLimiter(rl),
		WithBulkhead(b),
		WithTimeout(time.Second),
	)

	if e.circuitBreaker != cb {
		t.Error("CircuitBreaker not set")
	}
	if e.retry != retry {
		t.Error("Retry not set")
	}
	if e.rateLimiter != rl {
		t.Error("RateLimiter not set")
	}
	if e.bulkhead != b {
		t.Error("Bulkhead not set")
	}
	if e.timeout == nil {
		t.Error("Timeout not set")
	}
}

func TestExecutor_ExecuteNoPatterns(t *testing.T) {
	e := NewExecutor()

	executed := false
	err := e.Execute(context.Background(), func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if !executed {
		t.Error("Operation was not executed")
	}
}

func TestExecutor_ExecuteWithTimeout(t *testing.T) {
	e := NewExecutor(
		WithTimeout(20 * time.Millisecond),
	)

	t.Run("completes in time", func(t *testing.T) {
		err := e.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
	})

	t.Run("times out", func(t *testing.T) {
		err := e.Execute(context.Background(), func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		if err != ErrTimeout {
			t.Errorf("Execute() error = %v, want ErrTimeout", err)
		}
	})
}

func TestExecutor_ExecuteWithRetry(t *testing.T) {
	e := NewExecutor(
		WithRetry(NewRetry(RetryConfig{
			MaxAttempts:  3,
			InitialDelay: time.Millisecond,
			Jitter:       false,
		})),
	)

	attempts := 0
	testErr := errors.New("transient error")

	err := e.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return testErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestExecutor_ExecuteWithCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: time.Hour,
	})

	e := NewExecutor(
		WithCircuitBreaker(cb),
	)

	testErr := errors.New("test error")

	// Trigger circuit breaker
	for i := 0; i < 2; i++ {
		_ = e.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	// Should be blocked
	err := e.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Execute() error = %v, want ErrCircuitOpen", err)
	}
}

func TestExecutor_ExecuteWithRateLimiter(t *testing.T) {
	e := NewExecutor(
		WithRateLimiter(NewRateLimiter(RateLimiterConfig{
			Rate:  10,
			Burst: 1,
		})),
	)

	// First should succeed
	err := e.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("First Execute() error = %v", err)
	}

	// Second should be rate limited
	err = e.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != ErrRateLimitExceeded {
		t.Errorf("Second Execute() error = %v, want ErrRateLimitExceeded", err)
	}
}

func TestExecutor_ExecuteWithBulkhead(t *testing.T) {
	e := NewExecutor(
		WithBulkhead(NewBulkhead(BulkheadConfig{
			MaxConcurrent: 1,
		})),
	)

	// Acquire slot via direct bulkhead access
	done := make(chan struct{})
	started := make(chan struct{})

	go func() {
		_ = e.Execute(context.Background(), func(ctx context.Context) error {
			close(started)
			<-done
			return nil
		})
	}()

	<-started

	// Should be blocked
	err := e.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	close(done)

	if err != ErrBulkheadFull {
		t.Errorf("Execute() error = %v, want ErrBulkheadFull", err)
	}
}

func TestExecutor_ComposedPatterns(t *testing.T) {
	attempts := 0

	e := NewExecutor(
		WithRateLimiter(NewRateLimiter(RateLimiterConfig{
			Rate:  1000,
			Burst: 10,
		})),
		WithBulkhead(NewBulkhead(BulkheadConfig{
			MaxConcurrent: 10,
		})),
		WithCircuitBreaker(NewCircuitBreaker(CircuitBreakerConfig{
			MaxFailures: 10,
		})),
		WithRetry(NewRetry(RetryConfig{
			MaxAttempts:  3,
			InitialDelay: time.Millisecond,
			Jitter:       false,
		})),
		WithTimeout(time.Second),
	)

	testErr := errors.New("transient error")

	// Should retry and eventually succeed
	err := e.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return testErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestWithTimeoutConfig(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{Timeout: 5 * time.Second})
	e := NewExecutor(WithTimeoutConfig(timeout))

	if e.timeout != timeout {
		t.Error("Timeout not set correctly with WithTimeoutConfig")
	}
}
