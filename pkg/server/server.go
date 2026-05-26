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

// Server holds the HTTP server configuration and routes.
type Server struct {
	addr   string
	server *http.Server
}

// New creates a new Server configured to listen on addr.
func New(addr string) *Server {
	return &Server{addr: addr}
}

// routes registers all handler patterns and returns the composed handler.
func (s *Server) routes() http.Handler {
	// Per-route concurrency limits, scaled to available CPU cores.
	// Heavy CPU endpoints get 1×NumCPU, lighter I/O-bound get 4×NumCPU.
	cpu := runtime.NumCPU()
	valueLim := middleware.NewConcurrencyLimiter(4 * cpu)
	simLim := middleware.NewConcurrencyLimiter(cpu)
	annLim := middleware.NewConcurrencyLimiter(4 * cpu)
	reserveLim := middleware.NewConcurrencyLimiter(4 * cpu)

	// Result cache for idempotent pure-function endpoints.
	annCache := middleware.NewCache(1000)
	reserveCache := middleware.NewCache(1000)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.healthHandler)
	mux.Handle("POST /value", valueLim.Wrap(http.HandlerFunc(s.pvHandler)))
	mux.Handle("POST /simulate", simLim.Wrap(http.HandlerFunc(s.simulateHandler)))
	mux.Handle("POST /annuity", annCache.Wrap(annLim.Wrap(http.HandlerFunc(s.annuityHandler))))
	mux.Handle("POST /reserve", reserveCache.Wrap(reserveLim.Wrap(http.HandlerFunc(s.reserveHandler))))

	return middleware.CreateStack(
		middleware.Logging,
		middleware.CORS,
	)(mux)
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
