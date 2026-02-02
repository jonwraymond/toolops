package cache_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolops/cache"
)

func ExampleNewMemoryCache() {
	policy := cache.DefaultPolicy()
	c := cache.NewMemoryCache(policy)

	ctx := context.Background()

	// Store a value
	_ = c.Set(ctx, "my-key", []byte("hello"), 5*time.Minute)

	// Retrieve the value
	value, ok := c.Get(ctx, "my-key")
	if ok {
		fmt.Println("Value:", string(value))
	}
	// Output:
	// Value: hello
}

func ExampleMemoryCache_Get() {
	policy := cache.DefaultPolicy()
	c := cache.NewMemoryCache(policy)
	ctx := context.Background()

	// Miss - key doesn't exist
	_, ok := c.Get(ctx, "missing")
	fmt.Println("Missing key found:", ok)

	// Set and get
	_ = c.Set(ctx, "exists", []byte("data"), time.Hour)
	value, ok := c.Get(ctx, "exists")
	fmt.Println("Existing key found:", ok)
	fmt.Println("Value:", string(value))
	// Output:
	// Missing key found: false
	// Existing key found: true
	// Value: data
}

func ExampleMemoryCache_Set() {
	policy := cache.DefaultPolicy()
	c := cache.NewMemoryCache(policy)
	ctx := context.Background()

	// Normal set with TTL
	err := c.Set(ctx, "key1", []byte("value1"), 5*time.Minute)
	fmt.Println("Set error:", err)

	// Set with zero TTL is a no-op (no caching)
	err = c.Set(ctx, "key2", []byte("value2"), 0)
	fmt.Println("Zero TTL error:", err)

	// Verify zero TTL didn't cache
	_, ok := c.Get(ctx, "key2")
	fmt.Println("Zero TTL key cached:", ok)
	// Output:
	// Set error: <nil>
	// Zero TTL error: <nil>
	// Zero TTL key cached: false
}

func ExampleMemoryCache_Delete() {
	policy := cache.DefaultPolicy()
	c := cache.NewMemoryCache(policy)
	ctx := context.Background()

	// Set a value
	_ = c.Set(ctx, "to-delete", []byte("temporary"), time.Hour)

	// Verify it exists
	_, ok := c.Get(ctx, "to-delete")
	fmt.Println("Before delete:", ok)

	// Delete it
	err := c.Delete(ctx, "to-delete")
	fmt.Println("Delete error:", err)

	// Verify it's gone
	_, ok = c.Get(ctx, "to-delete")
	fmt.Println("After delete:", ok)

	// Delete is idempotent - no error on missing key
	err = c.Delete(ctx, "never-existed")
	fmt.Println("Delete missing:", err)
	// Output:
	// Before delete: true
	// Delete error: <nil>
	// After delete: false
	// Delete missing: <nil>
}

func ExampleNewDefaultKeyer() {
	keyer := cache.NewDefaultKeyer()

	// Simple input
	key1, _ := keyer.Key("github.search", map[string]any{"query": "test"})
	fmt.Println("Key format:", key1[:14]) // "cache:github.s..."

	// Deterministic - same input produces same key
	key2, _ := keyer.Key("github.search", map[string]any{"query": "test"})
	fmt.Println("Keys match:", key1 == key2)

	// Different input produces different key
	key3, _ := keyer.Key("github.search", map[string]any{"query": "other"})
	fmt.Println("Different input, different key:", key1 != key3)
	// Output:
	// Key format: cache:github.s
	// Keys match: true
	// Different input, different key: true
}

func ExampleDefaultKeyer_Key_mapOrdering() {
	keyer := cache.NewDefaultKeyer()

	// Map ordering doesn't affect key - keys are sorted internally
	input1 := map[string]any{"b": 2, "a": 1, "c": 3}
	input2 := map[string]any{"c": 3, "a": 1, "b": 2}

	key1, _ := keyer.Key("tool", input1)
	key2, _ := keyer.Key("tool", input2)

	fmt.Println("Same map, different order, same key:", key1 == key2)
	// Output:
	// Same map, different order, same key: true
}

func ExampleDefaultPolicy() {
	policy := cache.DefaultPolicy()

	fmt.Println("Default TTL:", policy.DefaultTTL)
	fmt.Println("Max TTL:", policy.MaxTTL)
	fmt.Println("Allow unsafe:", policy.AllowUnsafe)
	fmt.Println("Should cache:", policy.ShouldCache())
	// Output:
	// Default TTL: 5m0s
	// Max TTL: 1h0m0s
	// Allow unsafe: false
	// Should cache: true
}

func ExampleNoCachePolicy() {
	policy := cache.NoCachePolicy()

	fmt.Println("Should cache:", policy.ShouldCache())
	// Output:
	// Should cache: false
}

