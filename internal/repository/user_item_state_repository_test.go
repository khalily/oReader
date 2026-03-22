package repository

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/model"
)

// setupUserItemStateDB creates an in-memory database for testing
func setupUserItemStateDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate tables
	if err := db.AutoMigrate(
		&model.User{},
		&model.Feed{},
		&model.Item{},
		&model.UserItemState{},
	); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// TestBulkMarkRead_EmptyItemList tests that BulkMarkRead handles empty item list gracefully
func TestBulkMarkRead_EmptyItemList(t *testing.T) {
	db := setupUserItemStateDB(t)
	repo := NewUserItemStateRepository(db)
	userRepo := NewUserRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Call BulkMarkRead with empty list
	err := repo.BulkMarkRead(ctx, user.ID, []string{})
	if err != nil {
		t.Errorf("BulkMarkRead with empty list should return nil, got error: %v", err)
	}

	// Verify no states were created
	var count int64
	if err := db.Model(&model.UserItemState{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("Failed to count states: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 states, got %d", count)
	}
}

// TestBulkMarkRead_MixedExistingAndNew tests BulkMarkRead with mix of existing and new states
func TestBulkMarkRead_MixedExistingAndNew(t *testing.T) {
	db := setupUserItemStateDB(t)
	repo := NewUserItemStateRepository(db)
	userRepo := NewUserRepository(db)
	feedRepo := NewFeedRepository(db)
	itemRepo := NewItemRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create test feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := feedRepo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create test items
	items := make([]*model.Item, 4)
	for i := 0; i < 4; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   "guid-" + string(rune('0'+i)),
			Title:  "Item " + string(rune('0'+i)),
			Link:   "https://example.com/item/" + string(rune('0'+i)),
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := itemRepo.Create(ctx, items[i]); err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
	}

	// Pre-create a state for item[0] (existing state, not read yet)
	existingState := &model.UserItemState{
		UserID: user.ID,
		ItemID: items[0].ID,
		IsRead: false,
	}
	if err := existingState.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, existingState); err != nil {
		t.Fatalf("Failed to create existing state: %v", err)
	}

	// Bulk mark items 0, 1, 2 as read (item 0 has existing state, items 1 and 2 are new)
	itemIDs := []string{items[0].ID, items[1].ID, items[2].ID}
	err := repo.BulkMarkRead(ctx, user.ID, itemIDs)
	if err != nil {
		t.Fatalf("BulkMarkRead returned error: %v", err)
	}

	// Verify all three states exist and are read
	for _, itemID := range itemIDs {
		state, err := repo.GetByUserAndItem(ctx, user.ID, itemID)
		if err != nil {
			t.Errorf("State for item %s should exist, got error: %v", itemID, err)
			continue
		}
		if !state.IsRead {
			t.Errorf("State for item %s should be read", itemID)
		}
	}

	// Verify item[3] (not in the list) has no state
	_, err = repo.GetByUserAndItem(ctx, user.ID, items[3].ID)
	if err == nil {
		t.Error("Item 3 should not have a state")
	}
}

// TestMarkAllRead_NoItems tests MarkAllRead when feed has no items
func TestMarkAllRead_NoItems(t *testing.T) {
	db := setupUserItemStateDB(t)
	repo := NewUserItemStateRepository(db)
	userRepo := NewUserRepository(db)
	feedRepo := NewFeedRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create test feed (no items)
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := feedRepo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Call MarkAllRead
	err := repo.MarkAllRead(ctx, user.ID, feed.ID)
	if err != nil {
		t.Errorf("MarkAllRead on empty feed should return nil, got error: %v", err)
	}

	// Verify no states were created
	var count int64
	if err := db.Model(&model.UserItemState{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("Failed to count states: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 states, got %d", count)
	}
}

// TestMarkAllRead_TransactionIntegrity tests that MarkAllRead maintains transaction integrity
func TestMarkAllRead_TransactionIntegrity(t *testing.T) {
	db := setupUserItemStateDB(t)
	repo := NewUserItemStateRepository(db)
	userRepo := NewUserRepository(db)
	feedRepo := NewFeedRepository(db)
	itemRepo := NewItemRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create test feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := feedRepo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create test items
	const numItems = 5
	items := make([]*model.Item, numItems)
	for i := 0; i < numItems; i++ {
		items[i] = &model.Item{
			FeedID: feed.ID,
			GUID:   "guid-" + string(rune('0'+i)),
			Title:  "Item " + string(rune('0'+i)),
			Link:   "https://example.com/item/" + string(rune('0'+i)),
		}
		if err := items[i].GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := itemRepo.Create(ctx, items[i]); err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
	}

	// Pre-create some states (some read, some unread)
	// Item 0: already read
	state0 := &model.UserItemState{UserID: user.ID, ItemID: items[0].ID, IsRead: true}
	if err := state0.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, state0); err != nil {
		t.Fatal(err)
	}

	// Item 1: not read
	state1 := &model.UserItemState{UserID: user.ID, ItemID: items[1].ID, IsRead: false}
	if err := state1.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, state1); err != nil {
		t.Fatal(err)
	}

	// Items 2, 3, 4 have no state yet

	// Call MarkAllRead
	err := repo.MarkAllRead(ctx, user.ID, feed.ID)
	if err != nil {
		t.Fatalf("MarkAllRead returned error: %v", err)
	}

	// Verify all items are now read
	for i, item := range items {
		state, err := repo.GetByUserAndItem(ctx, user.ID, item.ID)
		if err != nil {
			t.Errorf("Item %d: should have a state, got error: %v", i, err)
			continue
		}
		if !state.IsRead {
			t.Errorf("Item %d: should be read", i)
		}
	}

	// Verify total state count
	var count int64
	if err := db.Model(&model.UserItemState{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("Failed to count states: %v", err)
	}
	if count != numItems {
		t.Errorf("Expected %d states, got %d", numItems, count)
	}
}
