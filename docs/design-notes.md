# toolops Design Notes

## Overview

toolops provides cross-cutting operational concerns for tool execution:

- **observe**: tracing, metrics, structured logging
- **cache**: deterministic caching with TTL policy
- **auth**: authentication + authorization primitives
- **health**: health checks and probe handlers
- **resilience**: retries, timeouts, circuit breakers, rate limits, bulkheads

## observe Package

### Design Decisions

1. **Observer abstraction**: Wraps OpenTelemetry tracer/meter/logger into one object.
2. **Middleware wrapping**: `Middleware` decorates an `ExecuteFunc` for execution telemetry.
3. **No execution dependency**: observe only instruments; it does not call tools.

### Contracts

- **Observer** must be shut down to flush exporters.
- **Middleware** wraps an `ExecuteFunc` and records telemetry per call.
- **ToolMeta** drives span naming (`tool.exec.<namespace>.<name>`).

## cache Package

### Design Decisions

1. **Deterministic keys**: Inputs are canonicalized and hashed (SHA‑256).
2. **Explicit policy**: TTL and unsafe tag handling are policy‑driven.
3. **No caching on error**: executor failures are never cached.

### Policy Semantics

- **DefaultTTL** controls caching enablement (0 disables).
- **MaxTTL** clamps overrides.
- **AllowUnsafe** gates caching for unsafe-tagged tools.

## auth Package

### Design Decisions

1. **Authenticator vs Authorizer**: Authentication returns identities; authorization enforces permissions.
2. **RBAC support**: Simple RBAC authorizer with role inheritance.
3. **Protocol-agnostic**: Works with any transport layer.

### Contracts

- **Authenticator** returns `AuthResult` for success/failure; errors indicate internal failure.
- **Authorizer** returns an `AuthzError` when denied.

## health Package

### Design Decisions

1. **Checker interface**: Components implement `Check(ctx)` returning `Result`.
2. **Aggregator**: Multiple checkers can be composed into liveness/readiness endpoints.
3. **HTTP handlers**: Built-in probe handlers for orchestration platforms.

### Contracts

- **Checker** returns a `Result` with status + details.
- **Aggregator** combines results and computes overall status.

## resilience Package

### Design Decisions

1. **Composable executor**: Pattern chain order is deterministic and documented.
2. **Minimal state**: Each pattern is isolated and configurable.
3. **Context-aware**: All patterns honor cancellation and deadlines.

### Execution Order

The executor composes patterns in this order:
1. Rate limiter
2. Bulkhead
3. Circuit breaker
4. Retry
5. Timeout

## Trade-offs

- **No built-in storage**: cache is in-memory by default; external backends are explicit.
- **No implicit telemetry**: callers must wire observe middleware to execution functions.
- **Fail-fast execution**: resilience executor stops on first failure after pattern handling.
