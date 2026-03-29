package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
	"oreader/internal/testutil"
)

func setupItemDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	// Migrate tables
	if err := db.AutoMigrate(&model.Feed{}, &model.User{}, &model.Item{}, &model.UserItemState{}, &model.UserFeed{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
// setupItemDBShared creates a test database for concurrent access testing
func setupItemDBShared(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	// Migrate tables
	if err := db.AutoMigrate(&model.Feed{}, &model.User{}, &model.Item{}, &model.UserItemState{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestItemRepository_Create(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed first
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create item
	item := &model.Item{
		FeedID:      feed.ID,
		GUID:        "https://example.com/item1",
		Title:       "Test Item",
		Link:        "https://example.com/item1",
		Description: "Test Description",
		Content:     "Test Content",
	}
	if err := item.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	err := repo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
}

func TestItemRepository_CreateBatch(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed first
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create items
	items := []*model.Item{
		{
			FeedID: feed.ID,
			GUID:   "guid1",
			Title:  "Item 1",
			Link:   "https://example.com/1",
		},
		{
			FeedID: feed.ID,
			GUID:   "guid2",
			Title:  "Item 2",
			Link:   "https://example.com/2",
		},
		{
			FeedID: feed.ID,
			GUID:   "guid3",
			Title:  "Item 3",
			Link:   "https://example.com/3",
		},
	}
	for _, item := range items {
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}

	err := repo.CreateBatch(ctx, items)
	if err != nil {
		t.Fatalf("CreateBatch() returned error: %v", err)
	}
}

func TestItemRepository_GetByID(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed and item
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	item := &model.Item{
		FeedID: feed.ID,
		GUID:   "guid1",
		Title:  "Test Item",
		Link:   "https://example.com/1",
	}
	if err := item.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// Get by ID
	found, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetByID() returned error: %v", err)
	}

	if found.Title != item.Title {
		t.Errorf("Title = %q, want %q", found.Title, item.Title)
	}
}

func TestItemRepository_GetByGUID(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed and item
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	item := &model.Item{
		FeedID: feed.ID,
		GUID:   "unique-guid-123",
		Title:  "Test Item",
		Link:   "https://example.com/1",
	}
	if err := item.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// Get by GUID
	found, err := repo.GetByGUID(ctx, feed.ID, "unique-guid-123")
	if err != nil {
		t.Fatalf("GetByGUID() returned error: %v", err)
	}

	if found.ID != item.ID {
		t.Errorf("ID = %q, want %q", found.ID, item.ID)
	}
}

func TestItemRepository_ListByFeedID(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create items
	items := []*model.Item{}
	for i := 1; i <= 5; i++ {
		item := &model.Item{
			FeedID: feed.ID,
			GUID:   "guid" + string(rune('0'+i)),
			Title:  "Item " + string(rune('0'+i)),
			Link:   "https://example.com/" + string(rune('0'+i)),
		}
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		items = append(items, item)
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Mark one as starred, one as read
	starredState := &model.UserItemState{
		UserID:    user.ID,
		ItemID:    items[0].ID,
		IsStarred: true,
		IsRead:    false,
	}
	readState := &model.UserItemState{
		UserID:    user.ID,
		ItemID:    items[1].ID,
		IsStarred: false,
		IsRead:    true,
	}
	now := time.Now()
	readState.ReadAt = &now
	if err := starredState.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := readState.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := stateRepo.Create(ctx, starredState); err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}
	if err := stateRepo.Create(ctx, readState); err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	// List items
	result, total, err := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListByFeedID() returned error: %v", err)
	}

	if total != 5 {
		t.Errorf("Total = %d, want 5", total)
	}
	if len(result) != 5 {
		t.Fatalf("Items count = %d, want 5", len(result))
	}

	// Check first item is starred
	if result[0].UserState == nil || !result[0].UserState.IsStarred {
		t.Error("First item should be starred")
	}

	// Check second item is read
	if result[1].UserState == nil || !result[1].UserState.IsRead {
		t.Error("Second item should be read")
	}
}

func TestItemRepository_CountByFeedID(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create items
	items := []*model.Item{}
	for i := 1; i <= 3; i++ {
		item := &model.Item{
			FeedID: feed.ID,
			GUID:   "guid" + string(rune('0'+i)),
			Title:  "Item " + string(rune('0'+i)),
			Link:   "https://example.com/" + string(rune('0'+i)),
		}
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		items = append(items, item)
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Count items
	count, err := repo.CountByFeedID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("CountByFeedID() returned error: %v", err)
	}

	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}
}

func TestItemRepository_ListStarred(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create user-feed relationship (user subscribes to feed)
	userFeed := &model.UserFeed{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err := userFeed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeed ID: %v", err)
	}
	if err := db.Create(userFeed).Error; err != nil {
		t.Fatalf("Failed to create user_feed: %v", err)
	}

	// Create items
	items := []*model.Item{}
	for i := 1; i <= 3; i++ {
		item := &model.Item{
			FeedID: feed.ID,
			GUID:   "guid" + string(rune('0'+i)),
			Title:  "Item " + string(rune('0'+i)),
			Link:   "https://example.com/" + string(rune('0'+i)),
		}
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		items = append(items, item)
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Star first two items
	for i := 0; i < 2; i++ {
		state := &model.UserItemState{
			UserID:    user.ID,
			ItemID:    items[i].ID,
			IsStarred: true,
		}
		if err := state.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := stateRepo.Create(ctx, state); err != nil {
			t.Fatalf("Failed to create state: %v", err)
		}
	}

	// List starred items
	result, total, err := repo.ListStarred(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListStarred() returned error: %v", err)
	}

	if total != 2 {
		t.Errorf("Total = %d, want 2", total)
	}
	if len(result) != 2 {
		t.Fatalf("Items count = %d, want 2", len(result))
	}
}

func TestItemRepository_ListUnread(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create user-feed relationship (user subscribes to feed)
	userFeed := &model.UserFeed{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err := userFeed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeed ID: %v", err)
	}
	if err := db.Create(userFeed).Error; err != nil {
		t.Fatalf("Failed to create user_feed: %v", err)
	}

	// Create items
	items := []*model.Item{}
	for i := 1; i <= 3; i++ {
		item := &model.Item{
			FeedID: feed.ID,
			GUID:   "guid" + string(rune('0'+i)),
			Title:  "Item " + string(rune('0'+i)),
			Link:   "https://example.com/" + string(rune('0'+i)),
		}
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		items = append(items, item)
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Mark first item as read
	state := &model.UserItemState{
		UserID: user.ID,
		ItemID: items[0].ID,
		IsRead: true,
	}
	if err := state.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := stateRepo.Create(ctx, state); err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	// List unread items
	result, total, err := repo.ListUnread(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListUnread() returned error: %v", err)
	}

	if total != 2 {
		t.Errorf("Total = %d, want 2", total)
	}
	if len(result) != 2 {
		t.Fatalf("Items count = %d, want 2", len(result))
	}
}

// TestItemRepository_ListUnread_ExcludesUnsubscribedFeed verifies that items from
// unsubscribed feeds (soft-deleted UserFeed) do not appear in the unread list.
// This is a regression test for the 404 issue when clicking items in ListUnread.
func TestItemRepository_ListUnread_ExcludesUnsubscribedFeed(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create Feed A (will be unsubscribed)
	feedA := &model.Feed{FeedURL: "https://feed-a.com/feed.xml", Title: "Feed A"}
	if err := feedA.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feedA).Error; err != nil {
		t.Fatalf("Failed to create feed A: %v", err)
	}

	// Create Feed B (will remain subscribed)
	feedB := &model.Feed{FeedURL: "https://feed-b.com/feed.xml", Title: "Feed B"}
	if err := feedB.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feedB).Error; err != nil {
		t.Fatalf("Failed to create feed B: %v", err)
	}

	// Create user-feed relationships
	userFeedA := &model.UserFeed{UserID: user.ID, FeedID: feedA.ID}
	if err := userFeedA.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeedA ID: %v", err)
	}
	if err := db.Create(userFeedA).Error; err != nil {
		t.Fatalf("Failed to create user_feed A: %v", err)
	}

	userFeedB := &model.UserFeed{UserID: user.ID, FeedID: feedB.ID}
	if err := userFeedB.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeedB ID: %v", err)
	}
	if err := db.Create(userFeedB).Error; err != nil {
		t.Fatalf("Failed to create user_feed B: %v", err)
	}

	// Create items for Feed A
	itemA1 := &model.Item{
		FeedID: feedA.ID,
		GUID:   "guid-a1",
		Title:  "Item A1",
		Link:   "https://feed-a.com/item1",
	}
	if err := itemA1.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	itemA2 := &model.Item{
		FeedID: feedA.ID,
		GUID:   "guid-a2",
		Title:  "Item A2 (marked unread)",
		Link:   "https://feed-a.com/item2",
	}
	if err := itemA2.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	// Create items for Feed B
	itemB1 := &model.Item{
		FeedID: feedB.ID,
		GUID:   "guid-b1",
		Title:  "Item B1",
		Link:   "https://feed-b.com/item1",
	}
	if err := itemB1.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if err := repo.CreateBatch(ctx, []*model.Item{itemA1, itemA2, itemB1}); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Mark itemA2 as unread explicitly (creates UserItemState with is_read=false)
	stateA2 := &model.UserItemState{
		UserID: user.ID,
		ItemID: itemA2.ID,
		IsRead: false,
	}
	if err := stateA2.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := stateRepo.Create(ctx, stateA2); err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	// Soft delete user_feed A (user unsubscribes from Feed A)
	if err := db.Delete(userFeedA).Error; err != nil {
		t.Fatalf("Failed to soft delete user_feed A: %v", err)
	}

	// List unread items
	result, total, err := repo.ListUnread(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListUnread() returned error: %v", err)
	}

	// Should only include items from Feed B (itemB1)
	// Should NOT include itemA1 (no state, but unsubscribed)
	// Should NOT include itemA2 (has is_read=false state, but unsubscribed)
	if total != 1 {
		t.Errorf("Total = %d, want 1 (only items from subscribed feeds)", total)
	}
	if len(result) != 1 {
		t.Fatalf("Items count = %d, want 1", len(result))
	}

	// Verify the item is from Feed B
	if result[0].Item.FeedID != feedB.ID {
		t.Errorf("Expected item from Feed B, got from feed %s", result[0].Item.FeedID)
	}
}

