# toolops

Operations layer providing observability, caching, authentication, health checks,
and resilience patterns for production deployments.

## Packages

| Package | Purpose |
|---------|---------|
| `observe` | OpenTelemetry-based tracing, metrics, and logging |
| `cache` | Deterministic caching with policies and middleware |
| `auth` | Authentication and authorization primitives |
| `health` | Health checks and HTTP probes |
| `resilience` | Circuit breakers, retries, rate limits, bulkheads |

## Installation

```bash
go get github.com/jonwraymond/toolops@latest
```

## Documentation Map

- [Architecture](architecture.md)
- [Schemas and Contracts](schemas.md)
- [Examples](examples.md)
- [Design Notes](design-notes.md)

## Quick Start: Observability

```go
import (
  "context"
  "log"

  "github.com/jonwraymond/toolops/observe"
)

obs, err := observe.NewObserver(context.Background(), observe.Config{
  ServiceName: "metatools-mcp",
  Tracing:     observe.TracingConfig{Enabled: true, Exporter: "otlp"},
  Metrics:     observe.MetricsConfig{Enabled: true, Exporter: "prometheus"},
  Logging:     observe.LoggingConfig{Enabled: true, Level: "info"},
})
if err != nil {
  log.Fatal(err)
}
defer obs.Shutdown(context.Background())

mw, _ := observe.MiddlewareFromObserver(obs)
wrapped := mw.Wrap(func(ctx context.Context, tool observe.ToolMeta, input any) (any, error) {
  return map[string]any{"ok": true}, nil
})

_, _ = wrapped(context.Background(), observe.ToolMeta{Name: "echo"}, map[string]any{"msg": "hi"})
```

## Quick Start: Cache

```go
import (
  "context"

  "github.com/jonwraymond/toolops/cache"
)

c := cache.NewMemoryCache(cache.DefaultPolicy())
keyer := cache.NewDefaultKeyer()
mw := cache.NewCacheMiddleware(c, keyer, cache.DefaultPolicy(), nil)

result, err := mw.Execute(context.Background(), "github:create_issue", map[string]any{"title": "Bug"}, []string{"issues"},
  func(ctx context.Context, toolID string, input any) ([]byte, error) {
    return []byte("{\"ok\":true}"), nil
  })
_ = result
_ = err
```
