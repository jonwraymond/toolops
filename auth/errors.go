package auth

import "errors"

// Sentinel errors for authentication and authorization.
var (
	// Authentication errors
	ErrMissingCredentials  = errors.New("auth: missing credentials")
	ErrInvalidCredentials  = errors.New("auth: invalid credentials")
	ErrTokenExpired        = errors.New("auth: token expired")
	ErrTokenMalformed      = errors.New("auth: token malformed")
	ErrTokenInactive       = errors.New("auth: token inactive")
	ErrIntrospectionFailed = errors.New("auth: introspection failed")
	ErrKeyNotFound         = errors.New("auth: signing key not found")

	// Authorization errors
	ErrForbidden = errors.New("auth: access denied")
)
