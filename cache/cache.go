package cache

import (
	"context"
	"errors"
	"strings"
	"time"
)

// MaxKeyLength is the maximum allowed length for a cache key.
const MaxKeyLength = 512

// Sentinel errors for cache operations.
var (
	ErrNilCache   = errors.New("cache: cache is nil")
	ErrInvalidKey = errors.New("cache: key is invalid")
	ErrKeyTooLong = errors.New("cache: key exceeds max length")
)

// Cache is the interface for caching tool execution results.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: methods should honor cancellation/deadlines where applicable.
// - Errors: Get should never error; it returns (nil, false) on miss.
type Cache interface {
	// Get retrieves a cached value. Returns (nil, false) on miss.
	Get(ctx context.Context, key string) ([]byte, bool)

	// Set stores a value with the given TTL. TTL=0 means no caching.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a cached value. Idempotent - no error on miss.
	Delete(ctx context.Context, key string) error
}

// ValidateKey checks if a key is valid for caching.
func ValidateKey(key string) error {
	if key == "" || strings.TrimSpace(key) == "" {
		return ErrInvalidKey
	}
	if len(key) > MaxKeyLength {
		return ErrKeyTooLong
	}
	// Reject keys with newlines or carriage returns
	if strings.ContainsAny(key, "\n\r") {
		return ErrInvalidKey
	}
	return nil
}
