// Package cache provides deterministic caching for tool executions.
//
// It provides a Cache interface with memory implementation, SHA-256-based
// key derivation, and TTL policies with unsafe tag handling.
//
// # Ecosystem Position
//
// cache sits between tool invocation and tool execution, intercepting requests
// to avoid redundant computation:
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│                      Tool Execution Flow                        │
//	├─────────────────────────────────────────────────────────────────┤
//	│                                                                 │
//	│   toolexec            cache               toolexec              │
//	│   ┌──────┐         ┌─────────┐          ┌─────────┐            │
//	│   │ Tool │────────▶│Middleware│─────────▶│Executor │            │
//	│   │ Call │         │         │          │         │            │
//	│   └──────┘         │ ┌─────┐ │   miss   └─────────┘            │
//	│       ▲            │ │Keyer│ │              │                   │
//	│       │            │ ├─────┤ │              │                   │
//	│       │            │ │Cache│◀──────────────┘                   │
//	│       │            │ ├─────┤ │   store                         │
//	│       │    hit     │ │Policy│ │                                 │
//	│       └────────────│ └─────┘ │                                 │
//	│                    └─────────┘                                 │
//	│                                                                 │
//	└─────────────────────────────────────────────────────────────────┘
//
// # Core Components
//
//   - [Cache]: Interface for caching tool execution results (Get/Set/Delete)
//   - [MemoryCache]: Thread-safe in-memory cache with TTL support
//   - [Keyer]: Interface for deterministic cache key generation
//   - [DefaultKeyer]: SHA-256 based keyer with canonical JSON serialization
//   - [Policy]: Configures TTL defaults, maximums, and unsafe tag handling
//   - [CacheMiddleware]: Transparent caching wrapper for tool execution
//
// # Quick Start
//
//	// Create cache with policy
//	policy := cache.DefaultPolicy() // 5min TTL, 1hr max
//	memCache := cache.NewMemoryCache(policy)
//	keyer := cache.NewDefaultKeyer()
//
//	// Create middleware
//	mw := cache.NewCacheMiddleware(memCache, keyer, policy, nil)
//
//	// Execute with caching
//	result, err := mw.Execute(ctx, "github.search", input, tags,
//	    func(ctx context.Context, toolID string, input any) ([]byte, error) {
//	        return actualExecutor(ctx, toolID, input)
//	    })
//
// # Key Generation
//
// The [DefaultKeyer] generates deterministic cache keys using:
//
//	cache:<toolID>:<hash>
//
// Where hash is the first 16 hex characters of SHA-256(canonical JSON(input)).
// Canonical JSON ensures map keys are sorted for deterministic serialization.
//
// # TTL Policies
//
// The [Policy] type controls caching behavior:
//
//   - DefaultTTL: Applied when no specific TTL is provided
//   - MaxTTL: Upper bound for any TTL (prevents excessive caching)
//   - AllowUnsafe: Whether to cache tools with unsafe tags
//
// Preset policies:
//
//   - [DefaultPolicy]: 5 minute default, 1 hour max, unsafe=false
//   - [NoCachePolicy]: Disabled (0 TTL)
//
// # Unsafe Tag Handling
//
// Tools with certain tags should not be cached because they have side effects:
//
//   - write, danger, unsafe, mutation, delete
//
// The [DefaultSkipRule] checks for these tags (case-insensitive) and skips
// caching. Override via [NewCacheMiddleware]'s skipRule parameter.
//
// # Thread Safety
//
// All exported types are safe for concurrent use:
//
//   - [MemoryCache]: sync.RWMutex protects all operations
//   - [DefaultKeyer]: Stateless, concurrent-safe
//   - [CacheMiddleware]: Delegates to thread-safe Cache/Keyer
//   - [Policy]: Immutable struct, concurrent-safe
//
// # Error Handling
//
// Sentinel errors (use errors.Is for checking):
//
//   - [ErrNilCache]: Cache is nil
//   - [ErrInvalidKey]: Key is empty, whitespace-only, or contains newlines
//   - [ErrKeyTooLong]: Key exceeds MaxKeyLength (512 characters)
//
// Note: Cache.Get never returns errors - it returns (nil, false) on miss.
// Key validation is performed via [ValidateKey] function.
//
// # Integration with ApertureStack
//
// cache integrates with other ApertureStack packages:
//
//   - toolexec: Wrap tool execution with CacheMiddleware
//   - observe: Log cache hits/misses via observability middleware
//   - resilience: Combine with retry/circuit breaker for robust caching
package cache
