package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"oreader/internal/model"
)

// mockItemRepository is a mock implementation of ItemRepository
type mockItemRepository struct {
	items            []*model.Item
	itemStates       []*model.UserItemState
	getByIDErr       error
	listByFeedIDErr  error
	listStarredErr   error
	listUnreadErr    error
	countByFeedIDErr error
}

func (m *mockItemRepository) Create(ctx context.Context, item *model.Item) error {
	m.items = append(m.items, item)
	return nil
}

func (m *mockItemRepository) CreateBatch(ctx context.Context, items []*model.Item) error {
	m.items = append(m.items, items...)
	return nil
}

func (m *mockItemRepository) GetByID(ctx context.Context, id string) (*model.Item, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, errors.New("item not found")
}

func (m *mockItemRepository) GetByGUID(ctx context.Context, feedID, guid string) (*model.Item, error) {
	for _, item := range m.items {
		if item.FeedID == feedID && item.GUID == guid {
			return item, nil
		}
	}
	return nil, errors.New("item not found")
}

func (m *mockItemRepository) ListByFeedID(ctx context.Context, feedID, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	if m.listByFeedIDErr != nil {
		return nil, 0, m.listByFeedIDErr
	}
	var result []*ItemWithState
	for _, item := range m.items {
		if item.FeedID == feedID {
			iws := &ItemWithState{Item: item}
			// Check for user state
			for _, state := range m.itemStates {
				if state.UserID == userID && state.ItemID == item.ID {
					iws.IsStarred = state.IsStarred
					iws.IsRead = state.IsRead
				}
			}
			result = append(result, iws)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockItemRepository) ListStarred(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	if m.listStarredErr != nil {
		return nil, 0, m.listStarredErr
	}
	var result []*ItemWithState
	for _, state := range m.itemStates {
		if state.UserID == userID && state.IsStarred {
			for _, item := range m.items {
				if item.ID == state.ItemID {
					iws := &ItemWithState{Item: item, IsStarred: true, IsRead: state.IsRead}
					result = append(result, iws)
					break
				}
			}
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockItemRepository) ListUnread(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	if m.listUnreadErr != nil {
		return nil, 0, m.listUnreadErr
	}
	var result []*ItemWithState
	// Items with explicit unread state
	for _, state := range m.itemStates {
		if state.UserID == userID && !state.IsRead {
			for _, item := range m.items {
				if item.ID == state.ItemID {
					iws := &ItemWithState{Item: item, IsStarred: state.IsStarred, IsRead: false}
					result = append(result, iws)
					break
				}
			}
		}
	}
	// Items without any state (implicitly unread)
	itemIDs := make(map[string]bool)
	for _, state := range m.itemStates {
		if state.UserID == userID {
			itemIDs[state.ItemID] = true
		}
	}
	for _, item := range m.items {
		if !itemIDs[item.ID] {
			iws := &ItemWithState{Item: item, IsRead: false}
			result = append(result, iws)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockItemRepository) CountByFeedID(ctx context.Context, feedID string) (int64, error) {
	if m.countByFeedIDErr != nil {
		return 0, m.countByFeedIDErr
	}
	count := int64(0)
	for _, item := range m.items {
		if item.FeedID == feedID {
			count++
		}
	}
	return count, nil
}

// mockUserItemStateRepository is a mock implementation of UserItemStateRepository
type mockUserItemStateRepository struct {
	states      []*model.UserItemState
	getByErr    error
	upsertErr   error
	markAllErr  error
}

func (m *mockUserItemStateRepository) Create(ctx context.Context, state *model.UserItemState) error {
	m.states = append(m.states, state)
	return nil
}

func (m *mockUserItemStateRepository) GetByUserAndItem(ctx context.Context, userID, itemID string) (*model.UserItemState, error) {
	if m.getByErr != nil {
		return nil, m.getByErr
	}
	for _, state := range m.states {
		if state.UserID == userID && state.ItemID == itemID {
			return state, nil
		}
	}
	return nil, errors.New("state not found")
}

func (m *mockUserItemStateRepository) Update(ctx context.Context, state *model.UserItemState) error {
	for i, s := range m.states {
		if s.ID == state.ID {
			m.states[i] = state
			return nil
		}
	}
	return errors.New("state not found")
}

func (m *mockUserItemStateRepository) Upsert(ctx context.Context, state *model.UserItemState) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	for i, s := range m.states {
		if s.UserID == state.UserID && s.ItemID == state.ItemID {
			m.states[i] = state
			return nil
		}
	}
	m.states = append(m.states, state)
	return nil
}

func (m *mockUserItemStateRepository) BulkMarkRead(ctx context.Context, userID string, itemIDs []string) error {
	for _, itemID := range itemIDs {
		found := false
		for i, s := range m.states {
			if s.UserID == userID && s.ItemID == itemID {
				m.states[i].IsRead = true
				now := time.Now()
				m.states[i].ReadAt = &now
				found = true
				break
			}
		}
		if !found {
			newState := &model.UserItemState{
				UserID:  userID,
				ItemID:  itemID,
				IsRead:  true,
				IsStarred: false,
			}
			newState.GenerateID()
			now := time.Now()
			newState.ReadAt = &now
			m.states = append(m.states, newState)
		}
	}
	return nil
}

func (m *mockUserItemStateRepository) MarkAllRead(ctx context.Context, userID, feedID string) error {
	if m.markAllErr != nil {
		return m.markAllErr
	}
	// Mark all items from this feed as read
	for i, s := range m.states {
		if s.UserID == userID {
			m.states[i].IsRead = true
			now := time.Now()
			m.states[i].ReadAt = &now
		}
	}
	return nil
}

// mockUserFeedRepository is a mock implementation of UserFeedRepository
type mockUserFeedRepository struct {
	userFeeds   []*model.UserFeed
	getByErr    error
}

func (m *mockUserFeedRepository) Create(ctx context.Context, userFeed *model.UserFeed) error {
	m.userFeeds = append(m.userFeeds, userFeed)
	return nil
}

func (m *mockUserFeedRepository) GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	if m.getByErr != nil {
		return nil, m.getByErr
	}
	for _, uf := range m.userFeeds {
		if uf.UserID == userID && uf.FeedID == feedID {
			return uf, nil
		}
	}
	return nil, errors.New("user feed not found")
}

func (m *mockUserFeedRepository) ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error) {
	var result []*model.UserFeed
	for _, uf := range m.userFeeds {
		if uf.UserID == userID {
			result = append(result, uf)
		}
	}
	return result, nil
}

func (m *mockUserFeedRepository) Delete(ctx context.Context, userID, feedID string) error {
	for i, uf := range m.userFeeds {
		if uf.UserID == userID && uf.FeedID == feedID {
			m.userFeeds = append(m.userFeeds[:i], m.userFeeds[i+1:]...)
			return nil
		}
	}
	return errors.New("user feed not found")
}

func (m *mockUserFeedRepository) GetMaxPosition(ctx context.Context, userID string) (int, error) {
	maxPos := 0
	for _, uf := range m.userFeeds {
		if uf.UserID == userID && uf.Position > maxPos {
			maxPos = uf.Position
		}
	}
	return maxPos, nil
}

// Helper function to create test items
func createTestItem(id, feedID, title string) *model.Item {
	item := &model.Item{
		FeedID:      feedID,
		GUID:        "guid-" + id,
		Title:       title,
		Link:        "https://example.com/" + id,
		Description: "Test description",
		Content:     "<p>Test content</p>",
	}
	item.Base.ID = id
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	return item
}

// Test ItemService.ListItems
func TestItemService_ListItems_AllItems(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
			createTestItem("item-2", feedID, "Item 2"),
			createTestItem("item-3", feedID, "Item 3"),
		},
	}
	stateRepo := &mockUserItemStateRepository{}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	opts := ListItemOptions{Limit: 10}
	result, err := service.ListItems(ctx, userID, opts)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if len(result.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Errorf("Expected total 3, got %d", result.Total)
	}
	if result.HasMore {
		t.Error("Expected HasMore to be false")
	}
}

func TestItemService_ListItems_WithFeedFilter(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
			createTestItem("item-2", "feed-2", "Item 2"),
		},
	}
	stateRepo := &mockUserItemStateRepository{}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{
			{UserID: userID, FeedID: feedID},
			{UserID: userID, FeedID: "feed-2"},
		},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	opts := ListItemOptions{Limit: 10, FeedID: feedID}
	result, err := service.ListItems(ctx, userID, opts)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].FeedID != feedID {
		t.Errorf("Expected feed_id %s, got %s", feedID, result.Items[0].FeedID)
	}
}

