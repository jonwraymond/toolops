package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig configures the JWT authenticator.
type JWTConfig struct {
	// Issuer is the expected token issuer (iss claim).
	Issuer string

	// Audience is the expected token audience (aud claim).
	Audience string

	// HeaderName is the header containing the token.
	// Default: "Authorization"
	HeaderName string

	// TokenPrefix is the prefix before the token in the header.
	// Default: "Bearer "
	TokenPrefix string

	// PrincipalClaim is the claim containing the user principal.
	// Default: "sub"
	PrincipalClaim string

	// TenantClaim is the claim containing the tenant ID.
	TenantClaim string

	// RolesClaim is the claim containing user roles.
	RolesClaim string
}

// KeyProvider retrieves signing keys for JWT validation.
type KeyProvider interface {
	// GetKey returns the key for the given key ID.
	GetKey(ctx context.Context, keyID string) (any, error)
}

// StaticKeyProvider provides a static signing key.
type StaticKeyProvider struct {
	key []byte
}

// NewStaticKeyProvider creates a static key provider.
func NewStaticKeyProvider(key []byte) *StaticKeyProvider {
	return &StaticKeyProvider{key: key}
}

// GetKey returns the static key.
func (p *StaticKeyProvider) GetKey(_ context.Context, _ string) (any, error) {
	return p.key, nil
}

// JWTAuthenticator validates JWT tokens.
type JWTAuthenticator struct {
	config      JWTConfig
	keyProvider KeyProvider
}

// NewJWTAuthenticator creates a new JWT authenticator.
func NewJWTAuthenticator(config JWTConfig, keyProvider KeyProvider) *JWTAuthenticator {
	// Apply defaults
	if config.HeaderName == "" {
		config.HeaderName = "Authorization"
	}
	if config.TokenPrefix == "" {
		config.TokenPrefix = "Bearer "
	}
	if config.PrincipalClaim == "" {
		config.PrincipalClaim = "sub"
	}

	return &JWTAuthenticator{
		config:      config,
		keyProvider: keyProvider,
	}
}

// Name returns "jwt".
func (a *JWTAuthenticator) Name() string {
	return "jwt"
}

// Supports returns true if the request contains a JWT token.
func (a *JWTAuthenticator) Supports(_ context.Context, req *AuthRequest) bool {
	header := req.GetHeader(a.config.HeaderName)
	return strings.HasPrefix(header, a.config.TokenPrefix)
}

// Authenticate validates the JWT token.
func (a *JWTAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	header := req.GetHeader(a.config.HeaderName)
	if header == "" {
		return AuthFailure(ErrMissingCredentials, "jwt"), nil
	}

	// Extract token
	tokenString := strings.TrimPrefix(header, a.config.TokenPrefix)
	if tokenString == header {
		return AuthFailure(ErrMissingCredentials, "jwt"), nil
	}
	tokenString = strings.TrimSpace(tokenString)

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Get key ID from header
		kid := ""
		if kidVal, ok := token.Header["kid"].(string); ok {
			kid = kidVal
		}

		return a.keyProvider.GetKey(ctx, kid)
	})

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return AuthFailure(ErrTokenExpired, "jwt"), nil
		}
		return AuthFailure(ErrTokenMalformed, "jwt"), nil
	}

	if !token.Valid {
		return AuthFailure(ErrInvalidCredentials, "jwt"), nil
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return AuthFailure(ErrTokenMalformed, "jwt"), nil
	}

	// Validate issuer if configured
	if a.config.Issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != a.config.Issuer {
			return AuthFailure(ErrInvalidCredentials, "jwt"), nil
		}
	}

	// Validate audience if configured
	if a.config.Audience != "" {
		aud := a.getAudience(claims)
		if !a.containsAudience(aud, a.config.Audience) {
			return AuthFailure(ErrInvalidCredentials, "jwt"), nil
		}
	}

	// Build identity
	identity := a.buildIdentity(claims)

	return AuthSuccess(identity), nil
}

func (a *JWTAuthenticator) getAudience(claims jwt.MapClaims) []string {
	switch v := claims["aud"].(type) {
	case string:
		return []string{v}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, a := range v {
			if s, ok := a.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func (a *JWTAuthenticator) containsAudience(audiences []string, target string) bool {
	for _, aud := range audiences {
		if aud == target {
			return true
		}
	}
	return false
}

func (a *JWTAuthenticator) buildIdentity(claims jwt.MapClaims) *Identity {
	identity := &Identity{
		Method: AuthMethodJWT,
		Claims: make(map[string]any),
	}

	// Copy claims
	for k, v := range claims {
		identity.Claims[k] = v
	}

	// Extract principal
	if principal, ok := claims[a.config.PrincipalClaim].(string); ok {
		identity.Principal = principal
	}

	// Extract tenant
	if a.config.TenantClaim != "" {
		if tenant, ok := claims[a.config.TenantClaim].(string); ok {
			identity.TenantID = tenant
		}
	}

	// Extract roles
	if a.config.RolesClaim != "" {
		if roles, ok := claims[a.config.RolesClaim].([]interface{}); ok {
			identity.Roles = make([]string, 0, len(roles))
			for _, r := range roles {
				if s, ok := r.(string); ok {
					identity.Roles = append(identity.Roles, s)
				}
			}
		}
	}

	// Extract expiration
	if exp, ok := claims["exp"].(float64); ok {
		identity.ExpiresAt = time.Unix(int64(exp), 0)
	}

	// Extract issued at
	if iat, ok := claims["iat"].(float64); ok {
		identity.IssuedAt = time.Unix(int64(iat), 0)
	}

	return identity
}

// Ensure JWTAuthenticator implements Authenticator
var _ Authenticator = (*JWTAuthenticator)(nil)

// Ensure StaticKeyProvider implements KeyProvider
var _ KeyProvider = (*StaticKeyProvider)(nil)

// Helper to format errors
func wrapJWTError(err error) error {
	return fmt.Errorf("jwt: %w", err)
}