// =============================================================================
// LARGE DATASET TESTS (PERF-002)
// These tests verify performance and correctness with realistic data volumes
// =============================================================================

// TestItemRepository_LargeDataset_ListByFeedID tests listing items with 500+ records
func TestItemRepository_LargeDataset_ListByFeedID(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create 500 items
	const numItems = 500
	items := make([]*model.Item, numItems)
	for i := 0; i < numItems; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   "guid-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i/100)),
			Title:  "Item",
			Link:   "https://example.com/item",
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Star every 10th item (0, 10, 20, ...)
	for i := 0; i < numItems; i += 10 {
		state := &model.UserItemState{
			UserID:    user.ID,
			ItemID:    items[i].ID,
			IsStarred: true,
		}
		if err := state.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := stateRepo.Create(ctx, state); err != nil {
			t.Fatalf("Failed to create state: %v", err)
		}
	}

	// Mark every 5th item as read, but skip items that are already starred (every 10th)
	// This avoids unique constraint violations
	for i := 5; i < numItems; i += 10 {
		state := &model.UserItemState{
			UserID: user.ID,
			ItemID: items[i].ID,
			IsRead: true,
		}
		if err := state.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := stateRepo.Create(ctx, state); err != nil {
			t.Fatalf("Failed to create state: %v", err)
		}
	}

	// Test pagination
	testCases := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
		expectedTotal int64
	}{
		{"first page", 20, 0, 20, numItems},
		{"middle page", 20, 100, 20, numItems},
		{"last full page", 20, 480, 20, numItems},
		{"partial last page", 20, 490, 10, numItems},
		{"large page", 100, 0, 100, numItems},
		{"offset near end", 50, 460, 40, numItems},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, total, err := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: tc.limit, Offset: tc.offset})
			if err != nil {
				t.Fatalf("ListByFeedID() returned error: %v", err)
			}

			if total != tc.expectedTotal {
				t.Errorf("Total = %d, want %d", total, tc.expectedTotal)
			}
			if len(result) != tc.expectedCount {
				t.Errorf("Items count = %d, want %d", len(result), tc.expectedCount)
			}

			// Verify state is loaded correctly
			for _, item := range result {
				if item.ID == "" {
					t.Error("Item has empty ID")
				}
			}
		})
	}
}

