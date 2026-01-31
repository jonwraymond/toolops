package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewTimeout(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{})

	if timeout.config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", timeout.config.Timeout)
	}
}

func TestTimeout_ExecuteSuccess(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: time.Second,
	})

	executed := false
	err := timeout.Execute(context.Background(), func(ctx context.Context) error {
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

func TestTimeout_ExecuteError(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: time.Second,
	})

	testErr := errors.New("test error")
	err := timeout.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}
}

func TestTimeout_ExecuteTimeout(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: 10 * time.Millisecond,
	})

	err := timeout.Execute(context.Background(), func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	if err != ErrTimeout {
		t.Errorf("Execute() error = %v, want ErrTimeout", err)
	}
}

func TestTimeout_ExecuteContextCancelled(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())

	err := timeout.Execute(ctx, func(ctx context.Context) error {
		cancel()
		<-ctx.Done()
		return ctx.Err()
	})

	if err != context.Canceled {
		t.Errorf("Execute() error = %v, want context.Canceled", err)
	}
}

func TestTimeout_OperationRespectsCancelledContext(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: 50 * time.Millisecond,
	})

	ctxDoneCh := make(chan bool, 1)
	err := timeout.Execute(context.Background(), func(ctx context.Context) error {
		// Wait for context cancellation
		select {
		case <-ctx.Done():
			ctxDoneCh <- true
			return ctx.Err()
		case <-time.After(time.Second):
			ctxDoneCh <- false
			return nil
		}
	})

	if err != ErrTimeout {
		t.Errorf("Execute() error = %v, want ErrTimeout", err)
	}

	// Wait for the operation goroutine to signal its result
	select {
	case ctxDone := <-ctxDoneCh:
		if !ctxDone {
			t.Error("Context was not cancelled")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Operation goroutine did not complete")
	}
}

func TestTimeout_Config(t *testing.T) {
	timeout := NewTimeout(TimeoutConfig{
		Timeout: 5 * time.Second,
	})

	config := timeout.Config()
	if config.Timeout != 5*time.Second {
		t.Errorf("Config().Timeout = %v, want 5s", config.Timeout)
	}
}

func TestExecuteWithTimeout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := ExecuteWithTimeout(context.Background(), time.Second, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("ExecuteWithTimeout() error = %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		err := ExecuteWithTimeout(context.Background(), 10*time.Millisecond, func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		if err != ErrTimeout {
			t.Errorf("ExecuteWithTimeout() error = %v, want ErrTimeout", err)
		}
	})
}
