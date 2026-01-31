package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Keyer generates deterministic cache keys from tool execution parameters.
//
// Contract:
// - Determinism: same inputs must produce same key, regardless of map iteration order.
// - Concurrency: implementations must be safe for concurrent use.
type Keyer interface {
	// Key generates a cache key from tool ID and input.
	Key(toolID string, input any) (string, error)
}

// DefaultKeyer generates SHA-256 based cache keys.
type DefaultKeyer struct{}

// NewDefaultKeyer creates a new default keyer.
func NewDefaultKeyer() *DefaultKeyer {
	return &DefaultKeyer{}
}

// Key generates a deterministic cache key.
// Format: cache:<toolID>:<hash>
// where hash is the first 16 characters of SHA-256(canonical JSON(input))
func (k *DefaultKeyer) Key(toolID string, input any) (string, error) {
	// Canonicalize input to ensure deterministic serialization
	canonical, err := canonicalize(input)
	if err != nil {
		return "", fmt.Errorf("cache: failed to canonicalize input: %w", err)
	}

	// Hash the canonical representation
	hash := sha256.Sum256(canonical)
	hashStr := hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars

	return fmt.Sprintf("cache:%s:%s", toolID, hashStr), nil
}

// canonicalize produces a deterministic JSON representation of the input.
// Maps are sorted by key to ensure consistent ordering.
func canonicalize(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	// For maps, sort keys for determinism
	switch val := v.(type) {
	case map[string]any:
		return canonicalizeMap(val)
	case []any:
		return canonicalizeSlice(val)
	default:
		// For other types, use standard JSON encoding
		return json.Marshal(v)
	}
}

func canonicalizeMap(m map[string]any) ([]byte, error) {
	// Sort keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build ordered JSON object
	result := []byte("{")
	for i, k := range keys {
		if i > 0 {
			result = append(result, ',')
		}

		// Key
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		result = append(result, keyBytes...)
		result = append(result, ':')

		// Value (recursively canonicalize)
		valBytes, err := canonicalize(m[k])
		if err != nil {
			return nil, err
		}
		result = append(result, valBytes...)
	}
	result = append(result, '}')

	return result, nil
}

func canonicalizeSlice(s []any) ([]byte, error) {
	result := []byte("[")
	for i, v := range s {
		if i > 0 {
			result = append(result, ',')
		}

		valBytes, err := canonicalize(v)
		if err != nil {
			return nil, err
		}
		result = append(result, valBytes...)
	}
	result = append(result, ']')

	return result, nil
}

// Ensure DefaultKeyer implements Keyer
var _ Keyer = (*DefaultKeyer)(nil)
