package auth

import (
	"context"
	"errors"
	"testing"
)

// mockAuthenticator is a test authenticator with configurable behavior.
type mockAuthenticator struct {
	name       string
	supports   bool
	result     *AuthResult
	err        error
	supportsFn func(ctx context.Context, req *AuthRequest) bool
	authFn     func(ctx context.Context, req *AuthRequest) (*AuthResult, error)
}

func (m *mockAuthenticator) Name() string {
	return m.name
}

func (m *mockAuthenticator) Supports(ctx context.Context, req *AuthRequest) bool {
	if m.supportsFn != nil {
		return m.supportsFn(ctx, req)
	}
	return m.supports
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	if m.authFn != nil {
		return m.authFn(ctx, req)
	}
	return m.result, m.err
}

func TestCompositeAuthenticator_Name(t *testing.T) {
	auth := NewCompositeAuthenticator()
	if auth.Name() != "composite" {
		t.Errorf("Name() = %v, want composite", auth.Name())
	}
}

func TestCompositeAuthenticator_NewWithAuthenticators(t *testing.T) {
	mock1 := &mockAuthenticator{name: "mock1"}
	mock2 := &mockAuthenticator{name: "mock2"}

	auth := NewCompositeAuthenticator(mock1, mock2)

	if len(auth.Authenticators) != 2 {
		t.Errorf("len(Authenticators) = %d, want 2", len(auth.Authenticators))
	}
	if !auth.StopOnFirst {
		t.Error("StopOnFirst should default to true")
	}
}

func TestCompositeAuthenticator_Supports(t *testing.T) {
	tests := []struct {
		name           string
		authenticators []Authenticator
		want           bool
	}{
		{
			name:           "no authenticators",
			authenticators: nil,
			want:           false,
		},
		{
			name: "none support",
			authenticators: []Authenticator{
				&mockAuthenticator{name: "mock1", supports: false},
				&mockAuthenticator{name: "mock2", supports: false},
			},
			want: false,
		},
		{
			name: "one supports",
			authenticators: []Authenticator{
				&mockAuthenticator{name: "mock1", supports: false},
				&mockAuthenticator{name: "mock2", supports: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewCompositeAuthenticator(tt.authenticators...)

			req := &AuthRequest{Headers: map[string][]string{}}
			if got := auth.Supports(context.Background(), req); got != tt.want {
				t.Errorf("Supports() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositeAuthenticator_Authenticate(t *testing.T) {
	tests := []struct {
		name           string
		authenticators []Authenticator
		wantAuth       bool
		wantErr        bool
		wantPrincipal  string
	}{
		{
			name:           "no authenticators",
			authenticators: nil,
			wantAuth:       false,
			wantErr:        false,
		},
		{
			name: "first succeeds",
			authenticators: []Authenticator{
				&mockAuthenticator{
					name:     "mock1",
					supports: true,
					result:   AuthSuccess(&Identity{Principal: "user1", Method: AuthMethodJWT}),
				},
				&mockAuthenticator{
					name:     "mock2",
					supports: true,
					result:   AuthSuccess(&Identity{Principal: "user2", Method: AuthMethodJWT}),
				},
			},
			wantAuth:      true,
			wantPrincipal: "user1",
		},
		{
			name: "first fails, second succeeds",
			authenticators: []Authenticator{
				&mockAuthenticator{
					name:     "mock1",
					supports: true,
					result:   AuthFailure(ErrInvalidCredentials, ""),
				},
				&mockAuthenticator{
					name:     "mock2",
					supports: true,
					result:   AuthSuccess(&Identity{Principal: "user2", Method: AuthMethodJWT}),
				},
			},
			wantAuth:      true,
			wantPrincipal: "user2",
		},
		{
			name: "all fail",
			authenticators: []Authenticator{
				&mockAuthenticator{
					name:     "mock1",
					supports: true,
					result:   AuthFailure(ErrInvalidCredentials, ""),
				},
				&mockAuthenticator{
					name:     "mock2",
					supports: true,
					result:   AuthFailure(ErrInvalidCredentials, ""),
				},
			},
			wantAuth: false,
		},
		{
			name: "error stops chain",
			authenticators: []Authenticator{
				&mockAuthenticator{
					name:     "mock1",
					supports: true,
					err:      errors.New("internal error"),
				},
				&mockAuthenticator{
					name:     "mock2",
					supports: true,
					result:   AuthSuccess(&Identity{Principal: "user2", Method: AuthMethodJWT}),
				},
			},
			wantErr: true,
		},
		{
			name: "skips unsupported",
			authenticators: []Authenticator{
				&mockAuthenticator{
					name:     "mock1",
					supports: false,
					result:   AuthSuccess(&Identity{Principal: "user1", Method: AuthMethodJWT}),
				},
				&mockAuthenticator{
					name:     "mock2",
					supports: true,
					result:   AuthSuccess(&Identity{Principal: "user2", Method: AuthMethodJWT}),
				},
			},
			wantAuth:      true,
			wantPrincipal: "user2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewCompositeAuthenticator(tt.authenticators...)

			req := &AuthRequest{Headers: map[string][]string{}}
			result, err := auth.Authenticate(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Error("Authenticate() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("Authenticate() error = %v", err)
				return
			}

			if result.Authenticated != tt.wantAuth {
				t.Errorf("Authenticated = %v, want %v", result.Authenticated, tt.wantAuth)
			}

			if tt.wantPrincipal != "" && result.Identity != nil {
				if result.Identity.Principal != tt.wantPrincipal {
					t.Errorf("Principal = %v, want %v", result.Identity.Principal, tt.wantPrincipal)
				}
			}
		})
	}
}

func TestCompositeAuthenticator_StopOnFirstFalse(t *testing.T) {
	auth := NewCompositeAuthenticator(
		&mockAuthenticator{
			name:     "mock1",
			supports: true,
			result:   AuthSuccess(&Identity{Principal: "user1", Method: AuthMethodJWT}),
		},
		&mockAuthenticator{
			name:     "mock2",
			supports: true,
			result:   AuthSuccess(&Identity{Principal: "user2", Method: AuthMethodJWT}),
		},
	)
	auth.StopOnFirst = false

	req := &AuthRequest{Headers: map[string][]string{}}
	result, err := auth.Authenticate(context.Background(), req)

	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if !result.Authenticated {
		t.Error("Authenticated = false, want true")
	}
	// Should return first success
	if result.Identity.Principal != "user1" {
		t.Errorf("Principal = %v, want user1", result.Identity.Principal)
	}
}
