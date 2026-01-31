package auth

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrMissingCredentials", ErrMissingCredentials},
		{"ErrInvalidCredentials", ErrInvalidCredentials},
		{"ErrTokenExpired", ErrTokenExpired},
		{"ErrTokenInactive", ErrTokenInactive},
		{"ErrKeyNotFound", ErrKeyNotFound},
		{"ErrForbidden", ErrForbidden},
		{"ErrIntrospectionFailed", ErrIntrospectionFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}

			// Check error message is not empty
			if tt.err.Error() == "" {
				t.Errorf("%s has empty message", tt.name)
			}
		})
	}
}

func TestErrorsIs(t *testing.T) {
	// Test that wrapped errors work with errors.Is
	wrapped := errors.New("wrapped: " + ErrInvalidCredentials.Error())

	// Direct error comparison
	if !errors.Is(ErrInvalidCredentials, ErrInvalidCredentials) {
		t.Error("errors.Is should match same error")
	}

	// Different errors should not match
	if errors.Is(ErrInvalidCredentials, ErrTokenExpired) {
		t.Error("errors.Is should not match different errors")
	}

	// Wrapped error string is different (not errors.Is compatible without proper wrapping)
	if errors.Is(wrapped, ErrInvalidCredentials) {
		t.Error("Simple string wrapping should not match with errors.Is")
	}
}
