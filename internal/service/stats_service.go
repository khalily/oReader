package service

import (
	"context"
)

// statsService implements StatsService interface
type statsService struct {
	statsRepo StatsRepository
}

// NewStatsService creates a new stats service
func NewStatsService(statsRepo StatsRepository) StatsService {
	return &statsService{
		statsRepo: statsRepo,
	}
}

// GetUserStats returns article statistics for a user
func (s *statsService) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	return s.statsRepo.GetUserStats(ctx, userID)
}
