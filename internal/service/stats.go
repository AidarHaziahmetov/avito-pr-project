package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserStats represents statistics for a user
type UserStats struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	ReviewAssignments int    `json:"review_assignments"`
	AuthoredPRs       int    `json:"authored_prs"`
	ActiveReviews     int    `json:"active_reviews"`
}

// PRStats represents overall PR statistics
type PRStats struct {
	TotalPRs       int `json:"total_prs"`
	OpenPRs        int `json:"open_prs"`
	MergedPRs      int `json:"merged_prs"`
	TotalReviewers int `json:"total_reviewers"`
}

// Stats represents combined statistics
type Stats struct {
	UserStats []UserStats `json:"user_stats"`
	PRStats   PRStats     `json:"pr_stats"`
}

// StatsService handles statistics queries
type StatsService struct {
	db *pgxpool.Pool
}

// NewStatsService creates a new StatsService
func NewStatsService(db *pgxpool.Pool) *StatsService {
	return &StatsService{db: db}
}

// GetStats returns overall statistics
func (s *StatsService) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}

	// Get user statistics
	userQuery := `
		SELECT 
			u.user_id,
			u.username,
			COUNT(DISTINCT prr.pull_request_id) as review_assignments,
			COUNT(DISTINCT pr.pull_request_id) as authored_prs,
			COUNT(DISTINCT CASE WHEN pr2.status = 'OPEN' THEN prr.pull_request_id END) as active_reviews
		FROM users u
		LEFT JOIN pr_reviewers prr ON u.user_id = prr.user_id
		LEFT JOIN pull_requests pr ON u.user_id = pr.author_id
		LEFT JOIN pr_reviewers prr2 ON u.user_id = prr2.user_id
		LEFT JOIN pull_requests pr2 ON prr2.pull_request_id = pr2.pull_request_id
		GROUP BY u.user_id, u.username
		ORDER BY review_assignments DESC, u.user_id
	`

	rows, err := s.db.Query(ctx, userQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var us UserStats
		if err := rows.Scan(&us.UserID, &us.Username, &us.ReviewAssignments, &us.AuthoredPRs, &us.ActiveReviews); err != nil {
			return nil, err
		}
		stats.UserStats = append(stats.UserStats, us)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get PR statistics
	prQuery := `
		SELECT 
			COUNT(*) as total_prs,
			COUNT(CASE WHEN status = 'OPEN' THEN 1 END) as open_prs,
			COUNT(CASE WHEN status = 'MERGED' THEN 1 END) as merged_prs,
			(SELECT COUNT(*) FROM pr_reviewers) as total_reviewers
		FROM pull_requests
	`

	if err := s.db.QueryRow(ctx, prQuery).Scan(
		&stats.PRStats.TotalPRs,
		&stats.PRStats.OpenPRs,
		&stats.PRStats.MergedPRs,
		&stats.PRStats.TotalReviewers,
	); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetUserStats returns statistics for a specific user
func (s *StatsService) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	query := `
		SELECT 
			u.user_id,
			u.username,
			COUNT(DISTINCT prr.pull_request_id) as review_assignments,
			COUNT(DISTINCT pr.pull_request_id) as authored_prs,
			COUNT(DISTINCT CASE WHEN pr2.status = 'OPEN' THEN prr.pull_request_id END) as active_reviews
		FROM users u
		LEFT JOIN pr_reviewers prr ON u.user_id = prr.user_id
		LEFT JOIN pull_requests pr ON u.user_id = pr.author_id
		LEFT JOIN pr_reviewers prr2 ON u.user_id = prr2.user_id
		LEFT JOIN pull_requests pr2 ON prr2.pull_request_id = pr2.pull_request_id
		WHERE u.user_id = $1
		GROUP BY u.user_id, u.username
	`

	var stats UserStats
	err := s.db.QueryRow(ctx, query, userID).Scan(
		&stats.UserID,
		&stats.Username,
		&stats.ReviewAssignments,
		&stats.AuthoredPRs,
		&stats.ActiveReviews,
	)

	return &stats, err
}
