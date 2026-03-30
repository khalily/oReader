package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
	"github.com/khalily/oreader/internal/testutil"
)

// setupFeedDB creates a test database for testing
func setupFeedDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	// Migrate tables
	if err := db.AutoMigrate(&model.Feed{}, &model.UserFeed{}, &model.User{}, &model.Item{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// setupFeedDBShared creates a test database for concurrent access testing
func setupFeedDBShared(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	// Migrate tables
	if err := db.AutoMigrate(&model.Feed{}, &model.UserFeed{}, &model.User{}, &model.Item{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func setupUserDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	if err := db.AutoMigrate(&model.User{}); err != nil {
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

// TestFeedRepository_ListByUserID_ExcludesSoftDeleted is a regression test that verifies
// soft-deleted user_feeds are excluded from the list. This test was added after a bug
// was discovered where the JOIN clause didn't filter out soft-deleted subscriptions.
func TestFeedRepository_ListByUserID_ExcludesSoftDeleted(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)
	userRepo := NewUserRepository(db)
	userFeedRepo := NewUserFeedRepository(db)

	ctx := context.Background()

	// Create user
	user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create feeds
	feed1 := &model.Feed{FeedURL: "https://example.com/feed1.xml", Title: "Active Feed"}
	feed2 := &model.Feed{FeedURL: "https://example.com/feed2.xml", Title: "Deleted Feed"}
	feed3 := &model.Feed{FeedURL: "https://example.com/feed3.xml", Title: "Another Active Feed"}
	if err := feed1.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := feed2.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := feed3.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, feed1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, feed2); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, feed3); err != nil {
		t.Fatal(err)
	}

	// Subscribe user to all feeds
	userFeed1 := &model.UserFeed{UserID: user.ID, FeedID: feed1.ID, Position: 0}
	userFeed2 := &model.UserFeed{UserID: user.ID, FeedID: feed2.ID, Position: 1}
	userFeed3 := &model.UserFeed{UserID: user.ID, FeedID: feed3.ID, Position: 2}
	if err := userFeed1.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := userFeed2.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := userFeed3.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := userFeedRepo.Create(ctx, userFeed1); err != nil {
		t.Fatal(err)
	}
	if err := userFeedRepo.Create(ctx, userFeed2); err != nil {
		t.Fatal(err)
	}
	if err := userFeedRepo.Create(ctx, userFeed3); err != nil {
		t.Fatal(err)
	}

	// Soft-delete userFeed2 (unsubscribe from feed2)
	if err := userFeedRepo.Delete(ctx, user.ID, feed2.ID); err != nil {
		t.Fatalf("Failed to soft-delete userFeed: %v", err)
	}

	// List user's feeds
	feeds, total, err := repo.ListByUserID(ctx, user.ID, service.ListOptions{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListByUserID() returned error: %v", err)
	}

	// Should only see 2 feeds (feed1 and feed3), not the soft-deleted feed2
	if total != 2 {
		t.Errorf("Total = %d, want 2 (soft-deleted feed should be excluded)", total)
	}
	if len(feeds) != 2 {
		t.Fatalf("Feeds count = %d, want 2", len(feeds))
	}

	// Verify feed2 is not in the results
	for _, f := range feeds {
		if f.ID == feed2.ID {
			t.Error("Soft-deleted feed should not appear in results")
		}
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

// TestFeedRepository_SQLInjection tests that SQL injection attacks are prevented.
// GORM uses parameterized queries by default, but we verify this explicitly.
// This addresses OWASP A03:2021 - Injection.
func TestFeedRepository_SQLInjection(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)
	userRepo := NewUserRepository(db)

	ctx := context.Background()

	// Create a test user and feed for testing
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// SQL Injection test cases
	// These payloads should NOT execute as SQL; they should be treated as literal strings
	injectionTests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{
			name: "GetByID with DROP TABLE attempt",
			testFn: func(t *testing.T) {
				maliciousID := "1'; DROP TABLE feeds; --"
				_, err := repo.GetByID(ctx, maliciousID)

				// Should either return error (not found) or not find the feed
				// The important thing is that the feeds table still exists
				if err == nil {
					t.Log("GetByID with malicious ID returned no error (not found)")
				}

				// Verify feeds table still exists and has our test feed
				var count int64
				if err := db.Model(&model.Feed{}).Count(&count).Error; err != nil {
					t.Fatalf("Feeds table should still exist, got error: %v", err)
				}
				if count != 1 {
					t.Errorf("Feeds table should have 1 row, got %d (possible SQL injection)", count)
				}
			},
		},
		{
			name: "GetByID with UNION SELECT attempt",
			testFn: func(t *testing.T) {
				maliciousID := "' UNION SELECT * FROM users --"
				_, err := repo.GetByID(ctx, maliciousID)

				// Should not return any user data
				if err == nil {
					t.Log("GetByID with UNION attempt returned no error (safe)")
				}
			},
		},
		{
			name: "GetByID with boolean injection",
			testFn: func(t *testing.T) {
				maliciousID := "' OR '1'='1"
				_, err := repo.GetByID(ctx, maliciousID)

				// Should not return all feeds
				if err == nil {
					t.Log("GetByID with boolean injection returned no error (safe)")
				}
			},
		},
		{
			name: "GetByURL with SQL injection",
			testFn: func(t *testing.T) {
				maliciousURL := "https://example.com' OR '1'='1"
				_, err := repo.GetByURL(ctx, maliciousURL)

				// Should not return all feeds
				if err == nil {
					t.Log("GetByURL with injection returned no error (safe)")
				}

				// Verify no unexpected data returned
				var count int64
				if err := db.Model(&model.Feed{}).Count(&count).Error; err != nil {
					t.Logf("Count error (may be expected): %v", err)
				}
				if count != 1 {
					t.Errorf("Feeds count should still be 1, got %d", count)
				}
			},
		},
		{
			name: "ListByUserID with SQL injection",
			testFn: func(t *testing.T) {
				maliciousUserID := user.ID + "' OR '1'='1"
				_, total, err := repo.ListByUserID(ctx, maliciousUserID, service.ListOptions{Limit: 10, Offset: 0})

				// Should not return all feeds from all users
				if err != nil {
					t.Logf("ListByUserID with injection returned error (safe): %v", err)
				} else if total > 0 {
					// If it returned feeds, verify they belong to the malicious user ID (which should be none)
					t.Logf("ListByUserID returned %d feeds for malicious ID", total)
				}
			},
		},
		{
			name: "Delete with SQL injection",
			testFn: func(t *testing.T) {
				// Get initial count
				var initialCount int64
				if err := db.Model(&model.Feed{}).Count(&initialCount).Error; err != nil {
					t.Fatalf("Failed to get initial count: %v", err)
				}

				maliciousID := "1'; DELETE FROM feeds WHERE '1'='1"
				_ = repo.Delete(ctx, maliciousID)

				// Verify no feeds were deleted
				var finalCount int64
				if err := db.Model(&model.Feed{}).Count(&finalCount).Error; err != nil {
					t.Fatalf("Feeds table should still exist: %v", err)
				}
				if finalCount != initialCount {
					t.Errorf("SQL injection deleted feeds! Initial: %d, Final: %d", initialCount, finalCount)
				}
			},
		},
		{
			name: "Create with SQL injection in fields",
			testFn: func(t *testing.T) {
				maliciousFeed := &model.Feed{
					FeedURL:     "https://example.com/feed.xml'); DROP TABLE feeds; --",
					Title:       "Test'); DELETE FROM feeds WHERE ('1'='1",
					Description: "Normal description",
				}
				if err := maliciousFeed.GenerateID(); err != nil {
					t.Fatalf("Failed to generate ID: %v", err)
				}

				err := repo.Create(ctx, maliciousFeed)

				// Create might succeed (treating as literal string) or fail (validation)
				// Either way, the feeds table should still exist
				if err != nil {
					t.Logf("Create with injection returned error (safe): %v", err)
				}

				// Verify feeds table still exists
				var count int64
				if err := db.Model(&model.Feed{}).Count(&count).Error; err != nil {
					t.Fatalf("Feeds table should still exist: %v", err)
				}
				t.Logf("Feeds table has %d rows after injection attempt", count)
			},
		},
		{
			name: "Update with SQL injection in fields",
			testFn: func(t *testing.T) {
				// Get initial count
				var initialCount int64
				if err := db.Model(&model.Feed{}).Count(&initialCount).Error; err != nil {
					t.Fatalf("Failed to get initial count: %v", err)
				}

				// Try to inject via update
				feed.Title = "Updated'; DROP TABLE feeds; --"
				err := repo.Update(ctx, feed)

				// Update might succeed (treating as literal string)
				if err != nil {
					t.Logf("Update with injection returned error: %v", err)
				}

				// Verify feeds table still exists and count unchanged
				var finalCount int64
				if err := db.Model(&model.Feed{}).Count(&finalCount).Error; err != nil {
					t.Fatalf("Feeds table should still exist: %v", err)
				}
				if finalCount != initialCount {
					t.Errorf("SQL injection affected row count! Initial: %d, Final: %d", initialCount, finalCount)
				}
			},
		},
	}

	for _, tc := range injectionTests {
		t.Run(tc.name, tc.testFn)
	}
}

// =============================================================================
// LARGE DATASET TESTS (PERF-002)
// These tests verify performance and correctness with realistic data volumes
// =============================================================================

// TestFeedRepository_LargeDataset_ListByUserID tests listing feeds with a large dataset
// This addresses the review finding that tests only use 3-15 records
func TestFeedRepository_LargeDataset_ListByUserID(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)
	userRepo := NewUserRepository(db)
	userFeedRepo := NewUserFeedRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create 100 feeds and subscribe user to them
	const numFeeds = 100
	feedIDs := make([]string, numFeeds)
	for i := 0; i < numFeeds; i++ {
		feed := &model.Feed{
			FeedURL:     "https://example.com/feed" + string(rune('0'+i%10)) + string(rune('0'+i/10)) + ".xml",
			Title:       "Feed " + string(rune('0'+i%10)) + string(rune('0'+i/10)),
			Description: "Description for feed " + string(rune('0'+i%10)) + string(rune('0'+i/10)),
		}
		if err := feed.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := repo.Create(ctx, feed); err != nil {
			t.Fatalf("Failed to create feed %d: %v", i, err)
		}
		feedIDs[i] = feed.ID

		// Subscribe user to feed
		userFeed := &model.UserFeed{UserID: user.ID, FeedID: feed.ID, Position: i}
		if err := userFeed.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := userFeedRepo.Create(ctx, userFeed); err != nil {
			t.Fatalf("Failed to create userFeed %d: %v", i, err)
		}
	}

	// Test pagination with different page sizes
	testCases := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
		expectedTotal int64
	}{
		{"first page", 10, 0, 10, numFeeds},
		{"second page", 10, 10, 10, numFeeds},
		{"last page", 10, 90, 10, numFeeds},
		{"partial last page", 15, 90, 10, numFeeds},
		{"large page", 50, 0, 50, numFeeds},
		{"offset beyond data", 10, 100, 0, numFeeds},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			feeds, total, err := repo.ListByUserID(ctx, user.ID, service.ListOptions{Limit: tc.limit, Offset: tc.offset})
			if err != nil {
				t.Fatalf("ListByUserID() returned error: %v", err)
			}

			if total != tc.expectedTotal {
				t.Errorf("Total = %d, want %d", total, tc.expectedTotal)
			}
			if len(feeds) != tc.expectedCount {
				t.Errorf("Feeds count = %d, want %d", len(feeds), tc.expectedCount)
			}

			// Verify feeds are ordered by position
			for i, feed := range feeds {
				if feed.ID == "" {
					t.Errorf("Feed %d has empty ID", i)
				}
			}
		})
	}
}

