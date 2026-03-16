package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
)

// userItemStateRepository implements UserItemStateRepository interface
type userItemStateRepository struct {
	db *gorm.DB
}

// NewUserItemStateRepository creates a new user item state repository
func NewUserItemStateRepository(db *gorm.DB) service.UserItemStateRepository {
	return &userItemStateRepository{db: db}
}

// Create creates a new user item state
func (r *userItemStateRepository) Create(ctx context.Context, state *model.UserItemState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

// GetByUserAndItem retrieves a user item state by user and item IDs
func (r *userItemStateRepository) GetByUserAndItem(ctx context.Context, userID, itemID string) (*model.UserItemState, error) {
	var state model.UserItemState
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		First(&state).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user item state not found")
		}
		return nil, err
	}
	return &state, nil
}

// Update updates a user item state
func (r *userItemStateRepository) Update(ctx context.Context, state *model.UserItemState) error {
	return r.db.WithContext(ctx).Save(state).Error
}

// Upsert creates or updates a user item state
func (r *userItemStateRepository) Upsert(ctx context.Context, state *model.UserItemState) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND item_id = ?", state.UserID, state.ItemID).
		Assign(state).
		FirstOrCreate(state).Error
}

// BulkMarkRead marks multiple items as read for a user
func (r *userItemStateRepository) BulkMarkRead(ctx context.Context, userID string, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Use a transaction to update existing states and create new ones
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update existing states
		if err := tx.Model(&model.UserItemState{}).
			Where("user_id = ? AND item_id IN ?", userID, itemIDs).
			Updates(map[string]interface{}{
				"is_read": true,
			}).Error; err != nil {
			return err
		}

		// Find items that don't have a state yet
		var existingItemIDs []string
		if err := tx.Model(&model.UserItemState{}).
			Where("user_id = ? AND item_id IN ?", userID, itemIDs).
			Pluck("item_id", &existingItemIDs).Error; err != nil {
			return err
		}

		// Create states for items without one
		itemIDMap := make(map[string]bool)
		for _, id := range existingItemIDs {
			itemIDMap[id] = true
		}

		newStates := make([]*model.UserItemState, 0)
		for _, itemID := range itemIDs {
			if !itemIDMap[itemID] {
				newState := &model.UserItemState{
					UserID:  userID,
					ItemID:  itemID,
					IsRead:  true,
				}
				if err := newState.GenerateID(); err != nil {
					return err
				}
				newStates = append(newStates, newState)
			}
		}

		if len(newStates) > 0 {
			if err := tx.Create(&newStates).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// MarkAllRead marks all items for a feed as read for a user
func (r *userItemStateRepository) MarkAllRead(ctx context.Context, userID, feedID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update existing states for this feed's items
		subQuery := tx.Model(&model.Item{}).
			Select("id").
			Where("feed_id = ?", feedID)

		if err := tx.Model(&model.UserItemState{}).
			Where("user_id = ? AND item_id IN (?)", userID, subQuery).
			Updates(map[string]interface{}{
				"is_read": true,
			}).Error; err != nil {
			return err
		}

		// Find items without state
		var itemsWithoutState []*model.Item
		if err := tx.Where("feed_id = ? AND id NOT IN (?)",
			feedID,
			tx.Model(&model.UserItemState{}).
				Select("item_id").
				Where("user_id = ?", userID)).
			Find(&itemsWithoutState).Error; err != nil {
			return err
		}

		// Create states for items without one
		newStates := make([]*model.UserItemState, len(itemsWithoutState))
		for i, item := range itemsWithoutState {
			newStates[i] = &model.UserItemState{
				UserID:  userID,
				ItemID:  item.ID,
				IsRead:  true,
			}
			if err := newStates[i].GenerateID(); err != nil {
				return err
			}
		}

		if len(newStates) > 0 {
			if err := tx.Create(&newStates).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
