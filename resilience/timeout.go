package resilience

import (
	"context"
	"time"
)

// TimeoutConfig configures the timeout wrapper.
type TimeoutConfig struct {
	// Timeout is the maximum duration for the operation.
	// Default: 30 seconds
	Timeout time.Duration
}

// Timeout wraps operations with a timeout.
type Timeout struct {
	config TimeoutConfig
}

// NewTimeout creates a new timeout wrapper.
func NewTimeout(config TimeoutConfig) *Timeout {
	// Apply defaults
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	return &Timeout{config: config}
}

// Execute runs the operation with a timeout.
func (t *Timeout) Execute(ctx context.Context, op func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- op(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return ErrTimeout
		}
		return ctx.Err()
	}
}

// Config returns the timeout configuration.
func (t *Timeout) Config() TimeoutConfig {
	return t.config
}

// ExecuteWithTimeout is a convenience function to run an operation with timeout.
func ExecuteWithTimeout(ctx context.Context, timeout time.Duration, op func(context.Context) error) error {
	t := NewTimeout(TimeoutConfig{Timeout: timeout})
	return t.Execute(ctx, op)
}
