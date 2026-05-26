package middleware

import (
	"encoding/json"
	"net/http"
)

// ConcurrencyLimiter restricts concurrent requests to a handler.
// Returns 503 Service Unavailable when the limit is reached.
type ConcurrencyLimiter struct {
	sem chan struct{}
}

// NewConcurrencyLimiter creates a limiter that allows up to max concurrent requests.
func NewConcurrencyLimiter(max int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		sem: make(chan struct{}, max),
	}
}

// Wrap returns an http.Handler that enforces the concurrency limit.
func (l *ConcurrencyLimiter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case l.sem <- struct{}{}:
			defer func() { <-l.sem }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "too many concurrent requests",
			})
		}
	})
}
