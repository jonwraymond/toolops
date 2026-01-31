package resilience

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{})

	if rl.config.Rate != 100 {
		t.Errorf("Rate = %f, want 100", rl.config.Rate)
	}
	if rl.config.Burst != 10 {
		t.Errorf("Burst = %d, want 10", rl.config.Burst)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  10, // 10 per second
		Burst: 5,
	})

	// Should allow burst
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Errorf("Allow() = false on attempt %d, want true", i)
		}
	}

	// Should deny after burst
	if rl.Allow() {
		t.Error("Allow() = true after burst exhausted, want false")
	}
}

func TestRateLimiter_AllowN(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  10,
		Burst: 5,
	})

	// Should allow N tokens
	if !rl.AllowN(3) {
		t.Error("AllowN(3) = false, want true")
	}

	// Should allow remaining tokens
	if !rl.AllowN(2) {
		t.Error("AllowN(2) = false, want true")
	}

	// Should deny when not enough tokens
	if rl.AllowN(1) {
		t.Error("AllowN(1) = true when empty, want false")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  1000, // 1000 per second = 1 per ms
		Burst: 5,
	})

	// Exhaust tokens
	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	// Wait for refill
	time.Sleep(10 * time.Millisecond)

	// Should have some tokens now
	if !rl.Allow() {
		t.Error("Allow() = false after refill, want true")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:    1000, // 1000 per second
		Burst:   1,
		MaxWait: 100 * time.Millisecond,
	})

	// Exhaust tokens
	rl.Allow()

	// Should wait and succeed
	ctx := context.Background()
	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}

	// Should have waited briefly
	if elapsed < time.Millisecond {
		t.Errorf("Wait() elapsed = %v, want > 1ms", elapsed)
	}
}

func TestRateLimiter_WaitTimeout(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:    0.1, // Very slow: 1 per 10 seconds
		Burst:   1,
		MaxWait: 10 * time.Millisecond,
	})

	// Exhaust tokens
	rl.Allow()

	// Should timeout
	err := rl.Wait(context.Background())
	if err != ErrRateLimitExceeded {
		t.Errorf("Wait() error = %v, want ErrRateLimitExceeded", err)
	}
}

func TestRateLimiter_WaitContextCancellation(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:    0.1,
		Burst:   1,
		MaxWait: time.Second,
	})

	// Exhaust tokens
	rl.Allow()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := rl.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
}

func TestRateLimiter_Execute(t *testing.T) {
	t.Run("without wait", func(t *testing.T) {
		rl := NewRateLimiter(RateLimiterConfig{
			Rate:        10,
			Burst:       1,
			WaitOnLimit: false,
		})

		// First should succeed
		err := rl.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("First Execute() error = %v", err)
		}

		// Second should fail
		err = rl.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != ErrRateLimitExceeded {
			t.Errorf("Second Execute() error = %v, want ErrRateLimitExceeded", err)
		}
	})

	t.Run("with wait", func(t *testing.T) {
		rl := NewRateLimiter(RateLimiterConfig{
			Rate:        1000,
			Burst:       1,
			WaitOnLimit: true,
			MaxWait:     100 * time.Millisecond,
		})

		// Exhaust tokens
		rl.Allow()

		// Should wait and succeed
		err := rl.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
	})
}

func TestRateLimiter_Tokens(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  100,
		Burst: 10,
	})

	tokens := rl.Tokens()
	if tokens != 10 {
		t.Errorf("Initial tokens = %f, want 10", tokens)
	}

	rl.Allow()
	rl.Allow()

	tokens = rl.Tokens()
	if tokens < 7.9 || tokens > 8.1 {
		t.Errorf("After 2 allows, tokens = %f, want ~8", tokens)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  100,
		Burst: 10,
	})

	// Exhaust tokens
	for i := 0; i < 10; i++ {
		rl.Allow()
	}

	tokens := rl.Tokens()
	if tokens > 0.5 {
		t.Errorf("Tokens after exhaust = %f, want ~0", tokens)
	}

	rl.Reset()

	tokens = rl.Tokens()
	if tokens != 10 {
		t.Errorf("Tokens after reset = %f, want 10", tokens)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		Rate:  1000,
		Burst: 100,
	})

	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should have allowed around 100 (burst size)
	if allowed < 90 || allowed > 110 {
		t.Errorf("Concurrent allowed = %d, want ~100", allowed)
	}
}
