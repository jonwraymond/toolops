package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuth2Config configures the OAuth2 token introspection authenticator.
type OAuth2Config struct {
	// IntrospectionEndpoint is the URL of the OAuth2 introspection endpoint.
	IntrospectionEndpoint string

	// ClientID is the client identifier for introspection requests.
	ClientID string

	// ClientSecret is the client secret for introspection requests.
	ClientSecret string

	// ClientAuthMethod is how to authenticate to the introspection endpoint.
	// Options: "client_secret_basic" (default), "client_secret_post"
	ClientAuthMethod string

	// CacheTTL is how long to cache positive introspection results.
	// Default: 5 minutes. Set to 0 to disable caching.
	CacheTTL time.Duration

	// Timeout is the HTTP request timeout for introspection calls.
	// Default: 10 seconds.
	Timeout time.Duration

	// PrincipalClaim is the claim containing the user principal.
	// Default: "sub"
	PrincipalClaim string

	// TenantClaim is the claim containing the tenant ID.
	TenantClaim string

	// RolesClaim is the claim containing user roles.
	RolesClaim string

	// ScopesClaim is the claim containing OAuth2 scopes.
	// Default: "scope" (space-separated string)
	ScopesClaim string

	// HTTPClient is the HTTP client to use. If nil, a default client is used.
	HTTPClient *http.Client
}

// OAuth2IntrospectionAuthenticator validates OAuth2 tokens via introspection.
type OAuth2IntrospectionAuthenticator struct {
	config     OAuth2Config
	httpClient *http.Client
	cache      *oauth2TokenCache
}

