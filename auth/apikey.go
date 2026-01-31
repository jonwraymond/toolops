package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// APIKeyConfig configures the API key authenticator.
type APIKeyConfig struct {
	// HeaderName is the header containing the API key.
	// Default: "X-API-Key"
	HeaderName string

	// HashAlgorithm is the algorithm used to hash stored keys.
	// Options: "sha256" (default), "plain" (not recommended)
	HashAlgorithm string
}

// APIKeyInfo contains information about a registered API key.
type APIKeyInfo struct {
	// ID is a unique identifier for this key.
	ID string

	// KeyHash is the hashed API key (SHA-256 hex).
	KeyHash string

	// Principal is the identity associated with this key.
	Principal string

	// TenantID is the tenant this key belongs to.
	TenantID string

	// Roles are the roles granted to this key.
	Roles []string

	// ExpiresAt is when this key expires (zero = never).
	ExpiresAt time.Time

	// Metadata contains additional key metadata.
	Metadata map[string]any
}

// APIKeyStore provides storage for API keys.
type APIKeyStore interface {
	// Lookup retrieves an API key by its hash.
	// Returns nil if not found.
	Lookup(ctx context.Context, keyHash string) (*APIKeyInfo, error)
}

// APIKeyAuthenticator validates API keys.
type APIKeyAuthenticator struct {
	config APIKeyConfig
	store  APIKeyStore
}

// NewAPIKeyAuthenticator creates a new API key authenticator.
func NewAPIKeyAuthenticator(config APIKeyConfig, store APIKeyStore) *APIKeyAuthenticator {
	// Apply defaults
	if config.HeaderName == "" {
		config.HeaderName = "X-API-Key"
	}
	if config.HashAlgorithm == "" {
		config.HashAlgorithm = "sha256"
	}

	return &APIKeyAuthenticator{
		config: config,
		store:  store,
	}
}

// Name returns "api_key".
func (a *APIKeyAuthenticator) Name() string {
	return "api_key"
}

// Supports returns true if the request contains an API key header.
func (a *APIKeyAuthenticator) Supports(_ context.Context, req *AuthRequest) bool {
	return req.GetHeader(a.config.HeaderName) != ""
}

// Authenticate validates the API key.
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	apiKey := req.GetHeader(a.config.HeaderName)
	if apiKey == "" {
		return AuthFailure(ErrMissingCredentials, "api_key"), nil
	}

	// Trim whitespace
	apiKey = strings.TrimSpace(apiKey)

	// Hash the provided key
	keyHash := a.hashKey(apiKey)

	// Look up the key
	info, err := a.store.Lookup(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	if info == nil {
		return AuthFailure(ErrInvalidCredentials, "api_key"), nil
	}

	// Check expiration
	if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
		return AuthFailure(ErrTokenExpired, "api_key"), nil
	}

	// Build identity
	identity := &Identity{
		Principal: info.Principal,
		TenantID:  info.TenantID,
		Roles:     info.Roles,
		Method:    AuthMethodAPIKey,
		ExpiresAt: info.ExpiresAt,
		Claims:    make(map[string]any),
	}

	// Add metadata to claims
	if info.Metadata != nil {
		for k, v := range info.Metadata {
			identity.Claims[k] = v
		}
	}
	identity.Claims["key_id"] = info.ID

	return AuthSuccess(identity), nil
}

func (a *APIKeyAuthenticator) hashKey(key string) string {
	switch a.config.HashAlgorithm {
	case "plain":
		return key
	case "sha256", "":
		hash := sha256.Sum256([]byte(key))
		return hex.EncodeToString(hash[:])
	default:
		// Default to SHA-256 for unknown algorithms
		hash := sha256.Sum256([]byte(key))
		return hex.EncodeToString(hash[:])
	}
}

// HashAPIKey hashes an API key using SHA-256 for storage.
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ConstantTimeCompare performs constant-time comparison of two strings.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// MemoryAPIKeyStore is an in-memory API key store.
type MemoryAPIKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*APIKeyInfo // keyed by hash
}

// NewMemoryAPIKeyStore creates a new in-memory API key store.
func NewMemoryAPIKeyStore() *MemoryAPIKeyStore {
	return &MemoryAPIKeyStore{
		keys: make(map[string]*APIKeyInfo),
	}
}

// Lookup retrieves an API key by its hash.
func (s *MemoryAPIKeyStore) Lookup(_ context.Context, keyHash string) (*APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.keys[keyHash], nil
}

// Add adds an API key to the store.
func (s *MemoryAPIKeyStore) Add(info *APIKeyInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[info.KeyHash] = info
	return nil
}

// Remove removes an API key from the store.
func (s *MemoryAPIKeyStore) Remove(keyHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, keyHash)
	return nil
}

// Ensure APIKeyAuthenticator implements Authenticator
var _ Authenticator = (*APIKeyAuthenticator)(nil)

// Ensure MemoryAPIKeyStore implements APIKeyStore
var _ APIKeyStore = (*MemoryAPIKeyStore)(nil)
