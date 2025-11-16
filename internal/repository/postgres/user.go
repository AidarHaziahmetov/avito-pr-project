package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aidar/avito-pr-project/internal/domain"
)

// UserRepository реализует repository.UserRepository для PostgreSQL
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository создает новый экземпляр UserRepository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// CreateOrUpdate создает нового пользователя или обновляет существующего
func (r *UserRepository) CreateOrUpdate(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET username = EXCLUDED.username,
		    team_name = EXCLUDED.team_name,
		    is_active = EXCLUDED.is_active,
		    updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query, user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

// GetByID получает пользователя по ID
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.UserID,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// SetIsActive обновляет статус активности пользователя
func (r *UserRepository) SetIsActive(ctx context.Context, userID string, isActive bool) error {
	query := `
		UPDATE users
		SET is_active = $1, updated_at = NOW()
		WHERE user_id = $2
	`

	result, err := r.db.Exec(ctx, query, isActive, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// GetActiveTeamMembers возвращает всех активных пользователей команды, исключая указанного
func (r *UserRepository) GetActiveTeamMembers(ctx context.Context, teamName, excludeUserID string) ([]*domain.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1 AND is_active = true AND user_id != $2
		ORDER BY user_id
	`

	rows, err := r.db.Query(ctx, query, teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

// GetTeamMembers возвращает всех пользователей команды
func (r *UserRepository) GetTeamMembers(ctx context.Context, teamName string) ([]*domain.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY user_id
	`

	rows, err := r.db.Query(ctx, query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}
