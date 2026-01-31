package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewRetry(t *testing.T) {
	r := NewRetry(RetryConfig{})

	if r.config.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", r.config.MaxAttempts)
	}
	if r.config.InitialDelay != 100*time.Millisecond {
		t.Errorf("InitialDelay = %v, want 100ms", r.config.InitialDelay)
	}
	if r.config.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", r.config.MaxDelay)
	}
	if r.config.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", r.config.Multiplier)
	}
}

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	r := NewRetry(RetryConfig{MaxAttempts: 3})

	attempts := 0
	err := r.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetry_SuccessOnRetry(t *testing.T) {
	r := NewRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		Jitter:       false,
	})

	attempts := 0
	testErr := errors.New("test error")

	err := r.Execute(context.Background(), func(ctx context.Context) error {
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

func TestRetry_ExhaustedAttempts(t *testing.T) {
	r := NewRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		Jitter:       false,
	})

	attempts := 0
	testErr := errors.New("persistent error")

	err := r.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		return testErr
	})

	if err != testErr {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	r := NewRetry(RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	testErr := errors.New("test error")

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := r.Execute(ctx, func(ctx context.Context) error {
		attempts++
		return testErr
	})

	if err != context.Canceled {
		t.Errorf("Execute() error = %v, want context.Canceled", err)
	}
}

func TestRetry_RetryIf(t *testing.T) {
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")

	r := NewRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		RetryIf: func(err error) bool {
			return err == retryableErr
		},
	})

	t.Run("retryable error", func(t *testing.T) {
		attempts := 0
		err := r.Execute(context.Background(), func(ctx context.Context) error {
			attempts++
			return retryableErr
		})

		if err != retryableErr {
			t.Errorf("Execute() error = %v, want %v", err, retryableErr)
		}
		if attempts != 3 {
			t.Errorf("attempts = %d, want 3", attempts)
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		attempts := 0
		err := r.Execute(context.Background(), func(ctx context.Context) error {
			attempts++
			return nonRetryableErr
		})

		if err != nonRetryableErr {
			t.Errorf("Execute() error = %v, want %v", err, nonRetryableErr)
		}
		if attempts != 1 {
			t.Errorf("attempts = %d, want 1", attempts)
		}
	})
}

func TestRetry_OnRetry(t *testing.T) {
	var callbacks []struct {
		attempt int
		delay   time.Duration
	}

	r := NewRetry(RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		Jitter:       false,
		OnRetry: func(attempt int, err error, delay time.Duration) {
			callbacks = append(callbacks, struct {
				attempt int
				delay   time.Duration
			}{attempt, delay})
		},
	})

	testErr := errors.New("test error")
	_ = r.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	if len(callbacks) != 2 {
		t.Errorf("callbacks = %d, want 2", len(callbacks))
	}
	if callbacks[0].attempt != 1 {
		t.Errorf("First callback attempt = %d, want 1", callbacks[0].attempt)
	}
}

func TestRetry_BackoffStrategies(t *testing.T) {
	t.Run("exponential", func(t *testing.T) {
		r := NewRetry(RetryConfig{
			MaxAttempts:  4,
			InitialDelay: 10 * time.Millisecond,
			Multiplier:   2.0,
			Strategy:     BackoffExponential,
			Jitter:       false,
		})

		// Delay for attempt 3 should be 10ms * 2^2 = 40ms
		delay := r.calculateDelay(3)
		if delay != 40*time.Millisecond {
			t.Errorf("Exponential delay for attempt 3 = %v, want 40ms", delay)
		}
	})

	t.Run("linear", func(t *testing.T) {
		r := NewRetry(RetryConfig{
			MaxAttempts:  4,
			InitialDelay: 10 * time.Millisecond,
			Strategy:     BackoffLinear,
			Jitter:       false,
		})

		// Delay for attempt 3 should be 10ms * 3 = 30ms
		delay := r.calculateDelay(3)
		if delay != 30*time.Millisecond {
			t.Errorf("Linear delay for attempt 3 = %v, want 30ms", delay)
		}
	})

	t.Run("constant", func(t *testing.T) {
		r := NewRetry(RetryConfig{
			MaxAttempts:  4,
			InitialDelay: 10 * time.Millisecond,
			Strategy:     BackoffConstant,
			Jitter:       false,
		})

		// Delay should always be 10ms
		delay := r.calculateDelay(3)
		if delay != 10*time.Millisecond {
			t.Errorf("Constant delay for attempt 3 = %v, want 10ms", delay)
		}
	})

	t.Run("max delay cap", func(t *testing.T) {
		r := NewRetry(RetryConfig{
			MaxAttempts:  10,
			InitialDelay: 1 * time.Second,
			MaxDelay:     5 * time.Second,
			Multiplier:   10.0,
			Strategy:     BackoffExponential,
			Jitter:       false,
		})

		// Delay should be capped at 5s
		delay := r.calculateDelay(5)
		if delay != 5*time.Second {
			t.Errorf("Capped delay = %v, want 5s", delay)
		}
	})
}

func TestRetry_Config(t *testing.T) {
	r := NewRetry(RetryConfig{
		MaxAttempts: 5,
	})

	config := r.Config()
	if config.MaxAttempts != 5 {
		t.Errorf("Config().MaxAttempts = %d, want 5", config.MaxAttempts)
	}
}
