package auth

import (
	"context"
	"testing"
)

func TestIdentityContext(t *testing.T) {
	ctx := context.Background()

	// Test with no identity
	if got := IdentityFromContext(ctx); got != nil {
		t.Errorf("IdentityFromContext() on empty context = %v, want nil", got)
	}

	// Test with identity
	identity := &Identity{Principal: "user123", Roles: []string{"admin"}}
	ctx = WithIdentity(ctx, identity)

	got := IdentityFromContext(ctx)
	if got == nil {
		t.Fatal("IdentityFromContext() = nil, want identity")
	}
	if got.Principal != "user123" {
		t.Errorf("Principal = %v, want user123", got.Principal)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin]", got.Roles)
	}
}

func TestPrincipalFromContext(t *testing.T) {
	ctx := context.Background()

	// No identity
	if got := PrincipalFromContext(ctx); got != "" {
		t.Errorf("PrincipalFromContext() = %v, want empty", got)
	}

	// With identity
	ctx = WithIdentity(ctx, &Identity{Principal: "user123"})
	if got := PrincipalFromContext(ctx); got != "user123" {
		t.Errorf("PrincipalFromContext() = %v, want user123", got)
	}
}

func TestTenantIDFromContext(t *testing.T) {
	ctx := context.Background()

	// No identity
	if got := TenantIDFromContext(ctx); got != "" {
		t.Errorf("TenantIDFromContext() = %v, want empty", got)
	}

	// With identity
	ctx = WithIdentity(ctx, &Identity{TenantID: "tenant1"})
	if got := TenantIDFromContext(ctx); got != "tenant1" {
		t.Errorf("TenantIDFromContext() = %v, want tenant1", got)
	}
}

func TestHeadersContext(t *testing.T) {
	ctx := context.Background()

	// Test with no headers
	if got := HeadersFromContext(ctx); got != nil {
		t.Errorf("HeadersFromContext() on empty context = %v, want nil", got)
	}

	// Test with headers
	headers := map[string][]string{
		"Authorization": {"Bearer token123"},
		"X-API-Key":     {"key456"},
	}
	ctx = WithHeaders(ctx, headers)

	got := HeadersFromContext(ctx)
	if got == nil {
		t.Fatal("HeadersFromContext() = nil, want headers")
	}

	authValues := got["Authorization"]
	if len(authValues) != 1 || authValues[0] != "Bearer token123" {
		t.Errorf("Authorization = %v, want [Bearer token123]", authValues)
	}

	apiKeyValues := got["X-API-Key"]
	if len(apiKeyValues) != 1 || apiKeyValues[0] != "key456" {
		t.Errorf("X-API-Key = %v, want [key456]", apiKeyValues)
	}
}

func TestGetHeader(t *testing.T) {
	ctx := context.Background()

	// Test with no headers in context
	if got := GetHeader(ctx, "Authorization"); got != "" {
		t.Errorf("GetHeader() on empty context = %v, want empty", got)
	}

	// Test with headers
	headers := map[string][]string{
		"Authorization": {"Bearer token123"},
	}
	ctx = WithHeaders(ctx, headers)

	if got := GetHeader(ctx, "Authorization"); got != "Bearer token123" {
		t.Errorf("GetHeader() = %v, want Bearer token123", got)
	}

	// Test missing header
	if got := GetHeader(ctx, "X-Missing"); got != "" {
		t.Errorf("GetHeader() for missing = %v, want empty", got)
	}

	// Test empty values
	ctx2 := WithHeaders(context.Background(), map[string][]string{"X-Empty": {}})
	if got := GetHeader(ctx2, "X-Empty"); got != "" {
		t.Errorf("GetHeader() for empty values = %v, want empty", got)
	}
}
