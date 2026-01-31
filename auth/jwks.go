package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// JWKSConfig configures the JWKS key provider.
type JWKSConfig struct {
	// URL is the JWKS endpoint URL.
	URL string

	// CacheTTL is how long to cache keys before refreshing.
	// Default: 1 hour
	CacheTTL time.Duration

	// HTTPClient is the HTTP client to use for requests.
	// If nil, a default client with 30s timeout is used.
	HTTPClient *http.Client
}

// JWKSKeyProvider retrieves signing keys from a JWKS endpoint.
// It implements the KeyProvider interface with caching support.
type JWKSKeyProvider struct {
	config JWKSConfig

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	cacheTime   time.Time
	lastFetched map[string]*rsa.PublicKey // backup for graceful degradation
	sfGroup     singleflight.Group        // prevents thundering herd
}

// NewJWKSKeyProvider creates a new JWKS key provider.
func NewJWKSKeyProvider(config JWKSConfig) *JWKSKeyProvider {
	// Apply defaults
	if config.CacheTTL == 0 {
		config.CacheTTL = time.Hour
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &JWKSKeyProvider{
		config:      config,
		keys:        make(map[string]*rsa.PublicKey),
		lastFetched: make(map[string]*rsa.PublicKey),
	}
}

// GetKey returns the key for the given key ID.
// If keyID is empty and there's exactly one key, that key is returned.
func (p *JWKSKeyProvider) GetKey(ctx context.Context, keyID string) (any, error) {
	// Check cache first
	p.mu.RLock()
	cacheValid := time.Since(p.cacheTime) < p.config.CacheTTL
	if cacheValid {
		key := p.lookupKeyLocked(keyID)
		p.mu.RUnlock()
		if key != nil {
			return key, nil
		}
		// Key not in cache, need to refresh
	} else {
		p.mu.RUnlock()
	}

	// Refresh keys using singleflight to prevent thundering herd
	_, err, _ := p.sfGroup.Do("refresh", func() (any, error) {
		return nil, p.refresh(ctx)
	})
	if err != nil {
		// On refresh failure, try to use cached key (graceful degradation)
		p.mu.RLock()
		key := p.lookupKeyLocked(keyID)
		if key == nil {
			// Also check lastFetched backup
			key = p.lookupFromBackupLocked(keyID)
		}
		p.mu.RUnlock()

		if key != nil {
			return key, nil
		}
		return nil, err
	}

	// Look up key after refresh
	p.mu.RLock()
	key := p.lookupKeyLocked(keyID)
	p.mu.RUnlock()

	if key == nil {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

// lookupKeyLocked finds a key by ID. Caller must hold at least RLock.
func (p *JWKSKeyProvider) lookupKeyLocked(keyID string) *rsa.PublicKey {
	if keyID == "" {
		// Return first key if no keyID specified
		for _, key := range p.keys {
			return key
		}
		return nil
	}
	return p.keys[keyID]
}

// lookupFromBackupLocked finds a key in the backup cache. Caller must hold at least RLock.
func (p *JWKSKeyProvider) lookupFromBackupLocked(keyID string) *rsa.PublicKey {
	if keyID == "" {
		for _, key := range p.lastFetched {
			return key
		}
		return nil
	}
	return p.lastFetched[keyID]
}

// refresh fetches keys from the JWKS endpoint.
func (p *JWKSKeyProvider) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config.URL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	// Parse all RSA keys
	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue // Skip non-RSA keys
		}

		pubKey, err := parseRSAPublicKey(jwk)
		if err != nil {
			continue // Skip invalid keys
		}

		keys[jwk.Kid] = pubKey
	}

	// Update cache
	p.mu.Lock()
	p.keys = keys
	p.cacheTime = time.Now()
	// Backup for graceful degradation
	for kid, key := range keys {
		p.lastFetched[kid] = key
	}
	p.mu.Unlock()

	return nil
}

// jwksResponse is the JWKS endpoint response format.
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

// jwkKey represents a single JWK.
type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// parseRSAPublicKey converts a JWK to an RSA public key.
func parseRSAPublicKey(jwk jwkKey) (*rsa.PublicKey, error) {
	if jwk.N == "" {
		return nil, fmt.Errorf("missing n parameter")
	}
	if jwk.E == "" {
		return nil, fmt.Errorf("missing e parameter")
	}

	// Decode modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// Ensure JWKSKeyProvider implements KeyProvider
var _ KeyProvider = (*JWKSKeyProvider)(nil)
