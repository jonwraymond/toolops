package cache

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_GetSetDelete(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	ctx := context.Background()

	// Test Get on empty cache
	val, ok := cache.Get(ctx, "nonexistent")
	if ok {
		t.Error("Get on empty cache should return ok=false")
	}
	if val != nil {
		t.Error("Get on empty cache should return nil value")
	}

	// Test Set
	key := "test-key"
	value := []byte("test-value")
	err := cache.Set(ctx, key, value, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get after Set
	got, ok := cache.Get(ctx, key)
	if !ok {
		t.Error("Get after Set should return ok=true")
	}
	if !bytes.Equal(got, value) {
		t.Errorf("Get returned %q, want %q", got, value)
	}

	// Test Delete
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Test Get after Delete
	val, ok = cache.Get(ctx, key)
	if ok {
		t.Error("Get after Delete should return ok=false")
	}
	if val != nil {
		t.Error("Get after Delete should return nil value")
	}

	// Test Delete is idempotent (no error on non-existent key)
	err = cache.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Delete on non-existent key should not error, got: %v", err)
	}
}

func TestMemoryCache_Expiry(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	ctx := context.Background()

	key := "expiring-key"
	value := []byte("expiring-value")

	// Set with very short TTL
	err := cache.Set(ctx, key, value, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should be present immediately
	got, ok := cache.Get(ctx, key)
	if !ok {
		t.Error("Get immediately after Set should return ok=true")
	}
	if !bytes.Equal(got, value) {
		t.Errorf("Get returned %q, want %q", got, value)
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	val, ok := cache.Get(ctx, key)
	if ok {
		t.Error("Get after expiry should return ok=false")
	}
	if val != nil {
		t.Error("Get after expiry should return nil value")
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	ctx := context.Background()

	const numGoroutines = 100
	const opsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := "concurrent-key"
				value := []byte("concurrent-value")

				// Mix of operations
				switch j % 3 {
				case 0:
					_ = cache.Set(ctx, key, value, 5*time.Minute)
				case 1:
					_, _ = cache.Get(ctx, key)
				case 2:
					_ = cache.Delete(ctx, key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestMemoryCache_SetOverwrite(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	ctx := context.Background()

	key := "overwrite-key"
	value1 := []byte("value1")
	value2 := []byte("value2")

	// Set initial value
	err := cache.Set(ctx, key, value1, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify initial value
	got, ok := cache.Get(ctx, key)
	if !ok {
		t.Error("Get should return ok=true")
	}
	if !bytes.Equal(got, value1) {
		t.Errorf("Get returned %q, want %q", got, value1)
	}

	// Overwrite with new value
	err = cache.Set(ctx, key, value2, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set (overwrite) failed: %v", err)
	}

	// Verify new value
	got, ok = cache.Get(ctx, key)
	if !ok {
		t.Error("Get after overwrite should return ok=true")
	}
	if !bytes.Equal(got, value2) {
		t.Errorf("Get returned %q, want %q", got, value2)
	}
}

func TestMemoryCache_ZeroTTL(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	ctx := context.Background()

	key := "zero-ttl-key"
	value := []byte("zero-ttl-value")

	// Set with TTL=0 (immediate expiry, no caching)
	err := cache.Set(ctx, key, value, 0)
	if err != nil {
		t.Fatalf("Set with TTL=0 failed: %v", err)
	}

	// Should not be stored (immediate expiry)
	val, ok := cache.Get(ctx, key)
	if ok {
		t.Error("Get after Set with TTL=0 should return ok=false")
	}
	if val != nil {
		t.Error("Get after Set with TTL=0 should return nil value")
	}
}

func TestMemoryCache_NilValue(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	ctx := context.Background()

	key := "nil-value-key"

	// Set nil value
	err := cache.Set(ctx, key, nil, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set with nil value failed: %v", err)
	}

	// Get should return nil value with ok=true
	got, ok := cache.Get(ctx, key)
	if !ok {
		t.Error("Get after Set with nil value should return ok=true")
	}
	if got != nil {
		t.Errorf("Get returned %q, want nil", got)
	}
}

func TestMemoryCache_ContextCancellation(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	key := "ctx-key"
	value := []byte("ctx-value")

	// Operations should still work with cancelled context
	// (memory cache doesn't block on context)
	err := cache.Set(ctx, key, value, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set with cancelled context failed: %v", err)
	}

	got, ok := cache.Get(ctx, key)
	if !ok {
		t.Error("Get with cancelled context should return ok=true")
	}
	if !bytes.Equal(got, value) {
		t.Errorf("Get returned %q, want %q", got, value)
	}

	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete with cancelled context failed: %v", err)
	}
}

func TestMemoryCache_LargeValues(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	ctx := context.Background()

	key := "large-value-key"
	// Create 1MB value
	value := make([]byte, 1024*1024)
	for i := range value {
		value[i] = byte(i % 256)
	}

	// Set large value
	err := cache.Set(ctx, key, value, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set with large value failed: %v", err)
	}

	// Get large value
	got, ok := cache.Get(ctx, key)
	if !ok {
		t.Error("Get large value should return ok=true")
	}
	if !bytes.Equal(got, value) {
		t.Error("Get returned different value than what was set")
	}
}

// Verify MemoryCache implements Cache interface at compile time
var _ Cache = (*MemoryCache)(nil)
