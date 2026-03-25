package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"oreader/internal/infra/logger"
	"oreader/internal/model"
	"oreader/internal/service"
)

// userFeedRepository implements UserFeedRepository interface
type userFeedRepository struct {
	db *gorm.DB
}

// NewUserFeedRepository creates a new user feed repository
func NewUserFeedRepository(db *gorm.DB) service.UserFeedRepository {
	return &userFeedRepository{db: db}
}

// Create creates a new user-feed relationship
func (r *userFeedRepository) Create(ctx context.Context, userFeed *model.UserFeed) error {
	return r.db.WithContext(ctx).Create(userFeed).Error
}

// GetByUserAndFeed retrieves a user-feed relationship by user and feed IDs
func (r *userFeedRepository) GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	logger.Debug().
		Str("user_id", userID).
		Str("feed_id", feedID).
		Msg("Querying user-feed relationship")

	var userFeed model.UserFeed
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		First(&userFeed).Error
	if err != nil {
		logger.Debug().
			Err(err).
			Str("user_id", userID).
			Str("feed_id", feedID).
			Msg("User-feed relationship not found")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user feed not found")
		}
		return nil, err
	}
	return &userFeed, nil
}

// GetByUserAndFeedIncludingDeleted retrieves a user-feed relationship including soft-deleted ones
func (r *userFeedRepository) GetByUserAndFeedIncludingDeleted(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	var userFeed model.UserFeed
	err := r.db.WithContext(ctx).
		Unscoped().
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		First(&userFeed).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user feed not found")
		}
		return nil, err
	}
	return &userFeed, nil
}

// ListByUserID retrieves all user-feed relationships for a user
func (r *userFeedRepository) ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error) {
	var userFeeds []*model.UserFeed
	err := r.db.WithContext(ctx).
		Preload("Feed").
		Where("user_id = ?", userID).
		Order("position ASC").
		Find(&userFeeds).Error
	if err != nil {
		return nil, err
	}
	return userFeeds, nil
}

// Delete deletes a user-feed relationship
func (r *userFeedRepository) Delete(ctx context.Context, userID, feedID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Delete(&model.UserFeed{}).Error
}

// GetMaxPosition gets the maximum position for a user's feeds
func (r *userFeedRepository) GetMaxPosition(ctx context.Context, userID string) (int, error) {
	var maxPosition int
	err := r.db.WithContext(ctx).
		Model(&model.UserFeed{}).
		Where("user_id = ?", userID).
		Select("COALESCE(MAX(position), -1)").
		Scan(&maxPosition).Error
	return maxPosition, err
}
