package http

import (
	"net/http"

	"github.com/go-chi/cors"
)

// corsMiddleware builds the CORS handler for the browser SPA. Credentials are
// enabled because the refresh token travels in a cookie; browsers forbid pairing
// credentials with a wildcard origin, which is why config validates that the
// allow-list contains only explicit origins.
func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
