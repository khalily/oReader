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
	offsetQuery := baseQuery.
		Preload("Feed").
		Order("created_at DESC").
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
			result[i] = &service.ItemWithState{
				Item: item,
			}
			if state, ok := stateMap[item.ID]; ok {
				result[i].IsStarred = state.IsStarred
				result[i].IsRead = state.IsRead
				if state.ReadAt != nil {
					readAt := state.ReadAt.Format(time.RFC3339)
					result[i].ReadAt = &readAt
				}
			}
		}

		return result, total, nil
	}

	// No user ID - return items without state
	result := make([]*service.ItemWithState, len(items))
	for i, item := range items {
		result[i] = &service.ItemWithState{
			Item: item,
		}
	}

	return result, total, nil
}

// ListStarred retrieves starred items for a user
func (r *itemRepository) ListStarred(ctx context.Context, userID string, opts service.ListOptions) ([]*service.ItemWithState, int64, error) {
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).
		Model(&model.UserItemState{}).
		Where("user_id = ? AND is_starred = ?", userID, true).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get states with items
	query := r.db.WithContext(ctx).
		Joins("JOIN items ON items.id = user_item_states.item_id").
		Where("user_item_states.user_id = ? AND user_item_states.is_starred = ?", userID, true).
		Order("user_item_states.created_at DESC").
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
		result[i] = &service.ItemWithState{
			Item:      state.Item,
			IsStarred: true,
			IsRead:    state.IsRead,
		}
		if state.ReadAt != nil {
			readAt := state.ReadAt.Format(time.RFC3339)
			result[i].ReadAt = &readAt
		}
	}

	return result, total, nil
}

// ListUnread retrieves unread items for a user
func (r *itemRepository) ListUnread(ctx context.Context, userID string, opts service.ListOptions) ([]*service.ItemWithState, int64, error) {
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).
		Model(&model.UserItemState{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Also count items without any state (unread by default)
	var itemsWithoutState int64
	if err := r.db.WithContext(ctx).
		Model(&model.Item{}).
		Where("id NOT IN (?)",
			r.db.WithContext(ctx).
				Model(&model.UserItemState{}).
				Select("item_id").
				Where("user_id = ?", userID)).
		Count(&itemsWithoutState).Error; err != nil {
		return nil, 0, err
	}

	total += itemsWithoutState

	// Get unread items with states first
	query := r.db.WithContext(ctx).
		Joins("JOIN items ON items.id = user_item_states.item_id").
		Where("user_item_states.user_id = ? AND user_item_states.is_read = ?", userID, false).
		Order("items.created_at DESC").
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
		result = append(result, &service.ItemWithState{
			Item:      state.Item,
			IsStarred: state.IsStarred,
			IsRead:    false,
		})
	}

	// If we need more items, get items without state
	remaining := opts.Limit - len(result)
	if opts.Limit > 0 && remaining > 0 {
		var items []*model.Item
		itemQuery := r.db.WithContext(ctx).
			Preload("Feed").
			Where("id NOT IN (?)",
				r.db.WithContext(ctx).
					Model(&model.UserItemState{}).
					Select("item_id").
					Where("user_id = ?", userID)).
			Order("created_at DESC").
			Limit(remaining)

		if err := itemQuery.Find(&items).Error; err != nil {
			return nil, 0, err
		}

		for _, item := range items {
			result = append(result, &service.ItemWithState{
				Item:   item,
				IsRead: false,
			})
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
