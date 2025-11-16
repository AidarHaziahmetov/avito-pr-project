package repository

import (
	"context"

	"github.com/aidar/avito-pr-project/internal/domain"
)

// UserRepository определяет методы для работы с данными пользователей
type UserRepository interface {
	// CreateOrUpdate создает нового пользователя или обновляет существующего
	CreateOrUpdate(ctx context.Context, user *domain.User) error

	// GetByID получает пользователя по ID
	GetByID(ctx context.Context, userID string) (*domain.User, error)

	// SetIsActive обновляет статус активности пользователя
	SetIsActive(ctx context.Context, userID string, isActive bool) error

	// GetActiveTeamMembers возвращает всех активных пользователей команды, исключая указанного
	GetActiveTeamMembers(ctx context.Context, teamName, excludeUserID string) ([]*domain.User, error)

	// GetTeamMembers возвращает всех пользователей команды
	GetTeamMembers(ctx context.Context, teamName string) ([]*domain.User, error)
}

// TeamRepository определяет методы для работы с данными команд
type TeamRepository interface {
	// Create создает новую команду
	Create(ctx context.Context, teamName string) error

	// GetByName получает команду со всеми участниками
	GetByName(ctx context.Context, teamName string) (*domain.Team, error)

	// Exists проверяет существование команды
	Exists(ctx context.Context, teamName string) (bool, error)
}

// PullRequestRepository определяет методы для работы с данными pull request'ов
type PullRequestRepository interface {
	// Create создает новый pull request с назначенными ревьюверами
	Create(ctx context.Context, pr *domain.PullRequest) error

	// GetByID получает pull request по ID
	GetByID(ctx context.Context, prID string) (*domain.PullRequest, error)

	// Merge помечает pull request как смерженный (идемпотентная операция)
	Merge(ctx context.Context, prID string) (*domain.PullRequest, error)

	// UpdateReviewers заменяет старого ревьювера на нового
	UpdateReviewers(ctx context.Context, prID, oldReviewerID, newReviewerID string) error

	// GetByReviewer возвращает все PR где пользователь назначен ревьювером
	GetByReviewer(ctx context.Context, userID string) ([]*domain.PullRequestShort, error)

	// Exists проверяет существование PR
	Exists(ctx context.Context, prID string) (bool, error)
}
