package auth

import (
	"context"
	"strings"
)

// RBACConfig configures the simple RBAC authorizer.
type RBACConfig struct {
	// Roles defines role configurations.
	Roles map[string]RoleConfig

	// DefaultRole is assigned to identities without explicit roles.
	DefaultRole string
}

// RoleConfig defines permissions for a role.
type RoleConfig struct {
	// Permissions are explicit permission strings (e.g., "tool:*:call").
	Permissions []string

	// Inherits lists roles this role inherits from.
	Inherits []string

	// AllowedTools is a list of tool names this role can access.
	AllowedTools []string

	// DeniedTools is a list of tool names this role cannot access.
	DeniedTools []string

	// AllowedActions is a list of actions this role can perform.
	AllowedActions []string
}

// SimpleRBACAuthorizer provides simple role-based access control.
type SimpleRBACAuthorizer struct {
	config RBACConfig
}

// NewSimpleRBACAuthorizer creates a new simple RBAC authorizer.
func NewSimpleRBACAuthorizer(config RBACConfig) *SimpleRBACAuthorizer {
	return &SimpleRBACAuthorizer{config: config}
}

// Name returns "simple_rbac".
func (a *SimpleRBACAuthorizer) Name() string {
	return "simple_rbac"
}

// Authorize checks if the identity is allowed to perform the action.
func (a *SimpleRBACAuthorizer) Authorize(_ context.Context, req *AuthzRequest) error {
	if req.Subject == nil {
		return &AuthzError{
			Resource: req.Resource,
			Action:   req.Action,
			Reason:   "no identity provided",
		}
	}

	// Collect all roles (including inherited)
	roles := a.collectRoles(req.Subject)

	// Check if any role permits this request
	for _, roleName := range roles {
		role, ok := a.config.Roles[roleName]
		if !ok {
			continue
		}

		if a.rolePermits(role, req) {
			return nil // Allowed
		}
	}

	return &AuthzError{
		Subject:  req.Subject.Principal,
		Resource: req.Resource,
		Action:   req.Action,
		Reason:   "no role permits this action",
	}
}

func (a *SimpleRBACAuthorizer) collectRoles(subject *Identity) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	// Start with subject's roles
	rolesToProcess := append([]string{}, subject.Roles...)

	// Add default role if no roles
	if len(rolesToProcess) == 0 && a.config.DefaultRole != "" {
		rolesToProcess = append(rolesToProcess, a.config.DefaultRole)
	}

	// Process roles with inheritance
	for len(rolesToProcess) > 0 {
		current := rolesToProcess[0]
		rolesToProcess = rolesToProcess[1:]

		if seen[current] {
			continue
		}
		seen[current] = true
		result = append(result, current)

		// Add inherited roles
		if role, ok := a.config.Roles[current]; ok {
			for _, inherited := range role.Inherits {
				if !seen[inherited] {
					rolesToProcess = append(rolesToProcess, inherited)
				}
			}
		}
	}

	return result
}

func (a *SimpleRBACAuthorizer) rolePermits(role RoleConfig, req *AuthzRequest) bool {
	toolName := req.ToolName()

	// Check denied tools first (deny takes precedence)
	for _, denied := range role.DeniedTools {
		if matchPattern(denied, toolName) {
			return false
		}
	}

	// Check allowed tools
	if len(role.AllowedTools) > 0 {
		allowed := false
		for _, allowedTool := range role.AllowedTools {
			if matchPattern(allowedTool, toolName) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check allowed actions
	if len(role.AllowedActions) > 0 {
		allowed := false
		for _, action := range role.AllowedActions {
			if action == "*" || action == req.Action {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check explicit permissions
	for _, perm := range role.Permissions {
		if matchPermission(perm, req) {
			return true
		}
	}

	// If we have allowed tools but no explicit permissions,
	// and tool passed the allowed check, permit
	if len(role.AllowedTools) > 0 {
		return true
	}

	return false
}

// matchPattern matches a pattern against a value.
// Supports "*" as a wildcard for any characters.
func matchPattern(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(value, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == value
}

// matchPermission checks if a permission string matches a request.
// Format: <resource_type>:<resource>:<action> or just <action>
// Wildcards (*) are supported.
func matchPermission(perm string, req *AuthzRequest) bool {
	parts := strings.Split(perm, ":")

	switch len(parts) {
	case 1:
		// Just action
		return parts[0] == "*" || parts[0] == req.Action
	case 2:
		// resource:action
		resource := parts[0]
		action := parts[1]
		resourceMatch := resource == "*" || resource == req.Resource || matchPattern(resource, req.ToolName())
		actionMatch := action == "*" || action == req.Action
		return resourceMatch && actionMatch
	case 3:
		// resource_type:resource:action
		resourceType := parts[0]
		resource := parts[1]
		action := parts[2]
		typeMatch := resourceType == "*" || resourceType == req.ResourceType
		resourceMatch := resource == "*" || resource == req.Resource || matchPattern(resource, req.ToolName())
		actionMatch := action == "*" || action == req.Action
		return typeMatch && resourceMatch && actionMatch
	default:
		return false
	}
}

// Ensure SimpleRBACAuthorizer implements Authorizer
var _ Authorizer = (*SimpleRBACAuthorizer)(nil)
