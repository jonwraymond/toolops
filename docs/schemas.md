# Schemas and Contracts

This document describes the configuration schemas and behavioral contracts for
`toolops`. These schemas are expressed as Go types with validation rules.

## observe

### Config

`observe.Config` is the root configuration for observability.

| Field | Type | Required | Notes |
|------|------|----------|-------|
| `ServiceName` | `string` | **Yes** | Required for tracing/metrics resource attribution. |
| `Version` | `string` | No | Optional service version label. |
| `Tracing` | `TracingConfig` | No | Enables and configures tracing. |
| `Metrics` | `MetricsConfig` | No | Enables and configures metrics. |
| `Logging` | `LoggingConfig` | No | Enables structured logs. |

Validation errors (sentinels):
- `ErrMissingServiceName`
- `ErrInvalidTracingExporter`
- `ErrInvalidSamplePct`
- `ErrInvalidMetricsExporter`
- `ErrInvalidLogLevel`

### TracingConfig

| Field | Type | Required | Notes |
|------|------|----------|-------|
| `Enabled` | `bool` | No | Enable tracing. |
| `Exporter` | `string` | No | `otlp`, `jaeger`, `stdout`, `none`. |
| `SamplePct` | `float64` | No | Range `0.0`–`1.0`. |

### MetricsConfig

| Field | Type | Required | Notes |
|------|------|----------|-------|
| `Enabled` | `bool` | No | Enable metrics. |
| `Exporter` | `string` | No | `otlp`, `prometheus`, `stdout`, `none`. |

### LoggingConfig

| Field | Type | Required | Notes |
|------|------|----------|-------|
| `Enabled` | `bool` | No | Enable logging. |
| `Level` | `string` | No | `debug`, `info`, `warn`, `error`. |

Redaction:
- Sensitive fields are automatically redacted using `observe.RedactedFields`.

## cache

### Policy

`cache.Policy` defines caching behavior.

| Field | Type | Required | Notes |
|------|------|----------|-------|
| `DefaultTTL` | `time.Duration` | No | `0` disables caching by default. |
| `MaxTTL` | `time.Duration` | No | Clamp TTL overrides; `0` = no max. |
| `AllowUnsafe` | `bool` | No | Allow caching tools tagged as unsafe. |

### Cache Contract

- `Get` returns `(nil, false)` on miss and must not error.
- `Set` with `ttl=0` disables caching.
- `Delete` is idempotent; no error on miss.

### Keyer Contract

- `cache.Keyer` must return deterministic, stable keys.
- Keys must pass `cache.ValidateKey` (non-empty, <=512 chars, no newlines).

## auth

Auth uses specific config types per mechanism:

| Type | Purpose |
|------|---------|
| `JWTConfig` | Validate JWT tokens and claims |
| `JWKSConfig` | JWKS URL + caching for JWT verification |
| `APIKeyConfig` | Static API key validation |
| `OAuth2Config` | Introspection settings |
| `RBACConfig` | Role-based access control |
| `RoleConfig` | Role definition + permissions |

Contracts:
- All auth checks are deterministic and side-effect free.
- Errors are explicit; deny-by-default on failure.

## health

| Type | Purpose |
|------|---------|
| `AggregatorConfig` | Aggregates multiple checks and configures thresholds |
| `MemoryCheckerConfig` | Memory health thresholds |

Contracts:
- Health checks are fast and non-blocking.
- Failures return structured errors with context.

## resilience

| Type | Purpose |
|------|---------|
| `RetryConfig` | Retry count, backoff, jitter |
| `CircuitBreakerConfig` | Failure thresholds, open/half-open timings |
| `RateLimiterConfig` | Rate, burst, time window |
| `BulkheadConfig` | Concurrency limits |
| `TimeoutConfig` | Max execution duration |

Contracts:
- All resilience middleware must be concurrency-safe.
- Timeouts and cancellations honor `context.Context`.
- Retry policies never retry on context cancellation.
