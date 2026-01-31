package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	if cb.State() != StateClosed {
		t.Errorf("Initial state = %v, want closed", cb.State())
	}
}

func TestNewCircuitBreaker_Defaults(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	if cb.config.MaxFailures != 5 {
		t.Errorf("MaxFailures = %d, want 5", cb.config.MaxFailures)
	}
	if cb.config.ResetTimeout != 30*time.Second {
		t.Errorf("ResetTimeout = %v, want 30s", cb.config.ResetTimeout)
	}
	if cb.config.HalfOpenMaxRequests != 1 {
		t.Errorf("HalfOpenMaxRequests = %d, want 1", cb.config.HalfOpenMaxRequests)
	}
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  3,
		ResetTimeout: time.Second,
	})

	testErr := errors.New("test error")

	// First 2 failures should not open
	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
		if err != testErr {
			t.Errorf("Execute() error = %v, want %v", err, testErr)
		}
		if cb.State() != StateClosed {
			t.Errorf("After %d failures, state = %v, want closed", i+1, cb.State())
		}
	}

	// Third failure should open
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}
	if cb.State() != StateOpen {
		t.Errorf("After 3 failures, state = %v, want open", cb.State())
	}

	// Next request should be rejected
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		t.Error("Should not be called when circuit is open")
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("Execute() when open = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: 10 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Open the circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	if cb.State() != StateOpen {
		t.Fatalf("State = %v, want open", cb.State())
	}

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// Should be half-open now
	if cb.State() != StateHalfOpen {
		t.Errorf("State = %v, want half-open", cb.State())
	}
}

func TestCircuitBreaker_RecoverySuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: 10 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Open the circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// Successful request should close circuit
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("State = %v, want closed", cb.State())
	}
}

func TestCircuitBreaker_RecoveryFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: 10 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// Open the circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// Failed request should re-open circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	if cb.State() != StateOpen {
		t.Errorf("State = %v, want open", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: time.Hour,
	})

	testErr := errors.New("test error")

	// Open the circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	if cb.State() != StateOpen {
		t.Fatalf("State = %v, want open", cb.State())
	}

	// Manual reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("After reset, state = %v, want closed", cb.State())
	}
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	var transitions []struct {
		from, to State
	}
	var mu sync.Mutex

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: 10 * time.Millisecond,
		OnStateChange: func(from, to State) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to State }{from, to})
			mu.Unlock()
		},
	})

	testErr := errors.New("test error")

	// Open the circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)
	_ = cb.State() // Trigger state check

	// Close the circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	mu.Lock()
	defer mu.Unlock()

	if len(transitions) < 2 {
		t.Errorf("Expected at least 2 transitions, got %d", len(transitions))
	}

	// Check first transition: closed -> open
	if transitions[0].from != StateClosed || transitions[0].to != StateOpen {
		t.Errorf("First transition: %v -> %v, want closed -> open", transitions[0].from, transitions[0].to)
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  3,
		ResetTimeout: time.Hour,
	})

	testErr := errors.New("test error")

	// Two failures
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	// One success should reset failure count
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	// Two more failures should not open (count was reset)
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	if cb.State() != StateClosed {
		t.Errorf("State = %v, want closed", cb.State())
	}
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 5,
	})

	testErr := errors.New("test error")

	// Generate some failures
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	metrics := cb.Metrics()

	if metrics.State != StateClosed {
		t.Errorf("Metrics.State = %v, want closed", metrics.State)
	}
	if metrics.Failures != 2 {
		t.Errorf("Metrics.Failures = %d, want 2", metrics.Failures)
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
