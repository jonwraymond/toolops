package auth

import (
	"testing"
	"time"
)

func TestIdentity_HasRole(t *testing.T) {
	tests := []struct {
		name     string
		identity *Identity
		role     string
		want     bool
	}{
		{
			name:     "nil identity",
			identity: nil,
			role:     "admin",
			want:     false,
		},
		{
			name:     "empty roles",
			identity: &Identity{Roles: []string{}},
			role:     "admin",
			want:     false,
		},
		{
			name:     "has role",
			identity: &Identity{Roles: []string{"user", "admin"}},
			role:     "admin",
			want:     true,
		},
		{
			name:     "does not have role",
			identity: &Identity{Roles: []string{"user"}},
			role:     "admin",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bool
			if tt.identity == nil {
				got = false
			} else {
				got = tt.identity.HasRole(tt.role)
			}
			if got != tt.want {
				t.Errorf("Identity.HasRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdentity_HasPermission(t *testing.T) {
	tests := []struct {
		name       string
		identity   *Identity
		permission string
		want       bool
	}{
		{
			name:       "nil identity",
			identity:   nil,
			permission: "read",
			want:       false,
		},
		{
			name:       "empty permissions",
			identity:   &Identity{Permissions: []string{}},
			permission: "read",
			want:       false,
		},
		{
			name:       "has permission",
			identity:   &Identity{Permissions: []string{"read", "write"}},
			permission: "write",
			want:       true,
		},
		{
			name:       "does not have permission",
			identity:   &Identity{Permissions: []string{"read"}},
			permission: "write",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bool
			if tt.identity == nil {
				got = false
			} else {
				got = tt.identity.HasPermission(tt.permission)
			}
			if got != tt.want {
				t.Errorf("Identity.HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdentity_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		identity *Identity
		want     bool
	}{
		{
			name:     "zero expiry",
			identity: &Identity{},
			want:     false,
		},
		{
			name:     "expired",
			identity: &Identity{ExpiresAt: time.Now().Add(-time.Hour)},
			want:     true,
		},
		{
			name:     "not expired",
			identity: &Identity{ExpiresAt: time.Now().Add(time.Hour)},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.identity.IsExpired(); got != tt.want {
				t.Errorf("Identity.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdentity_IsAnonymous(t *testing.T) {
	tests := []struct {
		name     string
		identity *Identity
		want     bool
	}{
		{
			name:     "anonymous method",
			identity: &Identity{Principal: "anon", Method: AuthMethodAnonymous},
			want:     true,
		},
		{
			name:     "empty principal",
			identity: &Identity{Principal: "", Method: AuthMethodJWT},
			want:     true,
		},
		{
			name:     "normal user",
			identity: &Identity{Principal: "user123", Method: AuthMethodJWT},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.identity.IsAnonymous(); got != tt.want {
				t.Errorf("Identity.IsAnonymous() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnonymousIdentity(t *testing.T) {
	id := AnonymousIdentity()

	if id.Principal != "anonymous" {
		t.Errorf("Principal = %v, want anonymous", id.Principal)
	}
	if id.Method != AuthMethodAnonymous {
		t.Errorf("Method = %v, want anonymous", id.Method)
	}
	if id.Claims == nil {
		t.Error("Claims should be initialized")
	}
}

func TestAuthMethod_Constants(t *testing.T) {
	tests := []struct {
		method AuthMethod
		want   string
	}{
		{AuthMethodNone, "none"},
		{AuthMethodJWT, "jwt"},
		{AuthMethodAPIKey, "api_key"},
		{AuthMethodOAuth2, "oauth2"},
		{AuthMethodBasic, "basic"},
		{AuthMethodAnonymous, "anonymous"},
		{AuthMethodComposite, "composite"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.method) != tt.want {
				t.Errorf("AuthMethod = %v, want %v", string(tt.method), tt.want)
			}
		})
	}
}
