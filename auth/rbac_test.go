package auth

import (
	"context"
	"testing"
)

func TestNewSimpleRBACAuthorizer(t *testing.T) {
	config := RBACConfig{
		Roles: map[string]RoleConfig{
			"admin": {Permissions: []string{"*"}},
		},
	}

	auth := NewSimpleRBACAuthorizer(config)

	if auth.Name() != "simple_rbac" {
		t.Errorf("Name() = %v, want simple_rbac", auth.Name())
	}
}

func TestSimpleRBACAuthorizer_Authorize(t *testing.T) {
	config := RBACConfig{
		Roles: map[string]RoleConfig{
			"admin": {
				AllowedTools:   []string{"*"},
				AllowedActions: []string{"*"},
			},
			"user": {
				AllowedTools:   []string{"calculator", "weather"},
				AllowedActions: []string{"call"},
			},
			"viewer": {
				AllowedTools:   []string{"*"},
				AllowedActions: []string{"read"},
				DeniedTools:    []string{"admin*"},
			},
			"inherits_user": {
				Inherits: []string{"user"},
			},
		},
		DefaultRole: "viewer",
	}

	auth := NewSimpleRBACAuthorizer(config)

	tests := []struct {
		name    string
		subject *Identity
		request *AuthzRequest
		wantErr bool
	}{
		{
			name:    "nil subject",
			subject: nil,
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "calculator",
				Action:       "call",
			},
			wantErr: true,
		},
		{
			name:    "admin can do anything",
			subject: &Identity{Roles: []string{"admin"}},
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "any-tool",
				Action:       "call",
			},
			wantErr: false,
		},
		{
			name:    "user can call allowed tool",
			subject: &Identity{Roles: []string{"user"}},
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "calculator",
				Action:       "call",
			},
			wantErr: false,
		},
		{
			name:    "user cannot call non-allowed tool",
			subject: &Identity{Roles: []string{"user"}},
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "admin-tool",
				Action:       "call",
			},
			wantErr: true,
		},
		{
			name:    "viewer can read but not call",
			subject: &Identity{Roles: []string{"viewer"}},
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "calculator",
				Action:       "read",
			},
			wantErr: false,
		},
		{
			name:    "viewer denied admin tools",
			subject: &Identity{Roles: []string{"viewer"}},
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "admin-panel",
				Action:       "read",
			},
			wantErr: true,
		},
		{
			name:    "inherited role permissions",
			subject: &Identity{Roles: []string{"inherits_user"}},
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "calculator",
				Action:       "call",
			},
			wantErr: false,
		},
		{
			name:    "default role when no roles",
			subject: &Identity{Roles: []string{}},
			request: &AuthzRequest{
				ResourceType: "tool",
				Resource:     "calculator",
				Action:       "read",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.request.Subject = tt.subject
			err := auth.Authorize(context.Background(), tt.request)

			if tt.wantErr && err == nil {
				t.Error("Authorize() should return error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Authorize() error = %v", err)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"*", "anything", true},
		{"admin", "admin", true},
		{"admin", "user", false},
		{"admin*", "admin", true},
		{"admin*", "admin-panel", true},
		{"admin*", "user", false},
		{"tool*", "tool-calculator", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			if got := matchPattern(tt.pattern, tt.value); got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

func TestMatchPermission(t *testing.T) {
	tests := []struct {
		perm    string
		request *AuthzRequest
		want    bool
	}{
		{
			perm:    "call",
			request: &AuthzRequest{Action: "call"},
			want:    true,
		},
		{
			perm:    "*",
			request: &AuthzRequest{Action: "anything"},
			want:    true,
		},
		{
			perm:    "calculator:call",
			request: &AuthzRequest{ResourceType: "tool", Resource: "calculator", Action: "call"},
			want:    true,
		},
		{
			perm:    "calculator:*",
			request: &AuthzRequest{ResourceType: "tool", Resource: "calculator", Action: "call"},
			want:    true,
		},
		{
			perm:    "tool:calculator:call",
			request: &AuthzRequest{ResourceType: "tool", Resource: "calculator", Action: "call"},
			want:    true,
		},
		{
			perm:    "tool:*:call",
			request: &AuthzRequest{ResourceType: "tool", Resource: "calculator", Action: "call"},
			want:    true,
		},
		{
			perm:    "*:*:*",
			request: &AuthzRequest{ResourceType: "tool", Resource: "calculator", Action: "call"},
			want:    true,
		},
		{
			perm:    "endpoint:users:read",
			request: &AuthzRequest{ResourceType: "tool", Resource: "calculator", Action: "call"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.perm, func(t *testing.T) {
			if got := matchPermission(tt.perm, tt.request); got != tt.want {
				t.Errorf("matchPermission(%q) = %v, want %v", tt.perm, got, tt.want)
			}
		})
	}
}

func TestAuthzRequest_ToolName(t *testing.T) {
	tests := []struct {
		name    string
		request *AuthzRequest
		want    string
	}{
		{
			name:    "tool prefix stripped",
			request: &AuthzRequest{Resource: "tool:calculator"},
			want:    "calculator",
		},
		{
			name:    "no tool prefix returns resource as-is",
			request: &AuthzRequest{Resource: "calculator"},
			want:    "calculator",
		},
		{
			name:    "endpoint resource returns as-is",
			request: &AuthzRequest{ResourceType: "endpoint", Resource: "/api/users"},
			want:    "/api/users",
		},
		{
			name:    "empty resource",
			request: &AuthzRequest{Resource: ""},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.request.ToolName(); got != tt.want {
				t.Errorf("ToolName() = %v, want %v", got, tt.want)
			}
		})
	}
}
