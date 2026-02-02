package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/jonwraymond/toolops/health"
)

func ExampleNewMemoryChecker() {
	checker := health.NewMemoryChecker(health.MemoryCheckerConfig{
		WarningThreshold:  0.80,
		CriticalThreshold: 0.95,
	})

	ctx := context.Background()
	result := checker.Check(ctx)

	fmt.Println("Checker name:", checker.Name())
	fmt.Println("Status is healthy:", result.Status == health.StatusHealthy)
	// Output:
	// Checker name: memory
	// Status is healthy: true
}

func ExampleNewCheckerFunc() {
	// Create a simple database ping checker
	dbChecker := health.NewCheckerFunc("database", func(ctx context.Context) health.Result {
		// Simulate a successful ping
		return health.Healthy("database connected")
	})

	ctx := context.Background()
	result := dbChecker.Check(ctx)

	fmt.Println("Checker name:", dbChecker.Name())
	fmt.Println("Status:", result.Status.String())
	fmt.Println("Message:", result.Message)
	// Output:
	// Checker name: database
	// Status: healthy
	// Message: database connected
}

func ExampleHealthy() {
	result := health.Healthy("all systems operational")

	fmt.Println("Status:", result.Status.String())
	fmt.Println("Message:", result.Message)
	// Output:
	// Status: healthy
	// Message: all systems operational
}

func ExampleDegraded() {
	result := health.Degraded("high latency detected")

	fmt.Println("Status:", result.Status.String())
	fmt.Println("Message:", result.Message)
	// Output:
	// Status: degraded
	// Message: high latency detected
}

func ExampleUnhealthy() {
	err := errors.New("connection refused")
	result := health.Unhealthy("database unreachable", err)

	fmt.Println("Status:", result.Status.String())
	fmt.Println("Message:", result.Message)
	fmt.Println("Has error:", result.Error != nil)
	// Output:
	// Status: unhealthy
	// Message: database unreachable
	// Has error: true
}

func ExampleResult_WithDetails() {
	result := health.Healthy("cache operational").WithDetails(map[string]any{
		"hit_rate":  0.95,
		"entries":   1234,
		"memory_mb": 56.7,
	})

	fmt.Println("Status:", result.Status.String())
	fmt.Println("Has details:", result.Details != nil)
	fmt.Printf("Hit rate: %.0f%%\n", result.Details["hit_rate"].(float64)*100)
	// Output:
	// Status: healthy
	// Has details: true
	// Hit rate: 95%
}

func ExampleResult_WithDuration() {
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	result := health.Healthy("check complete").WithDuration(time.Since(start))

	fmt.Println("Status:", result.Status.String())
	fmt.Println("Has duration:", result.Duration > 0)
	// Output:
	// Status: healthy
	// Has duration: true
}

func ExampleNewAggregator() {
	agg := health.NewAggregator()

	// Register checkers
	agg.Register("memory", health.NewMemoryChecker(health.MemoryCheckerConfig{}))
	agg.Register("service", health.NewCheckerFunc("service", func(ctx context.Context) health.Result {
		return health.Healthy("service running")
	}))

	fmt.Println("Registered checkers:", agg.CheckerNames())
	// Output:
	// Registered checkers: [memory service]
}

func ExampleAggregator_CheckAll() {
	agg := health.NewAggregator()

	// Register multiple checkers
	agg.Register("check1", health.NewCheckerFunc("check1", func(ctx context.Context) health.Result {
		return health.Healthy("check1 ok")
	}))
	agg.Register("check2", health.NewCheckerFunc("check2", func(ctx context.Context) health.Result {
		return health.Healthy("check2 ok")
	}))

	ctx := context.Background()
	results := agg.CheckAll(ctx)

	fmt.Println("Number of results:", len(results))
	fmt.Println("check1 status:", results["check1"].Status.String())
	fmt.Println("check2 status:", results["check2"].Status.String())
	// Output:
	// Number of results: 2
	// check1 status: healthy
	// check2 status: healthy
}

func ExampleAggregator_OverallStatus() {
	agg := health.NewAggregator()

	// All healthy
	results := map[string]health.Result{
		"a": health.Healthy("ok"),
		"b": health.Healthy("ok"),
	}
	fmt.Println("All healthy:", agg.OverallStatus(results).String())

	// One degraded
	results["c"] = health.Degraded("slow")
	fmt.Println("One degraded:", agg.OverallStatus(results).String())

	// One unhealthy
	results["d"] = health.Unhealthy("down", nil)
	fmt.Println("One unhealthy:", agg.OverallStatus(results).String())
	// Output:
	// All healthy: healthy
	// One degraded: degraded
	// One unhealthy: unhealthy
}

