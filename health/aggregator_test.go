package health

import (
	"context"
	"testing"
	"time"
)

func TestNewAggregator(t *testing.T) {
	agg := NewAggregator()

	if agg.config.Timeout != 10*time.Second {
		t.Errorf("Default timeout = %v, want 10s", agg.config.Timeout)
	}
	if !agg.config.Parallel {
		t.Error("Default Parallel should be true")
	}
}

func TestNewAggregator_WithConfig(t *testing.T) {
	agg := NewAggregator(AggregatorConfig{
		Timeout:  5 * time.Second,
		Parallel: false,
	})

	if agg.config.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", agg.config.Timeout)
	}
	if agg.config.Parallel {
		t.Error("Parallel should be false")
	}
}

func TestAggregator_Register(t *testing.T) {
	agg := NewAggregator()

	checker := NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("ok")
	})

	agg.Register("test", checker)

	names := agg.CheckerNames()
	if len(names) != 1 {
		t.Fatalf("Expected 1 checker, got %d", len(names))
	}
	if names[0] != "test" {
		t.Errorf("Checker name = %v, want 'test'", names[0])
	}
}

func TestAggregator_Unregister(t *testing.T) {
	agg := NewAggregator()

	checker := NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("ok")
	})

	agg.Register("test", checker)
	agg.Unregister("test")

	names := agg.CheckerNames()
	if len(names) != 0 {
		t.Errorf("Expected 0 checkers, got %d", len(names))
	}
}

func TestAggregator_Check(t *testing.T) {
	agg := NewAggregator()

	checker := NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("ok")
	})

	agg.Register("test", checker)

	result, err := agg.Check(context.Background(), "test")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Status != StatusHealthy {
		t.Errorf("Result.Status = %v, want StatusHealthy", result.Status)
	}
}

func TestAggregator_CheckNotFound(t *testing.T) {
	agg := NewAggregator()

	_, err := agg.Check(context.Background(), "nonexistent")
	if err != ErrCheckerNotFound {
		t.Errorf("Check() error = %v, want ErrCheckerNotFound", err)
	}
}

func TestAggregator_CheckAll(t *testing.T) {
	agg := NewAggregator()

	agg.Register("healthy", NewCheckerFunc("healthy", func(ctx context.Context) Result {
		return Healthy("ok")
	}))
	agg.Register("degraded", NewCheckerFunc("degraded", func(ctx context.Context) Result {
		return Degraded("slow")
	}))

	results := agg.CheckAll(context.Background())

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results["healthy"].Status != StatusHealthy {
		t.Errorf("healthy status = %v, want StatusHealthy", results["healthy"].Status)
	}
	if results["degraded"].Status != StatusDegraded {
		t.Errorf("degraded status = %v, want StatusDegraded", results["degraded"].Status)
	}
}

func TestAggregator_CheckAllEmpty(t *testing.T) {
	agg := NewAggregator()

	results := agg.CheckAll(context.Background())

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestAggregator_CheckAllSequential(t *testing.T) {
	agg := NewAggregator(AggregatorConfig{
		Parallel: false,
	})

	agg.Register("first", NewCheckerFunc("first", func(ctx context.Context) Result {
		return Healthy("ok")
	}))
	agg.Register("second", NewCheckerFunc("second", func(ctx context.Context) Result {
		return Healthy("ok")
	}))

	results := agg.CheckAll(context.Background())

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}

func TestAggregator_CheckAllTimeout(t *testing.T) {
	agg := NewAggregator(AggregatorConfig{
		Timeout: 50 * time.Millisecond,
	})

	agg.Register("slow", NewCheckerFunc("slow", func(ctx context.Context) Result {
		time.Sleep(200 * time.Millisecond)
		return Healthy("ok")
	}))

	results := agg.CheckAll(context.Background())

	if results["slow"].Status != StatusUnhealthy {
		t.Errorf("slow status = %v, want StatusUnhealthy", results["slow"].Status)
	}
	if results["slow"].Error != ErrCheckTimeout {
		t.Errorf("slow error = %v, want ErrCheckTimeout", results["slow"].Error)
	}
}

func TestAggregator_OverallStatus(t *testing.T) {
	agg := NewAggregator()

	tests := []struct {
		name    string
		results map[string]Result
		want    Status
	}{
		{
			name:    "empty",
			results: map[string]Result{},
			want:    StatusHealthy,
		},
		{
			name: "all healthy",
			results: map[string]Result{
				"a": Healthy("ok"),
				"b": Healthy("ok"),
			},
			want: StatusHealthy,
		},
		{
			name: "one degraded",
			results: map[string]Result{
				"a": Healthy("ok"),
				"b": Degraded("slow"),
			},
			want: StatusDegraded,
		},
		{
			name: "one unhealthy",
			results: map[string]Result{
				"a": Healthy("ok"),
				"b": Unhealthy("down", nil),
			},
			want: StatusUnhealthy,
		},
		{
			name: "unhealthy overrides degraded",
			results: map[string]Result{
				"a": Degraded("slow"),
				"b": Unhealthy("down", nil),
			},
			want: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agg.OverallStatus(tt.results)
			if got != tt.want {
				t.Errorf("OverallStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAggregator_Checker(t *testing.T) {
	agg := NewAggregator()

	agg.Register("healthy", NewCheckerFunc("healthy", func(ctx context.Context) Result {
		return Healthy("ok")
	}))

	checker := agg.Checker()

	if checker.Name() != "aggregate" {
		t.Errorf("Name() = %v, want 'aggregate'", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want StatusHealthy", result.Status)
	}
	if result.Details == nil {
		t.Error("Details should not be nil")
	}
}

func TestAggregator_CheckerWithUnhealthy(t *testing.T) {
	agg := NewAggregator()

	agg.Register("unhealthy", NewCheckerFunc("unhealthy", func(ctx context.Context) Result {
		return Unhealthy("down", nil)
	}))

	checker := agg.Checker()
	result := checker.Check(context.Background())

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want StatusUnhealthy", result.Status)
	}
	if result.Message != "some checks failed" {
		t.Errorf("Message = %v, want 'some checks failed'", result.Message)
	}
}

func TestAggregator_RegisterDuplicate(t *testing.T) {
	agg := NewAggregator()

	checker1 := NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("first")
	})
	checker2 := NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("second")
	})

	agg.Register("test", checker1)
	agg.Register("test", checker2) // Should replace

	names := agg.CheckerNames()
	if len(names) != 1 {
		t.Errorf("Expected 1 checker after duplicate, got %d", len(names))
	}

	result, _ := agg.Check(context.Background(), "test")
	if result.Message != "second" {
		t.Errorf("Message = %v, want 'second' (replacement)", result.Message)
	}
}
