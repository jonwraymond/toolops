package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// LivenessHandler returns an HTTP handler for liveness probes.
// This is a simple check that the service is running.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}

// ReadinessHandler returns an HTTP handler for readiness probes.
// This runs all health checks in the aggregator.
func ReadinessHandler(agg *Aggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		results := agg.CheckAll(ctx)
		status := agg.OverallStatus(results)

		w.Header().Set("Content-Type", "text/plain")

		switch status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		case StatusDegraded:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("DEGRADED"))
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("UNHEALTHY"))
		}
	}
}

// HealthResponse is the JSON response for the detailed health endpoint.
type HealthResponse struct {
	Status    string                   `json:"status"`
	Timestamp string                   `json:"timestamp"`
	Checks    map[string]CheckResponse `json:"checks,omitempty"`
}

// CheckResponse is the JSON response for a single health check.
type CheckResponse struct {
	Status   string         `json:"status"`
	Message  string         `json:"message,omitempty"`
	Duration string         `json:"duration,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// DetailedHandler returns an HTTP handler that provides detailed health information.
func DetailedHandler(agg *Aggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		results := agg.CheckAll(ctx)
		status := agg.OverallStatus(results)

		response := HealthResponse{
			Status:    status.String(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    make(map[string]CheckResponse, len(results)),
		}

		for name, result := range results {
			check := CheckResponse{
				Status:   result.Status.String(),
				Message:  result.Message,
				Duration: result.Duration.String(),
				Details:  result.Details,
			}
			if result.Error != nil {
				check.Error = result.Error.Error()
			}
			response.Checks[name] = check
		}

		w.Header().Set("Content-Type", "application/json")

		switch status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// SingleCheckHandler returns an HTTP handler for checking a single component.
func SingleCheckHandler(agg *Aggregator, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		result, err := agg.Check(ctx, name)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		response := CheckResponse{
			Status:   result.Status.String(),
			Message:  result.Message,
			Duration: result.Duration.String(),
			Details:  result.Details,
		}
		if result.Error != nil {
			response.Error = result.Error.Error()
		}

		w.Header().Set("Content-Type", "application/json")

		switch result.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// RegisterHandlers registers all health check handlers on the given mux.
func RegisterHandlers(mux *http.ServeMux, agg *Aggregator) {
	mux.HandleFunc("/healthz", LivenessHandler())
	mux.HandleFunc("/readyz", ReadinessHandler(agg))
	mux.HandleFunc("/health", DetailedHandler(agg))
}
