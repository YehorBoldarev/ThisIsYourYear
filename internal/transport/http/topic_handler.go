package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

// TopicHandler serves the topic endpoints. Every handler resolves the owner from
// the authenticated context, so the routes must be mounted behind the auth
// middleware.
type TopicHandler struct {
	svc    *service.TopicService
	logger *slog.Logger
}

// NewTopicHandler builds a TopicHandler.
func NewTopicHandler(svc *service.TopicService, logger *slog.Logger) *TopicHandler {
	return &TopicHandler{svc: svc, logger: logger}
}

// Create handles POST /topics.
func (h *TopicHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	var req CreateTopicRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	if err := validateStruct(req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	view, err := h.svc.Create(r.Context(), userID, req.toInput())
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, topicResponse(view))
}

// List handles GET /topics (year-scoped; ?year= and ?includeArchived=true).
func (h *TopicHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	year, err := parseYearQuery(r)
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	includeArchived := r.URL.Query().Get("includeArchived") == "true"

	views, err := h.svc.List(r.Context(), userID, year, includeArchived)
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, topicResponses(views))
}

// Get handles GET /topics/{id}.
func (h *TopicHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	view, err := h.svc.Get(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, topicResponse(view))
}

// Update handles PATCH /topics/{id}.
func (h *TopicHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		respondError(w, h.logger, r, domain.ErrUnauthenticated)
		return
	}
	var req UpdateTopicRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	if err := validateStruct(req); err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	view, err := h.svc.Update(r.Context(), userID, chi.URLParam(r, "id"), req.toDomain())
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	writeJSON(w, http.StatusOK, topicResponse(view))
}

// Delete handles DELETE /topics/{id}.
func (h *TopicHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// parseYearQuery reads an optional ?year= parameter. Absent -> nil (service
// defaults to the current year); present but non-numeric -> validation error.
func parseYearQuery(r *http.Request) (*int, error) {
	raw := r.URL.Query().Get("year")
	if raw == "" {
		return nil, nil
	}
	year, err := strconv.Atoi(raw)
	if err != nil {
		return nil, domain.ErrValidation
	}
	return &year, nil
}
