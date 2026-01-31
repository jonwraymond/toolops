package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusHealthy, "healthy"},
		{StatusDegraded, "degraded"},
		{StatusUnhealthy, "unhealthy"},
		{Status(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthy(t *testing.T) {
	result := Healthy("test message")

	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want StatusHealthy", result.Status)
	}
	if result.Message != "test message" {
		t.Errorf("Message = %v, want 'test message'", result.Message)
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestDegraded(t *testing.T) {
	result := Degraded("degraded message")

	if result.Status != StatusDegraded {
		t.Errorf("Status = %v, want StatusDegraded", result.Status)
	}
	if result.Message != "degraded message" {
		t.Errorf("Message = %v, want 'degraded message'", result.Message)
	}
}

func TestUnhealthy(t *testing.T) {
	testErr := errors.New("test error")
	result := Unhealthy("unhealthy message", testErr)

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want StatusUnhealthy", result.Status)
	}
	if result.Message != "unhealthy message" {
		t.Errorf("Message = %v, want 'unhealthy message'", result.Message)
	}
	if result.Error != testErr {
		t.Errorf("Error = %v, want %v", result.Error, testErr)
	}
}

func TestResult_WithDetails(t *testing.T) {
	details := map[string]any{"key": "value"}
	result := Healthy("test").WithDetails(details)

	if result.Details["key"] != "value" {
		t.Errorf("Details[key] = %v, want 'value'", result.Details["key"])
	}
}

func TestResult_WithDuration(t *testing.T) {
	duration := 100 * time.Millisecond
	result := Healthy("test").WithDuration(duration)

	if result.Duration != duration {
		t.Errorf("Duration = %v, want %v", result.Duration, duration)
	}
}

func TestCheckerFunc(t *testing.T) {
	checker := NewCheckerFunc("test-checker", func(ctx context.Context) Result {
		return Healthy("from func")
	})

	if checker.Name() != "test-checker" {
		t.Errorf("Name() = %v, want 'test-checker'", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("Check() Status = %v, want StatusHealthy", result.Status)
	}
	if result.Message != "from func" {
		t.Errorf("Check() Message = %v, want 'from func'", result.Message)
	}
}

func TestCheckerFunc_WithContext(t *testing.T) {
	checker := NewCheckerFunc("ctx-checker", func(ctx context.Context) Result {
		select {
		case <-ctx.Done():
			return Unhealthy("cancelled", ctx.Err())
		default:
			return Healthy("ok")
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := checker.Check(ctx)
	if result.Status != StatusUnhealthy {
		t.Errorf("Check() Status = %v, want StatusUnhealthy", result.Status)
	}
}
