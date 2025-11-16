package service

import (
	"context"

	"github.com/aidar/avito-pr-project/internal/domain"
	"github.com/aidar/avito-pr-project/internal/repository"
)

const maxReviewers = 2

// PullRequestService handles business logic for pull requests
type PullRequestService struct {
	prRepo           repository.PullRequestRepository
	userRepo         repository.UserRepository
	reviewerSelector *ReviewerSelector
}

// NewPullRequestService creates a new PullRequestService
func NewPullRequestService(
	prRepo repository.PullRequestRepository,
	userRepo repository.UserRepository,
	reviewerSelector *ReviewerSelector,
) *PullRequestService {
	return &PullRequestService{
		prRepo:           prRepo,
		userRepo:         userRepo,
		reviewerSelector: reviewerSelector,
	}
}

// CreatePR creates a new PR and automatically assigns up to 2 reviewers from author's team
func (s *PullRequestService) CreatePR(ctx context.Context, prID, prName, authorID string) (*domain.PullRequest, error) {
	// Check if PR already exists
	exists, err := s.prRepo.Exists(ctx, prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrPRExists
	}

	// Get author to find their team
	author, err := s.userRepo.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}

	// Get active team members excluding author
	candidates, err := s.userRepo.GetActiveTeamMembers(ctx, author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	// Select up to 2 reviewers
	reviewers := s.reviewerSelector.SelectReviewers(candidates, maxReviewers)

	// Create PR
	pr := &domain.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            domain.StatusOpen,
		AssignedReviewers: reviewers,
	}

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return nil, err
	}

	// Return the created PR
	return s.prRepo.GetByID(ctx, prID)
}

// MergePR marks a PR as merged (idempotent operation)
func (s *PullRequestService) MergePR(ctx context.Context, prID string) (*domain.PullRequest, error) {
	return s.prRepo.Merge(ctx, prID)
}

// ReassignReviewer replaces old reviewer with a new one from the old reviewer's team
func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*domain.PullRequest, string, error) {
	// Get PR
	pr, err := s.prRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	// Check if PR is merged
	if pr.IsMerged() {
		return nil, "", domain.ErrPRMerged
	}

	// Check if old reviewer is assigned
	if !pr.IsReviewerAssigned(oldReviewerID) {
		return nil, "", domain.ErrNotAssigned
	}

	// Get old reviewer to find their team
	oldReviewer, err := s.userRepo.GetByID(ctx, oldReviewerID)
	if err != nil {
		return nil, "", err
	}

	// Get active team members from old reviewer's team
	// Don't exclude anyone initially - we'll filter in SelectReplacement
	candidates, err := s.userRepo.GetActiveTeamMembers(ctx, oldReviewer.TeamName, "")
	if err != nil {
		return nil, "", err
	}

	// Select a replacement (excluding current reviewers)
	newReviewerID, err := s.reviewerSelector.SelectReplacement(candidates, pr.AssignedReviewers)
	if err != nil {
		return nil, "", err
	}

	// Update reviewers
	if err := s.prRepo.UpdateReviewers(ctx, prID, oldReviewerID, newReviewerID); err != nil {
		return nil, "", err
	}

	// Return updated PR
	updatedPR, errGet := s.prRepo.GetByID(ctx, prID)
	if errGet != nil {
		return nil, "", errGet
	}

	return updatedPR, newReviewerID, nil
}

// GetPRsByReviewer returns all PRs where user is assigned as reviewer
func (s *PullRequestService) GetPRsByReviewer(ctx context.Context, userID string) ([]*domain.PullRequestShort, error) {
	return s.prRepo.GetByReviewer(ctx, userID)
}

// GetByID retrieves a PR by ID
func (s *PullRequestService) GetByID(ctx context.Context, prID string) (*domain.PullRequest, error) {
	return s.prRepo.GetByID(ctx, prID)
}
