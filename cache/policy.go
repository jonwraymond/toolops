package cache

import "time"

// Policy configures caching behavior.
type Policy struct {
	// DefaultTTL is the TTL to use when none is specified.
	// If zero, caching is disabled by default.
	DefaultTTL time.Duration

	// MaxTTL is the maximum allowed TTL. Override TTLs are clamped to this.
	// If zero, no maximum is enforced.
	MaxTTL time.Duration

	// AllowUnsafe permits caching tools with unsafe tags (write, danger, etc.)
	AllowUnsafe bool
}

// DefaultPolicy returns the default caching policy.
// DefaultTTL: 5 minutes, MaxTTL: 1 hour, AllowUnsafe: false
func DefaultPolicy() Policy {
	return Policy{
		DefaultTTL:  5 * time.Minute,
		MaxTTL:      1 * time.Hour,
		AllowUnsafe: false,
	}
}

// NoCachePolicy returns a policy that disables caching entirely.
func NoCachePolicy() Policy {
	return Policy{
		DefaultTTL:  0,
		MaxTTL:      0,
		AllowUnsafe: false,
	}
}

// ShouldCache returns true if caching is enabled by this policy.
func (p Policy) ShouldCache() bool {
	return p.DefaultTTL > 0
}

// EffectiveTTL returns the TTL to use, applying defaults and clamping.
func (p Policy) EffectiveTTL(override time.Duration) time.Duration {
	// Use default if no override (or negative override)
	ttl := override
	if ttl <= 0 {
		ttl = p.DefaultTTL
	}

	// Clamp to MaxTTL if set
	if p.MaxTTL > 0 && ttl > p.MaxTTL {
		ttl = p.MaxTTL
	}

	return ttl
}
