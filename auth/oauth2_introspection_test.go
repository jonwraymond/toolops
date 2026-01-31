package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewOAuth2IntrospectionAuthenticator(t *testing.T) {
	config := OAuth2Config{
		IntrospectionEndpoint: "https://example.com/introspect",
		ClientID:              "client123",
		ClientSecret:          "secret456",
	}

	auth := NewOAuth2IntrospectionAuthenticator(config)

	if auth.Name() != "oauth2_introspection" {
		t.Errorf("Name() = %v, want oauth2_introspection", auth.Name())
	}
}

func TestOAuth2IntrospectionAuthenticator_Supports(t *testing.T) {
	auth := NewOAuth2IntrospectionAuthenticator(OAuth2Config{})

	tests := []struct {
		name    string
		headers map[string][]string
		want    bool
	}{
		{
			name:    "no authorization header",
			headers: map[string][]string{},
			want:    false,
		},
		{
			name:    "bearer token",
			headers: map[string][]string{"Authorization": {"Bearer token123"}},
			want:    true,
		},
		{
			name:    "wrong prefix",
			headers: map[string][]string{"Authorization": {"Basic abc"}},
			want:    false,
		},
		{
			name:    "case insensitive bearer",
			headers: map[string][]string{"Authorization": {"bearer token123"}},
			want:    true,
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

func TestOAuth2IntrospectionAuthenticator_Authenticate(t *testing.T) {
	t.Run("active token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method = %v, want POST", r.Method)
			}

			// Check content type
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %v, want application/x-www-form-urlencoded", ct)
			}

			// Check Basic auth header
			if auth := r.Header.Get("Authorization"); auth == "" {
				t.Error("Missing Authorization header")
			}

			resp := map[string]any{
				"active":    true,
				"sub":       "user123",
				"scope":     "read write",
				"exp":       time.Now().Add(time.Hour).Unix(),
				"iat":       time.Now().Unix(),
				"client_id": "client123",
				"tenant_id": "tenant1",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		config := OAuth2Config{
			IntrospectionEndpoint: server.URL,
			ClientID:              "client123",
			ClientSecret:          "secret456",
			PrincipalClaim:        "sub",
			TenantClaim:           "tenant_id",
			ScopesClaim:           "scope",
		}

		auth := NewOAuth2IntrospectionAuthenticator(config)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer test-token"}},
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
		if len(result.Identity.Permissions) != 2 {
			t.Errorf("Permissions = %v, want [read, write]", result.Identity.Permissions)
		}
	})

	t.Run("inactive token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{
				"active": false,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		config := OAuth2Config{
			IntrospectionEndpoint: server.URL,
			ClientID:              "client123",
			ClientSecret:          "secret456",
		}

		auth := NewOAuth2IntrospectionAuthenticator(config)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer inactive-token"}},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if result.Authenticated {
			t.Error("Authenticated = true for inactive token")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		auth := NewOAuth2IntrospectionAuthenticator(OAuth2Config{})

		req := &AuthRequest{
			Headers: map[string][]string{},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if result.Authenticated {
			t.Error("Authenticated = true for missing token")
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := OAuth2Config{
			IntrospectionEndpoint: server.URL,
			ClientID:              "client123",
			ClientSecret:          "secret456",
		}

		auth := NewOAuth2IntrospectionAuthenticator(config)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer test-token"}},
		}

		_, err := auth.Authenticate(context.Background(), req)
		if err == nil {
			t.Error("Authenticate() should return error for server error")
		}
	})

	t.Run("client_secret_post auth method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()

			// Check that credentials are in form body
			if r.Form.Get("client_id") != "client123" {
				t.Errorf("client_id = %v, want client123", r.Form.Get("client_id"))
			}
			if r.Form.Get("client_secret") != "secret456" {
				t.Errorf("client_secret not in form")
			}

			resp := map[string]any{
				"active": true,
				"sub":    "user123",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		config := OAuth2Config{
			IntrospectionEndpoint: server.URL,
			ClientID:              "client123",
			ClientSecret:          "secret456",
			ClientAuthMethod:      "client_secret_post",
		}

		auth := NewOAuth2IntrospectionAuthenticator(config)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer test-token"}},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if !result.Authenticated {
			t.Error("Authenticated = false, want true")
		}
	})

	t.Run("cached result", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			resp := map[string]any{
				"active": true,
				"sub":    "user123",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		config := OAuth2Config{
			IntrospectionEndpoint: server.URL,
			ClientID:              "client123",
			ClientSecret:          "secret456",
			CacheTTL:              time.Minute,
		}

		auth := NewOAuth2IntrospectionAuthenticator(config)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer cache-test-token"}},
		}

		// First call
		result1, _ := auth.Authenticate(context.Background(), req)
		if !result1.Authenticated {
			t.Error("First call: Authenticated = false")
		}

		// Second call should use cache
		result2, _ := auth.Authenticate(context.Background(), req)
		if !result2.Authenticated {
			t.Error("Second call: Authenticated = false")
		}

		if callCount != 1 {
			t.Errorf("Server called %d times, want 1 (cached)", callCount)
		}
	})
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		header    string
		wantToken string
		wantOk    bool
	}{
		{"", "", false},
		{"Bearer", "", false},
		{"Bearer ", "", false},
		{"Bearer token123", "token123", true},
		{"bearer token123", "token123", true},
		{"BEARER token123", "token123", true},
		{"Bearer  token123", "token123", true}, // extra space trimmed
		{"Basic abc123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			token, ok := extractBearerToken(tt.header)
			if ok != tt.wantOk {
				t.Errorf("extractBearerToken() ok = %v, want %v", ok, tt.wantOk)
			}
			if token != tt.wantToken {
				t.Errorf("extractBearerToken() = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

func TestHashTokenForCache(t *testing.T) {
	token := "test-token"
	hash := hashTokenForCache(token)

	if hash == "" {
		t.Error("hashTokenForCache returned empty string")
	}
	if hash == token {
		t.Error("hashTokenForCache should not return the raw token")
	}

	// Deterministic
	hash2 := hashTokenForCache(token)
	if hash != hash2 {
		t.Error("hashTokenForCache should be deterministic")
	}
}

func TestOAuth2TokenCache(t *testing.T) {
	cache := newOAuth2TokenCache()

	t.Run("get and set", func(t *testing.T) {
		identity := &Identity{Principal: "user123"}
		cache.Set("hash1", identity, time.Minute)

		got := cache.Get("hash1")
		if got == nil {
			t.Fatal("Get() = nil")
		}
		if got.Principal != "user123" {
			t.Errorf("Principal = %v, want user123", got.Principal)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		got := cache.Get("nonexistent")
		if got != nil {
			t.Errorf("Get() = %v, want nil", got)
		}
	})

	t.Run("expired entry", func(t *testing.T) {
		identity := &Identity{Principal: "expired-user"}
		cache.Set("expired-hash", identity, time.Nanosecond)

		// Wait for expiry
		time.Sleep(time.Millisecond)

		got := cache.Get("expired-hash")
		if got != nil {
			t.Errorf("Get() for expired = %v, want nil", got)
		}
	})
}
