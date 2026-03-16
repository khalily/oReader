package repository

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
)

func setupFeedDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate tables
	if err := db.AutoMigrate(&model.Feed{}, &model.UserFeed{}, &model.User{}, &model.Item{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestFeedRepository_Create(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()
	feed := &model.Feed{
		FeedURL:     "https://example.com/feed.xml",
		Title:       "Test Feed",
		Description: "Test Description",
		ImageURL:    "https://example.com/icon.png",
	}

	// Set ID for test
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	err := repo.Create(ctx, feed)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if feed.ID == "" {
		t.Error("Feed ID should be set after creation")
	}
}

func TestFeedRepository_GetByID(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()
	feed := &model.Feed{
		FeedURL:     "https://example.com/feed.xml",
		Title:       "Test Feed",
		Description: "Test Description",
	}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if err := repo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Get by ID
	found, err := repo.GetByID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("GetByID() returned error: %v", err)
	}

	if found.Title != feed.Title {
		t.Errorf("Title = %q, want %q", found.Title, feed.Title)
	}
	if found.FeedURL != feed.FeedURL {
		t.Errorf("FeedURL = %q, want %q", found.FeedURL, feed.FeedURL)
	}
}

func TestFeedRepository_GetByID_NotFound(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()
	_, err := repo.GetByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("GetByID() should return error for non-existent ID")
	}
}

func TestFeedRepository_GetByURL(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()
	feed := &model.Feed{
		FeedURL:     "https://example.com/feed.xml",
		Title:       "Test Feed",
		Description: "Test Description",
	}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if err := repo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Get by URL
	found, err := repo.GetByURL(ctx, feed.FeedURL)
	if err != nil {
		t.Fatalf("GetByURL() returned error: %v", err)
	}

	if found.ID != feed.ID {
		t.Errorf("ID = %q, want %q", found.ID, feed.ID)
	}
}

func TestFeedRepository_ListByUserID(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)
	userRepo := NewUserRepository(db)
	userFeedRepo := NewUserFeedRepository(db)

	ctx := context.Background()

	// Create users
	user1 := &model.User{Email: "user1@example.com", PasswordHash: "hash"}
	user2 := &model.User{Email: "user2@example.com", PasswordHash: "hash"}
	if err := user1.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := user2.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user1); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if err := userRepo.Create(ctx, user2); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feeds
	feed1 := &model.Feed{FeedURL: "https://example.com/feed1.xml", Title: "Feed 1"}
	feed2 := &model.Feed{FeedURL: "https://example.com/feed2.xml", Title: "Feed 2"}
	feed3 := &model.Feed{FeedURL: "https://example.com/feed3.xml", Title: "Feed 3"}
	if err := feed1.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := feed2.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := feed3.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, feed1); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}
	if err := repo.Create(ctx, feed2); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}
	if err := repo.Create(ctx, feed3); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// User1 subscribes to feed1 and feed2
	userFeed1 := &model.UserFeed{UserID: user1.ID, FeedID: feed1.ID, Position: 0}
	userFeed2 := &model.UserFeed{UserID: user1.ID, FeedID: feed2.ID, Position: 1}
	if err := userFeed1.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userFeed2.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userFeedRepo.Create(ctx, userFeed1); err != nil {
		t.Fatalf("Failed to create userFeed: %v", err)
	}
	if err := userFeedRepo.Create(ctx, userFeed2); err != nil {
		t.Fatalf("Failed to create userFeed: %v", err)
	}

	// User2 subscribes to feed3
	userFeed3 := &model.UserFeed{UserID: user2.ID, FeedID: feed3.ID, Position: 0}
	if err := userFeed3.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userFeedRepo.Create(ctx, userFeed3); err != nil {
		t.Fatalf("Failed to create userFeed: %v", err)
	}

	// List user1's feeds
	feeds, total, err := repo.ListByUserID(ctx, user1.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListByUserID() returned error: %v", err)
	}

	if total != 2 {
		t.Errorf("Total = %d, want 2", total)
	}
	if len(feeds) != 2 {
		t.Fatalf("Feeds count = %d, want 2", len(feeds))
	}
}

func TestFeedRepository_Update(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()
	feed := &model.Feed{
		FeedURL:     "https://example.com/feed.xml",
		Title:       "Original Title",
		Description: "Original Description",
	}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if err := repo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Update feed
	feed.Title = "Updated Title"
	feed.Description = "Updated Description"

	err := repo.Update(ctx, feed)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	// Verify update
	updated, err := repo.GetByID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("GetByID() returned error: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
	}
}

func TestFeedRepository_Delete(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()
	feed := &model.Feed{
		FeedURL:     "https://example.com/feed.xml",
		Title:       "Test Feed",
		Description: "Test Description",
	}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if err := repo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Delete feed
	err := repo.Delete(ctx, feed.ID)
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(ctx, feed.ID)
	if err == nil {
		t.Error("GetByID() should return error after deletion")
	}
}

func TestFeedRepository_ListAll(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()

	// Create feeds
	for i := 1; i <= 3; i++ {
		feed := &model.Feed{
			FeedURL:     "https://example.com/feed" + string(rune('0'+i)) + ".xml",
			Title:       "Feed " + string(rune('0'+i)),
			Description: "Description " + string(rune('0'+i)),
		}
		if err := feed.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := repo.Create(ctx, feed); err != nil {
			t.Fatalf("Failed to create feed: %v", err)
		}
	}

	// List all feeds
	feeds, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll() returned error: %v", err)
	}

	if len(feeds) != 3 {
		t.Errorf("Feeds count = %d, want 3", len(feeds))
	}
}
