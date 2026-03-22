package repository

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/model"
)

// setupUserFeedDB creates an in-memory database for testing
func setupUserFeedDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate tables
	if err := db.AutoMigrate(
		&model.User{},
		&model.Feed{},
		&model.UserFeed{},
	); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// TestGetByUserAndFeedIncludingDeleted tests retrieving soft-deleted user feeds
func TestGetByUserAndFeedIncludingDeleted(t *testing.T) {
	db := setupUserFeedDB(t)
	repo := NewUserFeedRepository(db)
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

	// Create test feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := feedRepo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Create user-feed relationship
	userFeed := &model.UserFeed{
		UserID:   user.ID,
		FeedID:   feed.ID,
		Position: 0,
	}
	if err := userFeed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, userFeed); err != nil {
		t.Fatalf("Failed to create user feed: %v", err)
	}

	// Get before deletion (should work with both methods)
	uf, err := repo.GetByUserAndFeed(ctx, user.ID, feed.ID)
	if err != nil {
		t.Fatalf("GetByUserAndFeed should succeed before deletion: %v", err)
	}
	if uf.ID != userFeed.ID {
		t.Errorf("ID = %s, want %s", uf.ID, userFeed.ID)
	}

	// Soft-delete the relationship
	if err := repo.Delete(ctx, user.ID, feed.ID); err != nil {
		t.Fatalf("Failed to delete user feed: %v", err)
	}

	// Regular GetByUserAndFeed should fail after deletion
	_, err = repo.GetByUserAndFeed(ctx, user.ID, feed.ID)
	if err == nil {
		t.Error("GetByUserAndFeed should fail after soft deletion")
	}

	// GetByUserAndFeedIncludingDeleted should still find the record
	ufDeleted, err := repo.GetByUserAndFeedIncludingDeleted(ctx, user.ID, feed.ID)
	if err != nil {
		t.Fatalf("GetByUserAndFeedIncludingDeleted should succeed after deletion: %v", err)
	}
	if ufDeleted.ID != userFeed.ID {
		t.Errorf("ID = %s, want %s", ufDeleted.ID, userFeed.ID)
	}
}

// TestGetMaxPosition_EmptyUser tests GetMaxPosition for user with no feeds
func TestGetMaxPosition_EmptyUser(t *testing.T) {
	db := setupUserFeedDB(t)
	repo := NewUserFeedRepository(db)
	userRepo := NewUserRepository(db)

	ctx := context.Background()

	// Create test user (no feeds)
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get max position for user with no feeds
	maxPos, err := repo.GetMaxPosition(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetMaxPosition returned error: %v", err)
	}

	// Should return -1 (COALESCE with empty table)
	if maxPos != -1 {
		t.Errorf("MaxPosition = %d, want -1", maxPos)
	}
}

// TestGetMaxPosition_WithFeeds tests GetMaxPosition for user with feeds
func TestGetMaxPosition_WithFeeds(t *testing.T) {
	db := setupUserFeedDB(t)
	repo := NewUserFeedRepository(db)
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

	// Create feeds with different positions
	positions := []int{0, 1, 5, 2}
	for i, pos := range positions {
		feed := &model.Feed{
			FeedURL: "https://example.com/feed" + string(rune('0'+i)) + ".xml",
			Title:   "Feed " + string(rune('0'+i)),
		}
		if err := feed.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := feedRepo.Create(ctx, feed); err != nil {
			t.Fatalf("Failed to create feed: %v", err)
		}

		userFeed := &model.UserFeed{
			UserID:   user.ID,
			FeedID:   feed.ID,
			Position: pos,
		}
		if err := userFeed.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := repo.Create(ctx, userFeed); err != nil {
			t.Fatalf("Failed to create user feed: %v", err)
		}
	}

	// Get max position
	maxPos, err := repo.GetMaxPosition(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetMaxPosition returned error: %v", err)
	}

	// Max position should be 5
	if maxPos != 5 {
		t.Errorf("MaxPosition = %d, want 5", maxPos)
	}
}
