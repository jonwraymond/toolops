package health

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkChecker_Check measures single check performance.
func BenchmarkChecker_Check(b *testing.B) {
	checker := NewCheckerFunc("bench", func(ctx context.Context) Result {
		return Healthy("ok")
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.Check(ctx)
	}
}

// BenchmarkMemoryChecker_Check measures memory checker performance.
func BenchmarkMemoryChecker_Check(b *testing.B) {
	checker := NewMemoryChecker(MemoryCheckerConfig{
		WarningThreshold:  0.80,
		CriticalThreshold: 0.95,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.Check(ctx)
	}
}

// BenchmarkAggregator_CheckAll_Sequential measures sequential check aggregation.
func BenchmarkAggregator_CheckAll_Sequential(b *testing.B) {
	agg := NewAggregator(AggregatorConfig{
		Timeout:  10 * time.Second,
		Parallel: false,
	})

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("check%d", i)
		agg.Register(name, NewCheckerFunc(name, func(ctx context.Context) Result {
			return Healthy("ok")
		}))
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agg.CheckAll(ctx)
	}
}

// BenchmarkAggregator_CheckAll_Parallel measures parallel check aggregation.
func BenchmarkAggregator_CheckAll_Parallel(b *testing.B) {
	agg := NewAggregator(AggregatorConfig{
		Timeout:  10 * time.Second,
		Parallel: true,
	})

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("check%d", i)
		agg.Register(name, NewCheckerFunc(name, func(ctx context.Context) Result {
			return Healthy("ok")
		}))
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agg.CheckAll(ctx)
	}
}

// BenchmarkAggregator_OverallStatus measures status computation.
func BenchmarkAggregator_OverallStatus(b *testing.B) {
	agg := NewAggregator()
	results := map[string]Result{
		"check1": Healthy("ok"),
		"check2": Healthy("ok"),
		"check3": Degraded("slow"),
		"check4": Healthy("ok"),
		"check5": Healthy("ok"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agg.OverallStatus(results)
	}
}

// BenchmarkAggregator_Register measures registration overhead.
func BenchmarkAggregator_Register(b *testing.B) {
	checker := NewCheckerFunc("bench", func(ctx context.Context) Result {
		return Healthy("ok")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agg := NewAggregator()
		agg.Register("check", checker)
	}
}

// BenchmarkAggregator_CheckerNames measures name retrieval.
func BenchmarkAggregator_CheckerNames(b *testing.B) {
	agg := NewAggregator()
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("check%d", i)
		agg.Register(name, NewCheckerFunc(name, func(ctx context.Context) Result {
			return Healthy("ok")
		}))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agg.CheckerNames()
	}
}

// BenchmarkAggregator_VaryingCheckers measures scaling with checker count.
func BenchmarkAggregator_VaryingCheckers(b *testing.B) {
	sizes := []int{1, 5, 10, 20}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("checkers=%d", size), func(b *testing.B) {
			agg := NewAggregator(AggregatorConfig{
				Timeout:  10 * time.Second,
				Parallel: true,
			})

			for i := 0; i < size; i++ {
				name := fmt.Sprintf("check%d", i)
				agg.Register(name, NewCheckerFunc(name, func(ctx context.Context) Result {
					return Healthy("ok")
				}))
			}
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = agg.CheckAll(ctx)
			}
		})
	}
}

// BenchmarkLivenessHandler_ServeHTTP measures liveness handler overhead.
func BenchmarkLivenessHandler_ServeHTTP(b *testing.B) {
	handler := LivenessHandler()
	req := httptest.NewRequest("GET", "/healthz", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkReadinessHandler_ServeHTTP measures readiness handler overhead.
func BenchmarkReadinessHandler_ServeHTTP(b *testing.B) {
	agg := NewAggregator()
	agg.Register("check", NewCheckerFunc("check", func(ctx context.Context) Result {
		return Healthy("ok")
	}))

	handler := ReadinessHandler(agg)
	req := httptest.NewRequest("GET", "/readyz", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkDetailedHandler_ServeHTTP measures detailed handler overhead.
func BenchmarkDetailedHandler_ServeHTTP(b *testing.B) {
	agg := NewAggregator()
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("check%d", i)
		agg.Register(name, NewCheckerFunc(name, func(ctx context.Context) Result {
			return Healthy("ok")
		}))
	}

	handler := DetailedHandler(agg)
	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkHealthy measures result creation.
func BenchmarkHealthy(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Healthy("message")
	}
}

// BenchmarkResult_WithDetails measures detail attachment.
func BenchmarkResult_WithDetails(b *testing.B) {
	result := Healthy("ok")
	details := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.WithDetails(details)
	}
}

// BenchmarkStatus_String measures status string conversion.
func BenchmarkStatus_String(b *testing.B) {
	statuses := []Status{StatusHealthy, StatusDegraded, StatusUnhealthy}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = statuses[i%3].String()
	}
}

// BenchmarkConcurrent_Aggregator measures concurrent aggregator usage.
func BenchmarkConcurrent_Aggregator(b *testing.B) {
	agg := NewAggregator()
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("check%d", i)
		agg.Register(name, NewCheckerFunc(name, func(ctx context.Context) Result {
			return Healthy("ok")
		}))
	}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = agg.CheckAll(ctx)
		}
	})
}
