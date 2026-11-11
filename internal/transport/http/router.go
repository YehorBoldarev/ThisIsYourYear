package http

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"kibou/thisisyouryear/internal/service"
)

// RouterDeps carries the collaborators the router needs to serve requests.
// Future steps extend this struct with services as new route groups are mounted.
type RouterDeps struct {
	Logger *slog.Logger
	Pinger Pinger

	// Authentication collaborators. These may be nil in tests that only exercise
	// the operational endpoints; the auth/user routes are wired regardless but
	// only touch these when called.
	AuthService      *service.AuthService
	UserService      *service.UserService
	TopicService     *service.TopicService
	GoalService      *service.GoalService
	DashboardService *service.DashboardService
	TokenVerifier    TokenVerifier
	RefreshCookie    RefreshCookieConfig

	// CORSOrigins is the allow-list of browser origins. When empty, no CORS
	// middleware is installed (same-origin only).
	CORSOrigins []string

	// RateLimit configures per-IP throttling of the public auth endpoints. When
	// RateLimit.AuthRequests is 0, no auth rate limiting is installed.
	RateLimit RateLimitConfig

	// ClientIP says which reverse proxies may state the client IP. The zero
	// value trusts none: the client IP is always the direct peer address.
	ClientIP ClientIPConfig

	// SPAFS, when non-nil, is the built single-page app served for any route the
	// API/ops groups don't match (with a client-side-routing fallback to
	// index.html). Nil ⇒ API-only: unmatched routes get the default 404.
	SPAFS fs.FS
}

// RateLimitConfig describes the per-IP fixed-window limit applied to the public
// auth endpoints. AuthRequests == 0 disables limiting.
type RateLimitConfig struct {
	AuthRequests int
	AuthWindow   time.Duration
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
	// vulnerable to IP spoofing. clientIPMiddleware instead honours a single-IP
	// header only from allow-listed proxy addresses (deps.ClientIP), and runs
	// before the logger and rate limiter so they all see the same client IP.
	r.Use(middleware.RequestID)
	r.Use(clientIPMiddleware(deps.ClientIP))
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Recoverer)
	// CORS must run before the request timeout and route handlers so browser
	// preflight (OPTIONS) is answered even for protected routes.
	if len(deps.CORSOrigins) > 0 {
		r.Use(corsMiddleware(deps.CORSOrigins))
	}
	r.Use(middleware.Timeout(30 * time.Second))

	// Operational endpoints live outside the versioned API namespace.
	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readinessHandler(deps.Pinger))
	// The API contract is published (unauthenticated) for client tooling.
	r.Get("/openapi.yaml", openapiHandler)

	authHandler := NewAuthHandler(deps.AuthService, deps.RefreshCookie, deps.Logger)
	userHandler := NewUserHandler(deps.UserService, deps.Logger)
	topicHandler := NewTopicHandler(deps.TopicService, deps.Logger)
	goalHandler := NewGoalHandler(deps.GoalService, deps.Logger)
	dashboardHandler := NewDashboardHandler(deps.DashboardService, deps.Logger)
	authenticator := NewAuthenticator(deps.TokenVerifier, deps.Logger)

	// Versioned API namespace. New feature route groups mount under here without
	// altering existing groups.
	r.Route("/api/v1", func(api chi.Router) {
		// Public authentication routes. These are the abuse-prone, unauthenticated
		// surface, so they carry a per-IP rate limit when one is configured.
		api.Route("/auth", func(a chi.Router) {
			if deps.RateLimit.AuthRequests > 0 {
				limiter := newRateLimiter(deps.RateLimit.AuthRequests, deps.RateLimit.AuthWindow, nil)
				a.Use(limiter.middleware)
			}
			a.Get("/registration", authHandler.RegistrationInfo)
			a.Post("/register", authHandler.Register)
			a.Post("/login", authHandler.Login)
			a.Post("/refresh", authHandler.Refresh)
			a.Post("/logout", authHandler.Logout)
		})

		// Authenticated routes. Every route added inside this group requires a
		// valid access token; this is the template later features reuse.
		api.Group(func(protected chi.Router) {
			protected.Use(authenticator.Middleware)
			protected.Get("/me", userHandler.Me)

			protected.Post("/topics", topicHandler.Create)
			protected.Get("/topics", topicHandler.List)
			protected.Get("/topics/{id}", topicHandler.Get)
			protected.Patch("/topics/{id}", topicHandler.Update)
			protected.Delete("/topics/{id}", topicHandler.Delete)

			// Goals live under their topic for creation/listing; individual
			// goals are addressed directly. {id} is the topic id on the nested
			// routes (kept consistent with the topic routes to avoid a chi
			// wildcard-name conflict).
			protected.Post("/topics/{id}/goals", goalHandler.Create)
			protected.Get("/topics/{id}/goals", goalHandler.List)
			protected.Patch("/goals/{id}", goalHandler.Update)
			protected.Delete("/goals/{id}", goalHandler.Delete)

			// Landing summary: aggregate progress + message + New-Year countdown.
			protected.Get("/dashboard", dashboardHandler.Get)
		})
	})

	// When a SPA is provided, serve it for every unmatched route (static file or
	// index.html fallback). This runs after all API/ops routes, so it never
	// shadows them; unmatched /api/** paths still return the JSON 404 envelope.
	if deps.SPAFS != nil {
		r.NotFound(newSPAHandler(deps.SPAFS, deps.Logger))
	}

	return r
}
