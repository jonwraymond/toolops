package cache

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockExecutor tracks calls and returns configured results
type mockExecutor struct {
	calls  int
	result []byte
	err    error
}

func (m *mockExecutor) execute(_ context.Context, _ string, _ any) ([]byte, error) {
	m.calls++
	return m.result, m.err
}

func TestMiddleware_CacheHit(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	keyer := NewDefaultKeyer()
	policy := DefaultPolicy()
	mw := NewCacheMiddleware(cache, keyer, policy, nil)

	executor := &mockExecutor{result: []byte(`{"status":"ok"}`)}

	ctx := context.Background()
	toolID := "test-tool"
	input := map[string]any{"query": "hello"}
	tags := []string{"read"}

	// First call - should execute
	result1, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call, got %d", executor.calls)
	}
	if string(result1) != `{"status":"ok"}` {
		t.Errorf("unexpected result: %s", result1)
	}

	// Second call - should return cached, executor NOT called
	result2, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if executor.calls != 1 {
		t.Errorf("expected executor to NOT be called again, got %d calls", executor.calls)
	}
	if string(result2) != `{"status":"ok"}` {
		t.Errorf("unexpected cached result: %s", result2)
	}
}

func TestMiddleware_CacheMiss(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	keyer := NewDefaultKeyer()
	policy := DefaultPolicy()
	mw := NewCacheMiddleware(cache, keyer, policy, nil)

	executor := &mockExecutor{result: []byte(`{"data":"value"}`)}

	ctx := context.Background()
	toolID := "test-tool"
	tags := []string{"read"}

	// First call with input A
	inputA := map[string]any{"query": "hello"}
	_, err := mw.Execute(ctx, toolID, inputA, tags, executor.execute)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call, got %d", executor.calls)
	}

	// Second call with different input B - should be cache miss
	inputB := map[string]any{"query": "world"}
	_, err = mw.Execute(ctx, toolID, inputB, tags, executor.execute)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if executor.calls != 2 {
		t.Errorf("expected 2 calls (cache miss), got %d", executor.calls)
	}
}

func TestMiddleware_SkipUnsafeTags(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	keyer := NewDefaultKeyer()
	policy := DefaultPolicy()
	mw := NewCacheMiddleware(cache, keyer, policy, nil)

	executor := &mockExecutor{result: []byte(`{"written":true}`)}

	ctx := context.Background()
	toolID := "write-tool"
	input := map[string]any{"data": "test"}
	tags := []string{"write"} // unsafe tag

	// First call - should execute but NOT cache
	_, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call, got %d", executor.calls)
	}

	// Second call - should execute again (not cached due to unsafe tag)
	_, err = mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if executor.calls != 2 {
		t.Errorf("expected 2 calls (skip caching for unsafe), got %d", executor.calls)
	}
}

func TestMiddleware_AllUnsafeTags(t *testing.T) {
	unsafeTags := []string{"write", "danger", "unsafe", "mutation", "delete"}

	for _, unsafeTag := range unsafeTags {
		t.Run(unsafeTag, func(t *testing.T) {
			cache := NewMemoryCache(DefaultPolicy())
			keyer := NewDefaultKeyer()
			policy := DefaultPolicy()
			mw := NewCacheMiddleware(cache, keyer, policy, nil)

			executor := &mockExecutor{result: []byte(`{"ok":true}`)}

			ctx := context.Background()
			toolID := "tool-" + unsafeTag
			input := map[string]any{"x": 1}
			tags := []string{unsafeTag}

			// First call
			_, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
			if err != nil {
				t.Fatalf("first call failed: %v", err)
			}

			// Second call - should execute again (not cached)
			_, err = mw.Execute(ctx, toolID, input, tags, executor.execute)
			if err != nil {
				t.Fatalf("second call failed: %v", err)
			}

			if executor.calls != 2 {
				t.Errorf("tag %q: expected 2 calls (skip caching), got %d", unsafeTag, executor.calls)
			}
		})
	}
}

func TestMiddleware_AllowUnsafeOverride(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	keyer := NewDefaultKeyer()
	policy := Policy{
		DefaultTTL:  5 * time.Minute,
		MaxTTL:      1 * time.Hour,
		AllowUnsafe: true, // Override: allow caching unsafe tools
	}
	mw := NewCacheMiddleware(cache, keyer, policy, nil)

	executor := &mockExecutor{result: []byte(`{"written":true}`)}

	ctx := context.Background()
	toolID := "write-tool"
	input := map[string]any{"data": "test"}
	tags := []string{"write"} // normally unsafe, but AllowUnsafe=true

	// First call - should execute and cache
	_, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call, got %d", executor.calls)
	}

	// Second call - should return cached (AllowUnsafe=true)
	_, err = mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call (cached despite unsafe tag), got %d", executor.calls)
	}
}

func TestMiddleware_CustomSkipRule(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	keyer := NewDefaultKeyer()
	policy := DefaultPolicy()

	// Custom skip rule: skip tools with "internal-" prefix
	customSkipRule := func(toolID string, _ []string) bool {
		return strings.HasPrefix(toolID, "internal-")
	}

	mw := NewCacheMiddleware(cache, keyer, policy, customSkipRule)

	executor := &mockExecutor{result: []byte(`{"internal":true}`)}

	ctx := context.Background()
	input := map[string]any{"x": 1}
	tags := []string{"read"} // safe tag

	// Tool with internal- prefix should skip caching
	toolID := "internal-secret-tool"
	_, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	_, err = mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if executor.calls != 2 {
		t.Errorf("expected 2 calls (custom skip rule), got %d", executor.calls)
	}

	// Tool without internal- prefix should cache
	executor2 := &mockExecutor{result: []byte(`{"public":true}`)}
	toolID2 := "public-tool"

	_, err = mw.Execute(ctx, toolID2, input, tags, executor2.execute)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	_, err = mw.Execute(ctx, toolID2, input, tags, executor2.execute)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if executor2.calls != 1 {
		t.Errorf("expected 1 call (cached), got %d", executor2.calls)
	}
}

