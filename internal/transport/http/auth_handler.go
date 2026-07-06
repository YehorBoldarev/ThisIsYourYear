package http

import (
	"log/slog"
	"net"
	"net/http"

	"kibou/thisisyouryear/internal/service"
)

// AuthHandler serves the authentication endpoints. Handlers stay thin: decode,
// validate, delegate to the service, render. The refresh token is read from and
// written to the HttpOnly cookie exclusively here.
type AuthHandler struct {
	svc    *service.AuthService
	cookie RefreshCookieConfig
	logger *slog.Logger
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(svc *service.AuthService, cookie RefreshCookieConfig, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, cookie: cookie, logger: logger}
}

// Register creates a new account and returns its public profile (201).
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	if err := validateStruct(req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}

	user, err := h.svc.Register(r.Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

// Login authenticates the user, sets the refresh cookie, and returns an access
// token with the user profile.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	if err := validateStruct(req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}

	creds, err := h.svc.Login(r.Context(), req.Email, req.Password, req.RememberMe, r.UserAgent(), clientIP(r))
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	h.cookie.set(w, creds.RefreshToken, creds.RefreshExpiresAt)
	writeJSON(w, http.StatusOK, tokenResponse(creds))
}

// Refresh rotates the refresh session (via the cookie) and returns a new access
// token. On any failure the cookie is cleared so the browser stops resending a
// dead token.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	creds, err := h.svc.Refresh(r.Context(), h.cookie.read(r), r.UserAgent(), clientIP(r))
	if err != nil {
		h.cookie.clear(w)
		respondError(w, h.logger, r, err)
		return
	}
	h.cookie.set(w, creds.RefreshToken, creds.RefreshExpiresAt)
	writeJSON(w, http.StatusOK, tokenResponse(creds))
}

// Logout revokes the current session and clears the cookie. It is idempotent and
// always succeeds with 204.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context(), h.cookie.read(r)); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	h.cookie.clear(w)
	w.WriteHeader(http.StatusNoContent)
}

// clientIP returns the peer address without the port. We intentionally do not
// trust forwarding headers (see the RealIP note in router.go); this is the
// direct connection's address, recorded on the session for auditability.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
