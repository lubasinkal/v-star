package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateStack(t *testing.T) {
	var calls []string
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "mw1")
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	stack := CreateStack(mw1, mw2)
	handler := stack(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "final")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if len(calls) != 3 || calls[0] != "mw1" || calls[1] != "mw2" || calls[2] != "final" {
		t.Errorf("unexpected call order: %v", calls)
	}
}

func TestCORS(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("sets headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("missing CORS header")
		}
	})

	t.Run("preflight returns 200", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("preflight status = %d, want 200", w.Code)
		}
	})
}

func TestLogging(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest("GET", "/teapot", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", w.Code)
	}
}

func TestConcurrencyLimiter_PassesThrough(t *testing.T) {
	limiter := NewConcurrencyLimiter(1)
	called := false
	handler := limiter.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestConcurrencyLimiter_RejectsWhenFull(t *testing.T) {
	limiter := NewConcurrencyLimiter(2)
	block := make(chan struct{})

	handler := limiter.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // blocks until we unblock
		w.WriteHeader(http.StatusOK)
	}))

	// Fill both slots.
	w1 := httptest.NewRecorder()
	w2 := httptest.NewRecorder()
	go handler.ServeHTTP(w1, httptest.NewRequest("GET", "/test", nil))
	go handler.ServeHTTP(w2, httptest.NewRequest("GET", "/test", nil))

	// Wait for goroutines to acquire slots.
	time.Sleep(50 * time.Millisecond)

	// Third request should get 503.
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, httptest.NewRequest("GET", "/test", nil))

	if w3.Code != http.StatusServiceUnavailable {
		t.Errorf("third request status = %d, want 503", w3.Code)
	}

	close(block)
}

func TestCache_Hit(t *testing.T) {
	cache := NewCache(100)
	callCount := 0
	handler := cache.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":42}`))
	}))

	// First request — cache miss, handler runs.
	req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"x":1}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if callCount != 1 {
		t.Errorf("callCount after first = %d, want 1", callCount)
	}
	if w.Header().Get("X-Cache") != "miss" {
		t.Errorf("X-Cache = %q, want miss", w.Header().Get("X-Cache"))
	}

	// Second request with same body — cache hit, handler not called.
	req2 := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"x":1}`)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if callCount != 1 {
		t.Errorf("callCount after second = %d, want 1", callCount)
	}
	if w2.Header().Get("X-Cache") != "hit" {
		t.Errorf("X-Cache = %q, want hit", w2.Header().Get("X-Cache"))
	}
	if w2.Body.String() != w.Body.String() {
		t.Errorf("body mismatch: %q vs %q", w2.Body.String(), w.Body.String())
	}
}

func TestCache_DifferentInputs(t *testing.T) {
	cache := NewCache(100)
	callCount := 0
	handler := cache.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(`{"ok":true}`))
	}))

	// Two requests with different bodies — both miss.
	req1 := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"x":1}`)))
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"x":2}`)))
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
	if w1.Header().Get("X-Cache") != "miss" || w2.Header().Get("X-Cache") != "miss" {
		t.Error("expected both to be cache misses")
	}
}

func TestCache_DifferentPaths(t *testing.T) {
	cache := NewCache(100)
	callCount := 0
	handler := cache.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(`{"ok":true}`))
	}))

	body := []byte(`{"x":1}`)
	req1 := httptest.NewRequest("POST", "/a", bytes.NewReader(body))
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	// Same body, different path — cache miss.
	req2 := httptest.NewRequest("POST", "/b", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestCache_MaxSizeEviction(t *testing.T) {
	cache := NewCache(2)
	callCount := 0
	handler := cache.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(`{}`))
	}))

	// Fill cache with 3 different requests (maxSize=2).
	for i := 0; i < 3; i++ {
		body := []byte(`{"i":` + string(rune('0'+i)) + `}`)
		req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// First request was evicted, so fourth call should be a miss.
	body := []byte(`{"i":0}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Cache") != "miss" {
		t.Errorf("expected miss for evicted entry, got %q", w.Header().Get("X-Cache"))
	}
	// callCount should be: 3 fills + 1 miss for evicted = 4
	if callCount != 4 {
		t.Errorf("callCount = %d, want 4", callCount)
	}
}
