package service

import (
	"context"
	"time"

	"oreader/internal/model"
)

// itemService implements ItemService interface
type itemService struct {
	itemRepo     ItemRepository
	stateRepo    UserItemStateRepository
	userFeedRepo UserFeedRepository
}

// NewItemService creates a new item service
func NewItemService(
	itemRepo ItemRepository,
	stateRepo UserItemStateRepository,
	userFeedRepo UserFeedRepository,
) ItemService {
	return &itemService{
		itemRepo:     itemRepo,
		stateRepo:    stateRepo,
		userFeedRepo: userFeedRepo,
	}
}

// ListItems retrieves items for a user with filtering and pagination
func (s *itemService) ListItems(ctx context.Context, userID string, opts ListItemOptions) (*ItemListResult, error) {
	var items []*ItemWithState
	var total int64
	var err error

	// Determine which list method to use based on filters
	if opts.Starred != nil && *opts.Starred {
		// List starred items
		listOpts := ListOptions{Limit: opts.Limit, Cursor: opts.Cursor}
		items, total, err = s.itemRepo.ListStarred(ctx, userID, listOpts)
		if err != nil {
			return nil, err
		}
	} else if opts.Read != nil && !*opts.Read {
		// List unread items
		listOpts := ListOptions{Limit: opts.Limit, Cursor: opts.Cursor}
		items, total, err = s.itemRepo.ListUnread(ctx, userID, listOpts)
		if err != nil {
			return nil, err
		}
	} else if opts.FeedID != "" {
		// List items for a specific feed - verify user has access
		_, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, opts.FeedID)
		if err != nil {
			return nil, ErrFeedNotFound
		}
		listOpts := ListOptions{Limit: opts.Limit, Cursor: opts.Cursor}
		items, total, err = s.itemRepo.ListByFeedID(ctx, opts.FeedID, userID, listOpts)
		if err != nil {
			return nil, err
		}
	} else {
		// List all items from user's subscribed feeds
		userFeeds, err := s.userFeedRepo.ListByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}

		// Collect items from all feeds
		itemMap := make(map[string]*ItemWithState)
		for _, uf := range userFeeds {
			listOpts := ListOptions{Limit: opts.Limit * 2, Cursor: opts.Cursor} // Get more to filter
			feedItems, _, err := s.itemRepo.ListByFeedID(ctx, uf.FeedID, userID, listOpts)
			if err != nil {
				continue
			}
			for _, item := range feedItems {
				itemMap[item.ID] = item
			}
		}

		items = make([]*ItemWithState, 0, len(itemMap))
		for _, item := range itemMap {
			items = append(items, item)
		}
		total = int64(len(items))

		// Apply limit
		if opts.Limit > 0 && len(items) > opts.Limit {
			items = items[:opts.Limit]
		}
	}

	// Calculate pagination metadata
	hasMore := opts.Limit > 0 && int64(len(items)) < total
	nextCursor := ""
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		nextCursor = lastItem.ID
	}

	return &ItemListResult{
		Items:      items,
		Total:      total,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

// GetItem retrieves a single item by ID with user state
func (s *itemService) GetItem(ctx context.Context, userID, itemID string) (*ItemWithState, error) {
	// Get the item
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Verify user has access to the feed
	_, err = s.userFeedRepo.GetByUserAndFeed(ctx, userID, item.FeedID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Get user state
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	result := &ItemWithState{Item: item}
	if err == nil {
		result.IsStarred = state.IsStarred
		result.IsRead = state.IsRead
		if state.ReadAt != nil {
			readAt := state.ReadAt.Format(time.RFC3339)
			result.ReadAt = &readAt
		}
	}

	return result, nil
}

// ToggleStar toggles the star status for an item
func (s *itemService) ToggleStar(ctx context.Context, userID, itemID string) (*ItemWithState, error) {
	// Get the item
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Verify user has access to the feed
	_, err = s.userFeedRepo.GetByUserAndFeed(ctx, userID, item.FeedID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Get current state
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		// Create new state
		now := time.Now()
		newState := &model.UserItemState{
			UserID:   userID,
			ItemID:   itemID,
			IsStarred: true,
			IsRead:   false,
		}
		if err := newState.GenerateID(); err != nil {
			return nil, err
		}
		newState.CreatedAt = now
		newState.UpdatedAt = now

		if err := s.stateRepo.Create(ctx, newState); err != nil {
			return nil, err
		}

		return &ItemWithState{
			Item:      item,
			IsStarred: true,
			IsRead:    false,
		}, nil
	}

	// Toggle star status
	state.IsStarred = !state.IsStarred
	state.UpdatedAt = time.Now()

	if err := s.stateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	result := &ItemWithState{
		Item:      item,
		IsStarred: state.IsStarred,
		IsRead:    state.IsRead,
	}
	if state.ReadAt != nil {
		readAt := state.ReadAt.Format(time.RFC3339)
		result.ReadAt = &readAt
	}

	return result, nil
}

// ToggleRead toggles the read status for an item
func (s *itemService) ToggleRead(ctx context.Context, userID, itemID string) (*ItemWithState, error) {
	// Get the item
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Verify user has access to the feed
	_, err = s.userFeedRepo.GetByUserAndFeed(ctx, userID, item.FeedID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Get current state
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		// Create new state with read=true
		now := time.Now()
		newState := &model.UserItemState{
			UserID:  userID,
			ItemID:  itemID,
			IsRead:  true,
			IsStarred: false,
			ReadAt:  &now,
		}
		if err := newState.GenerateID(); err != nil {
			return nil, err
		}
		newState.CreatedAt = now
		newState.UpdatedAt = now

		if err := s.stateRepo.Create(ctx, newState); err != nil {
			return nil, err
		}

		readAt := now.Format(time.RFC3339)
		return &ItemWithState{
			Item:      item,
			IsStarred: false,
			IsRead:    true,
			ReadAt:    &readAt,
		}, nil
	}

	// Toggle read status
	if state.IsRead {
		// Mark as unread - clear ReadAt
		state.IsRead = false
		state.ReadAt = nil
	} else {
		// Mark as read - set ReadAt
		now := time.Now()
		state.IsRead = true
		state.ReadAt = &now
	}
	state.UpdatedAt = time.Now()

	if err := s.stateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	result := &ItemWithState{
		Item:      item,
		IsStarred: state.IsStarred,
		IsRead:    state.IsRead,
	}
	if state.ReadAt != nil {
		readAt := state.ReadAt.Format(time.RFC3339)
		result.ReadAt = &readAt
	}

	return result, nil
}

// MarkAllRead marks all items in a feed as read for a user
func (s *itemService) MarkAllRead(ctx context.Context, userID, feedID string) (int, error) {
	// Verify user has access to the feed
	_, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		return 0, ErrFeedNotFound
	}

	// Get all items for the feed
	listOpts := ListOptions{Limit: 1000} // Max 1000 items at once
	items, total, err := s.itemRepo.ListByFeedID(ctx, feedID, userID, listOpts)
	if err != nil {
		return 0, err
	}

	// Collect item IDs
	itemIDs := make([]string, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID
	}

	// Mark all as read
	if len(itemIDs) > 0 {
		if err := s.stateRepo.BulkMarkRead(ctx, userID, itemIDs); err != nil {
			return 0, err
		}
	}

	return int(total), nil
}
