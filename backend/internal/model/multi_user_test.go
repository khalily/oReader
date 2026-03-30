package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/testutil"
)

// setupTestDB creates a MySQL database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	// Auto-migrate all models
	err := db.AutoMigrate(&User{}, &Feed{}, &UserFeed{}, &Item{}, &UserItemState{})
	require.NoError(t, err, "Failed to migrate test database")

	return db
}

// TestMultiUserScenario verifies that two users subscribing to the same feed
// have independent read/star states for the same item
func TestMultiUserScenario(t *testing.T) {
	db := setupTestDB(t)

	// Step 1: Create two users
	user1 := &User{
		Email:        "user1@example.com",
		PasswordHash: "hash1",
		Nickname:     "User One",
		AuthProvider: "email",
	}
	require.NoError(t, user1.GenerateID())

	user2 := &User{
		Email:        "user2@example.com",
		PasswordHash: "hash2",
		Nickname:     "User Two",
		AuthProvider: "email",
	}
	require.NoError(t, user2.GenerateID())

	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)

	t.Logf("Created users: %s, %s", user1.ID, user2.ID)

	// Step 2: Both users subscribe to the same feed
	feed := &Feed{
		FeedURL:     "https://example.com/rss.xml",
		Title:       "Shared Feed",
		Description: "A feed subscribed by multiple users",
	}
	require.NoError(t, feed.GenerateID())
	require.NoError(t, db.Create(feed).Error)

	userFeed1 := &UserFeed{UserID: user1.ID, FeedID: feed.ID, Position: 0}
	require.NoError(t, userFeed1.GenerateID())
	require.NoError(t, db.Create(userFeed1).Error)

	userFeed2 := &UserFeed{UserID: user2.ID, FeedID: feed.ID, Position: 1}
	require.NoError(t, userFeed2.GenerateID())
	require.NoError(t, db.Create(userFeed2).Error)

	t.Logf("Both users subscribed to feed: %s", feed.ID)

	// Step 3: Create one item in the feed (shared by all subscribers)
	item := &Item{
		FeedID:      feed.ID,
		GUID:        "article-123",
		Title:       "Shared Article",
		Link:        "https://example.com/article-123",
		Description: "An article read by multiple users",
	}
	require.NoError(t, item.GenerateID())
	require.NoError(t, db.Create(item).Error)

	t.Logf("Created item: %s", item.ID)

	// Step 4: User1 marks item as read (but NOT starred)
	now := time.Now()
	state1 := &UserItemState{
		UserID: user1.ID,
		ItemID: item.ID,
		IsRead: true,
		ReadAt: &now,
	}
	require.NoError(t, state1.GenerateID())
	require.NoError(t, db.Create(state1).Error)

	// Step 5: User2 marks item as starred (but NOT read)
	state2 := &UserItemState{
		UserID:    user2.ID,
		ItemID:    item.ID,
		IsStarred: true,
		IsRead:    false,
	}
	require.NoError(t, state2.GenerateID())
	require.NoError(t, db.Create(state2).Error)

	t.Log("User1 marked item as READ, User2 marked item as STARRED")

	// Step 6: Verify states are independent
	// Fetch User1's state
	var fetchedState1 UserItemState
	err := db.Where("user_id = ? AND item_id = ?", user1.ID, item.ID).First(&fetchedState1).Error
	require.NoError(t, err)

	// Fetch User2's state
	var fetchedState2 UserItemState
	err = db.Where("user_id = ? AND item_id = ?", user2.ID, item.ID).First(&fetchedState2).Error
	require.NoError(t, err)

	// Assertions - User1: read=true, starred=false
	assert.True(t, fetchedState1.IsRead, "User1's item should be read")
	assert.False(t, fetchedState1.IsStarred, "User1's item should NOT be starred")

	// Assertions - User2: read=false, starred=true
	assert.False(t, fetchedState2.IsRead, "User2's item should NOT be read")
	assert.True(t, fetchedState2.IsStarred, "User2's item should be starred")

	t.Log("SUCCESS: User states are independent!")

	// Step 7: Verify we can update User1's state without affecting User2
	readAt := time.Now()
	db.Model(&fetchedState1).Updates(map[string]interface{}{
		"is_starred": true,
		"read_at":    &readAt,
	})

	// Refetch User2's state to ensure it wasn't affected
	var refetchedState2 UserItemState
	err = db.Where("user_id = ? AND item_id = ?", user2.ID, item.ID).First(&refetchedState2).Error
	require.NoError(t, err)

	assert.False(t, refetchedState2.IsRead, "User2's read state should still be false after User1 update")
	assert.True(t, refetchedState2.IsStarred, "User2's starred state should still be true")

	t.Log("SUCCESS: User1's update did not affect User2's state!")
}

// TestUniqueUserItemStateConstraint verifies that a user can only have one state per item
func TestUniqueUserItemStateConstraint(t *testing.T) {
	db := setupTestDB(t)

	user := &User{Email: "test@example.com", PasswordHash: "hash"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, db.Create(user).Error)

	feed := &Feed{FeedURL: "https://example.com/rss.xml", Title: "Test"}
	require.NoError(t, feed.GenerateID())
	require.NoError(t, db.Create(feed).Error)

	item := &Item{FeedID: feed.ID, GUID: "item-1", Title: "Test Item"}
	require.NoError(t, item.GenerateID())
	require.NoError(t, db.Create(item).Error)

	// Create first state
	state1 := &UserItemState{UserID: user.ID, ItemID: item.ID, IsRead: true}
	require.NoError(t, state1.GenerateID())
	require.NoError(t, db.Create(state1).Error)

	// Try to create duplicate state - should fail due to unique constraint
	state2 := &UserItemState{UserID: user.ID, ItemID: item.ID, IsRead: false}
	require.NoError(t, state2.GenerateID())
	err := db.Create(state2).Error

	assert.Error(t, err, "Should not allow duplicate user_id + item_id combination")
	t.Logf("Correctly prevented duplicate: %v", err)
}

// TestMultipleUsersSameFeedCount verifies the feed sharing works correctly
func TestMultipleUsersSameFeedCount(t *testing.T) {
	db := setupTestDB(t)

	// Create 3 users
	users := make([]*User, 3)
	for i := 0; i < 3; i++ {
		users[i] = &User{
			Email:        "user" + string(rune('1'+i)) + "@example.com",
			PasswordHash: "hash",
		}
		require.NoError(t, users[i].GenerateID())
		require.NoError(t, db.Create(users[i]).Error)
	}

	// Create one feed
	feed := &Feed{FeedURL: "https://example.com/shared.xml", Title: "Shared"}
	require.NoError(t, feed.GenerateID())
	require.NoError(t, db.Create(feed).Error)

	// All 3 users subscribe to the same feed
	for _, user := range users {
		userFeed := &UserFeed{UserID: user.ID, FeedID: feed.ID}
		require.NoError(t, userFeed.GenerateID())
		require.NoError(t, db.Create(userFeed).Error)
	}

	// Verify: 1 feed, 3 user_feeds
	var feedCount int64
	db.Model(&Feed{}).Count(&feedCount)
	assert.Equal(t, int64(1), feedCount, "Should have exactly 1 feed")

	var userFeedCount int64
	db.Model(&UserFeed{}).Where("feed_id = ?", feed.ID).Count(&userFeedCount)
	assert.Equal(t, int64(3), userFeedCount, "Should have 3 user_feed subscriptions for the same feed")

	t.Log("SUCCESS: Multiple users can subscribe to the same feed!")
}
