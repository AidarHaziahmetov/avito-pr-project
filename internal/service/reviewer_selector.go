package service

import (
	"math/rand"
	"time"

	"github.com/aidar/avito-pr-project/internal/domain"
)

// ReviewerSelector handles the logic of selecting reviewers
type ReviewerSelector struct {
	rng *rand.Rand
}

// NewReviewerSelector creates a new ReviewerSelector with its own random source
func NewReviewerSelector() *ReviewerSelector {
	return &ReviewerSelector{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectReviewers randomly selects up to maxReviewers from candidates
// Returns the selected reviewer IDs
func (s *ReviewerSelector) SelectReviewers(candidates []*domain.User, maxReviewers int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	// If we have fewer candidates than needed, return all
	if len(candidates) <= maxReviewers {
		reviewers := make([]string, len(candidates))
		for i, c := range candidates {
			reviewers[i] = c.UserID
		}
		return reviewers
	}

	// Randomly shuffle candidates and take first maxReviewers
	shuffled := make([]*domain.User, len(candidates))
	copy(shuffled, candidates)
	s.rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	reviewers := make([]string, maxReviewers)
	for i := 0; i < maxReviewers; i++ {
		reviewers[i] = shuffled[i].UserID
	}

	return reviewers
}

// SelectReplacement randomly selects one replacement from candidates, excluding current reviewers
func (s *ReviewerSelector) SelectReplacement(candidates []*domain.User, currentReviewers []string) (string, error) {
	// Filter out users who are already assigned as reviewers
	available := make([]*domain.User, 0)
	for _, candidate := range candidates {
		isAlreadyAssigned := false
		for _, reviewerID := range currentReviewers {
			if candidate.UserID == reviewerID {
				isAlreadyAssigned = true
				break
			}
		}
		if !isAlreadyAssigned {
			available = append(available, candidate)
		}
	}

	if len(available) == 0 {
		return "", domain.ErrNoCandidate
	}

	// Randomly select one
	selected := available[s.rng.Intn(len(available))]
	return selected.UserID, nil
}
