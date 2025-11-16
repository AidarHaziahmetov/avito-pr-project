package service

import (
	"context"

	"github.com/aidar/avito-pr-project/internal/domain"
	"github.com/aidar/avito-pr-project/internal/repository"
)

// TeamService handles business logic for teams
type TeamService struct {
	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
}

// NewTeamService creates a new TeamService
func NewTeamService(teamRepo repository.TeamRepository, userRepo repository.UserRepository) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// AddTeam creates a new team with members (creates/updates users)
func (s *TeamService) AddTeam(ctx context.Context, team *domain.Team) (*domain.Team, error) {
	// Check if team already exists
	exists, err := s.teamRepo.Exists(ctx, team.TeamName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrTeamExists
	}

	// Create team
	if err := s.teamRepo.Create(ctx, team.TeamName); err != nil {
		return nil, err
	}

	// Create or update members
	for _, member := range team.Members {
		user := &domain.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		if err := s.userRepo.CreateOrUpdate(ctx, user); err != nil {
			return nil, err
		}
	}

	// Return the created team
	return s.teamRepo.GetByName(ctx, team.TeamName)
}

// GetTeam retrieves a team with all members
func (s *TeamService) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	return s.teamRepo.GetByName(ctx, teamName)
}
