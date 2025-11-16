package handler

import (
	"net/http"

	"github.com/aidar/avito-pr-project/internal/service"
)

// StatsHandler обрабатывает эндпоинты статистики
type StatsHandler struct {
	statsService *service.StatsService
}

// NewStatsHandler создает новый StatsHandler
func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

// GetStats обрабатывает GET /stats
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsService.GetStats(r.Context())
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, stats)
}

// GetUserStats обрабатывает GET /stats/user?user_id=...
func (h *StatsHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "user_id query parameter is required")
		return
	}

	stats, err := h.statsService.GetUserStats(r.Context(), userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, stats)
}
