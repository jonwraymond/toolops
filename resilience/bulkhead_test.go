package resilience

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewBulkhead(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{})

	if b.config.MaxConcurrent != 10 {
		t.Errorf("MaxConcurrent = %d, want 10", b.config.MaxConcurrent)
	}
}

func TestBulkhead_AcquireRelease(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 2,
	})

	// Acquire 2 slots
	if err := b.Acquire(context.Background()); err != nil {
		t.Errorf("First Acquire() error = %v", err)
	}
	if err := b.Acquire(context.Background()); err != nil {
		t.Errorf("Second Acquire() error = %v", err)
	}

	// Third should fail
	if err := b.Acquire(context.Background()); err != ErrBulkheadFull {
		t.Errorf("Third Acquire() error = %v, want ErrBulkheadFull", err)
	}

	// Release one
	b.Release()

	// Should be able to acquire again
	if err := b.Acquire(context.Background()); err != nil {
		t.Errorf("Acquire after release error = %v", err)
	}
}

func TestBulkhead_AcquireWithWait(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1,
		MaxWait:       100 * time.Millisecond,
	})

	// Acquire the only slot
	if err := b.Acquire(context.Background()); err != nil {
		t.Fatalf("First Acquire() error = %v", err)
	}

	// Start goroutine to release after delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		b.Release()
	}()

	// Should wait and succeed
	if err := b.Acquire(context.Background()); err != nil {
		t.Errorf("Second Acquire() error = %v", err)
	}
}

func TestBulkhead_AcquireTimeout(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1,
		MaxWait:       10 * time.Millisecond,
	})

	// Acquire the only slot
	if err := b.Acquire(context.Background()); err != nil {
		t.Fatalf("First Acquire() error = %v", err)
	}

	// Should timeout
	if err := b.Acquire(context.Background()); err != ErrBulkheadFull {
		t.Errorf("Second Acquire() error = %v, want ErrBulkheadFull", err)
	}
}

func TestBulkhead_ContextCancellation(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1,
		MaxWait:       time.Second,
	})

	// Acquire the only slot
	if err := b.Acquire(context.Background()); err != nil {
		t.Fatalf("First Acquire() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	if err := b.Acquire(ctx); err != context.Canceled {
		t.Errorf("Acquire() error = %v, want context.Canceled", err)
	}
}

func TestBulkhead_Execute(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1,
	})

	executed := false
	err := b.Execute(context.Background(), func(ctx context.Context) error {
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

func TestBulkhead_ExecuteFull(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 1,
	})

	// Acquire the slot
	if err := b.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	// Execute should fail
	err := b.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != ErrBulkheadFull {
		t.Errorf("Execute() error = %v, want ErrBulkheadFull", err)
	}
}

func TestBulkhead_Concurrent(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 5,
	})

	var (
		wg         sync.WaitGroup
		maxActive  int32
		currActive int32
	)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := b.Execute(context.Background(), func(ctx context.Context) error {
				curr := atomic.AddInt32(&currActive, 1)
				defer atomic.AddInt32(&currActive, -1)

				// Track max concurrent
				for {
					max := atomic.LoadInt32(&maxActive)
					if curr <= max || atomic.CompareAndSwapInt32(&maxActive, max, curr) {
						break
					}
				}

				time.Sleep(10 * time.Millisecond)
				return nil
			})

			if err != nil && err != ErrBulkheadFull {
				t.Errorf("Execute() error = %v", err)
			}
		}()
	}

	wg.Wait()

	max := atomic.LoadInt32(&maxActive)
	if max > 5 {
		t.Errorf("Max concurrent = %d, want <= 5", max)
	}
}

func TestBulkhead_Metrics(t *testing.T) {
	b := NewBulkhead(BulkheadConfig{
		MaxConcurrent: 3,
	})

	// Acquire 2 slots
	_ = b.Acquire(context.Background())
	_ = b.Acquire(context.Background())

	// Try to acquire when full
	b2 := NewBulkhead(BulkheadConfig{MaxConcurrent: 1})
	_ = b2.Acquire(context.Background())
	_ = b2.Acquire(context.Background()) // This will be rejected

	metrics := b.Metrics()

	if metrics.Active != 2 {
		t.Errorf("Metrics.Active = %d, want 2", metrics.Active)
	}
	if metrics.MaxActive != 2 {
		t.Errorf("Metrics.MaxActive = %d, want 2", metrics.MaxActive)
	}
	if metrics.Available != 1 {
		t.Errorf("Metrics.Available = %d, want 1", metrics.Available)
	}
	if metrics.MaxConcurrent != 3 {
		t.Errorf("Metrics.MaxConcurrent = %d, want 3", metrics.MaxConcurrent)
	}

	b2Metrics := b2.Metrics()
	if b2Metrics.Rejected != 1 {
		t.Errorf("Metrics.Rejected = %d, want 1", b2Metrics.Rejected)
	}
}