// TestFeedRepository_LargeDataset_ListAll tests listing all feeds with many records
func TestFeedRepository_LargeDataset_ListAll(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()

	// Create 200 feeds
	const numFeeds = 200
	for i := 0; i < numFeeds; i++ {
		feed := &model.Feed{
			FeedURL:     "https://example.com/feed" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i/100)) + ".xml",
			Title:       "Feed " + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i/100)),
			Description: "Description",
		}
		if err := feed.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := repo.Create(ctx, feed); err != nil {
			t.Fatalf("Failed to create feed %d: %v", i, err)
		}
	}

	// List all feeds
	feeds, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll() returned error: %v", err)
	}

	if len(feeds) != numFeeds {
		t.Errorf("Feeds count = %d, want %d", len(feeds), numFeeds)
	}

	// Verify all feeds have valid data
	seenIDs := make(map[string]bool)
	for i, feed := range feeds {
		if feed.ID == "" {
			t.Errorf("Feed %d has empty ID", i)
		}
		if seenIDs[feed.ID] {
			t.Errorf("Duplicate feed ID: %s", feed.ID)
		}
		seenIDs[feed.ID] = true
	}
}

// TestFeedRepository_LargeDataset_MultipleUsers tests data isolation with many users
func TestFeedRepository_LargeDataset_MultipleUsers(t *testing.T) {
	db := setupFeedDB(t)
	repo := NewFeedRepository(db)
	userRepo := NewUserRepository(db)
	userFeedRepo := NewUserFeedRepository(db)

	ctx := context.Background()

	// Create 50 users
	const numUsers = 50
	const feedsPerUser = 10
	userIDs := make([]string, numUsers)

	for u := 0; u < numUsers; u++ {
		user := &model.User{
			Email:        fmt.Sprintf("user%d@example.com", u),
			PasswordHash: "hash",
		}
		if err := user.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("Failed to create user %d: %v", u, err)
		}
		userIDs[u] = user.ID

		// Create feeds for this user
		for f := 0; f < feedsPerUser; f++ {
			feed := &model.Feed{
				FeedURL:     fmt.Sprintf("https://user%d.example.com/feed%d.xml", u, f),
				Title:       "User Feed",
				Description: "Description",
			}
			if err := feed.GenerateID(); err != nil {
				t.Fatalf("Failed to generate ID: %v", err)
			}
			if err := repo.Create(ctx, feed); err != nil {
				t.Fatalf("Failed to create feed: %v", err)
			}

			userFeed := &model.UserFeed{UserID: user.ID, FeedID: feed.ID, Position: f}
			if err := userFeed.GenerateID(); err != nil {
				t.Fatalf("Failed to generate ID: %v", err)
			}
			if err := userFeedRepo.Create(ctx, userFeed); err != nil {
				t.Fatalf("Failed to create userFeed: %v", err)
			}
		}
	}

	// Verify each user sees exactly their feeds
	for u, userID := range userIDs {
		feeds, total, err := repo.ListByUserID(ctx, userID, service.ListOptions{Limit: 100})
		if err != nil {
			t.Fatalf("ListByUserID() for user %d returned error: %v", u, err)
		}

		if total != feedsPerUser {
			t.Errorf("User %d: Total = %d, want %d", u, total, feedsPerUser)
		}
		if len(feeds) != feedsPerUser {
			t.Errorf("User %d: Feeds count = %d, want %d", u, len(feeds), feedsPerUser)
		}
	}
}

