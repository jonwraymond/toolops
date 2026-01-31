package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAuthHeaders(t *testing.T) {
	// Create a test handler that checks for headers in context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get headers from context
		headers := HeadersFromContext(r.Context())
		if headers == nil {
			t.Error("Headers not found in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Check specific header
		authHeader := GetHeader(r.Context(), "Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Authorization = %v, want Bearer test-token", authHeader)
		}

		customHeader := GetHeader(r.Context(), "X-Custom-Header")
		if customHeader != "custom-value" {
			t.Errorf("X-Custom-Header = %v, want custom-value", customHeader)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	handler := WithAuthHeaders(testHandler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Custom-Header", "custom-value")

	// Execute
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestWithAuthHeaders_MultipleValues(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := HeadersFromContext(r.Context())
		if headers == nil {
			t.Error("Headers not found in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Check multiple values
		acceptValues := headers["Accept"]
		if len(acceptValues) != 2 {
			t.Errorf("Accept has %d values, want 2", len(acceptValues))
		}

		// GetHeader should return first value
		accept := GetHeader(r.Context(), "Accept")
		if accept != "text/html" {
			t.Errorf("Accept = %v, want text/html", accept)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := WithAuthHeaders(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Add("Accept", "text/html")
	req.Header.Add("Accept", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}
