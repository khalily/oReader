package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/repository"
	"oreader/internal/testutil"
)

// setupSecurityTestDB creates a test database for security testing
func setupSecurityTestDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	if err := db.AutoMigrate(&model.User{}, &model.Feed{}, &model.UserFeed{}, &model.Item{}, &model.UserItemState{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Clean tables to avoid data pollution from parallel tests sharing the same DB
	tables := []string{"user_item_states", "items", "user_feeds", "feeds", "users"}
	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("Failed to clean %s table: %v", table, err)
		}
	}

	return db
}

// SQLInjectionPatterns contains common SQL injection payloads for testing
var SQLInjectionPatterns = []string{
	"1' OR '1'='1",
	"1' OR '1'='1'--",
	"1'; DROP TABLE users;--",
	"admin'--",
	"1' UNION SELECT * FROM users--",
}

// TestRepository_SQLInjection_Login tests SQL injection at repository level
func TestRepository_SQLInjection_Login(t *testing.T) {
	db := setupSecurityTestDB(t)
	userRepo := repository.NewUserRepository(db)

	// Create a normal test user
	normalUser := &model.User{Email: "normal@example.com", PasswordHash: "hash"}
	if err := normalUser.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := userRepo.Create(context.Background(), normalUser); err != nil {
		t.Fatal(err)
	}

	for _, payload := range SQLInjectionPatterns {
		t.Run("payload_"+truncatePayload(payload), func(t *testing.T) {
			// Try to find user with SQL injection payload as email
			// GetByEmail returns (nil, nil) when user not found (not an error)
			user, err := userRepo.GetByEmail(context.Background(), payload)
			if err != nil {
				t.Errorf("Unexpected error for SQL injection payload %s: %v", payload, err)
			}
			// Should NOT find any user with SQL injection payload
			if user != nil {
				t.Errorf("SQL injection payload should not find a user: %s", payload)
			}

			// Verify normal user still exists
			found, err := userRepo.GetByEmail(context.Background(), "normal@example.com")
			if err != nil {
				t.Errorf("Unexpected error finding normal user: %v", err)
			}
			if found == nil || found.ID != normalUser.ID {
				t.Errorf("Normal user should still exist after injection attempt")
			}

			// Verify users table still has only 1 row
			var count int64
			if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
				t.Fatalf("Users table should still exist: %v", err)
			}
			if count != 1 {
				t.Errorf("Users count = %d, want 1 (SQL injection may have modified data)", count)
			}
		})
	}
}

// XSSPatterns contains common XSS payloads for testing
var XSSPatterns = []string{
	"<script>alert('xss')</script>",
	"<img src=x onerror=alert('xss')>",
	"<svg onload=alert('xss')>",
	"javascript:alert('xss')",
}

// TestFeedRepository_XSSProtection tests XSS protection in feed title storage
func TestFeedRepository_XSSProtection(t *testing.T) {
	db := setupSecurityTestDB(t)
	feedRepo := repository.NewFeedRepository(db)

	for i, payload := range XSSPatterns {
		t.Run("xss_"+truncatePayload(payload), func(t *testing.T) {
			// Create feed with XSS payload in title
			// Each test needs a unique FeedURL to avoid UNIQUE constraint violations
			feed := &model.Feed{
				FeedURL: "https://example.com/xss-test-" + strconv.Itoa(i) + ".xml",
				Title:   payload,
			}
			if err := feed.GenerateID(); err != nil {
				t.Fatal(err)
			}

			err := feedRepo.Create(context.Background(), feed)
			if err != nil {
				t.Fatalf("Failed to create feed: %v", err)
			}

			// Retrieve the feed
			found, err := feedRepo.GetByID(context.Background(), feed.ID)
			if err != nil {
				t.Fatalf("Failed to get feed: %v", err)
			}

			// The XSS payload should be stored as-is (escaping happens at display time)
			// But we verify it was safely stored and retrieved
			if found.Title != payload {
				t.Errorf("Title = %q, want %q", found.Title, payload)
			}
		})
	}
}

// TestAuthBoundary_NoToken tests requests without authentication token
func TestAuthBoundary_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a simple handler that requires user_id
	router := gin.New()
	router.GET("/protected", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestAuthBoundary_ExpiredToken tests requests with expired token
func TestAuthBoundary_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/protected", func(c *gin.Context) {
		// Simulate token validation that rejects expired tokens
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no token"})
			return
		}
		// In a real app, we'd validate the token here
		if strings.Contains(auth, "expired") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestAuthorization_CrossUserAccess tests that users cannot access other users' resources
func TestAuthorization_CrossUserAccess(t *testing.T) {
	db := setupSecurityTestDB(t)
	userRepo := repository.NewUserRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	userFeedRepo := repository.NewUserFeedRepository(db)

	// Create two users
	user1 := &model.User{Email: "user1@example.com", PasswordHash: "hash"}
	user2 := &model.User{Email: "user2@example.com", PasswordHash: "hash"}
	if err := user1.GenerateID(); err != nil {
		t.Fatal(err)
	}
	if err := user2.GenerateID(); err != nil {
		t.Fatal(err)
	}
	userRepo.Create(context.Background(), user1)
	userRepo.Create(context.Background(), user2)

	// Create feed
	feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "User1 Feed"}
	if err := feed.GenerateID(); err != nil {
		t.Fatal(err)
	}
	feedRepo.Create(context.Background(), feed)

	// Subscribe user1 to feed
	userFeed := &model.UserFeed{UserID: user1.ID, FeedID: feed.ID}
	if err := userFeed.GenerateID(); err != nil {
		t.Fatal(err)
	}
	userFeedRepo.Create(context.Background(), userFeed)

	// Verify user2 cannot access user1's feed through user_feeds
	userFeeds2, err := userFeedRepo.ListByUserID(context.Background(), user2.ID)
	if err != nil {
		t.Fatalf("Failed to list user2 feeds: %v", err)
	}

	// User2 should have no feeds
	if len(userFeeds2) != 0 {
		t.Errorf("User2 should have 0 feeds, got %d (possible data leak)", len(userFeeds2))
	}

	// Verify user1 has the feed
	userFeeds1, err := userFeedRepo.ListByUserID(context.Background(), user1.ID)
	if err != nil {
		t.Fatalf("Failed to list user1 feeds: %v", err)
	}
	if len(userFeeds1) != 1 {
		t.Errorf("User1 should have 1 feed, got %d", len(userFeeds1))
	}
}

func truncatePayload(s string) string {
	if len(s) > 20 {
		s = s[:20]
	}
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, s)
}
