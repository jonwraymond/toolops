package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache is an in-memory cache implementation.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	policy  Policy
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache with the given policy.
func NewMemoryCache(policy Policy) *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]*cacheEntry),
		policy:  policy,
	}
}

// Get retrieves a value from the cache. Returns (nil, false) on miss or expiry.
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Check expiry
	if time.Now().After(entry.expiresAt) {
		// Expired - clean up lazily
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}

	return entry.value, true
}

// Set stores a value with the given TTL. TTL=0 means immediate expiry (no caching).
func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	// TTL=0 means don't cache
	if ttl <= 0 {
		return nil
	}

	c.mu.Lock()
	c.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()

	return nil
}

// Delete removes a value from the cache. Idempotent - no error on miss.
func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
	return nil
}

// Ensure MemoryCache implements Cache
var _ Cache = (*MemoryCache)(nil)
