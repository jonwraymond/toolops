package cache

import (
	"context"
	"strings"
)

// ExecutorFunc is the function signature for tool execution.
type ExecutorFunc func(ctx context.Context, toolID string, input any) ([]byte, error)

// SkipRule determines whether to skip caching for a given tool.
// Returns true if caching should be skipped.
type SkipRule func(toolID string, tags []string) bool

// UnsafeTags are tags that indicate a tool has side effects and should not be cached.
var UnsafeTags = []string{"write", "danger", "unsafe", "mutation", "delete"}

// DefaultSkipRule skips caching for tools with unsafe tags.
// Tag matching is case-insensitive.
func DefaultSkipRule(_ string, tags []string) bool {
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		for _, unsafe := range UnsafeTags {
			if tagLower == unsafe {
				return true
			}
		}
	}
	return false
}

// CacheMiddleware wraps tool execution with caching.
type CacheMiddleware struct {
	cache    Cache
	keyer    Keyer
	policy   Policy
	skipRule SkipRule
}

// NewCacheMiddleware creates a new cache middleware.
// If skipRule is nil, DefaultSkipRule is used.
func NewCacheMiddleware(cache Cache, keyer Keyer, policy Policy, skipRule SkipRule) *CacheMiddleware {
	if skipRule == nil {
		skipRule = DefaultSkipRule
	}
	return &CacheMiddleware{
		cache:    cache,
		keyer:    keyer,
		policy:   policy,
		skipRule: skipRule,
	}
}

// Execute runs the tool with caching.
// On cache hit, returns cached result without calling executor.
// On cache miss, calls executor and caches the result.
// Errors are NOT cached.
func (m *CacheMiddleware) Execute(
	ctx context.Context,
	toolID string,
	input any,
	tags []string,
	executor ExecutorFunc,
) ([]byte, error) {
	// Check if caching should be skipped
	if !m.policy.AllowUnsafe && m.skipRule(toolID, tags) {
		// Skip caching - execute directly
		return executor(ctx, toolID, input)
	}

	// Check if caching is enabled by policy
	if !m.policy.ShouldCache() {
		return executor(ctx, toolID, input)
	}

	// Generate cache key
	key, err := m.keyer.Key(toolID, input)
	if err != nil {
		// Key generation failed - execute without caching
		return executor(ctx, toolID, input)
	}

	// Check cache
	if cached, ok := m.cache.Get(ctx, key); ok {
		return cached, nil
	}

	// Cache miss - execute
	result, err := executor(ctx, toolID, input)
	if err != nil {
		// Don't cache errors
		return result, err
	}

	// Cache the result
	ttl := m.policy.EffectiveTTL(0)
	if ttl > 0 {
		_ = m.cache.Set(ctx, key, result, ttl)
	}

	return result, nil
}