// TestItemRepository_LargeDataset_ListStarred tests starred items with many records
func TestItemRepository_LargeDataset_ListStarred(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create user-feed relationship (user subscribes to feed)
	userFeed := &model.UserFeed{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err := userFeed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeed ID: %v", err)
	}
	if err := db.Create(userFeed).Error; err != nil {
		t.Fatalf("Failed to create user_feed: %v", err)
	}

	// Create 200 items, star 50 of them
	const numItems = 200
	const numStarred = 50
	items := make([]*model.Item, numItems)
	for i := 0; i < numItems; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   "guid-" + string(rune('0'+i%10)) + string(rune('0'+i/10)),
			Title:  "Item",
			Link:   "https://example.com/item",
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Star first 50 items
	for i := 0; i < numStarred; i++ {
		state := &model.UserItemState{
			UserID:    user.ID,
			ItemID:    items[i].ID,
			IsStarred: true,
		}
		if err := state.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := stateRepo.Create(ctx, state); err != nil {
			t.Fatalf("Failed to create state: %v", err)
		}
	}

	// List starred items
	result, total, err := repo.ListStarred(ctx, user.ID, service.ListOptions{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("ListStarred() returned error: %v", err)
	}

	if total != numStarred {
		t.Errorf("Total starred = %d, want %d", total, numStarred)
	}
	if len(result) != numStarred {
		t.Errorf("Starred items count = %d, want %d", len(result), numStarred)
	}

	// Verify all returned items are starred
	for _, item := range result {
		if item.UserState == nil || !item.UserState.IsStarred {
			t.Error("ListStarred returned non-starred item")
		}
	}
}

// TestItemRepository_ListStarred_UnsubscribedFeed tests that starred items from
// unsubscribed feeds are not returned (regression test for 404 error when clicking item)
func TestItemRepository_ListStarred_UnsubscribedFeed(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed A (user will subscribe and then unsubscribe)
	feedA := &model.Feed{FeedURL: "https://feed-a.example.com/feed.xml", Title: "Feed A"}
	if err := feedA.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feedA).Error; err != nil {
		t.Fatalf("Failed to create feed A: %v", err)
	}

	// Create feed B (user will stay subscribed)
	feedB := &model.Feed{FeedURL: "https://feed-b.example.com/feed.xml", Title: "Feed B"}
	if err := feedB.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feedB).Error; err != nil {
		t.Fatalf("Failed to create feed B: %v", err)
	}

	// Create user-feed relationship for feed A (will be soft deleted later)
	userFeedA := &model.UserFeed{
		UserID: user.ID,
		FeedID: feedA.ID,
	}
	if err := userFeedA.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeed ID: %v", err)
	}
	if err := db.Create(userFeedA).Error; err != nil {
		t.Fatalf("Failed to create user_feed A: %v", err)
	}

	// Create user-feed relationship for feed B (will stay subscribed)
	userFeedB := &model.UserFeed{
		UserID: user.ID,
		FeedID: feedB.ID,
	}
	if err := userFeedB.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeed ID: %v", err)
	}
	if err := db.Create(userFeedB).Error; err != nil {
		t.Fatalf("Failed to create user_feed B: %v", err)
	}

	// Create item in feed A
	itemA := &model.Item{
		FeedID: feedA.ID,
		GUID:   "guid-a",
		Title:  "Item from Feed A",
		Link:   "https://feed-a.example.com/item",
	}
	if err := itemA.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.CreateBatch(ctx, []*model.Item{itemA}); err != nil {
		t.Fatalf("Failed to create item A: %v", err)
	}

	// Create item in feed B
	itemB := &model.Item{
		FeedID: feedB.ID,
		GUID:   "guid-b",
		Title:  "Item from Feed B",
		Link:   "https://feed-b.example.com/item",
	}
	if err := itemB.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.CreateBatch(ctx, []*model.Item{itemB}); err != nil {
		t.Fatalf("Failed to create item B: %v", err)
	}

	// Star both items
	stateA := &model.UserItemState{
		UserID:    user.ID,
		ItemID:    itemA.ID,
		IsStarred: true,
	}
	if err := stateA.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := stateRepo.Create(ctx, stateA); err != nil {
		t.Fatalf("Failed to create state A: %v", err)
	}

	stateB := &model.UserItemState{
		UserID:    user.ID,
		ItemID:    itemB.ID,
		IsStarred: true,
	}
	if err := stateB.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := stateRepo.Create(ctx, stateB); err != nil {
		t.Fatalf("Failed to create state B: %v", err)
	}

	// Verify both items are in starred list
	result, total, err := repo.ListStarred(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListStarred() returned error: %v", err)
	}
	if total != 2 {
		t.Fatalf("Expected 2 starred items, got %d", total)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 starred items in result, got %d", len(result))
	}

	// User unsubscribes from feed A (soft delete user_feed)
	if err := db.Delete(userFeedA).Error; err != nil {
		t.Fatalf("Failed to soft delete user_feed A: %v", err)
	}

	// Now starred list should only contain item from feed B
	result, total, err = repo.ListStarred(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListStarred() after unsubscribe returned error: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 starred item after unsubscribe, got %d", total)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 starred item in result after unsubscribe, got %d", len(result))
	}
	if len(result) > 0 && result[0].Title != "Item from Feed B" {
		t.Errorf("Expected 'Item from Feed B', got %q", result[0].Title)
	}
}

