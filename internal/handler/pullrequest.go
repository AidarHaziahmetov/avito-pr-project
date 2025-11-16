package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aidar/avito-pr-project/internal/domain"
	"github.com/aidar/avito-pr-project/internal/service"
)

// PullRequestHandler обрабатывает эндпоинты pull request'ов
type PullRequestHandler struct {
	prService *service.PullRequestService
}

// NewPullRequestHandler создает новый PullRequestHandler
func NewPullRequestHandler(prService *service.PullRequestService) *PullRequestHandler {
	return &PullRequestHandler{
		prService: prService,
	}
}

// CreatePRRequest представляет тело запроса для создания PR
type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

// CreatePRResponse представляет ответ на создание PR
type CreatePRResponse struct {
	PR *domain.PullRequest `json:"pr"`
}

// CreatePR обрабатывает POST /pullRequest/create
func (h *PullRequestHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	// Валидация запроса
	if req.PullRequestID == "" || req.PullRequestName == "" || req.AuthorID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id, pull_request_name, and author_id are required")
		return
	}

	// Создаем PR (автоматически назначаются ревьюверы)
	pr, err := h.prService.CreatePR(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusCreated, CreatePRResponse{PR: pr})
}

// MergePRRequest представляет тело запроса для merge PR
type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

// MergePRResponse представляет ответ на merge PR
type MergePRResponse struct {
	PR *domain.PullRequest `json:"pr"`
}

// MergePR обрабатывает POST /pullRequest/merge (идемпотентная операция)
func (h *PullRequestHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.PullRequestID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id is required")
		return
	}

	// Мержим PR (идемпотентная операция)
	pr, err := h.prService.MergePR(r.Context(), req.PullRequestID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, MergePRResponse{PR: pr})
}

// ReassignRequest представляет тело запроса для переназначения ревьювера
type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

// ReassignResponse представляет ответ на переназначение ревьювера
type ReassignResponse struct {
	PR         *domain.PullRequest `json:"pr"`
	ReplacedBy string              `json:"replaced_by"`
}

// Reassign обрабатывает POST /pullRequest/reassign
func (h *PullRequestHandler) Reassign(w http.ResponseWriter, r *http.Request) {
	var req ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.PullRequestID == "" || req.OldUserID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id and old_user_id are required")
		return
	}

	// Переназначаем ревьювера
	pr, newReviewerID, err := h.prService.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithJSON(w, r, http.StatusOK, ReassignResponse{
		PR:         pr,
		ReplacedBy: newReviewerID,
	})
}
