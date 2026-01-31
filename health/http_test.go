package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLivenessHandler(t *testing.T) {
	handler := LivenessHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("Body = %v, want 'OK'", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type = %v, want 'text/plain'", rec.Header().Get("Content-Type"))
	}
}

func TestReadinessHandler_Healthy(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("ok")
	}))

	handler := ReadinessHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("Body = %v, want 'OK'", rec.Body.String())
	}
}

func TestReadinessHandler_Degraded(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Degraded("slow")
	}))

	handler := ReadinessHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d (degraded should still be OK)", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "DEGRADED" {
		t.Errorf("Body = %v, want 'DEGRADED'", rec.Body.String())
	}
}

func TestReadinessHandler_Unhealthy(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Unhealthy("down", nil)
	}))

	handler := ReadinessHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if rec.Body.String() != "UNHEALTHY" {
		t.Errorf("Body = %v, want 'UNHEALTHY'", rec.Body.String())
	}
}

func TestDetailedHandler_Healthy(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("ok").WithDetails(map[string]any{"key": "value"})
	}))

	handler := DetailedHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %v, want 'application/json'", rec.Header().Get("Content-Type"))
	}

	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Response.Status = %v, want 'healthy'", response.Status)
	}
	if response.Timestamp == "" {
		t.Error("Response.Timestamp should not be empty")
	}
	if check, ok := response.Checks["test"]; !ok {
		t.Error("Response.Checks should contain 'test'")
	} else {
		if check.Status != "healthy" {
			t.Errorf("Check.Status = %v, want 'healthy'", check.Status)
		}
	}
}

func TestDetailedHandler_Unhealthy(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Unhealthy("down", ErrCheckFailed)
	}))

	handler := DetailedHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("Response.Status = %v, want 'unhealthy'", response.Status)
	}
	if check := response.Checks["test"]; check.Error == "" {
		t.Error("Check.Error should contain error message")
	}
}

func TestSingleCheckHandler_Found(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("ok")
	}))

	handler := SingleCheckHandler(agg, "test")

	req := httptest.NewRequest(http.MethodGet, "/health/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response CheckResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Response.Status = %v, want 'healthy'", response.Status)
	}
}

func TestSingleCheckHandler_NotFound(t *testing.T) {
	agg := NewAggregator()

	handler := SingleCheckHandler(agg, "nonexistent")

	req := httptest.NewRequest(http.MethodGet, "/health/nonexistent", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSingleCheckHandler_Unhealthy(t *testing.T) {
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Unhealthy("down", nil)
	}))

	handler := SingleCheckHandler(agg, "test")

	req := httptest.NewRequest(http.MethodGet, "/health/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestRegisterHandlers(t *testing.T) {
	mux := http.NewServeMux()
	agg := NewAggregator()
	agg.Register("test", NewCheckerFunc("test", func(ctx context.Context) Result {
		return Healthy("ok")
	}))

	RegisterHandlers(mux, agg)

	// Test /healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("/healthz Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Test /readyz
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("/readyz Status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Test /health
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("/health Status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDetailedHandler_Timeout(t *testing.T) {
	agg := NewAggregator(AggregatorConfig{
		Timeout: 50 * time.Millisecond,
	})
	agg.Register("slow", NewCheckerFunc("slow", func(ctx context.Context) Result {
		time.Sleep(200 * time.Millisecond)
		return Healthy("ok")
	}))

	handler := DetailedHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d for timed out check", rec.Code, http.StatusServiceUnavailable)
	}

	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("Response.Status = %v, want 'unhealthy'", response.Status)
	}
}
