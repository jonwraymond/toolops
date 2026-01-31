package auth

import "net/http"

// WithAuthHeaders is HTTP middleware that extracts request headers
// into the context for use by authentication middleware.
//
// This middleware should wrap HTTP handlers that process requests,
// enabling authenticators to access headers like Authorization and X-API-Key.
//
// Usage:
//
//	mux.Handle("/api", auth.WithAuthHeaders(apiHandler))
func WithAuthHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract all headers into context
		ctx := WithHeaders(r.Context(), r.Header)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
