package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aidar/avito-pr-project/internal/service"
)

// AuthHandler обрабатывает эндпоинты аутентификации
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler создает новый AuthHandler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LoginRequest представляет тело запроса на логин
type LoginRequest struct {
	UserID string `json:"user_id"`
}

// LoginResponse представляет тело ответа на логин
type LoginResponse struct {
	Token string `json:"token"`
}

// Login обрабатывает POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.UserID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	token, err := h.authService.Login(r.Context(), req.UserID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, LoginResponse{Token: token})
}
