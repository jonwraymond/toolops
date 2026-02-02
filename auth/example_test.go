package auth_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolops/auth"
)

func ExampleNewJWTAuthenticator() {
	// Create a JWT authenticator with static key
	keyProvider := auth.NewStaticKeyProvider([]byte("my-secret-key"))
	authenticator := auth.NewJWTAuthenticator(auth.JWTConfig{
		Issuer:         "https://example.com",
		Audience:       "my-api",
		PrincipalClaim: "sub",
		RolesClaim:     "roles",
	}, keyProvider)

	fmt.Println("Authenticator name:", authenticator.Name())
	// Output:
	// Authenticator name: jwt
}

func ExampleNewAPIKeyAuthenticator() {
	// Create an in-memory key store
	store := auth.NewMemoryAPIKeyStore()

	// Add an API key
	keyHash := auth.HashAPIKey("sk_live_abc123")
	_ = store.Add(&auth.APIKeyInfo{
		ID:        "key-1",
		KeyHash:   keyHash,
		Principal: "user@example.com",
		TenantID:  "tenant-1",
		Roles:     []string{"admin"},
	})

	// Create authenticator
	authenticator := auth.NewAPIKeyAuthenticator(auth.APIKeyConfig{
		HeaderName: "X-API-Key",
	}, store)

	fmt.Println("Authenticator name:", authenticator.Name())

	// Authenticate a request
	ctx := context.Background()
	req := &auth.AuthRequest{
		Headers: map[string][]string{
			"X-API-Key": {"sk_live_abc123"},
		},
	}

	result, err := authenticator.Authenticate(ctx, req)
	if err == nil && result.Authenticated {
		fmt.Println("Principal:", result.Identity.Principal)
		fmt.Println("Tenant:", result.Identity.TenantID)
	}
	// Output:
	// Authenticator name: api_key
	// Principal: user@example.com
	// Tenant: tenant-1
}

func ExampleHashAPIKey() {
	// Hash an API key for storage
	apiKey := "sk_live_abc123"
	hash := auth.HashAPIKey(apiKey)

	// Hash is deterministic
	hash2 := auth.HashAPIKey(apiKey)

	fmt.Println("Hashes match:", hash == hash2)
	fmt.Println("Hash length:", len(hash)) // SHA-256 = 64 hex chars
	// Output:
	// Hashes match: true
	// Hash length: 64
}

func ExampleNewCompositeAuthenticator() {
	// Create individual authenticators
	jwtAuth := auth.NewJWTAuthenticator(
		auth.JWTConfig{Issuer: "issuer"},
		auth.NewStaticKeyProvider([]byte("secret")),
	)

	store := auth.NewMemoryAPIKeyStore()
	apiKeyAuth := auth.NewAPIKeyAuthenticator(
		auth.APIKeyConfig{HeaderName: "X-API-Key"},
		store,
	)

	// Combine them
	composite := auth.NewCompositeAuthenticator(jwtAuth, apiKeyAuth)

	fmt.Println("Authenticator name:", composite.Name())
	fmt.Println("Number of authenticators:", len(composite.Authenticators))
	// Output:
	// Authenticator name: composite
	// Number of authenticators: 2
}

func ExampleCompositeAuthenticator_Authenticate() {
	// Create a composite that tries API key first, then JWT
	store := auth.NewMemoryAPIKeyStore()
	hash := auth.HashAPIKey("valid-key")
	_ = store.Add(&auth.APIKeyInfo{
		ID:        "key-1",
		KeyHash:   hash,
		Principal: "api-user",
	})

	apiKeyAuth := auth.NewAPIKeyAuthenticator(
		auth.APIKeyConfig{HeaderName: "X-API-Key"},
		store,
	)

	jwtAuth := auth.NewJWTAuthenticator(
		auth.JWTConfig{},
		auth.NewStaticKeyProvider([]byte("secret")),
	)

	composite := auth.NewCompositeAuthenticator(apiKeyAuth, jwtAuth)

	// Request with API key
	ctx := context.Background()
	req := &auth.AuthRequest{
		Headers: map[string][]string{
			"X-API-Key": {"valid-key"},
		},
	}

	result, err := composite.Authenticate(ctx, req)
	if err == nil && result.Authenticated {
		fmt.Println("Method:", result.Method)
		fmt.Println("Principal:", result.Identity.Principal)
	}
	// Output:
	// Method: api_key
	// Principal: api-user
}

