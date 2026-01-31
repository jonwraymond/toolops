package auth

import (
	"context"
	"fmt"
	"strings"
)

// Authorizer determines if an identity is allowed to perform an action.
type Authorizer interface {
	// Authorize checks if the request is permitted.
	// Returns nil if authorized, or an error (typically *AuthzError) if denied.
	Authorize(ctx context.Context, req *AuthzRequest) error

	// Name returns a unique identifier for this authorizer.
	Name() string
}

// AuthzRequest contains the information needed for authorization.
type AuthzRequest struct {
	// Subject is the identity making the request.
	Subject *Identity

	// Resource is the target resource (e.g., "tool:search_tools").
	Resource string

	// Action is the requested action (e.g., "call", "list").
	Action string

	// ResourceType categorizes the resource (e.g., "tool", "namespace").
	ResourceType string
}

// ToolName extracts the tool name from the resource.
// Removes "tool:" prefix if present.
func (r *AuthzRequest) ToolName() string {
	if name, found := strings.CutPrefix(r.Resource, "tool:"); found {
		return name
	}
	return r.Resource
}

// AuthzError represents an authorization failure.
type AuthzError struct {
	// Subject is the identity that was denied.
	Subject string

	// Resource is the resource that was denied access to.
	Resource string

	// Action is the action that was denied.
	Action string

	// Reason explains why access was denied.
	Reason string

	// Cause is the underlying error if any.
	Cause error
}

// Error returns the error message.
func (e *AuthzError) Error() string {
	return fmt.Sprintf("authorization denied: subject=%q resource=%q action=%q reason=%q",
		e.Subject, e.Resource, e.Action, e.Reason)
}

// Unwrap returns the cause error for errors.Is/As support.
func (e *AuthzError) Unwrap() error {
	return e.Cause
}

// Is reports whether this error matches the target.
func (e *AuthzError) Is(target error) bool {
	return target == ErrForbidden
}

// AllowAllAuthorizer permits all requests.
type AllowAllAuthorizer struct{}

// Authorize always returns nil (permitted).
func (a AllowAllAuthorizer) Authorize(_ context.Context, _ *AuthzRequest) error {
	return nil
}

// Name returns "allow_all".
func (a AllowAllAuthorizer) Name() string {
	return "allow_all"
}

// DenyAllAuthorizer denies all requests.
type DenyAllAuthorizer struct{}

// Authorize always returns an error (denied).
func (a DenyAllAuthorizer) Authorize(_ context.Context, req *AuthzRequest) error {
	subject := ""
	if req.Subject != nil {
		subject = req.Subject.Principal
	}
	return &AuthzError{
		Subject:  subject,
		Resource: req.Resource,
		Action:   req.Action,
		Reason:   "all requests denied",
	}
}

// Name returns "deny_all".
func (a DenyAllAuthorizer) Name() string {
	return "deny_all"
}

// AuthorizerFunc is an adapter to allow use of ordinary functions as Authorizers.
type AuthorizerFunc func(ctx context.Context, req *AuthzRequest) error

// Authorize calls the function.
func (f AuthorizerFunc) Authorize(ctx context.Context, req *AuthzRequest) error {
	return f(ctx, req)
}

// Name returns "func" for function-based authorizers.
func (f AuthorizerFunc) Name() string {
	return "func"
}
