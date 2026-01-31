package resilience

import (
	"context"
	"sync"
	"time"
)

// BulkheadConfig configures the bulkhead.
type BulkheadConfig struct {
	// MaxConcurrent is the maximum number of concurrent operations.
	// Default: 10
	MaxConcurrent int

	// MaxWait is the maximum time to wait for a slot.
	// Default: 0 (no waiting, fail immediately)
	MaxWait time.Duration
}

// Bulkhead limits concurrent operations.
type Bulkhead struct {
	config BulkheadConfig
	sem    chan struct{}

	mu        sync.Mutex
	active    int
	maxActive int
	rejected  int64
}

// NewBulkhead creates a new bulkhead.
func NewBulkhead(config BulkheadConfig) *Bulkhead {
	// Apply defaults
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 10
	}

	return &Bulkhead{
		config: config,
		sem:    make(chan struct{}, config.MaxConcurrent),
	}
}

// Acquire acquires a slot in the bulkhead.
// Returns ErrBulkheadFull if no slot is available.
func (b *Bulkhead) Acquire(ctx context.Context) error {
	// Fast path: try non-blocking acquire
	select {
	case b.sem <- struct{}{}:
		b.mu.Lock()
		b.active++
		if b.active > b.maxActive {
			b.maxActive = b.active
		}
		b.mu.Unlock()
		return nil
	default:
		// Fall through to waiting logic
	}

	// No immediate slot available
	if b.config.MaxWait <= 0 {
		b.mu.Lock()
		b.rejected++
		b.mu.Unlock()
		return ErrBulkheadFull
	}

	// Wait for a slot
	timer := time.NewTimer(b.config.MaxWait)
	defer timer.Stop()

	select {
	case b.sem <- struct{}{}:
		b.mu.Lock()
		b.active++
		if b.active > b.maxActive {
			b.maxActive = b.active
		}
		b.mu.Unlock()
		return nil
	case <-timer.C:
		b.mu.Lock()
		b.rejected++
		b.mu.Unlock()
		return ErrBulkheadFull
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases a slot in the bulkhead.
func (b *Bulkhead) Release() {
	select {
	case <-b.sem:
		b.mu.Lock()
		b.active--
		b.mu.Unlock()
	default:
		// Semaphore was empty, this shouldn't happen in normal usage
	}
}

// Execute runs the operation within the bulkhead.
func (b *Bulkhead) Execute(ctx context.Context, op func(context.Context) error) error {
	if err := b.Acquire(ctx); err != nil {
		return err
	}
	defer b.Release()

	return op(ctx)
}

// Metrics returns current bulkhead metrics.
func (b *Bulkhead) Metrics() BulkheadMetrics {
	b.mu.Lock()
	defer b.mu.Unlock()

	return BulkheadMetrics{
		Active:        b.active,
		MaxActive:     b.maxActive,
		Available:     b.config.MaxConcurrent - b.active,
		MaxConcurrent: b.config.MaxConcurrent,
		Rejected:      b.rejected,
	}
}

// BulkheadMetrics contains bulkhead statistics.
type BulkheadMetrics struct {
	Active        int
	MaxActive     int
	Available     int
	MaxConcurrent int
	Rejected      int64
}
