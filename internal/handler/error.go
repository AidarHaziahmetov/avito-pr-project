package handler

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/aidar/avito-pr-project/internal/domain"
)

// ErrorResponse представляет ответ с ошибкой согласно OpenAPI спецификации
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail содержит код и описание ошибки
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondWithError отправляет ответ с ошибкой
func RespondWithError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	render.Status(r, statusCode)
	render.JSON(w, r, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// HandleError преобразует доменные ошибки в HTTP ответы
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case err == domain.ErrTeamExists:
		RespondWithError(w, r, http.StatusBadRequest, string(domain.CodeTeamExists), "team already exists")
	case err == domain.ErrPRExists:
		RespondWithError(w, r, http.StatusConflict, string(domain.CodePRExists), "pull request already exists")
	case err == domain.ErrPRMerged:
		RespondWithError(w, r, http.StatusConflict, string(domain.CodePRMerged), "cannot modify merged pull request")
	case err == domain.ErrNotAssigned:
		RespondWithError(w, r, http.StatusConflict, string(domain.CodeNotAssigned), "reviewer is not assigned to this PR")
	case err == domain.ErrNoCandidate:
		RespondWithError(w, r, http.StatusConflict, string(domain.CodeNoCandidate), "no active replacement candidate in team")
	case err == domain.ErrUserNotFound, err == domain.ErrTeamNotFound, err == domain.ErrPRNotFound, err == domain.ErrNotFound:
		RespondWithError(w, r, http.StatusNotFound, string(domain.CodeNotFound), "resource not found")
	case err == domain.ErrUnauthorized, err == domain.ErrInvalidToken:
		RespondWithError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
	default:
		RespondWithError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
