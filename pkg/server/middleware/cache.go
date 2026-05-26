package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// cachedResponse holds the serialized response for cache hits.
type cachedResponse struct {
	header     http.Header
	statusCode int
	body       []byte
}

// Cache caches successful (200) responses for idempotent handlers.
// Entries are evicted in FIFO order when the size limit is reached.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cachedResponse
	order   []string
	maxSize int
	maxBody int64
}

// NewCache creates a cache with the given maximum number of entries.
func NewCache(maxSize int) *Cache {
	return &Cache{
		entries: make(map[string]*cachedResponse),
		maxSize: maxSize,
		maxBody: 1 << 20, // 1 MB
	}
}

// Wrap returns an http.Handler that caches responses for identical requests.
func (c *Cache) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, c.maxBody))
		r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Restore body for downstream handlers.
		r.Body = io.NopCloser(bytes.NewReader(body))

		key := cacheKey(r.URL.Path, body)

		// Fast path: check cache.
		c.mu.RLock()
		if entry, ok := c.entries[key]; ok {
			c.mu.RUnlock()
			copyHeaders(w.Header(), entry.header)
			w.Header().Set("X-Cache", "hit")
			w.WriteHeader(entry.statusCode)
			w.Write(entry.body)
			return
		}
		c.mu.RUnlock()

		// Slow path: run the handler and capture its output.
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)

		// Copy response to client.
		copyHeaders(w.Header(), rec.Header())
		w.Header().Set("X-Cache", "miss")
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())

		// Store on success.
		if rec.Code == http.StatusOK {
			c.set(key, &cachedResponse{
				header:     cloneHeaders(rec.Header()),
				statusCode: rec.Code,
				body:       rec.Body.Bytes(),
			})
		}
	})
}

func (c *Cache) set(key string, entry *cachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[key]; ok {
		return
	}

	if len(c.entries) >= c.maxSize {
		oldest := c.order[0]
		delete(c.entries, oldest)
		c.order = c.order[1:]
	}

	c.entries[key] = entry
	c.order = append(c.order, key)
}

func cacheKey(path string, body []byte) string {
	h := sha256.Sum256(body)
	return path + ":" + hex.EncodeToString(h[:])
}

func copyHeaders(dst, src http.Header) {
	for k, v := range src {
		dst[k] = v
	}
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, v := range src {
		dst[k] = append([]string(nil), v...)
	}
	return dst
}
