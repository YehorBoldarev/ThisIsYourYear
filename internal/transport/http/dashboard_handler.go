package http

import (
	"log/slog"
	"net/http"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

// DashboardHandler serves the dashboard endpoint. It resolves the owner from the
// authenticated context, so it must be mounted behind the auth middleware.
type DashboardHandler struct {
	svc    *service.DashboardService
	logger *slog.Logger
}

// NewDashboardHandler builds a DashboardHandler.
func NewDashboardHandler(svc *service.DashboardService, logger *slog.Logger) *DashboardHandler {
	return &DashboardHandler{svc: svc, logger: logger}
}

// Get handles GET /dashboard.
func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	view, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboardResponse(view))
}
