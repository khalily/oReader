package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// statsRepository implements StatsRepository interface
type statsRepository struct {
	db *gorm.DB
}

// NewStatsRepository creates a new stats repository
func NewStatsRepository(db *gorm.DB) service.StatsRepository {
	return &statsRepository{db: db}
}

// GetUserStats returns article statistics for a user
func (r *statsRepository) GetUserStats(ctx context.Context, userID string) (*service.UserStats, error) {
	stats := &service.UserStats{}

	// Get user's feed IDs
	var feedIDs []string
	if err := r.db.WithContext(ctx).
		Model(&model.UserFeed{}).
		Where("user_id = ?", userID).
		Pluck("feed_id", &feedIDs).Error; err != nil {
		return nil, err
	}

	// If user has no feeds, return zero stats
	if len(feedIDs) == 0 {
		return stats, nil
	}

	// Total: count all items in user's subscribed feeds
	if err := r.db.WithContext(ctx).
		Model(&model.Item{}).
		Where("feed_id IN ?", feedIDs).
		Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Starred: count items where is_starred = true
	if err := r.db.WithContext(ctx).
		Model(&model.UserItemState{}).
		Where("user_id = ? AND is_starred = ?", userID, true).
		Count(&stats.Starred).Error; err != nil {
		return nil, err
	}

	// Read: count items where is_read = true
	var readCount int64
	if err := r.db.WithContext(ctx).
		Model(&model.UserItemState{}).
		Where("user_id = ? AND is_read = ?", userID, true).
		Count(&readCount).Error; err != nil {
		return nil, err
	}

	// Unread = Total - Read
	stats.Unread = stats.Total - readCount

	// Today: count items published today (UTC)
	todayStart := time.Now().UTC().Truncate(24 * time.Hour)
	if err := r.db.WithContext(ctx).
		Model(&model.Item{}).
		Where("feed_id IN ? AND pub_date >= ?", feedIDs, todayStart).
		Count(&stats.Today).Error; err != nil {
		return nil, err
	}

	return stats, nil
}
