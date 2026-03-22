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

// getFeedTitle extracts the feed title from an item's preloaded Feed relationship
func getFeedTitle(item *model.Item) string {
	if item.Feed != nil {
		return item.Feed.Title
	}
	return ""
}

// buildItemWithState creates an ItemWithState from an item and its user state
func buildItemWithState(item *model.Item, state *model.UserItemState) *ItemWithState {
	result := &ItemWithState{
		Item:      item,
		FeedTitle: getFeedTitle(item),
	}

	if state != nil {
		result.IsStarred = state.IsStarred
		result.IsRead = state.IsRead
		if state.ReadAt != nil {
			readAt := state.ReadAt.Format(time.RFC3339)
			result.ReadAt = &readAt
		}
	}

	return result
}

// verifyUserHasAccessToItem checks if a user has access to an item's feed
func (s *itemService) verifyUserHasAccessToItem(ctx context.Context, userID string, item *model.Item) error {
	_, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, item.FeedID)
	if err != nil {
		return ErrItemNotFound
	}
	return nil
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

	// Apply today filter if requested
	if opts.PublishedToday != nil && *opts.PublishedToday {
		todayStart := time.Now().UTC().Truncate(24 * time.Hour)
		filtered := make([]*ItemWithState, 0, len(items))
		for _, item := range items {
			if item.PubDate != nil && !item.PubDate.Before(todayStart) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
		total = int64(len(items))
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
	if err := s.verifyUserHasAccessToItem(ctx, userID, item); err != nil {
		return nil, err
	}

	// Get user state
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		// No state exists, return item without state
		return buildItemWithState(item, nil), nil
	}

	return buildItemWithState(item, state), nil
}

// ToggleStar toggles the star status for an item
// Deprecated: Use SetStar instead for spec-compliant behavior
func (s *itemService) ToggleStar(ctx context.Context, userID, itemID string) (*ItemWithState, error) {
	// Get current state to determine toggle value
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		// No state exists, toggle from false to true
		return s.SetStar(ctx, userID, itemID, true)
	}

	// Toggle the current value
	return s.SetStar(ctx, userID, itemID, !state.IsStarred)
}

// ToggleRead toggles the read status for an item
// Deprecated: Use SetRead instead for spec-compliant behavior
func (s *itemService) ToggleRead(ctx context.Context, userID, itemID string) (*ItemWithState, error) {
	// Get current state to determine toggle value
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		// No state exists, toggle from false to true
		return s.SetRead(ctx, userID, itemID, true)
	}

	// Toggle the current value
	return s.SetRead(ctx, userID, itemID, !state.IsRead)
}

// SetStar sets the star status for an item to the specified value (spec-compliant: sets, doesn't toggle)
func (s *itemService) SetStar(ctx context.Context, userID, itemID string, starred bool) (*ItemWithState, error) {
	// Get the item
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Verify user has access to the feed
	if err := s.verifyUserHasAccessToItem(ctx, userID, item); err != nil {
		return nil, err
	}

	// Get current state
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		// Create new state
		now := time.Now()
		newState := &model.UserItemState{
			UserID:    userID,
			ItemID:    itemID,
			IsStarred: starred,
			IsRead:    false,
		}
		if err := newState.GenerateID(); err != nil {
			return nil, err
		}
		newState.CreatedAt = now
		newState.UpdatedAt = now

		if err := s.stateRepo.Create(ctx, newState); err != nil {
			return nil, err
		}

		return buildItemWithState(item, newState), nil
	}

	// Set star status to the provided value (not toggle)
	state.IsStarred = starred
	state.UpdatedAt = time.Now()

	if err := s.stateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return buildItemWithState(item, state), nil
}

// SetRead sets the read status for an item to the specified value (spec-compliant: sets, doesn't toggle)
func (s *itemService) SetRead(ctx context.Context, userID, itemID string, read bool) (*ItemWithState, error) {
	// Get the item
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, ErrItemNotFound
	}

	// Verify user has access to the feed
	if err := s.verifyUserHasAccessToItem(ctx, userID, item); err != nil {
		return nil, err
	}

	// Get current state
	state, err := s.stateRepo.GetByUserAndItem(ctx, userID, itemID)
	if err != nil {
		// Create new state
		now := time.Now()
		newState := &model.UserItemState{
			UserID:    userID,
			ItemID:    itemID,
			IsStarred: false,
			IsRead:    read,
		}
		if read {
			newState.ReadAt = &now
		}
		if err := newState.GenerateID(); err != nil {
			return nil, err
		}
		newState.CreatedAt = now
		newState.UpdatedAt = now

		if err := s.stateRepo.Create(ctx, newState); err != nil {
			return nil, err
		}

		return buildItemWithState(item, newState), nil
	}

	// Set read status to the provided value (not toggle)
	state.IsRead = read
	if read {
		now := time.Now()
		state.ReadAt = &now
	} else {
		state.ReadAt = nil
	}
	state.UpdatedAt = time.Now()

	if err := s.stateRepo.Update(ctx, state); err != nil {
		return nil, err
	}

	return buildItemWithState(item, state), nil
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