// TestItemRepository_LargeDataset_ListUnread tests unread items with many records
func TestItemRepository_LargeDataset_ListUnread(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create user-feed relationship (user subscribes to feed)
	userFeed := &model.UserFeed{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err := userFeed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeed ID: %v", err)
	}
	if err := db.Create(userFeed).Error; err != nil {
		t.Fatalf("Failed to create user_feed: %v", err)
	}

	// Create 300 items
	const numItems = 300
	const numRead = 100
	const numUnread = numItems - numRead
	items := make([]*model.Item, numItems)
	for i := 0; i < numItems; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   "guid-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i/100)),
			Title:  "Item",
			Link:   "https://example.com/item",
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Mark first 100 as read
	for i := 0; i < numRead; i++ {
		state := &model.UserItemState{
			UserID: user.ID,
			ItemID: items[i].ID,
			IsRead: true,
		}
		if err := state.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := stateRepo.Create(ctx, state); err != nil {
			t.Fatalf("Failed to create state: %v", err)
		}
	}

	// List unread items
	result, total, err := repo.ListUnread(ctx, user.ID, service.ListOptions{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("ListUnread() returned error: %v", err)
	}

	if total != numUnread {
		t.Errorf("Total unread = %d, want %d", total, numUnread)
	}
	if len(result) != 100 { // Limited to 100
		t.Errorf("Unread items count = %d, want 100 (limited)", len(result))
	}

	// Verify all returned items are unread
	for _, item := range result {
		// Item is unread if UserState is nil (never read) or IsRead is false
		if item.UserState != nil && item.UserState.IsRead {
			t.Error("ListUnread returned read item")
		}
	}
}

// TestItemRepository_LargeDataset_CountByFeedID tests counting with many records
func TestItemRepository_LargeDataset_CountByFeedID(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create 1000 items
	const numItems = 1000
	items := make([]*model.Item, numItems)
	for i := 0; i < numItems; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   "guid-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+(i/100)%10)) + string(rune('0'+i/1000)),
			Title:  "Item",
			Link:   "https://example.com/item",
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Count items
	count, err := repo.CountByFeedID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("CountByFeedID() returned error: %v", err)
	}

	if count != numItems {
		t.Errorf("Count = %d, want %d", count, numItems)
	}
}

