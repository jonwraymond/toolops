package auth

import (
	"context"
	"testing"
)

func TestAuthRequest_GetHeader(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string][]string
		key     string
		want    string
	}{
		{
			name:    "nil headers",
			headers: nil,
			key:     "Authorization",
			want:    "",
		},
		{
			name:    "existing header",
			headers: map[string][]string{"Authorization": {"Bearer token123"}},
			key:     "Authorization",
			want:    "Bearer token123",
		},
		{
			name:    "missing header",
			headers: map[string][]string{"Content-Type": {"application/json"}},
			key:     "Authorization",
			want:    "",
		},
		{
			name:    "multiple values returns first",
			headers: map[string][]string{"Accept": {"text/html", "application/json"}},
			key:     "Accept",
			want:    "text/html",
		},
		{
			name:    "empty values slice",
			headers: map[string][]string{"X-Empty": {}},
			key:     "X-Empty",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AuthRequest{Headers: tt.headers}
			if got := req.GetHeader(tt.key); got != tt.want {
				t.Errorf("AuthRequest.GetHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthSuccess(t *testing.T) {
	identity := &Identity{Principal: "user123", Method: AuthMethodJWT}
	result := AuthSuccess(identity)

	if !result.Authenticated {
		t.Error("AuthSuccess should set Authenticated to true")
	}
	if result.Identity != identity {
		t.Error("AuthSuccess should set Identity")
	}
	if result.Error != nil {
		t.Error("AuthSuccess should not set Error")
	}
	if result.Method != "jwt" {
		t.Errorf("Method = %v, want jwt", result.Method)
	}
}

func TestAuthFailure(t *testing.T) {
	result := AuthFailure(ErrInvalidCredentials, "Bearer")

	if result.Authenticated {
		t.Error("AuthFailure should set Authenticated to false")
	}
	if result.Identity != nil {
		t.Error("AuthFailure should not set Identity")
	}
	if result.Error != ErrInvalidCredentials {
		t.Errorf("AuthFailure should set Error, got %v", result.Error)
	}
	if result.Method != "Bearer" {
		t.Errorf("Method = %v, want Bearer", result.Method)
	}
}

func TestNewAuthenticatorFunc(t *testing.T) {
	auth := NewAuthenticatorFunc(
		"test",
		func(_ context.Context, _ *AuthRequest) bool { return true },
		func(_ context.Context, _ *AuthRequest) (*AuthResult, error) {
			return AuthSuccess(&Identity{Principal: "test", Method: AuthMethodNone}), nil
		},
	)

	if auth.Name() != "test" {
		t.Errorf("Name() = %v, want test", auth.Name())
	}

	req := &AuthRequest{}
	if !auth.Supports(nil, req) {
		t.Error("Supports() = false, want true")
	}

	result, err := auth.Authenticate(nil, req)
	if err != nil {
		t.Errorf("Authenticate() error = %v", err)
	}
	if !result.Authenticated {
		t.Error("Authenticate() should succeed")
	}
}
