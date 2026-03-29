package handler

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"oreader/internal/model"
	"oreader/internal/repository"
	"oreader/internal/testutil"
)

// TestFeedRepository_SQLInjection_FeedURL tests SQL injection in feed URL at repository level
func TestFeedRepository_SQLInjection_FeedURL(t *testing.T) {
	db := setupFeedSecurityTestDB(t)
	feedRepo := repository.NewFeedRepository(db)

	// Create a normal feed first
	normalFeed := &model.Feed{
		FeedURL: "https://example.com/normal.xml",
		Title:   "Normal Feed",
	}
	if err := normalFeed.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := feedRepo.Create(context.Background(), normalFeed); err != nil {
		t.Fatal(err)
	}

	// SQL injection payloads in feed URL
	injectionPayloads := []string{
		"https://example.com/feed?id=1' OR '1'='1",
		"https://example.com/feed'; DROP TABLE feeds;--",
		"https://example.com/feed' UNION SELECT * FROM users--",
	}

	for _, payload := range injectionPayloads {
		t.Run("payload_"+truncatePayload(payload), func(t *testing.T) {
			// Try to find feed with SQL injection payload
			_, err := feedRepo.GetByURL(context.Background(), payload)

			// Should return error (not found) without causing damage
			if err == nil {
				t.Errorf("SQL injection payload should not find a feed: %s", payload)
			}

			// Verify feeds table still exists and has the normal feed
			var count int64
			if err := db.Model(&model.Feed{}).Count(&count).Error; err != nil {
				t.Fatalf("Feeds table should still exist: %v", err)
			}
			if count != 1 {
				t.Errorf("Feeds count = %d, want 1 (SQL injection may have modified data)", count)
			}

			// Verify normal feed still exists
			found, err := feedRepo.GetByURL(context.Background(), "https://example.com/normal.xml")
			if err != nil || found.ID != normalFeed.ID {
				t.Errorf("Normal feed should still exist after injection attempt")
			}
		})
	}
}

// TestFeedRepository_MaliciousInput tests handling of malicious input in feed fields
func TestFeedRepository_MaliciousInput(t *testing.T) {
	db := setupFeedSecurityTestDB(t)
	feedRepo := repository.NewFeedRepository(db)

	maliciousTests := []struct {
		name    string
		feedURL string
		title   string
	}{
		{
			name:    "null_byte_in_url",
			feedURL: "https://example.com/feed\x00.xml",
			title:   "Test Feed",
		},
		{
			name:    "null_byte_in_title",
			feedURL: "https://example.com/feed.xml",
			title:   "Test\x00Feed",
		},
		{
			name:    "very_long_url",
			feedURL: "https://example.com/" + string(make([]byte, 10000)),
			title:   "Test Feed",
		},
	}

	for _, tt := range maliciousTests {
		t.Run(tt.name, func(t *testing.T) {
			feed := &model.Feed{
				FeedURL: tt.feedURL,
				Title:   tt.title,
			}
			if err := feed.GenerateID(); err != nil {
				t.Fatal(err)
			}

			err := feedRepo.Create(context.Background(), feed)

			// The repository should either:
			// 1. Reject the input (return error)
			// 2. Safely store it (no crash, no data corruption)
			if err != nil {
				// Input was rejected - this is fine
				t.Logf("Malicious input rejected: %v", err)
			} else {
				// Input was stored - verify it can be retrieved safely
				found, err := feedRepo.GetByID(context.Background(), feed.ID)
				if err != nil {
					t.Errorf("Failed to retrieve stored feed: %v", err)
				}
				if found == nil {
					t.Error("Feed should exist")
				}
			}
		})
	}
}

// TestFeedRepository_SQLInjection_Delete tests SQL injection in delete operations
func TestFeedRepository_SQLInjection_Delete(t *testing.T) {
	db := setupSecurityTestDB(t)
	feedRepo := repository.NewFeedRepository(db)

	// Create test feeds
	for i := 0; i < 3; i++ {
		feed := &model.Feed{
			FeedURL: "https://example.com/feed" + string(rune('0'+i)) + ".xml",
			Title:   "Test Feed",
		}
		if err := feed.GenerateID(); err != nil {
			t.Fatal(err)
		}
		if err := feedRepo.Create(context.Background(), feed); err != nil {
			t.Fatal(err)
		}
	}

	// Try to delete with SQL injection
	maliciousID := "1'; DELETE FROM feeds WHERE '1'='1"
	_ = feedRepo.Delete(context.Background(), maliciousID)

	// Verify all feeds still exist
	var count int64
	if err := db.Model(&model.Feed{}).Count(&count).Error; err != nil {
		t.Fatalf("Feeds table should still exist: %v", err)
	}
	if count != 3 {
		t.Errorf("Feeds count = %d, want 3 (SQL injection may have deleted data)", count)
	}
}

// setupFeedSecurityTestDB creates a test database for security testing
func setupFeedSecurityTestDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	if err := db.AutoMigrate(&model.Feed{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