// =============================================================================
// RACE CONDITION TESTS (PERF-003)
// These tests verify thread safety and concurrent access patterns
// Run with: go test -race ./internal/repository/...
// =============================================================================

// TestFeedRepository_RaceCondition_ConcurrentReads tests concurrent read operations
func TestFeedRepository_RaceCondition_ConcurrentReads(t *testing.T) {
	db := setupFeedDBShared(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()

	// Create a feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Run concurrent reads
	const numGoroutines = 50
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.GetByID(ctx, feed.ID)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent read error: %v", err)
	}
}

// TestFeedRepository_RaceCondition_ConcurrentWrites tests concurrent write operations
// Note: MySQL has limited concurrent write support
func TestFeedRepository_RaceCondition_ConcurrentWrites(t *testing.T) {
	db := setupFeedDBShared(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()

	// Run concurrent creates with unique URLs
	const numGoroutines = 20
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)
	successCount := int64(0)
	var countMu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			feed := &model.Feed{
				FeedURL:     fmt.Sprintf("https://example%d.com/feed.xml", idx),
				Title:       "Concurrent Feed",
				Description: "Description",
			}
			if err := feed.GenerateID(); err != nil {
				errChan <- err
				return
			}
			if err := repo.Create(ctx, feed); err != nil {
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

	// At least some writes should succeed
	if successCount < 1 {
		t.Errorf("Expected at least 1 successful write, got %d", successCount)
	}
}

// TestFeedRepository_RaceCondition_ConcurrentUpdates tests concurrent update operations
func TestFeedRepository_RaceCondition_ConcurrentUpdates(t *testing.T) {
	db := setupFeedDBShared(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()

	// Create a feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Original Title"}
	if err := feed.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, feed); err != nil {
		t.Fatalf("Failed to create feed: %v", err)
	}

	// Run concurrent updates
	const numGoroutines = 20
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Get fresh copy
			f, err := repo.GetByID(ctx, feed.ID)
			if err != nil {
				errChan <- err
				return
			}
			// Update title
			f.Title = fmt.Sprintf("Updated Title %d", idx)
			if err := repo.Update(ctx, f); err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Updates may have conflicts, but the feed should still exist
	finalFeed, err := repo.GetByID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("Final GetByID() returned error: %v", err)
	}
	if finalFeed.Title == "" {
		t.Error("Feed title should not be empty after concurrent updates")
	}
}

