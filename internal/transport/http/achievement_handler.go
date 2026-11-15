package http

import (
	"log/slog"
	"net/http"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

// AchievementHandler serves the Wins list. It resolves the owner from the
// authenticated context, so it must be mounted behind the auth middleware.
type AchievementHandler struct {
	svc     *service.AchievementService
	seasons *SeasonResolver
	logger  *slog.Logger
}

// NewAchievementHandler builds an AchievementHandler.
func NewAchievementHandler(svc *service.AchievementService, seasons *SeasonResolver, logger *slog.Logger) *AchievementHandler {
	return &AchievementHandler{svc: svc, seasons: seasons, logger: logger}
}

// List handles GET /achievements (?year=, default the season year).
func (h *AchievementHandler) List(w http.ResponseWriter, r *http.Request) {
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
	if year == nil {
		seasonYear := h.seasons.Resolve(r).Year
		year = &seasonYear
	}
	wins, err := h.svc.List(r.Context(), userID, *year)
	if err != nil {
		respondError(w, h.logger, r, err)
		return
	}
	out := make([]AchievementResponse, 0, len(wins))
	for i := range wins {
		out = append(out, AchievementResponse{
			Goal:  goalResponse(&wins[i].Goal),
			Topic: topicSummaryResponse(wins[i].Topic),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
