# Examples

This page collects runnable examples for each toolops capability.

## Observability

```go
import (
  "context"
  "log"

  "github.com/jonwraymond/toolops/observe"
)

obs, err := observe.NewObserver(context.Background(), observe.Config{
  ServiceName: "metatools-mcp",
  Tracing:     observe.TracingConfig{Enabled: true, Exporter: "otlp", SamplePct: 1.0},
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

## Cache

```go
import (
  "context"

  "github.com/jonwraymond/toolops/cache"
)

c := cache.NewMemoryCache(cache.DefaultPolicy())
keyer := cache.NewDefaultKeyer()
policy := cache.DefaultPolicy()

mw := cache.NewCacheMiddleware(c, keyer, policy, nil)
result, err := mw.Execute(context.Background(), "github:create_issue", map[string]any{"title": "Bug"}, []string{"issues"},
  func(ctx context.Context, toolID string, input any) ([]byte, error) {
    return []byte("{\"ok\":true}"), nil
  })
_ = result
_ = err
```

## Auth (JWT)

```go
import "github.com/jonwraymond/toolops/auth"

validator := auth.NewJWTValidator(auth.JWTConfig{
  Issuer:   "https://issuer.example.com",
  Audience: "mcp",
})

ok, claims, err := validator.ValidateToken("<token>")
_ = ok
_ = claims
_ = err
```

## Health

```go
import "github.com/jonwraymond/toolops/health"

agg := health.NewAggregator(health.AggregatorConfig{ServiceName: "metatools-mcp"})
agg.Register("memory", health.NewMemoryChecker(health.MemoryCheckerConfig{MaxRSSBytes: 512 * 1024 * 1024}))

status := agg.Check(context.Background())
_ = status
```

## Resilience

```go
import "github.com/jonwraymond/toolops/resilience"

retry := resilience.NewRetry(resilience.RetryConfig{
  MaxAttempts: 3,
})

cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
  FailureThreshold: 5,
})

_ = retry
_ = cb
```
