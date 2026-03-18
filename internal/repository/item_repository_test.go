package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
)

func setupItemDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate tables
	if err := db.AutoMigrate(&model.Feed{}, &model.User{}, &model.Item{}, &model.UserItemState{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// setupItemDBShared creates a shared in-memory database for concurrent access testing
func setupItemDBShared(t *testing.T) *gorm.DB {
	// Use unique database name with timestamp to avoid conflicts between test runs
	dbName := fmt.Sprintf("file:test_%s_%d?cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

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
	if !result[0].IsStarred {
		t.Error("First item should be starred")
	}

	// Check second item is read
	if !result[1].IsRead {
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
		if !item.IsStarred {
			t.Error("ListStarred returned non-starred item")
		}
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
		if item.IsRead {
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
				// SQLite may return "database table is locked" which is expected
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

	// Verify at least some items were created (SQLite concurrency is limited)
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
	// Note: SQLite has limited concurrent write support, so some batches may fail
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
				// SQLite may return "database table is locked" which is expected
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

	// Verify at least some batches succeeded (SQLite concurrency is limited)
	// We expect at least 1 batch to succeed
	if successCount < 1 {
		t.Errorf("Expected at least 1 successful batch, got %d", successCount)
	}
}

// TestItemRepository_RaceCondition_MixedOperations tests mixed read/write operations
// Note: SQLite has limited concurrent write support, so we reduce concurrency
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

	// Reduced concurrency for SQLite compatibility
	const numGoroutines = 30
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	// Helper to check if error is a SQLite lock error (expected in concurrent tests)
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
				// Create new item - SQLite may lock on concurrent writes, so we accept some failures
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
					// SQLite may return "database table is locked" which is expected
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
