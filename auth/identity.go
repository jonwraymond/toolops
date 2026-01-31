package auth

import "time"

// AuthMethod indicates how authentication was performed.
type AuthMethod string

const (
	AuthMethodNone      AuthMethod = "none"
	AuthMethodJWT       AuthMethod = "jwt"
	AuthMethodAPIKey    AuthMethod = "api_key"
	AuthMethodOAuth2    AuthMethod = "oauth2"
	AuthMethodBasic     AuthMethod = "basic"
	AuthMethodAnonymous AuthMethod = "anonymous"
	AuthMethodComposite AuthMethod = "composite"
)

// Identity represents an authenticated principal.
type Identity struct {
	// Principal is the unique identifier (e.g., user ID, email).
	Principal string

	// TenantID is the tenant this identity belongs to (multi-tenancy).
	TenantID string

	// Roles are the roles assigned to this identity.
	Roles []string

	// Permissions are explicit permissions granted to this identity.
	Permissions []string

	// Method indicates how authentication was performed.
	Method AuthMethod

	// Claims contains the raw claims from the token.
	Claims map[string]any

	// ExpiresAt is when this identity expires.
	ExpiresAt time.Time

	// IssuedAt is when this identity was created.
	IssuedAt time.Time
}

// HasRole checks if the identity has a specific role.
func (id *Identity) HasRole(role string) bool {
	for _, r := range id.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the identity has a specific permission.
func (id *Identity) HasPermission(perm string) bool {
	for _, p := range id.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// IsExpired checks if the identity has expired.
func (id *Identity) IsExpired() bool {
	if id.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(id.ExpiresAt)
}

// IsAnonymous returns true if this is an anonymous identity.
func (id *Identity) IsAnonymous() bool {
	return id.Method == AuthMethodAnonymous || id.Principal == ""
}

// AnonymousIdentity creates a default anonymous identity.
func AnonymousIdentity() *Identity {
	return &Identity{
		Principal: "anonymous",
		Method:    AuthMethodAnonymous,
		Claims:    make(map[string]any),
	}
}
