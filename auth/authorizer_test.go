package auth

import (
	"context"
	"testing"
)

func TestAuthzError_Error(t *testing.T) {
	err := &AuthzError{
		Subject:  "user123",
		Resource: "tool/calculator",
		Action:   "call",
		Reason:   "permission denied",
	}

	expected := `authorization denied: subject="user123" resource="tool/calculator" action="call" reason="permission denied"`
	if got := err.Error(); got != expected {
		t.Errorf("AuthzError.Error() = %v, want %v", got, expected)
	}
}

func TestAuthzError_Is(t *testing.T) {
	err := &AuthzError{
		Subject:  "user123",
		Resource: "tool",
		Action:   "call",
		Reason:   "denied",
	}

	if !err.Is(ErrForbidden) {
		t.Error("AuthzError.Is(ErrForbidden) = false, want true")
	}
}

func TestAllowAllAuthorizer(t *testing.T) {
	auth := AllowAllAuthorizer{}

	if auth.Name() != "allow_all" {
		t.Errorf("Name() = %v, want allow_all", auth.Name())
	}

	req := &AuthzRequest{
		Subject:  &Identity{Principal: "user123"},
		Resource: "tool/calculator",
		Action:   "call",
	}

	err := auth.Authorize(context.Background(), req)
	if err != nil {
		t.Errorf("AllowAllAuthorizer.Authorize() error = %v", err)
	}
}

func TestDenyAllAuthorizer(t *testing.T) {
	auth := DenyAllAuthorizer{}

	if auth.Name() != "deny_all" {
		t.Errorf("Name() = %v, want deny_all", auth.Name())
	}

	req := &AuthzRequest{
		Subject:  &Identity{Principal: "user123"},
		Resource: "tool/calculator",
		Action:   "call",
	}

	err := auth.Authorize(context.Background(), req)
	if err == nil {
		t.Error("DenyAllAuthorizer.Authorize() should return error")
	}

	authzErr, ok := err.(*AuthzError)
	if !ok {
		t.Errorf("Expected *AuthzError, got %T", err)
	}
	if authzErr.Reason != "all requests denied" {
		t.Errorf("Reason = %v, want 'all requests denied'", authzErr.Reason)
	}
}

func TestAuthorizerFunc(t *testing.T) {
	called := false
	authz := AuthorizerFunc(func(_ context.Context, _ *AuthzRequest) error {
		called = true
		return nil
	})

	if authz.Name() != "func" {
		t.Errorf("Name() = %v, want func", authz.Name())
	}

	req := &AuthzRequest{
		Subject: &Identity{Principal: "user"},
		Action:  "call",
	}
	err := authz.Authorize(context.Background(), req)
	if err != nil {
		t.Errorf("Authorize() error = %v", err)
	}
	if !called {
		t.Error("AuthorizerFunc was not called")
	}
}
