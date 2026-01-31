package auth

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// AuthenticatorFactory creates an authenticator from configuration.
type AuthenticatorFactory func(cfg map[string]any) (Authenticator, error)

// AuthorizerFactory creates an authorizer from configuration.
type AuthorizerFactory func(cfg map[string]any) (Authorizer, error)

// Registry manages authenticator and authorizer factories.
type Registry struct {
	mu             sync.RWMutex
	authenticators map[string]AuthenticatorFactory
	authorizers    map[string]AuthorizerFactory
}

// NewRegistry creates a new auth registry.
func NewRegistry() *Registry {
	return &Registry{
		authenticators: make(map[string]AuthenticatorFactory),
		authorizers:    make(map[string]AuthorizerFactory),
	}
}

// RegisterAuthenticator adds an authenticator factory.
func (r *Registry) RegisterAuthenticator(name string, factory AuthenticatorFactory) error {
	if name == "" || factory == nil {
		return errors.New("invalid authenticator registration")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.authenticators[name]; exists {
		return fmt.Errorf("authenticator %q already registered", name)
	}

	r.authenticators[name] = factory
	return nil
}

// RegisterAuthorizer adds an authorizer factory.
func (r *Registry) RegisterAuthorizer(name string, factory AuthorizerFactory) error {
	if name == "" || factory == nil {
		return errors.New("invalid authorizer registration")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.authorizers[name]; exists {
		return fmt.Errorf("authorizer %q already registered", name)
	}

	r.authorizers[name] = factory
	return nil
}

// CreateAuthenticator instantiates an authenticator by name.
func (r *Registry) CreateAuthenticator(name string, cfg map[string]any) (Authenticator, error) {
	r.mu.RLock()
	factory, ok := r.authenticators[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("authenticator %q not found", name)
	}

	return factory(cfg)
}

// CreateAuthorizer instantiates an authorizer by name.
func (r *Registry) CreateAuthorizer(name string, cfg map[string]any) (Authorizer, error) {
	r.mu.RLock()
	factory, ok := r.authorizers[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("authorizer %q not found", name)
	}

	return factory(cfg)
}

// ListAuthenticators returns registered authenticator names.
func (r *Registry) ListAuthenticators() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.authenticators))
	for name := range r.authenticators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListAuthorizers returns registered authorizer names.
func (r *Registry) ListAuthorizers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.authorizers))
	for name := range r.authorizers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultRegistry is the global auth registry with built-in factories.
var DefaultRegistry = NewRegistry()

