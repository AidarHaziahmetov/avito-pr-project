package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aidar/avito-pr-project/internal/domain"
)

// PullRequestRepository реализует repository.PullRequestRepository для PostgreSQL
type PullRequestRepository struct {
	db *pgxpool.Pool
}

// NewPullRequestRepository создает новый экземпляр PullRequestRepository
func NewPullRequestRepository(db *pgxpool.Pool) *PullRequestRepository {
	return &PullRequestRepository{db: db}
}

// Create создает новый pull request с назначенными ревьюверами
func (r *PullRequestRepository) Create(ctx context.Context, pr *domain.PullRequest) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx) // Ignore error as it will fail if transaction was committed
	}()

	// Insert PR
	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	createdAt := time.Now()
	_, err = tx.Exec(ctx, query, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return domain.ErrPRExists
			}
			if pgErr.Code == "23503" { // foreign_key_violation
				return domain.ErrUserNotFound
			}
		}
		return err
	}

	// Insert reviewers
	if len(pr.AssignedReviewers) > 0 {
		reviewerQuery := `
			INSERT INTO pr_reviewers (pull_request_id, user_id)
			VALUES ($1, $2)
		`
		for _, reviewerID := range pr.AssignedReviewers {
			_, err = tx.Exec(ctx, reviewerQuery, pr.PullRequestID, reviewerID)
			if err != nil {
				return err
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	pr.CreatedAt = &createdAt
	return nil
}

// GetByID получает pull request по ID
func (r *PullRequestRepository) GetByID(ctx context.Context, prID string) (*domain.PullRequest, error) {
	// Get PR basic info
	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`

	var pr domain.PullRequest
	err := r.db.QueryRow(ctx, query, prID).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPRNotFound
		}
		return nil, err
	}

	// Get assigned reviewers
	reviewersQuery := `
		SELECT user_id
		FROM pr_reviewers
		WHERE pull_request_id = $1
		ORDER BY assigned_at
	`

	rows, err := r.db.Query(ctx, reviewersQuery, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}

	pr.AssignedReviewers = reviewers

	return &pr, rows.Err()
}

// Merge помечает pull request как смерженный (идемпотентная операция)
func (r *PullRequestRepository) Merge(ctx context.Context, prID string) (*domain.PullRequest, error) {
	query := `
		UPDATE pull_requests
		SET status = $1, merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $2
		RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	`

	var pr domain.PullRequest
	err := r.db.QueryRow(ctx, query, domain.StatusMerged, prID).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPRNotFound
		}
		return nil, err
	}

	// Get assigned reviewers
	reviewersQuery := `
		SELECT user_id
		FROM pr_reviewers
		WHERE pull_request_id = $1
		ORDER BY assigned_at
	`

	rows, err := r.db.Query(ctx, reviewersQuery, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}

	pr.AssignedReviewers = reviewers

	return &pr, rows.Err()
}

// UpdateReviewers заменяет старого ревьювера на нового
func (r *PullRequestRepository) UpdateReviewers(ctx context.Context, prID, oldReviewerID, newReviewerID string) error {
	query := `
		UPDATE pr_reviewers
		SET user_id = $1, assigned_at = NOW()
		WHERE pull_request_id = $2 AND user_id = $3
	`

	result, err := r.db.Exec(ctx, query, newReviewerID, prID, oldReviewerID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotAssigned
	}

	return nil
}

// GetByReviewer возвращает все PR где пользователь назначен ревьювером
func (r *PullRequestRepository) GetByReviewer(ctx context.Context, userID string) ([]*domain.PullRequestShort, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, &pr)
	}

	// Return empty array instead of nil if no PRs found
	if prs == nil {
		prs = []*domain.PullRequestShort{}
	}

	return prs, rows.Err()
}

// Exists проверяет существование PR
func (r *PullRequestRepository) Exists(ctx context.Context, prID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, prID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return exists, nil
}
