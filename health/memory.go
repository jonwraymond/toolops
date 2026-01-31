package health

import (
	"context"
	"fmt"
	"runtime"
)

// MemoryCheckerConfig configures the memory health checker.
type MemoryCheckerConfig struct {
	// WarningThreshold is the percentage of allocated memory that triggers degraded status.
	// Value should be between 0 and 1. Default: 0.8 (80%)
	WarningThreshold float64

	// CriticalThreshold is the percentage of allocated memory that triggers unhealthy status.
	// Value should be between 0 and 1. Default: 0.95 (95%)
	CriticalThreshold float64

	// MaxAlloc is the maximum expected allocation in bytes.
	// If zero, uses the system's total memory (approximated).
	// Default: 0 (auto-detect)
	MaxAlloc uint64
}

// MemoryChecker checks memory usage health.
type MemoryChecker struct {
	config MemoryCheckerConfig
}

// NewMemoryChecker creates a new memory health checker.
func NewMemoryChecker(config MemoryCheckerConfig) *MemoryChecker {
	if config.WarningThreshold <= 0 || config.WarningThreshold >= 1 {
		config.WarningThreshold = 0.8
	}
	if config.CriticalThreshold <= 0 || config.CriticalThreshold >= 1 {
		config.CriticalThreshold = 0.95
	}
	if config.CriticalThreshold < config.WarningThreshold {
		config.CriticalThreshold = config.WarningThreshold + 0.1
		if config.CriticalThreshold > 1 {
			config.CriticalThreshold = 0.99
		}
	}

	return &MemoryChecker{config: config}
}

// Name returns the name of this checker.
func (m *MemoryChecker) Name() string {
	return "memory"
}

// Check performs the memory health check.
func (m *MemoryChecker) Check(ctx context.Context) Result {
	// Check context first
	select {
	case <-ctx.Done():
		return Unhealthy("context cancelled", ctx.Err())
	default:
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	maxAlloc := m.config.MaxAlloc
	if maxAlloc == 0 {
		// Use a reasonable default based on current allocation
		// In production, this should be configured
		maxAlloc = stats.Sys
	}

	if maxAlloc == 0 {
		return Healthy("memory stats unavailable").WithDetails(map[string]any{
			"alloc":       stats.Alloc,
			"total_alloc": stats.TotalAlloc,
			"sys":         stats.Sys,
			"num_gc":      stats.NumGC,
		})
	}

	usageRatio := float64(stats.Alloc) / float64(maxAlloc)

	details := map[string]any{
		"alloc_bytes":    stats.Alloc,
		"alloc_mb":       float64(stats.Alloc) / (1024 * 1024),
		"max_alloc":      maxAlloc,
		"usage_percent":  usageRatio * 100,
		"heap_alloc":     stats.HeapAlloc,
		"heap_sys":       stats.HeapSys,
		"heap_idle":      stats.HeapIdle,
		"heap_in_use":    stats.HeapInuse,
		"heap_released":  stats.HeapReleased,
		"heap_objects":   stats.HeapObjects,
		"stack_in_use":   stats.StackInuse,
		"stack_sys":      stats.StackSys,
		"gc_pause_total": stats.PauseTotalNs,
		"num_gc":         stats.NumGC,
		"goroutines":     runtime.NumGoroutine(),
	}

	if usageRatio >= m.config.CriticalThreshold {
		return Unhealthy(
			fmt.Sprintf("memory usage critical: %.1f%%", usageRatio*100),
			ErrCheckFailed,
		).WithDetails(details)
	}

	if usageRatio >= m.config.WarningThreshold {
		return Degraded(
			fmt.Sprintf("memory usage high: %.1f%%", usageRatio*100),
		).WithDetails(details)
	}

	return Healthy(
		fmt.Sprintf("memory usage normal: %.1f%%", usageRatio*100),
	).WithDetails(details)
}

// ForceGC triggers a garbage collection.
// This is useful for tests or when you want to get accurate memory stats.
func (m *MemoryChecker) ForceGC() {
	runtime.GC()
}
