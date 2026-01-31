// Package health provides health checking primitives for tool operations.
//
// This package implements a generic health checking framework that can be used
// to monitor the health of various components in a tool execution system. It
// provides interfaces for defining health checks, aggregating results from
// multiple checkers, and exposing health status via HTTP endpoints.
//
// # Core Concepts
//
// A Checker is any component that can report its health status. The Status
// type represents the health state: Healthy, Degraded, or Unhealthy.
//
// # Basic Usage
//
//	// Create a memory checker
//	memCheck := health.NewMemoryChecker(health.MemoryCheckerConfig{
//	    WarningThreshold:  0.80,
//	    CriticalThreshold: 0.95,
//	})
//
//	// Check health
//	result := memCheck.Check(ctx)
//	if result.Status == health.StatusUnhealthy {
//	    log.Printf("Memory critical: %s", result.Message)
//	}
//
// # Aggregating Health Checks
//
// Use Aggregator to combine multiple health checks into a single composite check:
//
//	agg := health.NewAggregator()
//	agg.Register("memory", memChecker)
//	agg.Register("database", dbChecker)
//	agg.Register("cache", cacheChecker)
//
//	// Check all components
//	results := agg.CheckAll(ctx)
//	overall := agg.OverallStatus(results)
//
// # HTTP Endpoints
//
// The package provides HTTP handlers for common health check patterns:
//
//	// Liveness probe (for Kubernetes)
//	http.Handle("/healthz", health.LivenessHandler())
//
//	// Readiness probe with component checks
//	http.Handle("/readyz", health.ReadinessHandler(aggregator))
//
//	// Detailed health status
//	http.Handle("/health", health.DetailedHandler(aggregator))
package health
