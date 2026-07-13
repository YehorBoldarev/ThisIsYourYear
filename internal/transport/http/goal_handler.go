package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

// GoalHandler serves the goal endpoints. Every handler resolves the owner from
// the authenticated context, so the routes must be mounted behind the auth
// middleware.
type GoalHandler struct {
	svc    *service.GoalService
	logger *slog.Logger
}

// NewGoalHandler builds a GoalHandler.
func NewGoalHandler(svc *service.GoalService, logger *slog.Logger) *GoalHandler {
	return &GoalHandler{svc: svc, logger: logger}
}

// Create handles POST /topics/{id}/goals, where {id} is the parent topic.
func (h *GoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	topicID := chi.URLParam(r, "id")
	var req CreateGoalRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	if err := validateStruct(req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	goal, err := h.svc.Create(r.Context(), userID, topicID, req.toInput())
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, goalResponse(goal))
}

// List handles GET /topics/{id}/goals.
func (h *GoalHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	goals, err := h.svc.ListForTopic(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, goalResponses(goals))
}

// Update handles PATCH /goals/{id}.
func (h *GoalHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	var req UpdateGoalRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	if err := validateStruct(req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	goal, err := h.svc.Update(r.Context(), userID, chi.URLParam(r, "id"), req.toDomain())
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, goalResponse(goal))
}

// Delete handles DELETE /goals/{id}.
func (h *GoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	if err := h.svc.Delete(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
