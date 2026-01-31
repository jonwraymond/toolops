# toolops User Journey

## 1. Installation

```bash
go get github.com/jonwraymond/toolops@latest
```

## 2. Observability Middleware

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

## 3. Caching with Policies

```go
import (
  "context"

  "github.com/jonwraymond/toolops/cache"
)

policy := cache.DefaultPolicy()
c := cache.NewMemoryCache(policy)
keyer := cache.NewDefaultKeyer()
mw := cache.NewCacheMiddleware(c, keyer, policy, nil)

_, _ = mw.Execute(context.Background(), "github:list_issues", map[string]any{"repo": "toolops"}, []string{"issues"},
  func(ctx context.Context, toolID string, input any) ([]byte, error) {
    return []byte("{\"ok\":true}"), nil
  })
```

## 4. Authentication + Authorization

```go
import (
  "context"

  "github.com/jonwraymond/toolops/auth"
)

authenticator := auth.NewAPIKeyAuthenticator(auth.APIKeyConfig{
  Header: "X-API-Key",
  Keys:   map[string]string{"dev-key": "developer"},
})

authorizer := auth.NewSimpleRBACAuthorizer(auth.RBACConfig{
  DefaultRole: "reader",
  Roles: map[string]auth.RoleConfig{
    "reader": {AllowedTools: []string{"github:*"}, AllowedActions: []string{"list"}},
  },
})

req := &auth.AuthRequest{Headers: map[string][]string{"X-API-Key": {"dev-key"}}}
result, _ := authenticator.Authenticate(context.Background(), req)
if result != nil && result.Identity != nil {
  _ = authorizer.Authorize(context.Background(), &auth.AuthzRequest{
    Subject: result.Identity,
    Resource: "tool:github:list_issues",
    Action: "list",
  })
}
```

## 5. Health Checks

```go
import (
  "context"

  "github.com/jonwraymond/toolops/health"
)

agg := health.NewAggregator()
agg.Register("memory", health.NewMemoryChecker(health.MemoryCheckerConfig{
  WarningThreshold: 0.80,
  CriticalThreshold: 0.95,
}))

results := agg.CheckAll(context.Background())
overall := agg.OverallStatus(results)
_ = overall
```

## 6. Resilience Patterns

```go
import (
  "context"
  "time"

  "github.com/jonwraymond/toolops/resilience"
)

executor := resilience.NewExecutor(
  resilience.WithRetry(resilience.NewRetry(resilience.RetryConfig{
    MaxAttempts: 3,
  })),
  resilience.WithTimeout(2*time.Second),
)

_ = executor.Execute(context.Background(), func(ctx context.Context) error {
  return nil
})
```

## Next Steps

- Combine `observe` + `cache` + `resilience` in a middleware chain.
- Use `auth` + `health` to harden MCP endpoints.
