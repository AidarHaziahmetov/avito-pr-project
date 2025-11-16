package postgres

import (
	"context"
	"errors"

	"github.com/aidar/avito-pr-project/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TeamRepository реализует repository.TeamRepository для PostgreSQL
type TeamRepository struct {
	db *pgxpool.Pool
}

// NewTeamRepository создает новый экземпляр TeamRepository
func NewTeamRepository(db *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{db: db}
}

// Create создает новую команду
func (r *TeamRepository) Create(ctx context.Context, teamName string) error {
	query := `INSERT INTO teams (team_name) VALUES ($1)`

	_, err := r.db.Exec(ctx, query, teamName)
	if err != nil {
		// Check for unique constraint violation (team already exists)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return domain.ErrTeamExists
		}
		return err
	}

	return nil
}

// GetByName получает команду со всеми участниками
func (r *TeamRepository) GetByName(ctx context.Context, teamName string) (*domain.Team, error) {
	// First check if team exists
	exists, err := r.Exists(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrTeamNotFound
	}

	// Get all team members
	query := `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY user_id
	`

	rows, err := r.db.Query(ctx, query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.TeamMember
	for rows.Next() {
		var member domain.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	team := &domain.Team{
		TeamName: teamName,
		Members:  members,
	}

	return team, nil
}

// Exists проверяет существование команды
func (r *TeamRepository) Exists(ctx context.Context, teamName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, teamName).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return exists, nil
}
