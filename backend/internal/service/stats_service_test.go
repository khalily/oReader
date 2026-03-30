package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStatsRepository is a mock implementation of StatsRepository for testing
type mockStatsRepository struct {
	GetUserStatsFunc func(ctx context.Context, userID string) (*UserStats, error)
}

func (m *mockStatsRepository) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	if m.GetUserStatsFunc != nil {
		return m.GetUserStatsFunc(ctx, userID)
	}
	return &UserStats{}, nil
}

func TestStatsService_GetUserStats(t *testing.T) {
	ctx := context.Background()

	t.Run("returns stats from repository", func(t *testing.T) {
		expectedStats := &UserStats{
			Total:   100,
			Unread:  25,
			Starred: 10,
			Today:   5,
		}

		mockRepo := &mockStatsRepository{
			GetUserStatsFunc: func(ctx context.Context, userID string) (*UserStats, error) {
				assert.Equal(t, "user-123", userID)
				return expectedStats, nil
			},
		}

		service := NewStatsService(mockRepo)
		stats, err := service.GetUserStats(ctx, "user-123")

		require.NoError(t, err)
		assert.Equal(t, expectedStats, stats)
	})

	t.Run("returns zero stats for new user", func(t *testing.T) {
		mockRepo := &mockStatsRepository{
			GetUserStatsFunc: func(ctx context.Context, userID string) (*UserStats, error) {
				return &UserStats{
					Total:   0,
					Unread:  0,
					Starred: 0,
					Today:   0,
				}, nil
			},
		}

		service := NewStatsService(mockRepo)
		stats, err := service.GetUserStats(ctx, "new-user")

		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Total)
		assert.Equal(t, int64(0), stats.Unread)
		assert.Equal(t, int64(0), stats.Starred)
		assert.Equal(t, int64(0), stats.Today)
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		mockRepo := &mockStatsRepository{
			GetUserStatsFunc: func(ctx context.Context, userID string) (*UserStats, error) {
				return nil, assert.AnError
			},
		}

		service := NewStatsService(mockRepo)
		stats, err := service.GetUserStats(ctx, "user-123")

		require.Error(t, err)
		assert.Nil(t, stats)
	})
}