func TestMiddleware_ExecutorError(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	keyer := NewDefaultKeyer()
	policy := DefaultPolicy()
	mw := NewCacheMiddleware(cache, keyer, policy, nil)

	expectedErr := errors.New("execution failed")
	executor := &mockExecutor{result: nil, err: expectedErr}

	ctx := context.Background()
	toolID := "failing-tool"
	input := map[string]any{"x": 1}
	tags := []string{"read"}

	// First call - should return error
	_, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call, got %d", executor.calls)
	}

	// Second call - should execute again (errors are NOT cached)
	_, err = mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err == nil {
		t.Fatal("expected error on second call, got nil")
	}
	if executor.calls != 2 {
		t.Errorf("expected 2 calls (errors not cached), got %d", executor.calls)
	}
}

func TestMiddleware_NilResult(t *testing.T) {
	cache := NewMemoryCache(DefaultPolicy())
	keyer := NewDefaultKeyer()
	policy := DefaultPolicy()
	mw := NewCacheMiddleware(cache, keyer, policy, nil)

	executor := &mockExecutor{result: nil, err: nil} // nil result, no error

	ctx := context.Background()
	toolID := "nil-result-tool"
	input := map[string]any{"x": 1}
	tags := []string{"read"}

	// First call - should execute and cache nil result
	result1, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if result1 != nil {
		t.Errorf("expected nil result, got %v", result1)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call, got %d", executor.calls)
	}

	// Second call - should return cached nil result
	result2, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if result2 != nil {
		t.Errorf("expected nil cached result, got %v", result2)
	}
	if executor.calls != 1 {
		t.Errorf("expected 1 call (nil result cached), got %d", executor.calls)
	}
}

func TestMiddleware_CaseSensitiveTags(t *testing.T) {
	testCases := []struct {
		tag      string
		expected int // expected executor calls after 2 Execute calls
	}{
		{"WRITE", 2},    // uppercase - should skip
		{"Write", 2},    // mixed case - should skip
		{"wRiTe", 2},    // mixed case - should skip
		{"DANGER", 2},   // uppercase - should skip
		{"Unsafe", 2},   // mixed case - should skip
		{"MUTATION", 2}, // uppercase - should skip
		{"DELETE", 2},   // uppercase - should skip
	}

	for _, tc := range testCases {
		t.Run(tc.tag, func(t *testing.T) {
			cache := NewMemoryCache(DefaultPolicy())
			keyer := NewDefaultKeyer()
			policy := DefaultPolicy()
			mw := NewCacheMiddleware(cache, keyer, policy, nil)

			executor := &mockExecutor{result: []byte(`{"ok":true}`)}

			ctx := context.Background()
			toolID := "test-tool"
			input := map[string]any{"x": 1}
			tags := []string{tc.tag}

			// First call
			_, err := mw.Execute(ctx, toolID, input, tags, executor.execute)
			if err != nil {
				t.Fatalf("first call failed: %v", err)
			}

			// Second call
			_, err = mw.Execute(ctx, toolID, input, tags, executor.execute)
			if err != nil {
				t.Fatalf("second call failed: %v", err)
			}

			if executor.calls != tc.expected {
				t.Errorf("tag %q: expected %d calls, got %d", tc.tag, tc.expected, executor.calls)
			}
		})
	}
}

func TestDefaultSkipRule(t *testing.T) {
	testCases := []struct {
		name     string
		toolID   string
		tags     []string
		expected bool // true = skip caching
	}{
		// Unsafe tags should skip
		{"write tag", "tool", []string{"write"}, true},
		{"danger tag", "tool", []string{"danger"}, true},
		{"unsafe tag", "tool", []string{"unsafe"}, true},
		{"mutation tag", "tool", []string{"mutation"}, true},
		{"delete tag", "tool", []string{"delete"}, true},

		// Case insensitive
		{"WRITE uppercase", "tool", []string{"WRITE"}, true},
		{"Write mixed", "tool", []string{"Write"}, true},
		{"DANGER uppercase", "tool", []string{"DANGER"}, true},

		// Safe tags should NOT skip
		{"read tag", "tool", []string{"read"}, false},
		{"query tag", "tool", []string{"query"}, false},
		{"empty tags", "tool", []string{}, false},
		{"nil tags", "tool", nil, false},

		// Multiple tags - one unsafe should skip
		{"mixed tags with write", "tool", []string{"read", "write"}, true},
		{"mixed tags with danger", "tool", []string{"query", "danger"}, true},

		// Multiple safe tags
		{"multiple safe tags", "tool", []string{"read", "query", "list"}, false},

		// Tool ID doesn't affect default rule
		{"write-tool with safe tags", "write-tool", []string{"read"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DefaultSkipRule(tc.toolID, tc.tags)
			if result != tc.expected {
				t.Errorf("DefaultSkipRule(%q, %v) = %v, want %v",
					tc.toolID, tc.tags, result, tc.expected)
			}
		})
	}
}
