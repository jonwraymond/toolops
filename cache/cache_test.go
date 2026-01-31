package cache

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestCacheKey_Validation tests key validation rules.
func TestCacheKey_Validation(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"empty key", "", ErrInvalidKey},
		{"valid key", "cache:ns:tool:abc123", nil},
		{"too long", strings.Repeat("x", MaxKeyLength+1), ErrKeyTooLong},
		{"contains newline", "key\nwith\nnewlines", ErrInvalidKey},
		{"contains carriage return", "key\rwith\rreturns", ErrInvalidKey},
		{"whitespace only", "   ", ErrInvalidKey},
		{"max length exactly", strings.Repeat("x", MaxKeyLength), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateKey(%q) = %v, want nil", tt.key, err)
				}
			} else {
				if err != tt.wantErr {
					t.Errorf("ValidateKey(%q) = %v, want %v", tt.key, err, tt.wantErr)
				}
			}
		})
	}
}

// TestCacheInterface_CompileCheck verifies the Cache interface contract.
// This is a compile-time check enforced by implementing a mock.
func TestCacheInterface_CompileCheck(t *testing.T) {
	// mockCache implements Cache to verify interface contract at compile time
	var _ Cache = (*mockCache)(nil)
}

// mockCache is a test double that implements Cache interface.
type mockCache struct{}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, bool) {
	return nil, false
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

// TestSentinelErrors verifies sentinel errors are distinct and have expected messages.
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrNilCache", ErrNilCache, "cache: cache is nil"},
		{"ErrInvalidKey", ErrInvalidKey, "cache: key is invalid"},
		{"ErrKeyTooLong", ErrKeyTooLong, "cache: key exceeds max length"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("%s is nil", tt.name)
			}
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("%s.Error() = %q, want %q", tt.name, got, tt.wantMsg)
			}
		})
	}

	// Verify errors are distinct
	if ErrNilCache == ErrInvalidKey {
		t.Error("ErrNilCache and ErrInvalidKey should be distinct")
	}
	if ErrInvalidKey == ErrKeyTooLong {
		t.Error("ErrInvalidKey and ErrKeyTooLong should be distinct")
	}
	if ErrNilCache == ErrKeyTooLong {
		t.Error("ErrNilCache and ErrKeyTooLong should be distinct")
	}
}

// TestMaxKeyLength verifies the constant value.
func TestMaxKeyLength(t *testing.T) {
	if MaxKeyLength != 512 {
		t.Errorf("MaxKeyLength = %d, want 512", MaxKeyLength)
	}
}
