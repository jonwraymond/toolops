package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkMemoryCache_Get_Hit measures cache hit performance.
func BenchmarkMemoryCache_Get_Hit(b *testing.B) {
	policy := DefaultPolicy()
	c := NewMemoryCache(policy)
	ctx := context.Background()

	// Pre-populate
	_ = c.Set(ctx, "key", []byte("value"), time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get(ctx, "key")
	}
}

// BenchmarkMemoryCache_Get_Miss measures cache miss performance.
func BenchmarkMemoryCache_Get_Miss(b *testing.B) {
	policy := DefaultPolicy()
	c := NewMemoryCache(policy)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get(ctx, "missing")
	}
}

// BenchmarkMemoryCache_Set measures write performance.
func BenchmarkMemoryCache_Set(b *testing.B) {
	policy := DefaultPolicy()
	c := NewMemoryCache(policy)
	ctx := context.Background()
	value := []byte("test value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Set(ctx, fmt.Sprintf("key-%d", i), value, time.Hour)
	}
}

// BenchmarkMemoryCache_Set_SameKey measures overwrite performance.
func BenchmarkMemoryCache_Set_SameKey(b *testing.B) {
	policy := DefaultPolicy()
	c := NewMemoryCache(policy)
	ctx := context.Background()
	value := []byte("test value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Set(ctx, "same-key", value, time.Hour)
	}
}

// BenchmarkMemoryCache_Delete measures delete performance.
func BenchmarkMemoryCache_Delete(b *testing.B) {
	policy := DefaultPolicy()
	c := NewMemoryCache(policy)
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < b.N; i++ {
		_ = c.Set(ctx, fmt.Sprintf("key-%d", i), []byte("value"), time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Delete(ctx, fmt.Sprintf("key-%d", i))
	}
}

// BenchmarkMemoryCache_Concurrent_ReadWrite measures mixed concurrent operations.
func BenchmarkMemoryCache_Concurrent_ReadWrite(b *testing.B) {
	policy := DefaultPolicy()
	c := NewMemoryCache(policy)
	ctx := context.Background()

	// Pre-populate some entries
	for i := 0; i < 100; i++ {
		_ = c.Set(ctx, fmt.Sprintf("key-%d", i), []byte("value"), time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%100)
			if i%4 == 0 {
				// 25% writes
				_ = c.Set(ctx, key, []byte("new-value"), time.Hour)
			} else {
				// 75% reads
				_, _ = c.Get(ctx, key)
			}
			i++
		}
	})
}

// BenchmarkMemoryCache_Concurrent_ReadHeavy measures read-heavy workload.
func BenchmarkMemoryCache_Concurrent_ReadHeavy(b *testing.B) {
	policy := DefaultPolicy()
	c := NewMemoryCache(policy)
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 100; i++ {
		_ = c.Set(ctx, fmt.Sprintf("key-%d", i), []byte("value"), time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = c.Get(ctx, fmt.Sprintf("key-%d", i%100))
			i++
		}
	})
}

// BenchmarkDefaultKeyer_Key measures key generation.
func BenchmarkDefaultKeyer_Key(b *testing.B) {
	keyer := NewDefaultKeyer()
	input := map[string]any{
		"query": "test",
		"limit": 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = keyer.Key("github.search", input)
	}
}

// BenchmarkDefaultKeyer_Key_LargeInput measures key generation with large input.
func BenchmarkDefaultKeyer_Key_LargeInput(b *testing.B) {
	keyer := NewDefaultKeyer()
	input := map[string]any{
		"query":   "test query string",
		"limit":   100,
		"offset":  0,
		"filters": []any{"filter1", "filter2", "filter3"},
		"nested": map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = keyer.Key("complex.tool", input)
	}
}

// BenchmarkDefaultKeyer_Key_Concurrent measures concurrent key generation.
func BenchmarkDefaultKeyer_Key_Concurrent(b *testing.B) {
	keyer := NewDefaultKeyer()
	input := map[string]any{"query": "test"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = keyer.Key("tool", input)
		}
	})
}

// BenchmarkPolicy_EffectiveTTL measures TTL calculation.
func BenchmarkPolicy_EffectiveTTL(b *testing.B) {
	policy := DefaultPolicy()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.EffectiveTTL(10 * time.Minute)
	}
}

// BenchmarkPolicy_ShouldCache measures cache decision.
func BenchmarkPolicy_ShouldCache(b *testing.B) {
	policy := DefaultPolicy()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.ShouldCache()
	}
}

// BenchmarkDefaultSkipRule measures skip rule evaluation.
func BenchmarkDefaultSkipRule(b *testing.B) {
	tags := []string{"read", "query", "safe"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultSkipRule("tool.id", tags)
	}
}

// BenchmarkDefaultSkipRule_Unsafe measures skip rule with unsafe tag.
func BenchmarkDefaultSkipRule_Unsafe(b *testing.B) {
	tags := []string{"read", "write", "important"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultSkipRule("tool.id", tags)
	}
}

// BenchmarkValidateKey measures key validation.
func BenchmarkValidateKey(b *testing.B) {
	key := "cache:github.search:abc123def456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateKey(key)
	}
}

// BenchmarkCacheMiddleware_Execute_Hit measures middleware with cache hit.
func BenchmarkCacheMiddleware_Execute_Hit(b *testing.B) {
	policy := DefaultPolicy()
	memCache := NewMemoryCache(policy)
	keyer := NewDefaultKeyer()
	mw := NewCacheMiddleware(memCache, keyer, policy, nil)

	ctx := context.Background()
	executor := func(ctx context.Context, toolID string, input any) ([]byte, error) {
		return []byte("result"), nil
	}

	// Pre-warm cache
	_, _ = mw.Execute(ctx, "tool", "input", nil, executor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mw.Execute(ctx, "tool", "input", nil, executor)
	}
}

// BenchmarkCacheMiddleware_Execute_Miss measures middleware with cache miss.
func BenchmarkCacheMiddleware_Execute_Miss(b *testing.B) {
	policy := NoCachePolicy() // Ensure miss every time
	memCache := NewMemoryCache(policy)
	keyer := NewDefaultKeyer()
	mw := NewCacheMiddleware(memCache, keyer, policy, nil)

	ctx := context.Background()
	executor := func(ctx context.Context, toolID string, input any) ([]byte, error) {
		return []byte("result"), nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mw.Execute(ctx, "tool", "input", nil, executor)
	}
}

// BenchmarkCacheMiddleware_Concurrent measures concurrent middleware usage.
func BenchmarkCacheMiddleware_Concurrent(b *testing.B) {
	policy := DefaultPolicy()
	memCache := NewMemoryCache(policy)
	keyer := NewDefaultKeyer()
	mw := NewCacheMiddleware(memCache, keyer, policy, nil)

	ctx := context.Background()
	executor := func(ctx context.Context, toolID string, input any) ([]byte, error) {
		return []byte("result"), nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			toolID := fmt.Sprintf("tool-%d", i%10)
			_, _ = mw.Execute(ctx, toolID, "input", nil, executor)
			i++
		}
	})
}