func TestItemService_ListItems_WithStarredFilter(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
			createTestItem("item-2", feedID, "Item 2"),
		},
		itemStates: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsStarred: true},
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsStarred: true},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	starred := true
	opts := ListItemOptions{Limit: 10, Starred: &starred}
	result, err := service.ListItems(ctx, userID, opts)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 starred item, got %d", len(result.Items))
	}
	if !result.Items[0].IsStarred {
		t.Error("Expected item to be starred")
	}
}

func TestItemService_ListItems_WithReadFilter(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
			createTestItem("item-2", feedID, "Item 2"),
		},
		itemStates: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsRead: true},
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsRead: true},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	read := false
	opts := ListItemOptions{Limit: 10, Read: &read}
	result, err := service.ListItems(ctx, userID, opts)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// Should include item-2 (no state = unread) and exclude item-1 (read)
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 unread item, got %d", len(result.Items))
	}
	if result.Items[0].IsRead {
		t.Error("Expected item to be unread")
	}
}

func TestItemService_ListItems_Pagination(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	items := make([]*model.Item, 25)
	for i := 0; i < 25; i++ {
		items[i] = createTestItem("item-"+string(rune('1'+i)), feedID, "Item "+string(rune('1'+i)))
	}

	itemRepo := &mockItemRepository{items: items}
	stateRepo := &mockUserItemStateRepository{}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	opts := ListItemOptions{Limit: 10}
	result, err := service.ListItems(ctx, userID, opts)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(result.Items) != 10 {
		t.Errorf("Expected 10 items, got %d", len(result.Items))
	}
	if !result.HasMore {
		t.Error("Expected HasMore to be true")
	}
	if result.NextCursor == "" {
		t.Error("Expected NextCursor to be set")
	}
}

