package health

import (
	"context"
	"sync"
	"time"
)

// AggregatorConfig configures the health aggregator.
type AggregatorConfig struct {
	// Timeout is the maximum time to wait for all checks.
	// Default: 10 seconds
	Timeout time.Duration

	// Parallel runs health checks in parallel when true.
	// Default: true
	Parallel bool
}

// Aggregator combines multiple health checkers into a single composite check.
type Aggregator struct {
	config   AggregatorConfig
	mu       sync.RWMutex
	checkers map[string]Checker
	order    []string // Maintains registration order
}

// NewAggregator creates a new health aggregator.
func NewAggregator(config ...AggregatorConfig) *Aggregator {
	cfg := AggregatorConfig{
		Timeout:  10 * time.Second,
		Parallel: true,
	}
	if len(config) > 0 {
		cfg = config[0]
		if cfg.Timeout <= 0 {
			cfg.Timeout = 10 * time.Second
		}
	}

	return &Aggregator{
		config:   cfg,
		checkers: make(map[string]Checker),
		order:    make([]string, 0),
	}
}

// Register adds a health checker to the aggregator.
func (a *Aggregator) Register(name string, checker Checker) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.checkers[name]; !exists {
		a.order = append(a.order, name)
	}
	a.checkers[name] = checker
}

// Unregister removes a health checker from the aggregator.
func (a *Aggregator) Unregister(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.checkers, name)

	// Remove from order
	for i, n := range a.order {
		if n == name {
			a.order = append(a.order[:i], a.order[i+1:]...)
			break
		}
	}
}

// CheckerNames returns the names of all registered checkers.
func (a *Aggregator) CheckerNames() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	names := make([]string, len(a.order))
	copy(names, a.order)
	return names
}

// Check runs a single named health check.
func (a *Aggregator) Check(ctx context.Context, name string) (Result, error) {
	a.mu.RLock()
	checker, ok := a.checkers[name]
	a.mu.RUnlock()

	if !ok {
		return Result{}, ErrCheckerNotFound
	}

	return a.runCheck(ctx, checker), nil
}

// CheckAll runs all registered health checks and returns the results.
func (a *Aggregator) CheckAll(ctx context.Context) map[string]Result {
	a.mu.RLock()
	checkers := make(map[string]Checker, len(a.checkers))
	for name, checker := range a.checkers {
		checkers[name] = checker
	}
	a.mu.RUnlock()

	if len(checkers) == 0 {
		return make(map[string]Result)
	}

	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	results := make(map[string]Result, len(checkers))

	if a.config.Parallel {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for name, checker := range checkers {
			wg.Add(1)
			go func(name string, checker Checker) {
				defer wg.Done()
				result := a.runCheck(ctx, checker)
				mu.Lock()
				results[name] = result
				mu.Unlock()
			}(name, checker)
		}

		wg.Wait()
	} else {
		for name, checker := range checkers {
			results[name] = a.runCheck(ctx, checker)
		}
	}

	return results
}

// OverallStatus computes the overall health status from a set of results.
// Returns Unhealthy if any check is unhealthy.
// Returns Degraded if any check is degraded but none are unhealthy.
// Returns Healthy if all checks are healthy.
func (a *Aggregator) OverallStatus(results map[string]Result) Status {
	if len(results) == 0 {
		return StatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

func (a *Aggregator) runCheck(ctx context.Context, checker Checker) Result {
	start := time.Now()

	// Use a channel to handle timeout
	resultCh := make(chan Result, 1)

	go func() {
		result := checker.Check(ctx)
		result.Duration = time.Since(start)
		if result.Timestamp.IsZero() {
			result.Timestamp = start
		}
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result
	case <-ctx.Done():
		return Result{
			Status:    StatusUnhealthy,
			Message:   "check timed out",
			Error:     ErrCheckTimeout,
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}
}

// Checker returns a single Checker interface for the aggregator.
// This allows the aggregator to be used as a checker itself.
func (a *Aggregator) Checker() Checker {
	return &aggregatorChecker{agg: a}
}

type aggregatorChecker struct {
	agg *Aggregator
}

func (c *aggregatorChecker) Name() string {
	return "aggregate"
}

func (c *aggregatorChecker) Check(ctx context.Context) Result {
	results := c.agg.CheckAll(ctx)
	status := c.agg.OverallStatus(results)

	details := make(map[string]any, len(results))
	for name, result := range results {
		details[name] = map[string]any{
			"status":   result.Status.String(),
			"message":  result.Message,
			"duration": result.Duration.String(),
		}
	}

	var message string
	switch status {
	case StatusHealthy:
		message = "all checks passed"
	case StatusDegraded:
		message = "some checks degraded"
	case StatusUnhealthy:
		message = "some checks failed"
	}

	return Result{
		Status:    status,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}
}
