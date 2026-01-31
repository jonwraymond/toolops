package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewJWTAuthenticator(t *testing.T) {
	config := JWTConfig{
		Issuer:   "test-issuer",
		Audience: "test-audience",
	}
	keyProvider := NewStaticKeyProvider([]byte("secret"))

	auth := NewJWTAuthenticator(config, keyProvider)

	if auth.Name() != "jwt" {
		t.Errorf("Name() = %v, want jwt", auth.Name())
	}
}

func TestJWTAuthenticator_Supports(t *testing.T) {
	auth := NewJWTAuthenticator(JWTConfig{}, NewStaticKeyProvider([]byte("secret")))

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
			name:    "custom header without bearer prefix",
			headers: map[string][]string{"X-Custom": {"token123"}},
			want:    false,
		},
		{
			name:    "wrong prefix",
			headers: map[string][]string{"Authorization": {"Basic abc123"}},
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

func TestJWTAuthenticator_Supports_CustomHeader(t *testing.T) {
	// Custom header with Bearer prefix
	config := JWTConfig{
		HeaderName:  "X-JWT-Token",
		TokenPrefix: "Bearer ",
	}
	auth := NewJWTAuthenticator(config, NewStaticKeyProvider([]byte("secret")))

	tests := []struct {
		name    string
		headers map[string][]string
		want    bool
	}{
		{
			name:    "custom header with bearer token",
			headers: map[string][]string{"X-JWT-Token": {"Bearer token123"}},
			want:    true,
		},
		{
			name:    "authorization header ignored",
			headers: map[string][]string{"Authorization": {"Bearer token123"}},
			want:    false,
		},
		{
			name:    "custom header without prefix",
			headers: map[string][]string{"X-JWT-Token": {"token123"}},
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

func TestJWTAuthenticator_Authenticate(t *testing.T) {
	secret := []byte("test-secret-key-at-least-32-bytes")
	keyProvider := NewStaticKeyProvider(secret)

	config := JWTConfig{
		Issuer:         "test-issuer",
		Audience:       "test-audience",
		PrincipalClaim: "sub",
		RolesClaim:     "roles",
		TenantClaim:    "tenant_id",
	}

	auth := NewJWTAuthenticator(config, keyProvider)

	t.Run("valid token", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":       "user123",
			"iss":       "test-issuer",
			"aud":       "test-audience",
			"exp":       time.Now().Add(time.Hour).Unix(),
			"iat":       time.Now().Unix(),
			"roles":     []any{"admin", "user"},
			"tenant_id": "tenant1",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString(secret)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer " + tokenStr}},
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
		if len(result.Identity.Roles) != 2 {
			t.Errorf("Roles = %v, want [admin, user]", result.Identity.Roles)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "test-issuer",
			"aud": "test-audience",
			"exp": time.Now().Add(-time.Hour).Unix(),
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString(secret)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer " + tokenStr}},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if result.Authenticated {
			t.Error("Authenticated = true for expired token")
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "wrong-issuer",
			"aud": "test-audience",
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString(secret)

		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer " + tokenStr}},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if result.Authenticated {
			t.Error("Authenticated = true for wrong issuer")
		}
	})

	t.Run("missing token", func(t *testing.T) {
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

	t.Run("invalid token format", func(t *testing.T) {
		req := &AuthRequest{
			Headers: map[string][]string{"Authorization": {"Bearer invalid.token"}},
		}

		result, err := auth.Authenticate(context.Background(), req)
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}

		if result.Authenticated {
			t.Error("Authenticated = true for invalid token")
		}
	})
}

func TestStaticKeyProvider(t *testing.T) {
	secret := []byte("my-secret")
	provider := NewStaticKeyProvider(secret)

	key, err := provider.GetKey(context.Background(), "any-key-id")
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}

	keyBytes, ok := key.([]byte)
	if !ok {
		t.Fatalf("GetKey() returned %T, want []byte", key)
	}

	if string(keyBytes) != string(secret) {
		t.Errorf("GetKey() = %v, want %v", string(keyBytes), string(secret))
	}
}
