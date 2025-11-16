package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aidar/avito-pr-project/internal/domain"
	"github.com/aidar/avito-pr-project/internal/service"
)

// UserHandler обрабатывает эндпоинты пользователей
type UserHandler struct {
	userService *service.UserService
	prService   *service.PullRequestService
}

// NewUserHandler создает новый UserHandler
func NewUserHandler(userService *service.UserService, prService *service.PullRequestService) *UserHandler {
	return &UserHandler{
		userService: userService,
		prService:   prService,
	}
}

// SetIsActiveRequest представляет тело запроса для установки флага активности
type SetIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

// SetIsActiveResponse представляет ответ на установку флага активности
type SetIsActiveResponse struct {
	User *domain.User `json:"user"`
}

// SetIsActive обрабатывает POST /users/setIsActive
func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req SetIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.UserID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	user, err := h.userService.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, SetIsActiveResponse{User: user})
}

// GetReviewResponse представляет ответ со списком PR пользователя
type GetReviewResponse struct {
	UserID       string                     `json:"user_id"`
	PullRequests []*domain.PullRequestShort `json:"pull_requests"`
}

// GetReview обрабатывает GET /users/getReview?user_id=...
func (h *UserHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "user_id query parameter is required")
		return
	}

	prs, err := h.prService.GetPRsByReviewer(r.Context(), userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, GetReviewResponse{
		UserID:       userID,
		PullRequests: prs,
	})
}
