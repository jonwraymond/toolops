package auth

import (
	"context"
)

// Context keys for auth-related values.
type contextKey int

const (
	identityKey contextKey = iota
	headersKey
)

// WithIdentity returns a new context with the given identity attached.
func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// IdentityFromContext retrieves the identity from the context.
// Returns nil if no identity is present.
func IdentityFromContext(ctx context.Context) *Identity {
	id, _ := ctx.Value(identityKey).(*Identity)
	return id
}

// PrincipalFromContext retrieves the principal from the context.
// Returns empty string if no identity is present.
func PrincipalFromContext(ctx context.Context) string {
	id := IdentityFromContext(ctx)
	if id == nil {
		return ""
	}
	return id.Principal
}

// TenantIDFromContext retrieves the tenant ID from the context.
// Returns empty string if no identity is present or tenant is not set.
func TenantIDFromContext(ctx context.Context) string {
	id := IdentityFromContext(ctx)
	if id == nil {
		return ""
	}
	return id.TenantID
}

// WithHeaders returns a new context with the given HTTP headers attached.
// These headers are used by authenticators to extract credentials.
func WithHeaders(ctx context.Context, headers map[string][]string) context.Context {
	return context.WithValue(ctx, headersKey, headers)
}

// HeadersFromContext retrieves HTTP headers from the context.
// Returns nil if no headers are present.
func HeadersFromContext(ctx context.Context) map[string][]string {
	h, _ := ctx.Value(headersKey).(map[string][]string)
	return h
}

// GetHeader retrieves a single header value from the context.
// Returns the first value if multiple values exist, or empty string if not found.
func GetHeader(ctx context.Context, key string) string {
	headers := HeadersFromContext(ctx)
	if headers == nil {
		return ""
	}
	values := headers[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
