package resilience

import (
	"context"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed means the circuit is operating normally.
	StateClosed State = iota
	// StateOpen means the circuit is blocking all requests.
	StateOpen
	// StateHalfOpen means the circuit is testing if the service recovered.
	StateHalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures the circuit breaker.
type CircuitBreakerConfig struct {
	// MaxFailures is the number of failures before opening the circuit.
	// Default: 5
	MaxFailures int

	// ResetTimeout is how long to wait before attempting recovery.
	// Default: 30 seconds
	ResetTimeout time.Duration

	// HalfOpenMaxRequests is the max requests allowed in half-open state.
	// Default: 1
	HalfOpenMaxRequests int

	// OnStateChange is called when the circuit state changes.
	OnStateChange func(from, to State)

	// IsFailure determines if an error should count as a failure.
	// Default: all non-nil errors are failures.
	IsFailure func(err error) bool
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu            sync.Mutex
	state         State
	failures      int
	successes     int
	lastFailure   time.Time
	halfOpenCount int
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	// Apply defaults
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = 1
	}
	if config.IsFailure == nil {
		config.IsFailure = func(err error) bool { return err != nil }
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute runs the operation through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, op func(context.Context) error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	err := op(ctx)
	cb.afterRequest(err)
	return err
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentStateLocked()
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCount = 0

	if oldState != StateClosed && cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, StateClosed)
	}
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentStateLocked()

	switch state {
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.config.HalfOpenMaxRequests {
			return ErrCircuitOpen
		}
		cb.halfOpenCount++
	}

	return nil
}

func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	isFailure := cb.config.IsFailure(err)
	oldState := cb.state

	switch cb.state {
	case StateClosed:
		if isFailure {
			cb.failures++
			cb.lastFailure = time.Now()
			if cb.failures >= cb.config.MaxFailures {
				cb.setState(StateOpen)
			}
		} else {
			// Reset failure count on success
			cb.failures = 0
		}

	case StateHalfOpen:
		if isFailure {
			// Failed during probe, go back to open
			cb.lastFailure = time.Now() // Reset timeout for open state
			cb.setState(StateOpen)
		} else {
			cb.successes++
			// Successful probe, close the circuit
			cb.setState(StateClosed)
			cb.failures = 0
			cb.successes = 0
		}
	}

	if oldState != cb.state && cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, cb.state)
	}
}

func (cb *CircuitBreaker) currentStateLocked() State {
	if cb.state == StateOpen && time.Since(cb.lastFailure) >= cb.config.ResetTimeout {
		cb.state = StateHalfOpen
		cb.halfOpenCount = 0
		if cb.config.OnStateChange != nil {
			cb.config.OnStateChange(StateOpen, StateHalfOpen)
		}
	}
	return cb.state
}

func (cb *CircuitBreaker) setState(state State) {
	cb.state = state
	if state == StateHalfOpen {
		cb.halfOpenCount = 0
	}
}

// Metrics returns current circuit breaker metrics.
func (cb *CircuitBreaker) Metrics() CircuitBreakerMetrics {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return CircuitBreakerMetrics{
		State:       cb.currentStateLocked(),
		Failures:    cb.failures,
		Successes:   cb.successes,
		LastFailure: cb.lastFailure,
	}
}

// CircuitBreakerMetrics contains circuit breaker statistics.
type CircuitBreakerMetrics struct {
	State       State
	Failures    int
	Successes   int
	LastFailure time.Time
}
