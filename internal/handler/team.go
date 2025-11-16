package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aidar/avito-pr-project/internal/domain"
	"github.com/aidar/avito-pr-project/internal/service"
)

// TeamHandler обрабатывает эндпоинты команд
type TeamHandler struct {
	teamService *service.TeamService
}

// NewTeamHandler создает новый TeamHandler
func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{
		teamService: teamService,
	}
}

// AddTeamResponse представляет ответ на создание команды
type AddTeamResponse struct {
	Team *domain.Team `json:"team"`
}

// AddTeam обрабатывает POST /team/add
func (h *TeamHandler) AddTeam(w http.ResponseWriter, r *http.Request) {
	var team domain.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	// Валидация запроса
	if team.TeamName == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	// Создаем команду
	createdTeam, err := h.teamService.AddTeam(r.Context(), &team)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusCreated, AddTeamResponse{Team: createdTeam})
}

// GetTeam обрабатывает GET /team/get?team_name=...
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "team_name query parameter is required")
		return
	}

	team, err := h.teamService.GetTeam(r.Context(), teamName)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, team)
}