func ExamplePolicy_EffectiveTTL() {
	policy := cache.Policy{
		DefaultTTL: 5 * time.Minute,
		MaxTTL:     1 * time.Hour,
	}

	// No override - uses default
	fmt.Println("No override:", policy.EffectiveTTL(0))

	// Reasonable override - uses as-is
	fmt.Println("10min override:", policy.EffectiveTTL(10*time.Minute))

	// Excessive override - clamped to max
	fmt.Println("2hr override (clamped):", policy.EffectiveTTL(2*time.Hour))
	// Output:
	// No override: 5m0s
	// 10min override: 10m0s
	// 2hr override (clamped): 1h0m0s
}

func ExampleNewCacheMiddleware() {
	policy := cache.DefaultPolicy()
	memCache := cache.NewMemoryCache(policy)
	keyer := cache.NewDefaultKeyer()

	mw := cache.NewCacheMiddleware(memCache, keyer, policy, nil)

	ctx := context.Background()
	executorCalls := 0

	executor := func(ctx context.Context, toolID string, input any) ([]byte, error) {
		executorCalls++
		return []byte("result"), nil
	}

	// First call - cache miss
	result1, _ := mw.Execute(ctx, "tool1", "input", nil, executor)
	fmt.Println("Call 1 result:", string(result1))
	fmt.Println("Executor calls after 1:", executorCalls)

	// Second call - cache hit
	result2, _ := mw.Execute(ctx, "tool1", "input", nil, executor)
	fmt.Println("Call 2 result:", string(result2))
	fmt.Println("Executor calls after 2:", executorCalls) // Still 1 - cached!
	// Output:
	// Call 1 result: result
	// Executor calls after 1: 1
	// Call 2 result: result
	// Executor calls after 2: 1
}

func ExampleCacheMiddleware_Execute_unsafeTags() {
	policy := cache.DefaultPolicy() // AllowUnsafe: false
	memCache := cache.NewMemoryCache(policy)
	keyer := cache.NewDefaultKeyer()
	mw := cache.NewCacheMiddleware(memCache, keyer, policy, nil)

	ctx := context.Background()
	executorCalls := 0

	executor := func(ctx context.Context, toolID string, input any) ([]byte, error) {
		executorCalls++
		return []byte("executed"), nil
	}

	// Tool with "write" tag - not cached
	_, _ = mw.Execute(ctx, "fs.write", nil, []string{"write"}, executor)
	_, _ = mw.Execute(ctx, "fs.write", nil, []string{"write"}, executor)
	fmt.Println("Write tool executor calls:", executorCalls) // Called twice

	// Reset
	executorCalls = 0

	// Tool without unsafe tags - cached
	_, _ = mw.Execute(ctx, "fs.read", nil, []string{"read"}, executor)
	_, _ = mw.Execute(ctx, "fs.read", nil, []string{"read"}, executor)
	fmt.Println("Read tool executor calls:", executorCalls) // Called once
	// Output:
	// Write tool executor calls: 2
	// Read tool executor calls: 1
}

func ExampleDefaultSkipRule() {
	// Unsafe tags
	fmt.Println("write tag:", cache.DefaultSkipRule("tool", []string{"write"}))
	fmt.Println("danger tag:", cache.DefaultSkipRule("tool", []string{"danger"}))
	fmt.Println("UNSAFE tag (case-insensitive):", cache.DefaultSkipRule("tool", []string{"UNSAFE"}))

	// Safe tags
	fmt.Println("read tag:", cache.DefaultSkipRule("tool", []string{"read"}))
	fmt.Println("query tag:", cache.DefaultSkipRule("tool", []string{"query"}))
	// Output:
	// write tag: true
	// danger tag: true
	// UNSAFE tag (case-insensitive): true
	// read tag: false
	// query tag: false
}

func ExampleValidateKey() {
	// Valid keys
	fmt.Println("normal key:", cache.ValidateKey("my-key") == nil)
	fmt.Println("with colons:", cache.ValidateKey("cache:tool:hash") == nil)

	// Invalid keys
	fmt.Println("empty:", errors.Is(cache.ValidateKey(""), cache.ErrInvalidKey))
	fmt.Println("whitespace:", errors.Is(cache.ValidateKey("   "), cache.ErrInvalidKey))
	fmt.Println("with newline:", errors.Is(cache.ValidateKey("key\nvalue"), cache.ErrInvalidKey))

	// Too long
	longKey := make([]byte, 600)
	for i := range longKey {
		longKey[i] = 'x'
	}
	fmt.Println("too long:", errors.Is(cache.ValidateKey(string(longKey)), cache.ErrKeyTooLong))
	// Output:
	// normal key: true
	// with colons: true
	// empty: true
	// whitespace: true
	// with newline: true
	// too long: true
}
