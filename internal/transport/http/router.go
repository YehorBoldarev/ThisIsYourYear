package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterDeps carries the collaborators the router needs to serve requests.
// Future steps extend this struct with services as new route groups are mounted.
type RouterDeps struct {
	Logger *slog.Logger
	Pinger Pinger
}

// NewRouter builds the application's HTTP handler: the shared middleware chain
// plus all mounted routes. Adding a new feature means mounting a new sub-router
// here, without altering existing groups.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Order matters: assign a request ID first so it is available to every
	// subsequent middleware and handler, recover from panics before they reach
	// the server, and bound total handler time last.
	//
	// We deliberately do not use middleware.RealIP: it trusts client-supplied
	// forwarding headers (X-Forwarded-For / X-Real-IP) unconditionally and is
	// vulnerable to IP spoofing. If the service is later deployed behind a
	// trusted proxy, we will derive the client IP from a vetted, allow-listed
	// source at that point.
	r.Use(middleware.RequestID)
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Operational endpoints live outside the versioned API namespace.
	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readinessHandler(deps.Pinger))

	// Versioned API namespace. Feature route groups are mounted under here by
	// later steps (e.g. r.Mount("/auth", authRouter)).
	r.Route("/api/v1", func(_ chi.Router) {})

	return r
}
