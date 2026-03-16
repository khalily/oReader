package repository

import (
	"context"
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
