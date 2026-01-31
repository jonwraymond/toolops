package auth

import (
	"context"
	"testing"
)

func TestNewAPIKeyAuthenticator(t *testing.T) {
	config := APIKeyConfig{
		HeaderName: "X-API-Key",
	}
	store := NewMemoryAPIKeyStore()

	auth := NewAPIKeyAuthenticator(config, store)

	if auth.Name() != "api_key" {
		t.Errorf("Name() = %v, want api_key", auth.Name())
	}
}

func TestAPIKeyAuthenticator_Supports(t *testing.T) {
	config := APIKeyConfig{
		HeaderName: "X-API-Key",
	}
	store := NewMemoryAPIKeyStore()
	auth := NewAPIKeyAuthenticator(config, store)

	tests := []struct {
		name    string
		headers map[string][]string
		want    bool
	}{
		{
			name:    "no api key header",
			headers: map[string][]string{},
			want:    false,
		},
		{
			name:    "has api key header",
			headers: map[string][]string{"X-API-Key": {"key123"}},
			want:    true,
		},
		{
			name:    "wrong header",
			headers: map[string][]string{"Authorization": {"Bearer token"}},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AuthRequest{Headers: tt.headers}
			if got := auth.Supports(context.Background(), req); got != tt.want {
				t.Errorf("Supports() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKeyAuthenticator_Authenticate(t *testing.T) {
	config := APIKeyConfig{
		HeaderName:    "X-API-Key",
		HashAlgorithm: "sha256",
	}
	store := NewMemoryAPIKeyStore()

	// Add a test key
	keyInfo := &APIKeyInfo{
		ID:        "key1",
		KeyHash:   HashAPIKey("test-api-key"),
		Principal: "user123",
		TenantID:  "tenant1",
		Roles:     []string{"admin"},
	}
	_ = store.Add(keyInfo)

	auth := NewAPIKeyAuthenticator(config, store)

	t.Run("valid key", func(t *testing.T) {
		req := &AuthRequest{
			Headers: map[string][]string{"X-API-Key": {"test-api-key"}},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if !result.Authenticated {
			t.Error("Authenticated = false, want true")
		}
		if result.Identity == nil {
			t.Fatal("Identity = nil")
		}
		if result.Identity.Principal != "user123" {
			t.Errorf("Principal = %v, want user123", result.Identity.Principal)
		}
		if result.Identity.TenantID != "tenant1" {
			t.Errorf("TenantID = %v, want tenant1", result.Identity.TenantID)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		req := &AuthRequest{
			Headers: map[string][]string{"X-API-Key": {"wrong-key"}},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if result.Authenticated {
			t.Error("Authenticated = true for invalid key")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		req := &AuthRequest{
			Headers: map[string][]string{},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if result.Authenticated {
			t.Error("Authenticated = true for missing key")
		}
	})
}

func TestMemoryAPIKeyStore(t *testing.T) {
	store := NewMemoryAPIKeyStore()

	keyInfo := &APIKeyInfo{
		ID:        "key1",
		KeyHash:   "hash123",
		Principal: "user123",
	}

	t.Run("add and lookup", func(t *testing.T) {
		err := store.Add(keyInfo)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		got, err := store.Lookup(context.Background(), "hash123")
		if err != nil {
			t.Fatalf("Lookup() error = %v", err)
		}
		if got == nil {
			t.Fatal("Lookup() = nil")
		}
		if got.Principal != "user123" {
			t.Errorf("Principal = %v, want user123", got.Principal)
		}
	})

	t.Run("lookup not found", func(t *testing.T) {
		got, err := store.Lookup(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("Lookup() error = %v", err)
		}
		if got != nil {
			t.Errorf("Lookup() = %v, want nil", got)
		}
	})

	t.Run("remove", func(t *testing.T) {
		err := store.Remove("hash123")
		if err != nil {
			t.Fatalf("Remove() error = %v", err)
		}

		got, _ := store.Lookup(context.Background(), "hash123")
		if got != nil {
			t.Error("Key should be removed")
		}
	})
}

func TestHashAPIKey(t *testing.T) {
	key := "test-key"
	hash := HashAPIKey(key)

	if hash == "" {
		t.Error("HashAPIKey returned empty string")
	}
	if hash == key {
		t.Error("HashAPIKey should not return the raw key")
	}

	// Same input should produce same output
	hash2 := HashAPIKey(key)
	if hash != hash2 {
		t.Error("HashAPIKey should be deterministic")
	}

	// Different inputs should produce different outputs
	hash3 := HashAPIKey("another-key")
	if hash == hash3 {
		t.Error("HashAPIKey should produce different hashes for different keys")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want bool
	}{
		{"hello", "hello", true},
		{"hello", "world", false},
		{"", "", true},
		{"a", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			if got := ConstantTimeCompare(tt.a, tt.b); got != tt.want {
				t.Errorf("ConstantTimeCompare(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
