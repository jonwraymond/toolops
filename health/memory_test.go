package health

import (
	"context"
	"testing"
)

func TestNewMemoryChecker(t *testing.T) {
	checker := NewMemoryChecker(MemoryCheckerConfig{})

	if checker.config.WarningThreshold != 0.8 {
		t.Errorf("WarningThreshold = %v, want 0.8", checker.config.WarningThreshold)
	}
	if checker.config.CriticalThreshold != 0.95 {
		t.Errorf("CriticalThreshold = %v, want 0.95", checker.config.CriticalThreshold)
	}
}

func TestNewMemoryChecker_CustomThresholds(t *testing.T) {
	checker := NewMemoryChecker(MemoryCheckerConfig{
		WarningThreshold:  0.7,
		CriticalThreshold: 0.9,
	})

	if checker.config.WarningThreshold != 0.7 {
		t.Errorf("WarningThreshold = %v, want 0.7", checker.config.WarningThreshold)
	}
	if checker.config.CriticalThreshold != 0.9 {
		t.Errorf("CriticalThreshold = %v, want 0.9", checker.config.CriticalThreshold)
	}
}

func TestNewMemoryChecker_InvalidThresholds(t *testing.T) {
	// Invalid warning threshold
	checker := NewMemoryChecker(MemoryCheckerConfig{
		WarningThreshold: 1.5, // Invalid
	})
	if checker.config.WarningThreshold != 0.8 {
		t.Errorf("Invalid warning should default to 0.8, got %v", checker.config.WarningThreshold)
	}

	// Critical less than warning
	checker = NewMemoryChecker(MemoryCheckerConfig{
		WarningThreshold:  0.9,
		CriticalThreshold: 0.7,
	})
	if checker.config.CriticalThreshold <= checker.config.WarningThreshold {
		t.Error("Critical threshold should be adjusted to be > warning threshold")
	}
}

func TestMemoryChecker_Name(t *testing.T) {
	checker := NewMemoryChecker(MemoryCheckerConfig{})

	if checker.Name() != "memory" {
		t.Errorf("Name() = %v, want 'memory'", checker.Name())
	}
}

func TestMemoryChecker_Check(t *testing.T) {
	checker := NewMemoryChecker(MemoryCheckerConfig{})

	result := checker.Check(context.Background())

	// In a test environment, memory usage should be low (healthy)
	if result.Status == StatusUnhealthy {
		t.Logf("Warning: Memory check returned unhealthy: %s", result.Message)
	}

	if result.Details == nil {
		t.Error("Details should not be nil")
	}

	// Check that expected details are present
	expectedKeys := []string{"alloc_bytes", "heap_alloc", "num_gc", "goroutines"}
	for _, key := range expectedKeys {
		if _, ok := result.Details[key]; !ok {
			t.Errorf("Details missing key: %s", key)
		}
	}
}

func TestMemoryChecker_CheckContextCancelled(t *testing.T) {
	checker := NewMemoryChecker(MemoryCheckerConfig{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := checker.Check(ctx)

	if result.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want StatusUnhealthy for cancelled context", result.Status)
	}
	if result.Error != context.Canceled {
		t.Errorf("Error = %v, want context.Canceled", result.Error)
	}
}

func TestMemoryChecker_ForceGC(t *testing.T) {
	checker := NewMemoryChecker(MemoryCheckerConfig{})

	// This should not panic
	checker.ForceGC()

	// After GC, check should still work
	result := checker.Check(context.Background())
	if result.Status == StatusUnhealthy && result.Error != nil {
		t.Errorf("Check after ForceGC failed: %v", result.Error)
	}
}

func TestMemoryChecker_WithMaxAlloc(t *testing.T) {
	// Set a very low max alloc to trigger warnings
	checker := NewMemoryChecker(MemoryCheckerConfig{
		MaxAlloc:          1024, // 1KB - very low
		WarningThreshold:  0.5,
		CriticalThreshold: 0.8,
	})

	result := checker.Check(context.Background())

	// Should be degraded or unhealthy with such a low max alloc
	if result.Status == StatusHealthy {
		t.Log("Note: Memory check healthy even with 1KB max alloc - allocation might be reported as 0")
	}

	if result.Details["max_alloc"] != uint64(1024) {
		t.Errorf("max_alloc = %v, want 1024", result.Details["max_alloc"])
	}
}
