package middleware

import (
	"encoding/json"
	"net/http"
	"time"
)

// ConcurrencyLimiter restricts concurrent requests to a handler.
// When all slots are busy, new requests block up to a timeout
// waiting for a slot to open. If the timeout expires, the
// request gets a 503.
//
// This absorbs brief traffic bursts without rejection — clients
// just wait a few hundred milliseconds instead of getting an error.
type ConcurrencyLimiter struct {
	sem     chan struct{}
	timeout time.Duration
}

// NewConcurrencyLimiter creates a limiter with max concurrent slots
// and a timeout for how long a request waits before getting 503.
func NewConcurrencyLimiter(max int, timeout time.Duration) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		sem:     make(chan struct{}, max),
		timeout: timeout,
	}
}

// NewConcurrencyLimiterV creates a limiter with a 5-second timeout.
func NewConcurrencyLimiterV(max int) *ConcurrencyLimiter {
	return NewConcurrencyLimiter(max, 5*time.Second)
}

// Wrap returns an http.Handler that enforces the concurrency limit.
func (l *ConcurrencyLimiter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(l.timeout)
		defer timer.Stop()

		select {
		case l.sem <- struct{}{}:
			defer func() { <-l.sem }()
			next.ServeHTTP(w, r)
		case <-timer.C:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "server busy, try again later",
			})
		}
	})
}