func ExampleNewSimpleRBACAuthorizer() {
	// Define roles and permissions
	rbac := auth.NewSimpleRBACAuthorizer(auth.RBACConfig{
		Roles: map[string]auth.RoleConfig{
			"admin": {
				AllowedTools:   []string{"*"},
				AllowedActions: []string{"*"},
			},
			"reader": {
				AllowedTools:   []string{"search_*", "list_*", "get_*"},
				AllowedActions: []string{"call"},
			},
		},
		DefaultRole: "reader",
	})

	fmt.Println("Authorizer name:", rbac.Name())
	// Output:
	// Authorizer name: simple_rbac
}

func ExampleSimpleRBACAuthorizer_Authorize() {
	rbac := auth.NewSimpleRBACAuthorizer(auth.RBACConfig{
		Roles: map[string]auth.RoleConfig{
			"admin": {
				AllowedTools: []string{"*"},
			},
			"user": {
				AllowedTools: []string{"read_*", "search_*"},
				DeniedTools:  []string{"admin_*"}, // Deny tools starting with admin_
			},
		},
	})

	ctx := context.Background()

	// Admin can access anything
	adminReq := &auth.AuthzRequest{
		Subject:  &auth.Identity{Principal: "admin", Roles: []string{"admin"}},
		Resource: "tool:delete_user",
		Action:   "call",
	}
	fmt.Println("Admin delete_user:", rbac.Authorize(ctx, adminReq) == nil)

	// User can read
	userReq := &auth.AuthzRequest{
		Subject:  &auth.Identity{Principal: "user1", Roles: []string{"user"}},
		Resource: "tool:read_file",
		Action:   "call",
	}
	fmt.Println("User read_file:", rbac.Authorize(ctx, userReq) == nil)

	// User cannot access admin tools (denied by prefix pattern)
	userAdminReq := &auth.AuthzRequest{
		Subject:  &auth.Identity{Principal: "user1", Roles: []string{"user"}},
		Resource: "tool:admin_panel",
		Action:   "call",
	}
	fmt.Println("User admin_panel:", rbac.Authorize(ctx, userAdminReq) == nil)
	// Output:
	// Admin delete_user: true
	// User read_file: true
	// User admin_panel: false
}

func ExampleWithIdentity() {
	// Create an identity
	identity := &auth.Identity{
		Principal: "user@example.com",
		TenantID:  "tenant-123",
		Roles:     []string{"admin", "user"},
		Method:    auth.AuthMethodJWT,
	}

	// Attach to context
	ctx := auth.WithIdentity(context.Background(), identity)

	// Retrieve from context
	retrieved := auth.IdentityFromContext(ctx)
	fmt.Println("Principal:", retrieved.Principal)
	fmt.Println("Tenant:", retrieved.TenantID)
	// Output:
	// Principal: user@example.com
	// Tenant: tenant-123
}

func ExampleIdentityFromContext() {
	// Context with identity
	identity := &auth.Identity{Principal: "alice"}
	ctx := auth.WithIdentity(context.Background(), identity)
	fmt.Println("With identity:", auth.IdentityFromContext(ctx) != nil)

	// Context without identity
	emptyCtx := context.Background()
	fmt.Println("Without identity:", auth.IdentityFromContext(emptyCtx) == nil)
	// Output:
	// With identity: true
	// Without identity: true
}

func ExamplePrincipalFromContext() {
	identity := &auth.Identity{Principal: "alice@example.com"}
	ctx := auth.WithIdentity(context.Background(), identity)

	fmt.Println("Principal:", auth.PrincipalFromContext(ctx))
	// Output:
	// Principal: alice@example.com
}

func ExampleTenantIDFromContext() {
	identity := &auth.Identity{
		Principal: "alice",
		TenantID:  "acme-corp",
	}
	ctx := auth.WithIdentity(context.Background(), identity)

	fmt.Println("Tenant:", auth.TenantIDFromContext(ctx))
	// Output:
	// Tenant: acme-corp
}

func ExampleIdentity_HasRole() {
	identity := &auth.Identity{
		Principal: "alice",
		Roles:     []string{"admin", "user"},
	}

	fmt.Println("Has admin:", identity.HasRole("admin"))
	fmt.Println("Has guest:", identity.HasRole("guest"))
	// Output:
	// Has admin: true
	// Has guest: false
}

func ExampleIdentity_HasPermission() {
	identity := &auth.Identity{
		Principal:   "alice",
		Permissions: []string{"tool:read", "tool:write"},
	}

	fmt.Println("Has tool:read:", identity.HasPermission("tool:read"))
	fmt.Println("Has tool:delete:", identity.HasPermission("tool:delete"))
	// Output:
	// Has tool:read: true
	// Has tool:delete: false
}