// Test ItemService.GetItem
func TestItemService_GetItem_Success(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsStarred: true, IsRead: true},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	item, err := service.GetItem(ctx, userID, "item-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if item == nil {
		t.Fatal("Expected item, got nil")
	}
	if item.ID != "item-1" {
		t.Errorf("Expected item-1, got %s", item.ID)
	}
	if !item.IsStarred {
		t.Error("Expected item to be starred")
	}
	if !item.IsRead {
		t.Error("Expected item to be read")
	}
}

func TestItemService_GetItem_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"

	itemRepo := &mockItemRepository{
		getByIDErr: errors.New("item not found"),
	}
	stateRepo := &mockUserItemStateRepository{}
	userFeedRepo := &mockUserFeedRepository{}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	_, err := service.GetItem(ctx, userID, "non-existent")

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("Expected ErrItemNotFound, got %v", err)
	}
}

func TestItemService_GetItem_UnauthorizedFeed(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
		},
	}
	stateRepo := &mockUserItemStateRepository{}
	userFeedRepo := &mockUserFeedRepository{
		getByErr: errors.New("user feed not found"),
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	_, err := service.GetItem(ctx, userID, "item-1")

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("Expected ErrItemNotFound, got %v", err)
	}
}

// Test ItemService.ToggleStar
func TestItemService_ToggleStar_Enable(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsStarred: false},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	item, err := service.ToggleStar(ctx, userID, "item-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !item.IsStarred {
		t.Error("Expected item to be starred")
	}
}

func TestItemService_ToggleStar_Disable(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsStarred: true},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	item, err := service.ToggleStar(ctx, userID, "item-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if item.IsStarred {
		t.Error("Expected item to not be starred")
	}
}

func TestItemService_ToggleStar_CreateNewState(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{}, // No existing state
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	item, err := service.ToggleStar(ctx, userID, "item-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !item.IsStarred {
		t.Error("Expected item to be starred")
	}
}

// Test ItemService.ToggleRead
func TestItemService_ToggleRead_Enable(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsRead: false},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	item, err := service.ToggleRead(ctx, userID, "item-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !item.IsRead {
		t.Error("Expected item to be read")
	}
	if item.ReadAt == nil {
		t.Error("Expected ReadAt to be set")
	}
}

func TestItemService_ToggleRead_Disable(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsRead: true},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	item, err := service.ToggleRead(ctx, userID, "item-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if item.IsRead {
		t.Error("Expected item to not be read")
	}
	if item.ReadAt != nil {
		t.Error("Expected ReadAt to be nil")
	}
}

// Test ItemService.MarkAllRead
func TestItemService_MarkAllRead_Success(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{
		items: []*model.Item{
			createTestItem("item-1", feedID, "Item 1"),
			createTestItem("item-2", feedID, "Item 2"),
			createTestItem("item-3", feedID, "Item 3"),
		},
	}
	stateRepo := &mockUserItemStateRepository{
		states: []*model.UserItemState{
			{Base: model.Base{ID: "state-1"}, UserID: userID, ItemID: "item-1", IsRead: false},
			{Base: model.Base{ID: "state-2"}, UserID: userID, ItemID: "item-2", IsRead: false},
		},
	}
	userFeedRepo := &mockUserFeedRepository{
		userFeeds: []*model.UserFeed{{UserID: userID, FeedID: feedID}},
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	count, err := service.MarkAllRead(ctx, userID, feedID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestItemService_MarkAllRead_FeedNotFound(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"

	itemRepo := &mockItemRepository{items: []*model.Item{}}
	stateRepo := &mockUserItemStateRepository{}
	userFeedRepo := &mockUserFeedRepository{
		getByErr: errors.New("user feed not found"),
	}

	service := NewItemService(itemRepo, stateRepo, userFeedRepo)

	_, err := service.MarkAllRead(ctx, userID, feedID)

	if err == nil {
		t.Error("Expected error, got nil")
	}
}
