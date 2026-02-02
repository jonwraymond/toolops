package auth

import "context"

// Authenticator validates credentials and returns an identity.
//
// # Contract
//
// Concurrency:
//   - Implementations must be safe for concurrent use from multiple goroutines.
//   - Internal state (if any) must be protected by appropriate synchronization.
//
// Context:
//   - All methods accepting context.Context must honor cancellation and deadlines.
//   - Long-running operations (e.g., remote token introspection) should check ctx.Done().
//
// Errors:
//   - Authenticate returns (nil, error) ONLY for internal/infrastructure errors
//     (e.g., network failure, database unavailable).
//   - Authentication failures (invalid token, expired, etc.) return (*AuthResult, nil)
//     with result.Authenticated=false and result.Error set to the auth error.
//   - Use sentinel errors from this package for auth failures: [ErrInvalidCredentials],
//     [ErrTokenExpired], [ErrTokenMalformed], [ErrMissingCredentials].
//
// Ownership:
//   - The caller owns the AuthRequest; implementations must not modify it.
//   - The returned AuthResult is owned by the caller.
type Authenticator interface {
	// Name returns a unique identifier for this authenticator.
	// Must be constant for the lifetime of the authenticator.
	Name() string

	// Supports returns true if this authenticator can handle the request.
	// Should be a fast check (e.g., header presence) without network calls.
	Supports(ctx context.Context, req *AuthRequest) bool

	// Authenticate validates credentials and returns a result.
	// Returns (result, nil) for success/failure, (nil, error) for internal errors.
	Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error)
}

// AuthRequest contains the information needed for authentication.
type AuthRequest struct {
	// Headers contains HTTP headers (Authorization, X-API-Key, etc.)
	Headers map[string][]string

	// Resource is the target resource (optional, for context).
	Resource string

	// Metadata contains additional request metadata.
	Metadata map[string]any
}

// GetHeader returns the first value for a header, or empty string.
func (r *AuthRequest) GetHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	values := r.Headers[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// AuthResult is the result of an authentication attempt.
type AuthResult struct {
	// Authenticated is true if authentication succeeded.
	Authenticated bool

	// Identity is the authenticated identity (only if Authenticated=true).
	Identity *Identity

	// Error is the authentication error (only if Authenticated=false).
	Error error

	// Method indicates which authenticator method was used.
	Method string
}

// AuthSuccess creates a successful authentication result.
func AuthSuccess(identity *Identity) *AuthResult {
	return &AuthResult{
		Authenticated: true,
		Identity:      identity,
		Method:        string(identity.Method),
	}
}

// AuthFailure creates a failed authentication result.
func AuthFailure(err error, method string) *AuthResult {
	return &AuthResult{
		Authenticated: false,
		Error:         err,
		Method:        method,
	}
}

// AuthenticatorFunc is an adapter to allow use of ordinary functions as Authenticators.
type AuthenticatorFunc struct {
	name     string
	supports func(ctx context.Context, req *AuthRequest) bool
	auth     func(ctx context.Context, req *AuthRequest) (*AuthResult, error)
}

// Name returns the authenticator name.
func (f *AuthenticatorFunc) Name() string {
	return f.name
}

// Supports returns true if this authenticator can handle the request.
func (f *AuthenticatorFunc) Supports(ctx context.Context, req *AuthRequest) bool {
	return f.supports(ctx, req)
}

// Authenticate validates credentials.
func (f *AuthenticatorFunc) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	return f.auth(ctx, req)
}

// NewAuthenticatorFunc creates an AuthenticatorFunc.
func NewAuthenticatorFunc(
	name string,
	supports func(ctx context.Context, req *AuthRequest) bool,
	auth func(ctx context.Context, req *AuthRequest) (*AuthResult, error),
) *AuthenticatorFunc {
	return &AuthenticatorFunc{
		name:     name,
		supports: supports,
		auth:     auth,
	}
}
