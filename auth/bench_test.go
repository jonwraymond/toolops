package auth

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkAPIKeyAuthenticator_Authenticate measures API key validation.
func BenchmarkAPIKeyAuthenticator_Authenticate(b *testing.B) {
	store := NewMemoryAPIKeyStore()
	hash := HashAPIKey("test-api-key")
	_ = store.Add(&APIKeyInfo{
		ID:        "key-1",
		KeyHash:   hash,
		Principal: "user@example.com",
		TenantID:  "tenant-1",
		Roles:     []string{"admin"},
	})

	auth := NewAPIKeyAuthenticator(APIKeyConfig{}, store)
	ctx := context.Background()
	req := &AuthRequest{
		Headers: map[string][]string{
			"X-API-Key": {"test-api-key"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.Authenticate(ctx, req)
	}
}

// BenchmarkAPIKeyAuthenticator_Supports measures support check.
func BenchmarkAPIKeyAuthenticator_Supports(b *testing.B) {
	store := NewMemoryAPIKeyStore()
	auth := NewAPIKeyAuthenticator(APIKeyConfig{}, store)
	ctx := context.Background()
	req := &AuthRequest{
		Headers: map[string][]string{
			"X-API-Key": {"some-key"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = auth.Supports(ctx, req)
	}
}

// BenchmarkHashAPIKey measures key hashing.
func BenchmarkHashAPIKey(b *testing.B) {
	value := "example-key-test-12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashAPIKey(value)
	}
}

// BenchmarkMemoryAPIKeyStore_Lookup measures store lookup.
func BenchmarkMemoryAPIKeyStore_Lookup(b *testing.B) {
	store := NewMemoryAPIKeyStore()
	for i := 0; i < 100; i++ {
		hash := HashAPIKey(fmt.Sprintf("key-%d", i))
		_ = store.Add(&APIKeyInfo{
			ID:        fmt.Sprintf("key-%d", i),
			KeyHash:   hash,
			Principal: fmt.Sprintf("user-%d", i),
		})
	}

	ctx := context.Background()
	targetHash := HashAPIKey("key-50")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Lookup(ctx, targetHash)
	}
}

// BenchmarkMemoryAPIKeyStore_Concurrent measures concurrent lookups.
func BenchmarkMemoryAPIKeyStore_Concurrent(b *testing.B) {
	store := NewMemoryAPIKeyStore()
	hashes := make([]string, 100)
	for i := 0; i < 100; i++ {
		hash := HashAPIKey(fmt.Sprintf("key-%d", i))
		hashes[i] = hash
		_ = store.Add(&APIKeyInfo{
			ID:        fmt.Sprintf("key-%d", i),
			KeyHash:   hash,
			Principal: fmt.Sprintf("user-%d", i),
		})
	}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = store.Lookup(ctx, hashes[i%100])
			i++
		}
	})
}

// BenchmarkSimpleRBACAuthorizer_Authorize measures RBAC authorization.
func BenchmarkSimpleRBACAuthorizer_Authorize(b *testing.B) {
	rbac := NewSimpleRBACAuthorizer(RBACConfig{
		Roles: map[string]RoleConfig{
			"admin": {
				AllowedTools:   []string{"*"},
				AllowedActions: []string{"*"},
			},
			"user": {
				AllowedTools:   []string{"read_*", "list_*"},
				AllowedActions: []string{"call"},
			},
		},
	})

	ctx := context.Background()
	req := &AuthzRequest{
		Subject:  &Identity{Principal: "user1", Roles: []string{"user"}},
		Resource: "tool:read_file",
		Action:   "call",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rbac.Authorize(ctx, req)
	}
}

// BenchmarkSimpleRBACAuthorizer_WithInheritance measures RBAC with role inheritance.
func BenchmarkSimpleRBACAuthorizer_WithInheritance(b *testing.B) {
	rbac := NewSimpleRBACAuthorizer(RBACConfig{
		Roles: map[string]RoleConfig{
			"reader": {
				AllowedTools: []string{"read_*"},
			},
			"writer": {
				Inherits:     []string{"reader"},
				AllowedTools: []string{"write_*"},
			},
			"admin": {
				Inherits:     []string{"writer"},
				AllowedTools: []string{"*"},
			},
		},
	})

	ctx := context.Background()
	req := &AuthzRequest{
		Subject:  &Identity{Principal: "admin1", Roles: []string{"admin"}},
		Resource: "tool:read_file",
		Action:   "call",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rbac.Authorize(ctx, req)
	}
}

