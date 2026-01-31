package resilience

import (
	"context"
	"sync"
	"time"
)

// RateLimiterConfig configures the rate limiter.
type RateLimiterConfig struct {
	// Rate is the number of operations allowed per second.
	// Default: 100
	Rate float64

	// Burst is the maximum burst size.
	// Default: 10
	Burst int

	// WaitOnLimit waits for a token instead of returning error.
	// Default: false
	WaitOnLimit bool

	// MaxWait is the maximum time to wait for a token.
	// Default: 1 second
	MaxWait time.Duration
}

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	config RateLimiterConfig

	mu          sync.Mutex
	tokens      float64
	lastRefresh time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	// Apply defaults
	if config.Rate <= 0 {
		config.Rate = 100
	}
	if config.Burst <= 0 {
		config.Burst = 10
	}
	if config.MaxWait <= 0 {
		config.MaxWait = time.Second
	}

	return &RateLimiter{
		config:      config,
		tokens:      float64(config.Burst),
		lastRefresh: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit.
func (rl *RateLimiter) Allow() bool {
	return rl.AllowN(1)
}

// AllowN checks if n requests are allowed.
func (rl *RateLimiter) AllowN(n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillLocked()

	if rl.tokens >= float64(n) {
		rl.tokens -= float64(n)
		return true
	}

	return false
}

// Wait blocks until a token is available or context is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.WaitN(ctx, 1)
}

// WaitN blocks until n tokens are available.
func (rl *RateLimiter) WaitN(ctx context.Context, n int) error {
	// Check context first
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if rl.AllowN(n) {
		return nil
	}

	// Calculate wait time
	rl.mu.Lock()
	tokensNeeded := float64(n) - rl.tokens
	waitTime := time.Duration(tokensNeeded / rl.config.Rate * float64(time.Second))
	rl.mu.Unlock()

	// Cap wait time - but still allow context cancellation during the capped wait
	if waitTime > rl.config.MaxWait {
		waitTime = rl.config.MaxWait
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		// Try again after waiting
		if rl.AllowN(n) {
			return nil
		}
		return ErrRateLimitExceeded
	}
}

// Execute runs the operation if allowed by rate limit.
func (rl *RateLimiter) Execute(ctx context.Context, op func(context.Context) error) error {
	if rl.config.WaitOnLimit {
		if err := rl.Wait(ctx); err != nil {
			return err
		}
	} else if !rl.Allow() {
		return ErrRateLimitExceeded
	}

	return op(ctx)
}

func (rl *RateLimiter) refillLocked() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefresh)
	rl.lastRefresh = now

	// Add tokens based on elapsed time
	tokensToAdd := elapsed.Seconds() * rl.config.Rate
	rl.tokens += tokensToAdd

	// Cap at burst size
	if rl.tokens > float64(rl.config.Burst) {
		rl.tokens = float64(rl.config.Burst)
	}
}

// Tokens returns the current number of available tokens.
func (rl *RateLimiter) Tokens() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.refillLocked()
	return rl.tokens
}

// Reset resets the rate limiter to full capacity.
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.tokens = float64(rl.config.Burst)
	rl.lastRefresh = time.Now()
}
