package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"
)

// ServerConfig holds the timeouts and address the HTTP server is built with.
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Server wraps an *http.Server with start/stop lifecycle methods that integrate
// with context cancellation for graceful shutdown.
type Server struct {
	httpServer      *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// NewServer constructs a Server bound to the configured port with the given
// handler and timeouts.
func NewServer(cfg ServerConfig, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         net.JoinHostPort("", strconv.Itoa(cfg.Port)),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

// Run starts serving and blocks until ctx is cancelled, at which point it
// gracefully shuts the server down within the configured shutdown timeout.
// It returns nil on a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)

	go func() {
		s.logger.Info("http server started", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	s.logger.Info("http server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	s.logger.Info("http server stopped")
	return nil
}
