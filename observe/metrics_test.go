package observe

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestMetrics_TotalCounterIncrements verifies tool.exec.total is incremented.
func TestMetrics_TotalCounterIncrements(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	m, err := newMetrics(meter)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{
		Namespace: "test",
		Name:      "my_tool",
	}

	m.RecordExecution(context.Background(), meta, 100*time.Millisecond, nil)

	// Collect and verify metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	found := findMetric(rm, "tool.exec.total")
	if found == nil {
		t.Fatal("tool.exec.total metric not found")
	}

	sum, ok := found.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", found.Data)
	}
	if len(sum.DataPoints) == 0 {
		t.Fatal("no data points")
	}
	if sum.DataPoints[0].Value != 1 {
		t.Errorf("expected count 1, got %d", sum.DataPoints[0].Value)
	}
}

// TestMetrics_ErrorCounterOnSuccess verifies errors counter NOT incremented on success.
func TestMetrics_ErrorCounterOnSuccess(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	m, err := newMetrics(meter)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{Name: "success_tool"}
	m.RecordExecution(context.Background(), meta, 50*time.Millisecond, nil)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	found := findMetric(rm, "tool.exec.errors")
	if found == nil {
		// If metric doesn't exist at all (no errors recorded), that's acceptable
		return
	}

	sum, ok := found.Data.(metricdata.Sum[int64])
	if !ok {
		return // Different type, skip
	}
	if len(sum.DataPoints) > 0 && sum.DataPoints[0].Value != 0 {
		t.Errorf("expected errors count 0, got %d", sum.DataPoints[0].Value)
	}
}

// TestMetrics_ErrorCounterOnFailure verifies errors counter incremented on failure.
func TestMetrics_ErrorCounterOnFailure(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	m, err := newMetrics(meter)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{Name: "failing_tool"}
	testErr := errors.New("execution failed")
	m.RecordExecution(context.Background(), meta, 50*time.Millisecond, testErr)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	found := findMetric(rm, "tool.exec.errors")
	if found == nil {
		t.Fatal("tool.exec.errors metric not found")
	}

	sum, ok := found.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", found.Data)
	}
	if len(sum.DataPoints) == 0 {
		t.Fatal("no data points")
	}
	if sum.DataPoints[0].Value != 1 {
		t.Errorf("expected errors count 1, got %d", sum.DataPoints[0].Value)
	}
}

// TestMetrics_DurationHistogramRecords verifies duration is recorded.
func TestMetrics_DurationHistogramRecords(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	m, err := newMetrics(meter)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{Name: "timed_tool"}
	duration := 50 * time.Millisecond
	m.RecordExecution(context.Background(), meta, duration, nil)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	found := findMetric(rm, "tool.exec.duration_ms")
	if found == nil {
		t.Fatal("tool.exec.duration_ms metric not found")
	}

	hist, ok := found.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram[float64], got %T", found.Data)
	}
	if len(hist.DataPoints) == 0 {
		t.Fatal("no data points")
	}

	// Verify sum is approximately 50ms
	dp := hist.DataPoints[0]
	if dp.Sum < 40 || dp.Sum > 60 {
		t.Errorf("expected duration ~50ms, got %f", dp.Sum)
	}
}

// TestMetrics_LabelsApplied verifies labels include tool metadata.
func TestMetrics_LabelsApplied(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	m, err := newMetrics(meter)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{
		Namespace: "github",
		Name:      "create_issue",
	}
	m.RecordExecution(context.Background(), meta, 10*time.Millisecond, nil)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	found := findMetric(rm, "tool.exec.total")
	if found == nil {
		t.Fatal("tool.exec.total metric not found")
	}

	sum, ok := found.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", found.Data)
	}
	if len(sum.DataPoints) == 0 {
		t.Fatal("no data points")
	}

	// Verify attributes
	attrs := sum.DataPoints[0].Attributes
	var foundID, foundNS, foundName bool
	for iter := attrs.Iter(); iter.Next(); {
		kv := iter.Attribute()
		switch string(kv.Key) {
		case "tool.id":
			foundID = true
			if kv.Value.AsString() != "github.create_issue" {
				t.Errorf("expected tool.id='github.create_issue', got %q", kv.Value.AsString())
			}
		case "tool.namespace":
			foundNS = true
			if kv.Value.AsString() != "github" {
				t.Errorf("expected tool.namespace='github', got %q", kv.Value.AsString())
			}
		case "tool.name":
			foundName = true
			if kv.Value.AsString() != "create_issue" {
				t.Errorf("expected tool.name='create_issue', got %q", kv.Value.AsString())
			}
		}
	}

	if !foundID {
		t.Error("tool.id attribute not found")
	}
	if !foundNS {
		t.Error("tool.namespace attribute not found")
	}
	if !foundName {
		t.Error("tool.name attribute not found")
	}
}

// TestMetrics_ConcurrentRecording verifies thread safety.
func TestMetrics_ConcurrentRecording(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	m, err := newMetrics(meter)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	meta := ToolMeta{Name: "concurrent_tool"}
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			m.RecordExecution(context.Background(), meta, time.Millisecond, nil)
		}()
	}

	wg.Wait()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	found := findMetric(rm, "tool.exec.total")
	if found == nil {
		t.Fatal("tool.exec.total metric not found")
	}

	sum, ok := found.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", found.Data)
	}
	if len(sum.DataPoints) == 0 {
		t.Fatal("no data points")
	}
	if sum.DataPoints[0].Value != numGoroutines {
		t.Errorf("expected count %d, got %d", numGoroutines, sum.DataPoints[0].Value)
	}
}

// findMetric searches for a metric by name in ResourceMetrics.
func findMetric(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

// Silence unused import warning
var _ = attribute.String