func ExampleAggregator_Check() {
	agg := health.NewAggregator()
	agg.Register("mycheck", health.NewCheckerFunc("mycheck", func(ctx context.Context) health.Result {
		return health.Healthy("specific check passed")
	}))

	ctx := context.Background()

	// Check specific component
	result, err := agg.Check(ctx, "mycheck")
	if err == nil {
		fmt.Println("Status:", result.Status.String())
		fmt.Println("Message:", result.Message)
	}

	// Check non-existent component
	_, err = agg.Check(ctx, "unknown")
	fmt.Println("Unknown checker error:", errors.Is(err, health.ErrCheckerNotFound))
	// Output:
	// Status: healthy
	// Message: specific check passed
	// Unknown checker error: true
}

func ExampleAggregator_Checker() {
	agg := health.NewAggregator()
	agg.Register("sub1", health.NewCheckerFunc("sub1", func(ctx context.Context) health.Result {
		return health.Healthy("sub1 ok")
	}))
	agg.Register("sub2", health.NewCheckerFunc("sub2", func(ctx context.Context) health.Result {
		return health.Healthy("sub2 ok")
	}))

	// Use aggregator as a single checker
	checker := agg.Checker()
	ctx := context.Background()
	result := checker.Check(ctx)

	fmt.Println("Checker name:", checker.Name())
	fmt.Println("Overall status:", result.Status.String())
	fmt.Println("Has sub-check details:", result.Details != nil)
	// Output:
	// Checker name: aggregate
	// Overall status: healthy
	// Has sub-check details: true
}

func ExampleNewAggregator_withConfig() {
	// Configure aggregator
	agg := health.NewAggregator(health.AggregatorConfig{
		Timeout:  5 * time.Second,
		Parallel: false, // Run checks sequentially
	})

	agg.Register("check1", health.NewCheckerFunc("check1", func(ctx context.Context) health.Result {
		return health.Healthy("sequential check")
	}))

	ctx := context.Background()
	results := agg.CheckAll(ctx)

	fmt.Println("Check completed:", len(results) == 1)
	// Output:
	// Check completed: true
}

func ExampleStatus_String() {
	statuses := []health.Status{
		health.StatusHealthy,
		health.StatusDegraded,
		health.StatusUnhealthy,
	}

	for _, s := range statuses {
		fmt.Println(s.String())
	}
	// Output:
	// healthy
	// degraded
	// unhealthy
}

func ExampleLivenessHandler() {
	handler := health.LivenessHandler()

	// Simulate HTTP request
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	fmt.Println("Status code:", rec.Code)
	fmt.Println("Body:", rec.Body.String())
	// Output:
	// Status code: 200
	// Body: OK
}

func ExampleReadinessHandler() {
	agg := health.NewAggregator()
	agg.Register("component", health.NewCheckerFunc("component", func(ctx context.Context) health.Result {
		return health.Healthy("ready")
	}))

	handler := health.ReadinessHandler(agg)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	fmt.Println("Status code:", rec.Code)
	fmt.Println("Body:", rec.Body.String())
	// Output:
	// Status code: 200
	// Body: OK
}

func ExampleDetailedHandler() {
	agg := health.NewAggregator()
	agg.Register("api", health.NewCheckerFunc("api", func(ctx context.Context) health.Result {
		return health.Healthy("api responding")
	}))

	handler := health.DetailedHandler(agg)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	fmt.Println("Status code:", rec.Code)
	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))

	// Parse response
	var response health.HealthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	fmt.Println("Overall status:", response.Status)
	fmt.Println("Has checks:", len(response.Checks) > 0)
	// Output:
	// Status code: 200
	// Content-Type: application/json
	// Overall status: healthy
	// Has checks: true
}

func ExampleRegisterHandlers() {
	agg := health.NewAggregator()
	agg.Register("test", health.NewCheckerFunc("test", func(ctx context.Context) health.Result {
		return health.Healthy("ok")
	}))

	mux := http.NewServeMux()
	health.RegisterHandlers(mux, agg)

	// Test that handlers are registered
	endpoints := []string{"/healthz", "/readyz", "/health"}
	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		fmt.Printf("%s: %d\n", ep, rec.Code)
	}
	// Output:
	// /healthz: 200
	// /readyz: 200
	// /health: 200
}