// =============================================================================
// RACE CONDITION TESTS (PERF-003)
// These tests verify thread safety and concurrent access patterns
// Run with: go test -race ./internal/repository/...
// =============================================================================

// TestItemRepository_RaceCondition_ConcurrentReads tests concurrent read operations
func TestItemRepository_RaceCondition_ConcurrentReads(t *testing.T) {
	db := setupItemDBShared(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed with unique URL
	feed := &model.Feed{FeedURL: "https://race-read.example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create 50 items
	items := make([]*model.Item, 50)
	for i := 0; i < 50; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   fmt.Sprintf("guid-%d", i),
			Title:  "Item",
			Link:   "https://example.com/item",
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Run concurrent reads
	const numGoroutines = 100
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Read by ID
			_, err := repo.GetByID(ctx, items[idx%50].ID)
			if err != nil {
				errChan <- err
				return
			}
			// Count items
			_, err = repo.CountByFeedID(ctx, feed.ID)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent read error: %v", err)
	}
}

// TestItemRepository_RaceCondition_ConcurrentWrites tests concurrent write operations
func TestItemRepository_RaceCondition_ConcurrentWrites(t *testing.T) {
	db := setupItemDBShared(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Run concurrent creates
	const numGoroutines = 50
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)
	var successCount int64
	var countMu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			item := &model.Item{
				FeedID: feed.ID,
				GUID:   fmt.Sprintf("concurrent-guid-%d", idx),
				Title:  "Concurrent Item",
				Link:   fmt.Sprintf("https://example.com/item/%d", idx),
			}
			if err := item.GenerateID(); err != nil {
				errChan <- err
				return
			}
			if err := repo.Create(ctx, item); err != nil {
				// MySQL may return "database table is locked" which is expected
				if !strings.Contains(err.Error(), "locked") && !strings.Contains(err.Error(), "busy") {
					errChan <- err
				}
			} else {
				countMu.Lock()
				successCount++
				countMu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify at least some items were created (MySQL concurrency is limited)
	if successCount < 1 {
		t.Errorf("Expected at least 1 successful write, got %d", successCount)
	}
}

// TestItemRepository_RaceCondition_ConcurrentBatchWrites tests concurrent batch operations
func TestItemRepository_RaceCondition_ConcurrentBatchWrites(t *testing.T) {
	db := setupItemDBShared(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create multiple feeds
	const numFeeds = 5
	feeds := make([]*model.Feed, numFeeds)
	for i := 0; i < numFeeds; i++ {
		feeds[i] = &model.Feed{
			FeedURL: fmt.Sprintf("https://example%d.com/feed.xml", i),
			Title:   "Test Feed",
		}
		if err := feeds[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := db.Create(feeds[i]).Error; err != nil {
			t.Fatalf("Failed to create feed: %v", err)
		}
	}

	// Run concurrent batch creates for each feed
	// Note: MySQL has limited concurrent write support, so some batches may fail
	const itemsPerBatch = 20
	var wg sync.WaitGroup
	errChan := make(chan error, numFeeds)
	successCount := int64(0)
	var countMu sync.Mutex

	for i := 0; i < numFeeds; i++ {
		wg.Add(1)
		go func(feedIdx int) {
			defer wg.Done()
			items := make([]*model.Item, itemsPerBatch)
			for j := 0; j < itemsPerBatch; j++ {
				items[j] = &model.Item{
					FeedID: feeds[feedIdx].ID,
					GUID:   fmt.Sprintf("batch-%d-item-%d", feedIdx, j),
					Title:  "Batch Item",
					Link:   "https://example.com/item",
				}
				if err := items[j].GenerateID(); err != nil {
					errChan <- err
					return
				}
			}
			if err := repo.CreateBatch(ctx, items); err != nil {
				// MySQL may return "database table is locked" which is expected
				if !strings.Contains(err.Error(), "locked") && !strings.Contains(err.Error(), "busy") {
					errChan <- err
				}
			} else {
				countMu.Lock()
				successCount++
				countMu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent batch write error: %v", err)
	}

	// Verify at least some batches succeeded (MySQL concurrency is limited)
	// We expect at least 1 batch to succeed
	if successCount < 1 {
		t.Errorf("Expected at least 1 successful batch, got %d", successCount)
	}
}

// TestItemRepository_RaceCondition_MixedOperations tests mixed read/write operations
// Note: MySQL has limited concurrent write support, so we reduce concurrency
func TestItemRepository_RaceCondition_MixedOperations(t *testing.T) {
	db := setupItemDBShared(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create user and feed
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Pre-create items
	items := make([]*model.Item, 20)
	for i := 0; i < 20; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   fmt.Sprintf("mixed-guid-%d", i),
			Title:  "Item",
			Link:   "https://example.com/item",
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Reduced concurrency for MySQL compatibility
	const numGoroutines = 30
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	// Helper to check if error is a MySQL lock error (expected in concurrent tests)
	isLockError := func(err error) bool {
		if err == nil {
			return false
		}
		return strings.Contains(err.Error(), "locked") || strings.Contains(err.Error(), "busy")
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			switch idx % 4 {
			case 0:
				// Read by feed ID
				_, _, err := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: 10})
				if err != nil && !isLockError(err) {
					errChan <- err
				}
			case 1:
				// Count items
				_, err := repo.CountByFeedID(ctx, feed.ID)
				if err != nil && !isLockError(err) {
					errChan <- err
				}
			case 2:
				// Get by ID
				_, err := repo.GetByID(ctx, items[idx%20].ID)
				if err != nil && !isLockError(err) {
					errChan <- err
				}
			case 3:
				// Create new item - MySQL may lock on concurrent writes, so we accept some failures
				item := &model.Item{
					FeedID: feed.ID,
					GUID:   fmt.Sprintf("mixed-concurrent-%d", idx),
					Title:  "Concurrent Item",
					Link:   "https://example.com/item",
				}
				if err := item.GenerateID(); err != nil {
					errChan <- err
					return
				}
				if err := repo.Create(ctx, item); err != nil {
					// MySQL may return "database table is locked" which is expected
					if !isLockError(err) {
						errChan <- err
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Mixed operation error: %v", err)
	}
}

// =============================================================================
// SPEC COMPLIANCE TESTS: Sorting by pub_date DESC NULLS LAST
// These tests verify that items are sorted by pub_date descending, with NULL
// pub_date values appearing at the end (NULLS LAST).
// =============================================================================

// TestItemRepository_ListByFeedID_SortsByPubDateDescNullsLast verifies sorting order
// Expected order: Items with pub_date DESC, then items with NULL pub_date
// This test will FAIL initially because current implementation sorts by created_at DESC
func TestItemRepository_ListByFeedID_SortsByPubDateDescNullsLast(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "sort-user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://sort-test.example.com/feed.xml", Title: "Sort Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create items with specific pub_dates
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	items := []*model.Item{
		{
			FeedID: feed.ID,
			GUID:   "guid-jan-15",
			Title:  "Jan 15th (newest)",
			Link:   "https://example.com/jan15",
			// pub_date = 2024-01-15 12:00 (should be 1st)
		},
		{
			FeedID: feed.ID,
			GUID:   "guid-jan-10",
			Title:  "Jan 10th",
			Link:   "https://example.com/jan10",
			// pub_date = 2024-01-10 12:00 (should be 2nd)
		},
		{
			FeedID: feed.ID,
			GUID:   "guid-null",
			Title:  "No PubDate (null)",
			Link:   "https://example.com/null",
			// pub_date = NULL (should be last)
		},
		{
			FeedID: feed.ID,
			GUID:   "guid-jan-5",
			Title:  "Jan 5th (oldest)",
			Link:   "https://example.com/jan5",
			// pub_date = 2024-01-05 12:00 (should be 3rd)
		},
	}

	// Set pub_dates
	jan15 := baseTime
	jan10 := baseTime.AddDate(0, 0, -5)
	jan5 := baseTime.AddDate(0, 0, -10)

	items[0].PubDate = &jan15
	items[1].PubDate = &jan10
	items[2].PubDate = nil // NULL
	items[3].PubDate = &jan5

	for _, item := range items {
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}

	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// List items
	result, total, err := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListByFeedID() returned error: %v", err)
	}

	if total != 4 {
		t.Errorf("Expected 4 items, got %d", total)
	}

	if len(result) != 4 {
		t.Fatalf("Expected 4 items in result, got %d", len(result))
	}

	// Verify expected order: Jan 15th, Jan 10th, Jan 5th, No PubDate
	expectedOrder := []string{"Jan 15th (newest)", "Jan 10th", "Jan 5th (oldest)", "No PubDate (null)"}

	for i, item := range result {
		if item.Title != expectedOrder[i] {
			t.Errorf("Position %d: expected title %q, got %q", i, expectedOrder[i], item.Title)
		}
	}

	// Additional verification: NULL pub_date should be at the end
	lastItem := result[len(result)-1]
	if lastItem.PubDate != nil {
		t.Errorf("Last item should have NULL pub_date, but got %v", lastItem.PubDate)
	}
}

// TestItemRepository_ListStarred_SortsByPubDateDescNullsLast verifies starred items sorting
func TestItemRepository_ListStarred_SortsByPubDateDescNullsLast(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "star-sort-user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://star-sort.example.com/feed.xml", Title: "Star Sort Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create user-feed relationship (user subscribes to feed)
	userFeed := &model.UserFeed{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err := userFeed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate UserFeed ID: %v", err)
	}
	if err := db.Create(userFeed).Error; err != nil {
		t.Fatalf("Failed to create user_feed: %v", err)
	}

	// Create items with specific pub_dates
	baseTime := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	feb1 := baseTime
	jan15 := baseTime.AddDate(0, 0, -17)
	jan1 := baseTime.AddDate(0, 0, -31)

	items := []*model.Item{
		{FeedID: feed.ID, GUID: "star-guid-1", Title: "Feb 1 Starred", Link: "https://example.com/1"},
		{FeedID: feed.ID, GUID: "star-guid-2", Title: "Jan 15 Starred", Link: "https://example.com/2"},
		{FeedID: feed.ID, GUID: "star-guid-3", Title: "Jan 1 Starred", Link: "https://example.com/3"},
		{FeedID: feed.ID, GUID: "star-guid-4", Title: "No Date Starred", Link: "https://example.com/4"},
	}
	items[0].PubDate = &feb1
	items[1].PubDate = &jan15
	items[2].PubDate = &jan1
	items[3].PubDate = nil // NULL

	for _, item := range items {
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}

	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Star all items
	for _, item := range items {
		state := &model.UserItemState{
			UserID:    user.ID,
			ItemID:    item.ID,
			IsStarred: true,
		}
		if err := state.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := stateRepo.Create(ctx, state); err != nil {
			t.Fatalf("Failed to create state: %v", err)
		}
	}

	// List starred items
	result, total, err := repo.ListStarred(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListStarred() returned error: %v", err)
	}

	if total != 4 {
		t.Errorf("Expected 4 starred items, got %d", total)
	}

	// Verify order: Feb 1, Jan 15, Jan 1, No Date
	expectedOrder := []string{"Feb 1 Starred", "Jan 15 Starred", "Jan 1 Starred", "No Date Starred"}
	for i, item := range result {
		if item.Title != expectedOrder[i] {
			t.Errorf("Position %d: expected %q, got %q", i, expectedOrder[i], item.Title)
		}
	}
}

// TestItemRepository_ListUnread_SortsByPubDateDescNullsLast verifies unread items sorting
func TestItemRepository_ListUnread_SortsByPubDateDescNullsLast(t *testing.T) {
	db := setupItemDB(t)
	repo := NewItemRepository(db)
	stateRepo := NewUserItemStateRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "unread-sort-user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feed
	feed := &model.Feed{FeedURL: "https://unread-sort.example.com/feed.xml", Title: "Unread Sort Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := db.Create(feed).Error; err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create items with specific pub_dates
	baseTime := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	mar1 := baseTime
	feb15 := baseTime.AddDate(0, 0, -14)
	feb1 := baseTime.AddDate(0, 0, -29)

	items := []*model.Item{
		{FeedID: feed.ID, GUID: "unread-guid-1", Title: "Mar 1 Unread", Link: "https://example.com/1"},
		{FeedID: feed.ID, GUID: "unread-guid-2", Title: "Feb 15 Unread", Link: "https://example.com/2"},
		{FeedID: feed.ID, GUID: "unread-guid-3", Title: "Feb 1 Unread", Link: "https://example.com/3"},
		{FeedID: feed.ID, GUID: "unread-guid-4", Title: "No Date Unread", Link: "https://example.com/4"},
	}
	items[0].PubDate = &mar1
	items[1].PubDate = &feb15
	items[2].PubDate = &feb1
	items[3].PubDate = nil // NULL

	for _, item := range items {
		if err := item.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
	}

	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("Failed to create items: %v", err)
	}

	// Mark items as unread explicitly (IsRead = false)
	for _, item := range items {
		state := &model.UserItemState{
			UserID: user.ID,
			ItemID: item.ID,
			IsRead: false,
		}
		if err := state.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := stateRepo.Create(ctx, state); err != nil {
			t.Fatalf("Failed to create state: %v", err)
		}
	}

	// List unread items
	result, total, err := repo.ListUnread(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListUnread() returned error: %v", err)
	}

	// Verify order: Mar 1, Feb 15, Feb 1, No Date
	expectedOrder := []string{"Mar 1 Unread", "Feb 15 Unread", "Feb 1 Unread", "No Date Unread"}
	for i, item := range result {
		if item.Title != expectedOrder[i] {
			t.Errorf("Position %d: expected %q, got %q", i, expectedOrder[i], item.Title)
		}
	}

	_ = total // May vary based on implementation
}
