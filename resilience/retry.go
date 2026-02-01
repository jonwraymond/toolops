package resilience

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// BackoffStrategy defines how delays increase between retries.
type BackoffStrategy int

const (
	// BackoffExponential doubles the delay each attempt with jitter.
	BackoffExponential BackoffStrategy = iota
	// BackoffLinear increases delay linearly.
	BackoffLinear
	// BackoffConstant uses the same delay for all retries.
	BackoffConstant
)

// RetryConfig configures the retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including initial).
	// Default: 3
	MaxAttempts int

	// InitialDelay is the delay before the first retry.
	// Default: 100ms
	InitialDelay time.Duration

	// MaxDelay caps the maximum delay between retries.
	// Default: 30s
	MaxDelay time.Duration

	// Multiplier is the backoff multiplier for exponential backoff.
	// Default: 2.0
	Multiplier float64

	// Strategy is the backoff strategy.
	// Default: BackoffExponential
	Strategy BackoffStrategy

	// Jitter adds randomness to delays to prevent thundering herd.
	// Default: true
	Jitter bool

	// RetryIf determines if an error should trigger a retry.
	// Default: all non-nil errors trigger retry.
	RetryIf func(err error) bool

	// OnRetry is called before each retry attempt.
	OnRetry func(attempt int, err error, delay time.Duration)
}

// Retry implements retry with backoff.
type Retry struct {
	config RetryConfig
}

// NewRetry creates a new retry handler.
func NewRetry(config RetryConfig) *Retry {
	// Apply defaults
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.RetryIf == nil {
		config.RetryIf = func(err error) bool { return err != nil }
	}

	return &Retry{config: config}
}

// Execute runs the operation with retry logic.
func (r *Retry) Execute(ctx context.Context, op func(context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		err := op(ctx)

		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !r.config.RetryIf(err) {
			return err
		}

		// Don't retry if this was the last attempt
		if attempt >= r.config.MaxAttempts {
			break
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Callback before retry
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, err, delay)
		}

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

func (r *Retry) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch r.config.Strategy {
	case BackoffConstant:
		delay = r.config.InitialDelay

	case BackoffLinear:
		delay = r.config.InitialDelay * time.Duration(attempt)

	case BackoffExponential:
		multiplier := math.Pow(r.config.Multiplier, float64(attempt-1))
		delay = time.Duration(float64(r.config.InitialDelay) * multiplier)
	}

	// Cap at max delay
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	// Add jitter if enabled
	if r.config.Jitter && delay > 0 {
		// Add up to 25% jitter
		// #nosec G404 -- jitter is non-cryptographic timing variance.
		jitter := time.Duration(rand.Int64N(int64(delay / 4)))
		delay = delay + jitter
	}

	return delay
}

// Config returns the retry configuration.
func (r *Retry) Config() RetryConfig {
	return r.config
}