func init() {
	// Register JWT authenticator
	_ = DefaultRegistry.RegisterAuthenticator("jwt", func(cfg map[string]any) (Authenticator, error) {
		config := JWTConfig{}

		if issuer, ok := cfg["issuer"].(string); ok {
			config.Issuer = issuer
		}
		if audience, ok := cfg["audience"].(string); ok {
			config.Audience = audience
		}
		if headerName, ok := cfg["header_name"].(string); ok {
			config.HeaderName = headerName
		}
		if tokenPrefix, ok := cfg["token_prefix"].(string); ok {
			config.TokenPrefix = tokenPrefix
		}
		if principalClaim, ok := cfg["principal_claim"].(string); ok {
			config.PrincipalClaim = principalClaim
		}
		if tenantClaim, ok := cfg["tenant_claim"].(string); ok {
			config.TenantClaim = tenantClaim
		}
		if rolesClaim, ok := cfg["roles_claim"].(string); ok {
			config.RolesClaim = rolesClaim
		}

		// Get key provider - support JWKS URL or static secret
		var keyProvider KeyProvider
		if jwksURL, ok := cfg["jwks_url"].(string); ok {
			jwksConfig := JWKSConfig{URL: jwksURL}
			if cacheTTL, ok := cfg["cache_ttl"].(string); ok {
				if d, err := time.ParseDuration(cacheTTL); err == nil {
					jwksConfig.CacheTTL = d
				}
			}
			keyProvider = NewJWKSKeyProvider(jwksConfig)
		} else if secret, ok := cfg["secret"].(string); ok {
			keyProvider = NewStaticKeyProvider([]byte(secret))
		} else {
			// Default to empty key (will fail validation)
			keyProvider = NewStaticKeyProvider([]byte{})
		}

		return NewJWTAuthenticator(config, keyProvider), nil
	})

	// Register API key authenticator
	_ = DefaultRegistry.RegisterAuthenticator("api_key", func(cfg map[string]any) (Authenticator, error) {
		config := APIKeyConfig{}

		if headerName, ok := cfg["header_name"].(string); ok {
			config.HeaderName = headerName
		}
		if algorithm, ok := cfg["hash_algorithm"].(string); ok {
			config.HashAlgorithm = algorithm
		}

		// For MVP, use memory store
		store := NewMemoryAPIKeyStore()

		// Pre-populate keys if provided
		if keys, ok := cfg["keys"].([]any); ok {
			for _, k := range keys {
				if keyMap, ok := k.(map[string]any); ok {
					info := &APIKeyInfo{}
					if id, ok := keyMap["id"].(string); ok {
						info.ID = id
					}
					if hash, ok := keyMap["hash"].(string); ok {
						info.KeyHash = hash
					}
					if principal, ok := keyMap["principal"].(string); ok {
						info.Principal = principal
					}
					if tenantID, ok := keyMap["tenant_id"].(string); ok {
						info.TenantID = tenantID
					}
					if roles, ok := keyMap["roles"].([]any); ok {
						for _, r := range roles {
							if s, ok := r.(string); ok {
								info.Roles = append(info.Roles, s)
							}
						}
					}
					_ = store.Add(info)
				}
			}
		}

		return NewAPIKeyAuthenticator(config, store), nil
	})

	// Register simple RBAC authorizer
	_ = DefaultRegistry.RegisterAuthorizer("simple_rbac", func(cfg map[string]any) (Authorizer, error) {
		config := RBACConfig{
			Roles: make(map[string]RoleConfig),
		}

		if defaultRole, ok := cfg["default_role"].(string); ok {
			config.DefaultRole = defaultRole
		}

		if roles, ok := cfg["roles"].(map[string]any); ok {
			for roleName, roleData := range roles {
				roleConfig := RoleConfig{}
				if rd, ok := roleData.(map[string]any); ok {
					if perms, ok := rd["permissions"].([]any); ok {
						for _, p := range perms {
							if s, ok := p.(string); ok {
								roleConfig.Permissions = append(roleConfig.Permissions, s)
							}
						}
					}
					if inherits, ok := rd["inherits"].([]any); ok {
						for _, i := range inherits {
							if s, ok := i.(string); ok {
								roleConfig.Inherits = append(roleConfig.Inherits, s)
							}
						}
					}
					if tools, ok := rd["allowed_tools"].([]any); ok {
						for _, t := range tools {
							if s, ok := t.(string); ok {
								roleConfig.AllowedTools = append(roleConfig.AllowedTools, s)
							}
						}
					}
					if tools, ok := rd["denied_tools"].([]any); ok {
						for _, t := range tools {
							if s, ok := t.(string); ok {
								roleConfig.DeniedTools = append(roleConfig.DeniedTools, s)
							}
						}
					}
					if actions, ok := rd["allowed_actions"].([]any); ok {
						for _, a := range actions {
							if s, ok := a.(string); ok {
								roleConfig.AllowedActions = append(roleConfig.AllowedActions, s)
							}
						}
					}
				}
				config.Roles[roleName] = roleConfig
			}
		}

		return NewSimpleRBACAuthorizer(config), nil
	})

	// Register allow_all authorizer
	_ = DefaultRegistry.RegisterAuthorizer("allow_all", func(cfg map[string]any) (Authorizer, error) {
		return AllowAllAuthorizer{}, nil
	})

	// Register deny_all authorizer
	_ = DefaultRegistry.RegisterAuthorizer("deny_all", func(cfg map[string]any) (Authorizer, error) {
		return DenyAllAuthorizer{}, nil
	})

	// Register OAuth2 introspection authenticator
	_ = DefaultRegistry.RegisterAuthenticator("oauth2_introspection", func(cfg map[string]any) (Authenticator, error) {
		config := OAuth2Config{}

		if endpoint, ok := cfg["introspection_endpoint"].(string); ok {
			config.IntrospectionEndpoint = endpoint
		}
		if clientID, ok := cfg["client_id"].(string); ok {
			config.ClientID = clientID
		}
		if clientSecret, ok := cfg["client_secret"].(string); ok {
			config.ClientSecret = clientSecret
		}
		if authMethod, ok := cfg["client_auth_method"].(string); ok {
			config.ClientAuthMethod = authMethod
		}
		if cacheTTL, ok := cfg["cache_ttl"].(string); ok {
			if d, err := time.ParseDuration(cacheTTL); err == nil {
				config.CacheTTL = d
			}
		}
		if timeout, ok := cfg["timeout"].(string); ok {
			if d, err := time.ParseDuration(timeout); err == nil {
				config.Timeout = d
			}
		}
		if principalClaim, ok := cfg["principal_claim"].(string); ok {
			config.PrincipalClaim = principalClaim
		}
		if tenantClaim, ok := cfg["tenant_claim"].(string); ok {
			config.TenantClaim = tenantClaim
		}
		if rolesClaim, ok := cfg["roles_claim"].(string); ok {
			config.RolesClaim = rolesClaim
		}
		if scopesClaim, ok := cfg["scopes_claim"].(string); ok {
			config.ScopesClaim = scopesClaim
		}

		return NewOAuth2IntrospectionAuthenticator(config), nil
	})
}