// BenchmarkCompositeAuthenticator_Authenticate measures composite auth.
func BenchmarkCompositeAuthenticator_Authenticate(b *testing.B) {
	store := NewMemoryAPIKeyStore()
	hash := HashAPIKey("api-key")
	_ = store.Add(&APIKeyInfo{
		ID:        "key-1",
		KeyHash:   hash,
		Principal: "user",
	})

	apiKey := NewAPIKeyAuthenticator(APIKeyConfig{}, store)
	jwt := NewJWTAuthenticator(JWTConfig{}, NewStaticKeyProvider([]byte("secret")))

	composite := NewCompositeAuthenticator(apiKey, jwt)

	ctx := context.Background()
	req := &AuthRequest{
		Headers: map[string][]string{
			"X-API-Key": {"api-key"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = composite.Authenticate(ctx, req)
	}
}

// BenchmarkCompositeAuthenticator_Supports measures composite support check.
func BenchmarkCompositeAuthenticator_Supports(b *testing.B) {
	store := NewMemoryAPIKeyStore()
	apiKey := NewAPIKeyAuthenticator(APIKeyConfig{}, store)
	jwt := NewJWTAuthenticator(JWTConfig{}, NewStaticKeyProvider([]byte("secret")))

	composite := NewCompositeAuthenticator(apiKey, jwt)

	ctx := context.Background()
	req := &AuthRequest{
		Headers: map[string][]string{
			"X-API-Key": {"some-key"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = composite.Supports(ctx, req)
	}
}

// BenchmarkWithIdentity measures context identity attachment.
func BenchmarkWithIdentity(b *testing.B) {
	ctx := context.Background()
	identity := &Identity{
		Principal: "user@example.com",
		TenantID:  "tenant-1",
		Roles:     []string{"admin", "user"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithIdentity(ctx, identity)
	}
}

// BenchmarkIdentityFromContext measures context identity retrieval.
func BenchmarkIdentityFromContext(b *testing.B) {
	identity := &Identity{Principal: "user"}
	ctx := WithIdentity(context.Background(), identity)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IdentityFromContext(ctx)
	}
}

// BenchmarkIdentity_HasRole measures role checking.
func BenchmarkIdentity_HasRole(b *testing.B) {
	identity := &Identity{
		Principal: "user",
		Roles:     []string{"admin", "user", "reader", "writer", "moderator"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = identity.HasRole("moderator") // Last role for worst case
	}
}

// BenchmarkIdentity_HasPermission measures permission checking.
func BenchmarkIdentity_HasPermission(b *testing.B) {
	identity := &Identity{
		Principal:   "user",
		Permissions: []string{"read", "write", "delete", "admin", "execute"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = identity.HasPermission("execute") // Last permission for worst case
	}
}

// BenchmarkIdentity_IsExpired measures expiry checking.
func BenchmarkIdentity_IsExpired(b *testing.B) {
	identity := &Identity{
		Principal: "user",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = identity.IsExpired()
	}
}

// BenchmarkAuthRequest_GetHeader measures header retrieval.
func BenchmarkAuthRequest_GetHeader(b *testing.B) {
	req := &AuthRequest{
		Headers: map[string][]string{
			"Authorization": {"Bearer token"},
			"X-API-Key":     {"key"},
			"Content-Type":  {"application/json"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.GetHeader("Authorization")
	}
}

// BenchmarkAllowAllAuthorizer measures allow all authorization.
func BenchmarkAllowAllAuthorizer(b *testing.B) {
	authz := AllowAllAuthorizer{}
	ctx := context.Background()
	req := &AuthzRequest{
		Subject:  &Identity{Principal: "user"},
		Resource: "tool:action",
		Action:   "call",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = authz.Authorize(ctx, req)
	}
}

// BenchmarkDenyAllAuthorizer measures deny all authorization.
func BenchmarkDenyAllAuthorizer(b *testing.B) {
	authz := DenyAllAuthorizer{}
	ctx := context.Background()
	req := &AuthzRequest{
		Subject:  &Identity{Principal: "user"},
		Resource: "tool:action",
		Action:   "call",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = authz.Authorize(ctx, req)
	}
}

// BenchmarkAuthzRequest_ToolName measures tool name extraction.
func BenchmarkAuthzRequest_ToolName(b *testing.B) {
	req := &AuthzRequest{
		Resource: "tool:search_files",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.ToolName()
	}
}

// BenchmarkConstantTimeCompare measures constant-time comparison.
func BenchmarkConstantTimeCompare(b *testing.B) {
	a := "abcdefghijklmnopqrstuvwxyz123456"
	bStr := "abcdefghijklmnopqrstuvwxyz123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConstantTimeCompare(a, bStr)
	}
}
