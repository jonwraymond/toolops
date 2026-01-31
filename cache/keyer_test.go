package cache

import (
	"strings"
	"testing"
)

func TestKeyer_DeterministicForMaps(t *testing.T) {
	keyer := NewDefaultKeyer()

	// Same content, different insertion order
	map1 := map[string]any{"b": 2, "a": 1, "c": 3}
	map2 := map[string]any{"a": 1, "c": 3, "b": 2}
	map3 := map[string]any{"c": 3, "b": 2, "a": 1}

	key1, err := keyer.Key("test-tool", map1)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	key2, err := keyer.Key("test-tool", map2)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	key3, err := keyer.Key("test-tool", map3)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	if key1 != key2 {
		t.Errorf("Keys should be equal for same content:\n  key1=%s\n  key2=%s", key1, key2)
	}
	if key2 != key3 {
		t.Errorf("Keys should be equal for same content:\n  key2=%s\n  key3=%s", key2, key3)
	}
}

func TestKeyer_ArrayOrderPreserved(t *testing.T) {
	keyer := NewDefaultKeyer()

	// Different array order should produce different keys
	input1 := map[string]any{"items": []any{1, 2, 3}}
	input2 := map[string]any{"items": []any{3, 2, 1}}

	key1, err := keyer.Key("test-tool", input1)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	key2, err := keyer.Key("test-tool", input2)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	if key1 == key2 {
		t.Errorf("Keys should differ for different array order:\n  key1=%s\n  key2=%s", key1, key2)
	}
}

func TestKeyer_SameInputsSameKey(t *testing.T) {
	keyer := NewDefaultKeyer()

	input := map[string]any{"query": "test", "limit": 10}

	// Call multiple times
	keys := make([]string, 5)
	for i := 0; i < 5; i++ {
		key, err := keyer.Key("search-tool", input)
		if err != nil {
			t.Fatalf("Key() iteration %d error = %v", i, err)
		}
		keys[i] = key
	}

	// All keys should be identical
	for i := 1; i < len(keys); i++ {
		if keys[i] != keys[0] {
			t.Errorf("Key should be consistent across calls:\n  keys[0]=%s\n  keys[%d]=%s", keys[0], i, keys[i])
		}
	}
}

func TestKeyer_DifferentToolsDifferentKeys(t *testing.T) {
	keyer := NewDefaultKeyer()

	input := map[string]any{"query": "test"}

	key1, err := keyer.Key("tool-a", input)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	key2, err := keyer.Key("tool-b", input)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	if key1 == key2 {
		t.Errorf("Keys should differ for different tools:\n  key1=%s\n  key2=%s", key1, key2)
	}
}

func TestKeyer_KeyFormat(t *testing.T) {
	keyer := NewDefaultKeyer()

	input := map[string]any{"test": "value"}
	toolID := "my-tool"

	key, err := keyer.Key(toolID, input)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	// Format: cache:<toolID>:<hash>
	// Hash should be 16 hex characters
	prefix := "cache:" + toolID + ":"
	if !strings.HasPrefix(key, prefix) {
		t.Errorf("Key should have prefix %q, got %q", prefix, key)
	}

	hash := strings.TrimPrefix(key, prefix)
	if len(hash) != 16 {
		t.Errorf("Hash should be 16 characters, got %d: %q", len(hash), hash)
	}

	// Verify hash is valid hex
	for _, c := range hash {
		isLowerHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
		if !isLowerHex {
			t.Errorf("Hash should be lowercase hex, got character %q in %q", string(c), hash)
			break
		}
	}
}

func TestKeyer_NestedMaps(t *testing.T) {
	keyer := NewDefaultKeyer()

	// Nested maps with different insertion order
	nested1 := map[string]any{
		"outer": map[string]any{
			"z": 26,
			"a": 1,
			"m": 13,
		},
		"other": "value",
	}
	nested2 := map[string]any{
		"other": "value",
		"outer": map[string]any{
			"a": 1,
			"m": 13,
			"z": 26,
		},
	}

	key1, err := keyer.Key("test-tool", nested1)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	key2, err := keyer.Key("test-tool", nested2)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	if key1 != key2 {
		t.Errorf("Keys should be equal for nested maps with same content:\n  key1=%s\n  key2=%s", key1, key2)
	}
}

func TestKeyer_NilInput(t *testing.T) {
	keyer := NewDefaultKeyer()

	// nil input should be valid and deterministic
	key1, err := keyer.Key("test-tool", nil)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	key2, err := keyer.Key("test-tool", nil)
	if err != nil {
		t.Fatalf("Key() error = %v", err)
	}

	if key1 != key2 {
		t.Errorf("Keys should be equal for nil input:\n  key1=%s\n  key2=%s", key1, key2)
	}

	// Verify format is still correct
	if !strings.HasPrefix(key1, "cache:test-tool:") {
		t.Errorf("Key should have correct prefix, got %q", key1)
	}
}

func TestKeyer_EmptyInput(t *testing.T) {
	keyer := NewDefaultKeyer()

	// Empty map vs nil should produce different keys
	emptyMap := map[string]any{}

	keyNil, err := keyer.Key("test-tool", nil)
	if err != nil {
		t.Fatalf("Key() for nil error = %v", err)
	}

	keyEmpty, err := keyer.Key("test-tool", emptyMap)
	if err != nil {
		t.Fatalf("Key() for empty map error = %v", err)
	}

	if keyNil == keyEmpty {
		t.Errorf("Keys should differ for nil vs empty map:\n  keyNil=%s\n  keyEmpty=%s", keyNil, keyEmpty)
	}
}