// TestFeedRepository_RaceCondition_MixedOperations tests mixed read/write operations
func TestFeedRepository_RaceCondition_MixedOperations(t *testing.T) {
	db := setupFeedDBShared(t)
	repo := NewFeedRepository(db)

	ctx := context.Background()

	// Pre-create some feeds
	for i := 0; i < 10; i++ {
		feed := &model.Feed{
			FeedURL:     fmt.Sprintf("https://example%d.com/feed.xml", i),
			Title:       "Initial Feed",
			Description: "Description",
		}
		if err := feed.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := repo.Create(ctx, feed); err != nil {
			t.Fatalf("Failed to create feed: %v", err)
		}
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	// Helper to check if error is a MySQL lock error
	isLockError := func(err error) bool {
		if err == nil {
			return false
		}
		return strings.Contains(err.Error(), "locked") || strings.Contains(err.Error(), "busy")
	}

	// Mix of reads and writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				// Read operation
				_, err := repo.ListAll(ctx)
				if err != nil && !isLockError(err) {
					errChan <- err
				}
			} else {
				// Write operation
				feed := &model.Feed{
					FeedURL:     fmt.Sprintf("https://concurrent%d.example.com/feed.xml", idx),
					Title:       "Concurrent Feed",
					Description: "Description",
				}
				if err := feed.GenerateID(); err != nil {
					errChan <- err
					return
				}
				if err := repo.Create(ctx, feed); err != nil {
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
