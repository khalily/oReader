// Package testutil provides shared utilities for testing.
package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestUser represents a test user fixture.
type TestUser struct {
	ID       uint
	Email    string
	Password string // Plain text password for testing
}

// CreateTestUser creates a test user in the database.
// The password is hashed before storage.
//
// Usage:
//
//	user := testutil.CreateTestUser(t, db, "test@example.com")
//	fmt.Println(user.ID) // Use in tests
func CreateTestUser(t *testing.T, db *gorm.DB, email string) TestUser {
	t.Helper()

	// This is a simplified version - adapt to your actual User model
	user := map[string]any{
		"email":        email,
		"password_hash": HashPassword("testpassword123"), // Implement based on your auth
	}

	result := db.Table("users").Create(user)
	require.NoError(t, result.Error, "Failed to create test user")

	var id uint
	db.Table("users").Where("email = ?", email).Select("id").Scan(&id)

	return TestUser{
		ID:       id,
		Email:    email,
		Password: "testpassword123",
	}
}

// TestFeed represents a test feed fixture.
type TestFeed struct {
	ID    uint
	Title string
	Link  string
}

// CreateTestFeed creates a test feed in the database for a user.
//
// Usage:
//
//	user := testutil.CreateTestUser(t, db, "test@example.com")
//	feed := testutil.CreateTestFeed(t, db, user.ID, "https://example.com/feed.xml")
func CreateTestFeed(t *testing.T, db *gorm.DB, userID uint, link string) TestFeed {
	t.Helper()

	feed := map[string]any{
		"user_id":     userID,
		"link":        link,
		"title":       "Test Feed",
		"description": "A test feed for testing",
	}

	result := db.Table("feeds").Create(feed)
	require.NoError(t, result.Error, "Failed to create test feed")

	var id uint
	db.Table("feeds").Where("link = ? AND user_id = ?", link, userID).Select("id").Scan(&id)

	return TestFeed{
		ID:    id,
		Title: "Test Feed",
		Link:  link,
	}
}

// TestItem represents a test item fixture.
type TestItem struct {
	ID      uint
	Title   string
	Link    string
	FeedID  uint
}

// CreateTestItem creates a test item in the database for a feed.
//
// Usage:
//
//	item := testutil.CreateTestItem(t, db, feed.ID, "https://example.com/article/1")
func CreateTestItem(t *testing.T, db *gorm.DB, feedID uint, link string) TestItem {
	t.Helper()

	item := map[string]any{
		"feed_id":     feedID,
		"link":        link,
		"title":       "Test Item",
		"description": "A test item for testing",
	}

	result := db.Table("items").Create(item)
	require.NoError(t, result.Error, "Failed to create test item")

	var id uint
	db.Table("items").Where("link = ? AND feed_id = ?", link, feedID).Select("id").Scan(&id)

	return TestItem{
		ID:     id,
		Title:  "Test Item",
		Link:   link,
		FeedID: feedID,
	}
}

// CleanupTable truncates a table after the test completes.
// Use this to ensure test isolation when not using :memory: database.
//
// Usage:
//
//	db := testutil.SetupTestDB(t)
//	testutil.CleanupTable(t, db, "feeds")
func CleanupTable(t *testing.T, db *gorm.DB, table string) {
	t.Helper()

	t.Cleanup(func() {
		db.Exec("DELETE FROM " + table)
	})
}

// CountRows counts the number of rows in a table with optional conditions.
//
// Usage:
//
//	count := testutil.CountRows(t, db, "feeds", "user_id = ?", user.ID)
//	assert.Equal(t, 2, count)
func CountRows(t *testing.T, db *gorm.DB, table string, where string, args ...any) int64 {
	t.Helper()

	var count int64
	result := db.Table(table).Where(where, args...).Count(&count)
	require.NoError(t, result.Error, "Failed to count rows")

	return count
}

// HashPassword is a placeholder - implement based on your actual password hashing.
// This should be replaced with your actual hashing implementation.
func HashPassword(password string) string {
	// TODO: Replace with actual password hashing
	// Example: return bcrypt.HashPassword(password)
	return "hashed_" + password
}
