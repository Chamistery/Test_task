package service

import (
	"math/rand"
	"time"

	"github.com/Chamistery/Test_task/internal/storage"
)

type ReviewerService struct {
	storage storage.Storage
	rand    *rand.Rand
}

func NewReviewerService(storage storage.Storage) *ReviewerService {
	return &ReviewerService{
		storage: storage,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *ReviewerService) AssignReviewers(authorID string) ([]string, error) {
	teamName, err := s.storage.GetUserTeam(authorID)
	if err != nil {
		return nil, err
	}

	if teamName == "" {
		return []string{}, nil
	}

	candidates, err := s.storage.GetActiveCandidates(teamName, []string{authorID})
	if err != nil {
		return nil, err
	}

	s.rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	maxReviewers := 2
	if len(candidates) < maxReviewers {
		maxReviewers = len(candidates)
	}

	return candidates[:maxReviewers], nil
}

func (s *ReviewerService) FindReplacementReviewer(prID string, oldUserID string) (string, error) {
	teamName, err := s.storage.GetUserTeam(oldUserID)
	if err != nil {
		return "", err
	}

	pr, err := s.storage.GetPullRequest(prID)
	if err != nil || pr == nil {
		return "", err
	}

	excludeIDs := append(pr.AssignedReviewers, pr.AuthorID)

	candidates, err := s.storage.GetActiveCandidates(teamName, excludeIDs)
	if err != nil {
		return "", err
	}

	if len(candidates) == 0 {
		return "", nil
	}

	return candidates[s.rand.Intn(len(candidates))], nil
}
