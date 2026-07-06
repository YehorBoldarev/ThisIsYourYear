package http

import (
	"log/slog"
	"net/http"
	"strings"

	"kibou/thisisyouryear/internal/auth"
	"kibou/thisisyouryear/internal/domain"
)

// TokenVerifier verifies an access token and returns its claims. The middleware
// depends on this narrow interface rather than the concrete issuer so it stays
// unit-testable.
type TokenVerifier interface {
	Verify(token string) (auth.Claims, error)
}

// Authenticator is middleware that requires a valid Bearer access token and
// injects the authenticated user's id into the request context.
type Authenticator struct {
	tokens TokenVerifier
	logger *slog.Logger
}

// NewAuthenticator builds an Authenticator.
func NewAuthenticator(tokens TokenVerifier, logger *slog.Logger) *Authenticator {
	return &Authenticator{tokens: tokens, logger: logger}
}

// Middleware verifies the access token and, on success, calls the next handler
// with the user id available via UserID(ctx). Any failure short-circuits with a
// 401 through the shared error mapper.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			respondError(w, a.logger, r, domain.ErrUnauthenticated)
			return
		}
		claims, err := a.tokens.Verify(token)
		if err != nil {
			respondError(w, a.logger, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), claims.UserID)))
	})
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	return token, token != ""
}
