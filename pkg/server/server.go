// Package server provides an HTTP API for v-star actuarial calculations.
// This allows Python, R, Excel, and other non-Go users to access v-star functionality.
package server

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/lubasinkal/v-star/pkg/server/middleware"
)

// Server holds the HTTP server configuration and routes.
type Server struct {
	addr              string
	MortalityTableDir string
	server            *http.Server
}

// New creates a new Server configured to listen on addr.
func New(addr string) *Server {
	return &Server{addr: addr, MortalityTableDir: "mortality"}
}

// routes registers all handler patterns and returns the composed handler.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.healthHandler)
	mux.HandleFunc("POST /value", s.pvHandler)
	mux.HandleFunc("POST /montecarlo", s.monteCarloHandler)
	mux.HandleFunc("POST /convert-rate", s.convertRateHandler)
	mux.HandleFunc("GET /mortality/", s.mortalityHandler)
	mux.HandleFunc("POST /export/csv", s.exportCSVHandler)
	mux.HandleFunc("POST /export/report", s.exportReportHandler)
	mux.HandleFunc("POST /upload/csv", s.StreamCSVHandler)

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