func ExampleIdentity_IsExpired() {
	// Non-expiring identity
	noExpiry := &auth.Identity{Principal: "alice"}
	fmt.Println("No expiry is expired:", noExpiry.IsExpired())

	// Future expiry
	future := &auth.Identity{
		Principal: "bob",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	fmt.Println("Future expiry is expired:", future.IsExpired())

	// Past expiry
	past := &auth.Identity{
		Principal: "charlie",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	fmt.Println("Past expiry is expired:", past.IsExpired())
	// Output:
	// No expiry is expired: false
	// Future expiry is expired: false
	// Past expiry is expired: true
}

func ExampleAnonymousIdentity() {
	anon := auth.AnonymousIdentity()

	fmt.Println("Principal:", anon.Principal)
	fmt.Println("Method:", anon.Method)
	fmt.Println("Is anonymous:", anon.IsAnonymous())
	// Output:
	// Principal: anonymous
	// Method: anonymous
	// Is anonymous: true
}

func ExampleAuthSuccess() {
	identity := &auth.Identity{
		Principal: "alice",
		Method:    auth.AuthMethodAPIKey,
	}

	result := auth.AuthSuccess(identity)

	fmt.Println("Authenticated:", result.Authenticated)
	fmt.Println("Method:", result.Method)
	fmt.Println("Has error:", result.Error != nil)
	// Output:
	// Authenticated: true
	// Method: api_key
	// Has error: false
}

func ExampleAuthFailure() {
	result := auth.AuthFailure(auth.ErrInvalidCredentials, "jwt")

	fmt.Println("Authenticated:", result.Authenticated)
	fmt.Println("Method:", result.Method)
	fmt.Println("Error is invalid credentials:", errors.Is(result.Error, auth.ErrInvalidCredentials))
	// Output:
	// Authenticated: false
	// Method: jwt
	// Error is invalid credentials: true
}

func ExampleAllowAllAuthorizer() {
	authz := auth.AllowAllAuthorizer{}

	ctx := context.Background()
	req := &auth.AuthzRequest{
		Subject:  &auth.Identity{Principal: "anyone"},
		Resource: "tool:anything",
		Action:   "call",
	}

	err := authz.Authorize(ctx, req)
	fmt.Println("Allowed:", err == nil)
	fmt.Println("Name:", authz.Name())
	// Output:
	// Allowed: true
	// Name: allow_all
}

func ExampleDenyAllAuthorizer() {
	authz := auth.DenyAllAuthorizer{}

	ctx := context.Background()
	req := &auth.AuthzRequest{
		Subject:  &auth.Identity{Principal: "anyone"},
		Resource: "tool:anything",
		Action:   "call",
	}

	err := authz.Authorize(ctx, req)
	fmt.Println("Denied:", err != nil)
	fmt.Println("Is forbidden:", errors.Is(err, auth.ErrForbidden))
	fmt.Println("Name:", authz.Name())
	// Output:
	// Denied: true
	// Is forbidden: true
	// Name: deny_all
}

func ExampleAuthzError() {
	err := &auth.AuthzError{
		Subject:  "alice",
		Resource: "tool:admin_panel",
		Action:   "access",
		Reason:   "insufficient permissions",
	}

	fmt.Println("Is forbidden:", errors.Is(err, auth.ErrForbidden))
	// Output:
	// Is forbidden: true
}

func ExampleNewAuthenticatorFunc() {
	// Create a custom authenticator using a function
	customAuth := auth.NewAuthenticatorFunc(
		"custom",
		func(ctx context.Context, req *auth.AuthRequest) bool {
			// Support requests with X-Custom-Auth header
			return req.GetHeader("X-Custom-Auth") != ""
		},
		func(ctx context.Context, req *auth.AuthRequest) (*auth.AuthResult, error) {
			token := req.GetHeader("X-Custom-Auth")
			if token == "valid-token" {
				return auth.AuthSuccess(&auth.Identity{
					Principal: "custom-user",
					Method:    "custom",
				}), nil
			}
			return auth.AuthFailure(auth.ErrInvalidCredentials, "custom"), nil
		},
	)

	fmt.Println("Authenticator name:", customAuth.Name())

	ctx := context.Background()
	req := &auth.AuthRequest{
		Headers: map[string][]string{
			"X-Custom-Auth": {"valid-token"},
		},
	}

	result, _ := customAuth.Authenticate(ctx, req)
	fmt.Println("Authenticated:", result.Authenticated)
	// Output:
	// Authenticator name: custom
	// Authenticated: true
}
