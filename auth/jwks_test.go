package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewJWKSKeyProvider(t *testing.T) {
	config := JWKSConfig{
		URL:      "https://example.com/.well-known/jwks.json",
		CacheTTL: time.Hour,
	}

	provider := NewJWKSKeyProvider(config)

	if provider.config.URL != config.URL {
		t.Errorf("URL = %v, want %v", provider.config.URL, config.URL)
	}
	if provider.config.CacheTTL != time.Hour {
		t.Errorf("CacheTTL = %v, want %v", provider.config.CacheTTL, time.Hour)
	}
}

func TestNewJWKSKeyProvider_Defaults(t *testing.T) {
	config := JWKSConfig{
		URL: "https://example.com/.well-known/jwks.json",
	}

	provider := NewJWKSKeyProvider(config)

	// Default CacheTTL should be 1 hour
	if provider.config.CacheTTL != time.Hour {
		t.Errorf("Default CacheTTL = %v, want %v", provider.config.CacheTTL, time.Hour)
	}

	// Default HTTPClient should be created
	if provider.config.HTTPClient == nil {
		t.Error("Default HTTPClient should be created")
	}
}

func TestJWKSKeyProvider_GetKey(t *testing.T) {
	// Generate a test RSA key
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	// Create JWKS response
	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": "key1",
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	config := JWKSConfig{
		URL:      server.URL,
		CacheTTL: time.Hour,
	}

	provider := NewJWKSKeyProvider(config)

	t.Run("get key by ID", func(t *testing.T) {
		key, err := provider.GetKey(context.Background(), "key1")
		if err != nil {
			t.Fatalf("GetKey() error = %v", err)
		}

		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			t.Fatalf("GetKey() returned %T, want *rsa.PublicKey", key)
		}

		if rsaKey.N.Cmp(publicKey.N) != 0 {
			t.Error("Key modulus does not match")
		}
	})

	t.Run("get key without ID returns first", func(t *testing.T) {
		key, err := provider.GetKey(context.Background(), "")
		if err != nil {
			t.Fatalf("GetKey() error = %v", err)
		}

		if key == nil {
			t.Error("GetKey() = nil")
		}
	})

	t.Run("key not found", func(t *testing.T) {
		_, err := provider.GetKey(context.Background(), "nonexistent")
		if err != ErrKeyNotFound {
			t.Errorf("GetKey() error = %v, want ErrKeyNotFound", err)
		}
	})
}

func TestJWKSKeyProvider_Caching(t *testing.T) {
	callCount := 0

	// Generate a test RSA key
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": "key1",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	config := JWKSConfig{
		URL:      server.URL,
		CacheTTL: time.Hour,
	}

	provider := NewJWKSKeyProvider(config)

	// First call
	_, err := provider.GetKey(context.Background(), "key1")
	if err != nil {
		t.Fatalf("First GetKey() error = %v", err)
	}

	// Second call should use cache
	_, err = provider.GetKey(context.Background(), "key1")
	if err != nil {
		t.Fatalf("Second GetKey() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("Server called %d times, want 1 (cached)", callCount)
	}
}

func TestJWKSKeyProvider_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := JWKSConfig{
		URL:      server.URL,
		CacheTTL: time.Nanosecond, // Very short TTL to force refresh
	}

	provider := NewJWKSKeyProvider(config)

	_, err := provider.GetKey(context.Background(), "key1")
	if err == nil {
		t.Error("GetKey() should return error for server error")
	}
}

func TestJWKSKeyProvider_GracefulDegradation(t *testing.T) {
	callCount := 0

	// Generate a test RSA key
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": "key1",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount > 1 {
			// Fail on subsequent calls
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	config := JWKSConfig{
		URL:      server.URL,
		CacheTTL: time.Nanosecond, // Very short TTL to force refresh
	}

	provider := NewJWKSKeyProvider(config)

	// First call succeeds
	key1, err := provider.GetKey(context.Background(), "key1")
	if err != nil {
		t.Fatalf("First GetKey() error = %v", err)
	}

	// Wait for cache expiry
	time.Sleep(time.Millisecond)

	// Second call should use backup (graceful degradation)
	key2, err := provider.GetKey(context.Background(), "key1")
	if err != nil {
		t.Fatalf("Second GetKey() error = %v (should use backup)", err)
	}

	// Should return same key from backup
	rsaKey1 := key1.(*rsa.PublicKey)
	rsaKey2 := key2.(*rsa.PublicKey)
	if rsaKey1.N.Cmp(rsaKey2.N) != 0 {
		t.Error("Backup key should match original")
	}
}

func TestParseRSAPublicKey(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	t.Run("valid key", func(t *testing.T) {
		jwk := jwkKey{
			Kty: "RSA",
			Kid: "test",
			N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
		}

		parsed, err := parseRSAPublicKey(jwk)
		if err != nil {
			t.Fatalf("parseRSAPublicKey() error = %v", err)
		}

		if parsed.N.Cmp(publicKey.N) != 0 {
			t.Error("Parsed modulus does not match")
		}
		if parsed.E != publicKey.E {
			t.Errorf("Parsed exponent = %d, want %d", parsed.E, publicKey.E)
		}
	})

	t.Run("missing n parameter", func(t *testing.T) {
		jwk := jwkKey{
			Kty: "RSA",
			Kid: "test",
			N:   "",
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(65537).Bytes()),
		}

		_, err := parseRSAPublicKey(jwk)
		if err == nil {
			t.Error("parseRSAPublicKey() should error on missing n")
		}
	})

	t.Run("missing e parameter", func(t *testing.T) {
		jwk := jwkKey{
			Kty: "RSA",
			Kid: "test",
			N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
			E:   "",
		}

		_, err := parseRSAPublicKey(jwk)
		if err == nil {
			t.Error("parseRSAPublicKey() should error on missing e")
		}
	})

	t.Run("invalid n encoding", func(t *testing.T) {
		jwk := jwkKey{
			Kty: "RSA",
			Kid: "test",
			N:   "not-valid-base64!!!",
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(65537).Bytes()),
		}

		_, err := parseRSAPublicKey(jwk)
		if err == nil {
			t.Error("parseRSAPublicKey() should error on invalid n encoding")
		}
	})
}
