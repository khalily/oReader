package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
)

// itemRepository implements ItemRepository interface
type itemRepository struct {
	db *gorm.DB
}

// NewItemRepository creates a new item repository
func NewItemRepository(db *gorm.DB) service.ItemRepository {
	return &itemRepository{db: db}
}

// buildFeedResponse creates a service.FeedResponse from a model.Feed
func buildFeedResponse(feed *model.Feed) *service.FeedResponse {
	if feed == nil {
		return nil
	}
	result := &service.FeedResponse{
		ID:          feed.ID,
		Title:       feed.Title,
		FeedURL:     feed.FeedURL,
		Description: feed.Description,
	}
	// Only set ImageURL if it's not empty
	if feed.ImageURL != "" {
		result.ImageURL = &feed.ImageURL
	}
	return result
}

// buildUserItemStateResponse creates a service.UserItemStateResponse from a model.UserItemState
func buildUserItemStateResponse(state *model.UserItemState) *service.UserItemStateResponse {
	if state == nil {
		return nil
	}
	result := &service.UserItemStateResponse{
		ItemID:    state.ItemID,
		IsStarred: state.IsStarred,
		IsRead:    state.IsRead,
	}
	if state.ReadAt != nil {
		readAt := state.ReadAt.Format(time.RFC3339)
		result.ReadAt = &readAt
	}
	return result
}

// buildItemWithState creates an ItemWithState from an item and optional state
func buildItemWithState(item *model.Item, state *model.UserItemState) *service.ItemWithState {
	result := &service.ItemWithState{
		Item:      item,
		Feed:      buildFeedResponse(item.Feed),
		UserState: buildUserItemStateResponse(state),
	}

	return result
}

// Create creates a new item
func (r *itemRepository) Create(ctx context.Context, item *model.Item) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// CreateBatch creates multiple items in a single transaction
func (r *itemRepository) CreateBatch(ctx context.Context, items []*model.Item) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

// GetByID retrieves an item by ID
func (r *itemRepository) GetByID(ctx context.Context, id string) (*model.Item, error) {
	var item model.Item
	err := r.db.WithContext(ctx).Preload("Feed").Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("item not found")
		}
		return nil, err
	}
	return &item, nil
}

// Update updates an existing item
func (r *itemRepository) Update(ctx context.Context, item *model.Item) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// GetByGUID retrieves an item by feed ID and GUID

// GetByGUID retrieves an item by feed ID and GUID
func (r *itemRepository) GetByGUID(ctx context.Context, feedID, guid string) (*model.Item, error) {
	var item model.Item
	err := r.db.WithContext(ctx).
		Where("feed_id = ? AND guid = ?", feedID, guid).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("item not found")
		}
		return nil, err
	}
	return &item, nil
}

// ListByFeedID retrieves items for a feed with user state
func (r *itemRepository) ListByFeedID(ctx context.Context, feedID, userID string, opts service.ListOptions) ([]*service.ItemWithState, int64, error) {
	var items []*model.Item
	var total int64

	// Build query with user state
	baseQuery := r.db.WithContext(ctx).
		Model(&model.Item{}).
		Where("feed_id = ?", feedID)

	// Count total
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get items with pagination (preload Feed for frontend display)
	// Sort by pub_date DESC NULLS LAST (spec-compliant sorting)
	offsetQuery := baseQuery.
		Preload("Feed").
		Order("pub_date DESC NULLS LAST").
		Offset(opts.Offset)

	if opts.Limit > 0 {
		offsetQuery = offsetQuery.Limit(opts.Limit)
	}

	if err := offsetQuery.Find(&items).Error; err != nil {
		return nil, 0, err
	}

	// Get user states for these items
	if len(items) > 0 && userID != "" {
		itemIDs := make([]string, len(items))
		for i, item := range items {
			itemIDs[i] = item.ID
		}

		var states []model.UserItemState
		if err := r.db.WithContext(ctx).
			Where("user_id = ? AND item_id IN ?", userID, itemIDs).
			Find(&states).Error; err != nil {
			return nil, 0, err
		}

		// Create state map
		stateMap := make(map[string]*model.UserItemState)
		for i := range states {
			stateMap[states[i].ItemID] = &states[i]
		}

		// Build result with state
		result := make([]*service.ItemWithState, len(items))
		for i, item := range items {
			state := stateMap[item.ID]
			result[i] = buildItemWithState(item, state)
		}

		return result, total, nil
	}

	// No user ID - return items without state
	result := make([]*service.ItemWithState, len(items))
	for i, item := range items {
		result[i] = buildItemWithState(item, nil)
	}

	return result, total, nil
}

