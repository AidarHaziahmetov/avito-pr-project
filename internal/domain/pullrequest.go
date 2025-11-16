package domain

import "time"

// PullRequestStatus представляет статус pull request'а
type PullRequestStatus string

// Возможные статусы pull request'а
const (
	StatusOpen   PullRequestStatus = "OPEN"   // PR открыт и может быть изменен
	StatusMerged PullRequestStatus = "MERGED" // PR смержен и не может быть изменен
)

// PullRequest представляет pull request с назначенными ревьюверами
type PullRequest struct {
	PullRequestID     string            `json:"pull_request_id"`
	PullRequestName   string            `json:"pull_request_name"`
	AuthorID          string            `json:"author_id"`
	Status            PullRequestStatus `json:"status"`
	AssignedReviewers []string          `json:"assigned_reviewers"` // До 2 ревьюверов
	CreatedAt         *time.Time        `json:"createdAt,omitempty"`
	MergedAt          *time.Time        `json:"mergedAt,omitempty"`
}

// PullRequestShort представляет сокращенную информацию о PR (используется в списках)
type PullRequestShort struct {
	PullRequestID   string            `json:"pull_request_id"`
	PullRequestName string            `json:"pull_request_name"`
	AuthorID        string            `json:"author_id"`
	Status          PullRequestStatus `json:"status"`
}

// IsMerged возвращает true если PR находится в статусе MERGED
func (pr *PullRequest) IsMerged() bool {
	return pr.Status == StatusMerged
}

// IsReviewerAssigned проверяет, назначен ли пользователь ревьювером этого PR
func (pr *PullRequest) IsReviewerAssigned(userID string) bool {
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == userID {
			return true
		}
	}
	return false
}