// NewOAuth2IntrospectionAuthenticator creates a new OAuth2 introspection authenticator.
func NewOAuth2IntrospectionAuthenticator(config OAuth2Config) *OAuth2IntrospectionAuthenticator {
	// Apply defaults
	if config.ClientAuthMethod == "" {
		config.ClientAuthMethod = "client_secret_basic"
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.PrincipalClaim == "" {
		config.PrincipalClaim = "sub"
	}
	if config.ScopesClaim == "" {
		config.ScopesClaim = "scope"
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	return &OAuth2IntrospectionAuthenticator{
		config:     config,
		httpClient: httpClient,
		cache:      newOAuth2TokenCache(),
	}
}

// Name returns "oauth2_introspection".
func (a *OAuth2IntrospectionAuthenticator) Name() string {
	return "oauth2_introspection"
}

// Supports returns true if the request contains a Bearer token.
func (a *OAuth2IntrospectionAuthenticator) Supports(_ context.Context, req *AuthRequest) bool {
	header := req.GetHeader("Authorization")
	_, ok := extractBearerToken(header)
	return ok
}

// Authenticate validates the token via introspection.
func (a *OAuth2IntrospectionAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	header := req.GetHeader("Authorization")
	token, ok := extractBearerToken(header)
	if !ok {
		return AuthFailure(ErrMissingCredentials, "Bearer"), nil
	}

	// Check cache first
	tokenHash := hashTokenForCache(token)
	if identity := a.cache.Get(tokenHash); identity != nil {
		return AuthSuccess(identity), nil
	}

	// Perform introspection
	introspectionResult, err := a.introspect(ctx, token)
	if err != nil {
		return nil, err
	}

	if !introspectionResult.Active {
		// Don't cache negative results
		return AuthFailure(ErrTokenInactive, "Bearer"), nil
	}

	// Build identity from introspection response
	identity := a.buildIdentity(introspectionResult)

	// Cache positive result
	a.cache.Set(tokenHash, identity, a.config.CacheTTL)

	return AuthSuccess(identity), nil
}

// introspectionResult represents the OAuth2 introspection response.
type introspectionResult struct {
	Active   bool           `json:"active"`
	Sub      string         `json:"sub"`
	Scope    string         `json:"scope"`
	Exp      int64          `json:"exp"`
	Iat      int64          `json:"iat"`
	ClientID string         `json:"client_id"`
	Claims   map[string]any `json:"-"` // Raw claims
}

func (a *OAuth2IntrospectionAuthenticator) introspect(ctx context.Context, token string) (*introspectionResult, error) {
	// Build request body
	form := url.Values{}
	form.Set("token", token)

	if a.config.ClientAuthMethod == "client_secret_post" {
		form.Set("client_id", a.config.ClientID)
		form.Set("client_secret", a.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.config.IntrospectionEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Add Basic auth if using client_secret_basic
	if a.config.ClientAuthMethod == "client_secret_basic" {
		credentials := base64.StdEncoding.EncodeToString([]byte(a.config.ClientID + ":" + a.config.ClientSecret))
		req.Header.Set("Authorization", "Basic "+credentials)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIntrospectionFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrIntrospectionFailed, resp.StatusCode)
	}

	// Parse response as generic map first to capture all claims
	var rawResponse map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("%w: decode error: %v", ErrIntrospectionFailed, err)
	}

	result := &introspectionResult{
		Claims: rawResponse,
	}

	// Extract standard fields
	if active, ok := rawResponse["active"].(bool); ok {
		result.Active = active
	}
	if sub, ok := rawResponse["sub"].(string); ok {
		result.Sub = sub
	}
	if scope, ok := rawResponse["scope"].(string); ok {
		result.Scope = scope
	}
	if exp, ok := rawResponse["exp"].(float64); ok {
		result.Exp = int64(exp)
	}
	if iat, ok := rawResponse["iat"].(float64); ok {
		result.Iat = int64(iat)
	}
	if clientID, ok := rawResponse["client_id"].(string); ok {
		result.ClientID = clientID
	}

	return result, nil
}

func (a *OAuth2IntrospectionAuthenticator) buildIdentity(result *introspectionResult) *Identity {
	identity := &Identity{
		Method: AuthMethodOAuth2,
		Claims: result.Claims,
	}

	// Extract principal
	if a.config.PrincipalClaim != "" {
		if principal, ok := result.Claims[a.config.PrincipalClaim].(string); ok {
			identity.Principal = principal
		}
	}

	// Extract tenant
	if a.config.TenantClaim != "" {
		if tenant, ok := result.Claims[a.config.TenantClaim].(string); ok {
			identity.TenantID = tenant
		}
	}

	// Extract roles
	if a.config.RolesClaim != "" {
		if roles, ok := result.Claims[a.config.RolesClaim].([]any); ok {
			identity.Roles = make([]string, 0, len(roles))
			for _, r := range roles {
				if s, ok := r.(string); ok {
					identity.Roles = append(identity.Roles, s)
				}
			}
		}
	}

	// Extract scopes as permissions
	if a.config.ScopesClaim != "" {
		if scope, ok := result.Claims[a.config.ScopesClaim].(string); ok && scope != "" {
			identity.Permissions = strings.Split(scope, " ")
		}
	}

	// Extract expiration
	if result.Exp > 0 {
		identity.ExpiresAt = time.Unix(result.Exp, 0)
	}

	// Extract issued at
	if result.Iat > 0 {
		identity.IssuedAt = time.Unix(result.Iat, 0)
	}

	return identity
}

// extractBearerToken extracts the token from a Bearer authorization header.
func extractBearerToken(header string) (string, bool) {
	if header == "" {
		return "", false
	}

	// Case-insensitive "Bearer " prefix check
	if len(header) < 7 {
		return "", false
	}

	prefix := strings.ToLower(header[:7])
	if prefix != "bearer " {
		return "", false
	}

	token := strings.TrimSpace(header[7:])
	if token == "" {
		return "", false
	}

	return token, true
}

// hashTokenForCache creates a SHA256 hash of the token for cache keys.
func hashTokenForCache(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// oauth2TokenCache provides thread-safe caching for introspection results.
type oauth2TokenCache struct {
	mu      sync.RWMutex
	entries map[string]*oauth2CacheEntry
}

type oauth2CacheEntry struct {
	identity  *Identity
	expiresAt time.Time
}

func newOAuth2TokenCache() *oauth2TokenCache {
	return &oauth2TokenCache{
		entries: make(map[string]*oauth2CacheEntry),
	}
}

// Get retrieves an identity from the cache.
func (c *oauth2TokenCache) Get(tokenHash string) *Identity {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[tokenHash]
	if !ok {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.identity
}

// Set stores an identity in the cache.
func (c *oauth2TokenCache) Set(tokenHash string, identity *Identity, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[tokenHash] = &oauth2CacheEntry{
		identity:  identity,
		expiresAt: time.Now().Add(ttl),
	}
}

// Ensure OAuth2IntrospectionAuthenticator implements Authenticator
var _ Authenticator = (*OAuth2IntrospectionAuthenticator)(nil)