// ListStarred retrieves starred items for a user
func (r *itemRepository) ListStarred(ctx context.Context, userID string, opts service.ListOptions) ([]*service.ItemWithState, int64, error) {
	var total int64

	// Count total - only count items from feeds the user is subscribed to
	// Note: explicit deleted_at IS NULL check because raw JOINs don't apply GORM soft delete filter
	if err := r.db.WithContext(ctx).
		Model(&model.UserItemState{}).
		Joins("JOIN items ON items.id = user_item_states.item_id").
		Joins("JOIN user_feeds ON user_feeds.feed_id = items.feed_id AND user_feeds.user_id = ? AND user_feeds.deleted_at IS NULL", userID).
		Where("user_item_states.user_id = ? AND user_item_states.is_starred = ?", userID, true).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get states with items, sorted by pub_date DESC NULLS LAST
	// Only get items from feeds the user is subscribed to
	query := r.db.WithContext(ctx).
		Joins("JOIN items ON items.id = user_item_states.item_id").
		Joins("JOIN user_feeds ON user_feeds.feed_id = items.feed_id AND user_feeds.user_id = ? AND user_feeds.deleted_at IS NULL", userID).
		Where("user_item_states.user_id = ? AND user_item_states.is_starred = ?", userID, true).
		Order("items.pub_date DESC NULLS LAST").
		Offset(opts.Offset)

	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}

	var states []model.UserItemState
	if err := query.Preload("Item.Feed").Find(&states).Error; err != nil {
		return nil, 0, err
	}

	// Build result
	result := make([]*service.ItemWithState, len(states))
	for i, state := range states {
		result[i] = buildItemWithState(state.Item, &state)
	}

	return result, total, nil
}

// ListUnread retrieves unread items for a user
func (r *itemRepository) ListUnread(ctx context.Context, userID string, opts service.ListOptions) ([]*service.ItemWithState, int64, error) {
	var total int64

	// Count items with is_read=false state, only from subscribed feeds
	if err := r.db.WithContext(ctx).
		Model(&model.UserItemState{}).
		Joins("JOIN items ON items.id = user_item_states.item_id").
		Joins("JOIN user_feeds ON user_feeds.feed_id = items.feed_id AND user_feeds.user_id = ? AND user_feeds.deleted_at IS NULL", userID).
		Where("user_item_states.user_id = ? AND user_item_states.is_read = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Also count items without any state (unread by default)
	// Only count items from feeds the user is subscribed to
	var itemsWithoutState int64
	if err := r.db.WithContext(ctx).
		Model(&model.Item{}).
		Where("feed_id IN (?)",
			r.db.WithContext(ctx).
				Model(&model.UserFeed{}).
				Select("feed_id").
				Where("user_id = ?", userID)).
		Where("id NOT IN (?)",
			r.db.WithContext(ctx).
				Model(&model.UserItemState{}).
				Select("item_id").
				Where("user_id = ?", userID)).
		Count(&itemsWithoutState).Error; err != nil {
		return nil, 0, err
	}

	total += itemsWithoutState

	// Get unread items with states first (sorted by pub_date DESC NULLS LAST)
	// Only get items from feeds the user is subscribed to
	query := r.db.WithContext(ctx).
		Joins("JOIN items ON items.id = user_item_states.item_id").
		Joins("JOIN user_feeds ON user_feeds.feed_id = items.feed_id AND user_feeds.user_id = ? AND user_feeds.deleted_at IS NULL", userID).
		Where("user_item_states.user_id = ? AND user_item_states.is_read = ?", userID, false).
		Order("items.pub_date DESC NULLS LAST").
		Offset(opts.Offset)

	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}

	var states []model.UserItemState
	if err := query.Preload("Item.Feed").Find(&states).Error; err != nil {
		return nil, 0, err
	}

	// Build result from states
	result := make([]*service.ItemWithState, 0, len(states))
	for _, state := range states {
		result = append(result, buildItemWithState(state.Item, &state))
	}

	// If we need more items, get items without state
	// Only get items from feeds the user is subscribed to
	remaining := opts.Limit - len(result)
	if opts.Limit > 0 && remaining > 0 {
		var items []*model.Item
		itemQuery := r.db.WithContext(ctx).
			Preload("Feed").
			Where("feed_id IN (?)",
				r.db.WithContext(ctx).
					Model(&model.UserFeed{}).
					Select("feed_id").
					Where("user_id = ?", userID)).
			Where("id NOT IN (?)",
				r.db.WithContext(ctx).
					Model(&model.UserItemState{}).
					Select("item_id").
					Where("user_id = ?", userID)).
			Order("pub_date DESC NULLS LAST").
			Limit(remaining)

		if err := itemQuery.Find(&items).Error; err != nil {
			return nil, 0, err
		}

		for _, item := range items {
			result = append(result, buildItemWithState(item, nil))
		}
	}

	return result, total, nil
}

// CountByFeedID counts items for a feed
func (r *itemRepository) CountByFeedID(ctx context.Context, feedID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Item{}).
		Where("feed_id = ?", feedID).
		Count(&count).Error
	return count, err
}
