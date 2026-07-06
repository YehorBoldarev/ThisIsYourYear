package http

import (
	"log/slog"
	"net/http"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

// UserHandler serves authenticated user endpoints.
type UserHandler struct {
	svc    *service.UserService
	logger *slog.Logger
}

// NewUserHandler builds a UserHandler.
func NewUserHandler(svc *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, logger: logger}
}

// Me returns the profile of the authenticated user. It must be mounted behind
// the auth middleware, which populates the user id in the request context.
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	user, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}
