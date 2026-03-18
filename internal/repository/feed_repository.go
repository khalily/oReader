package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
)

// feedRepository implements FeedRepository interface
type feedRepository struct {
	db *gorm.DB
}

// NewFeedRepository creates a new feed repository
func NewFeedRepository(db *gorm.DB) service.FeedRepository {
	return &feedRepository{db: db}
}

// Create creates a new feed
func (r *feedRepository) Create(ctx context.Context, feed *model.Feed) error {
	return r.db.WithContext(ctx).Create(feed).Error
}

// GetByID retrieves a feed by ID
func (r *feedRepository) GetByID(ctx context.Context, id string) (*model.Feed, error) {
	var feed model.Feed
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&feed).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("feed not found")
		}
		return nil, err
	}
	return &feed, nil
}

// GetByURL retrieves a feed by URL
func (r *feedRepository) GetByURL(ctx context.Context, url string) (*model.Feed, error) {
	var feed model.Feed
	err := r.db.WithContext(ctx).Where("feed_url = ?", url).First(&feed).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("feed not found")
		}
		return nil, err
	}
	return &feed, nil
}

// ListByUserID retrieves all feeds for a user
func (r *feedRepository) ListByUserID(ctx context.Context, userID string, opts service.ListOptions) ([]*model.Feed, int64, error) {
	var feeds []*model.Feed
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.Feed{}).
		Joins("JOIN user_feeds ON user_feeds.feed_id = feeds.id AND user_feeds.deleted_at IS NULL").
		Where("user_feeds.user_id = ?", userID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	offsetQuery := query.
		Order("user_feeds.position ASC").
		Offset(opts.Offset)

	if opts.Limit > 0 {
		offsetQuery = offsetQuery.Limit(opts.Limit)
	}

	if err := offsetQuery.Find(&feeds).Error; err != nil {
		return nil, 0, err
	}

	return feeds, total, nil
}

// Update updates a feed
func (r *feedRepository) Update(ctx context.Context, feed *model.Feed) error {
	return r.db.WithContext(ctx).Save(feed).Error
}

// Delete deletes a feed by ID
func (r *feedRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Feed{}, "id = ?", id).Error
}

// ListAll retrieves all feeds
func (r *feedRepository) ListAll(ctx context.Context) ([]*model.Feed, error) {
	var feeds []*model.Feed
	err := r.db.WithContext(ctx).Find(&feeds).Error
	if err != nil {
		return nil, err
	}
	return feeds, nil
}
