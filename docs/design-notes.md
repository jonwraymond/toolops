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

## cache Package

### Design Decisions

1. **Deterministic keys**: Inputs are canonicalized and hashed (SHA‑256).
2. **Explicit policy**: TTL and unsafe tag handling are policy‑driven.
3. **No caching on error**: executor failures are never cached.

## auth Package

### Design Decisions

1. **Authenticator vs Authorizer**: Authentication returns identities; authorization enforces permissions.
2. **RBAC support**: Simple RBAC authorizer with role inheritance.
3. **Protocol-agnostic**: Works with any transport layer.

## health Package

### Design Decisions

1. **Checker interface**: Components implement `Check(ctx)` returning `Result`.
2. **Aggregator**: Multiple checkers can be composed into liveness/readiness endpoints.
3. **HTTP handlers**: Built-in probe handlers for orchestration platforms.

## resilience Package

### Design Decisions

1. **Composable executor**: Pattern chain order is deterministic and documented.
2. **Minimal state**: Each pattern is isolated and configurable.
3. **Context-aware**: All patterns honor cancellation and deadlines.

## Trade-offs

- **No built-in storage**: cache is in-memory by default; external backends are explicit.
- **No implicit telemetry**: callers must wire observe middleware to execution functions.
- **Fail-fast execution**: resilience executor stops on first failure after pattern handling.
