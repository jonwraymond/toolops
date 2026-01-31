package auth

import "context"

// CompositeAuthenticator tries multiple authenticators in sequence.
// It returns on the first successful authentication or after all fail.
type CompositeAuthenticator struct {
	// Authenticators is the ordered list of authenticators to try.
	Authenticators []Authenticator

	// StopOnFirst stops on the first successful authentication.
	// Default: true
	StopOnFirst bool
}

// NewCompositeAuthenticator creates a composite authenticator.
func NewCompositeAuthenticator(auths ...Authenticator) *CompositeAuthenticator {
	return &CompositeAuthenticator{
		Authenticators: auths,
		StopOnFirst:    true,
	}
}

// Name returns "composite".
func (c *CompositeAuthenticator) Name() string {
	return "composite"
}

// Supports returns true if any authenticator supports the request.
func (c *CompositeAuthenticator) Supports(ctx context.Context, req *AuthRequest) bool {
	for _, auth := range c.Authenticators {
		if auth.Supports(ctx, req) {
			return true
		}
	}
	return false
}

// Authenticate tries each authenticator in sequence.
func (c *CompositeAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	if len(c.Authenticators) == 0 {
		return AuthFailure(ErrMissingCredentials, ""), nil
	}

	var lastResult *AuthResult
	var firstSuccess *AuthResult

	for _, auth := range c.Authenticators {
		// Skip authenticators that don't support this request
		if !auth.Supports(ctx, req) {
			continue
		}

		result, err := auth.Authenticate(ctx, req)
		if err != nil {
			// Propagate errors immediately
			return nil, err
		}

		lastResult = result

		if result.Authenticated {
			if c.StopOnFirst {
				return result, nil
			}
			if firstSuccess == nil {
				firstSuccess = result
			}
		}
	}

	// Return first success if we found one (StopOnFirst=false case)
	if firstSuccess != nil {
		return firstSuccess, nil
	}

	// No authenticator succeeded
	if lastResult != nil {
		return lastResult, nil
	}

	return AuthFailure(ErrMissingCredentials, ""), nil
}

// Ensure CompositeAuthenticator implements Authenticator
var _ Authenticator = (*CompositeAuthenticator)(nil)
