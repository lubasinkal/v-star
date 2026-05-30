// Package server provides an HTTP API for v-star actuarial calculations.
// This allows Python, R, Excel, and other non-Go users to access v-star functionality.
package server

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/lubasinkal/v-star/pkg/server/middleware"
)

// DefaultCacheSize is the maximum number of entries in the per-endpoint caches.
const DefaultCacheSize = 1000

// Server holds the HTTP server configuration and routes.
type Server struct {
	addr   string
	server *http.Server
}

// New creates a new Server configured to listen on addr.
func New(addr string) *Server {
	return &Server{addr: addr}
}

// Handler returns the composed HTTP handler with all routes, middleware,
// concurrency limiters, and caches configured.
func (s *Server) Handler() http.Handler {
	return s.routes()
}

// routes registers all handler patterns and returns the composed handler.
func (s *Server) routes() http.Handler {
	cpu := runtime.NumCPU()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.healthHandler)
	s.handle(mux, "POST /value", s.pvHandler, 4*cpu, false)
	s.handle(mux, "POST /simulate", s.simulateHandler, cpu, false)
	s.handle(mux, "POST /annuity", s.annuityHandler, 4*cpu, true)
	s.handle(mux, "POST /reserve", s.reserveHandler, 4*cpu, true)
	s.handle(mux, "POST /profit", s.profitHandler, 4*cpu, true)
	return middleware.CreateStack(middleware.Logging, middleware.CORS)(mux)
}

// handle registers a route with a per-route concurrency limiter and optional cache.
// The limiter has a 5-second wait timeout (queues brief bursts instead of 503).
// Cache is applied outside the limiter so cache hits skip the slot wait.
func (s *Server) handle(mux *http.ServeMux, pattern string, handler http.HandlerFunc, concurrency int, cached bool) {
	h := http.Handler(handler)
	h = middleware.NewConcurrencyLimiterV(concurrency).Wrap(h)
	if cached {
		h = middleware.NewCache(DefaultCacheSize).Wrap(h)
	}
	mux.Handle(pattern, h)
}

// Start registers routes and begins listening.
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      s.routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s.server.ListenAndServe()
}

// StartWithGracefulShutdown starts the server and blocks until SIGINT/SIGTERM.
func (s *Server) StartWithGracefulShutdown() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("v-star server listening on %s", s.addr)
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(shutdownCtx)
}
