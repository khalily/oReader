//go:build integration
// +build integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"oreader/internal/model"
	"oreader/internal/repository"
)

// itemIntegrationTestSetup creates a test environment with in-memory database
func itemIntegrationTestSetup(t *testing.T) (ItemService, *repository.ItemRepository, *repository.UserFeedRepository, string, string) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.User{}, &model.Feed{}, &model.Item{}, &model.UserFeed{}, &model.UserItemState{})
	require.NoError(t, err)

	// Create test user
	user := &model.User{Email: "test@example.com"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, db.Create(user).Error)

	// Create test feed
	feed := &model.Feed{
		Title:   "Test Feed",
		Link:    "https://example.com/feed.xml",
		FeedURL: "https://example.com/feed.xml",
	}
	require.NoError(t, feed.GenerateID())
	require.NoError(t, db.Create(feed).Error)

	// Create user-feed subscription
	userFeed := &model.UserFeed{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	require.NoError(t, userFeed.GenerateID())
	require.NoError(t, db.Create(userFeed).Error)

	// Create test items with different pub_dates
	now := time.Now()
	items := []*model.Item{
		{
			FeedID:  feed.ID,
			Title:   "Item 1 - Oldest",
			Link:    "https://example.com/item1",
			GUID:    "guid-1",
			PubDate: ptrTime(now.Add(-2 * time.Hour)), // 2 hours ago
		},
		{
			FeedID:  feed.ID,
			Title:   "Item 2 - Newest",
			Link:    "https://example.com/item2",
			GUID:    "guid-2",
			PubDate: ptrTime(now), // Now
		},
		{
			FeedID:  feed.ID,
			Title:   "Item 3 - Middle",
			Link:    "https://example.com/item3",
			GUID:    "guid-3",
			PubDate: ptrTime(now.Add(-1 * time.Hour)), // 1 hour ago
		},
		{
			FeedID:  feed.ID,
			Title:   "Item 4 - Null PubDate",
			Link:    "https://example.com/item4",
			GUID:    "guid-4",
			PubDate: nil, // Test NULLS LAST
		},
	}

	for _, item := range items {
		require.NoError(t, item.GenerateID())
		require.NoError(t, db.Create(item).Error)
	}

	itemRepo := repository.NewItemRepository(db)
	userFeedRepo := repository.NewUserFeedRepository(db)
	stateRepo := repository.NewUserItemStateRepository(db)

	svc := NewItemService(itemRepo, stateRepo, userFeedRepo)

	return svc, itemRepo, userFeedRepo, user.ID, feed.ID
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// =============================================================================
// SPEC COMPLIANCE INTEGRATION TESTS
// =============================================================================

// TestItemService_Integration_SetStar_SetsValue tests that SetStar sets the value (not toggles)
func TestItemService_Integration_SetStar_SetsValue(t *testing.T) {
	svc, _, _, userID, feedID := itemIntegrationTestSetup(t)
	ctx := context.Background()

	// Get items to find an item ID
	result, err := svc.ListItems(ctx, userID, ListItemOptions{FeedID: feedID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 4)

	itemID := result.Items[0].ID

	// Set star to true
	item, err := svc.SetStar(ctx, userID, itemID, true)
	require.NoError(t, err)
	assert.True(t, item.IsStarred, "First SetStar(true) should set starred to true")

	// Set star to true again (idempotent)
	item, err = svc.SetStar(ctx, userID, itemID, true)
	require.NoError(t, err)
	assert.True(t, item.IsStarred, "Second SetStar(true) should keep starred as true")

	// Set star to false
	item, err = svc.SetStar(ctx, userID, itemID, false)
	require.NoError(t, err)
	assert.False(t, item.IsStarred, "SetStar(false) should set starred to false")

	// Set star to false again (idempotent)
	item, err = svc.SetStar(ctx, userID, itemID, false)
	require.NoError(t, err)
	assert.False(t, item.IsStarred, "Second SetStar(false) should keep starred as false")
}

// TestItemService_Integration_SetRead_SetsValue tests that SetRead sets the value (not toggles)
func TestItemService_Integration_SetRead_SetsValue(t *testing.T) {
	svc, _, _, userID, feedID := itemIntegrationTestSetup(t)
	ctx := context.Background()

	// Get items to find an item ID
	result, err := svc.ListItems(ctx, userID, ListItemOptions{FeedID: feedID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 4)

	itemID := result.Items[0].ID

	// Set read to true
	item, err := svc.SetRead(ctx, userID, itemID, true)
	require.NoError(t, err)
	assert.True(t, item.IsRead, "First SetRead(true) should set read to true")
	assert.NotNil(t, item.ReadAt, "SetRead(true) should set read_at timestamp")

	// Set read to true again (idempotent)
	item, err = svc.SetRead(ctx, userID, itemID, true)
	require.NoError(t, err)
	assert.True(t, item.IsRead, "Second SetRead(true) should keep read as true")

	// Set read to false
	item, err = svc.SetRead(ctx, userID, itemID, false)
	require.NoError(t, err)
	assert.False(t, item.IsRead, "SetRead(false) should set read to false")
	assert.Nil(t, item.ReadAt, "SetRead(false) should clear read_at timestamp")

	// Set read to false again (idempotent)
	item, err = svc.SetRead(ctx, userID, itemID, false)
	require.NoError(t, err)
	assert.False(t, item.IsRead, "Second SetRead(false) should keep read as false")
}

// TestItemService_Integration_SortsByPubDateDescNullsLast tests sorting spec compliance
func TestItemService_Integration_SortsByPubDateDescNullsLast(t *testing.T) {
	svc, _, _, userID, feedID := itemIntegrationTestSetup(t)
	ctx := context.Background()

	// List items
	result, err := svc.ListItems(ctx, userID, ListItemOptions{FeedID: feedID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 4)

	// Verify sorting: newest first, nulls last
	// Expected order: Item 2 (now), Item 3 (1h ago), Item 1 (2h ago), Item 4 (null)
	assert.Equal(t, "Item 2 - Newest", result.Items[0].Title, "First item should be newest")
	assert.Equal(t, "Item 3 - Middle", result.Items[1].Title, "Second item should be middle")
	assert.Equal(t, "Item 1 - Oldest", result.Items[2].Title, "Third item should be oldest")
	assert.Equal(t, "Item 4 - Null PubDate", result.Items[3].Title, "Last item should be null pub_date")
}

// TestItemService_Integration_IncludesFeedTitle tests feed_title in response
func TestItemService_Integration_IncludesFeedTitle(t *testing.T) {
	svc, _, _, userID, feedID := itemIntegrationTestSetup(t)
	ctx := context.Background()

	// List items
	result, err := svc.ListItems(ctx, userID, ListItemOptions{FeedID: feedID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 4)

	// All items should have feed_title
	for _, item := range result.Items {
		assert.Equal(t, "Test Feed", item.FeedTitle, "Each item should include feed_title")
	}

	// Get single item
	singleItem, err := svc.GetItem(ctx, userID, result.Items[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Feed", singleItem.FeedTitle, "GetItem should include feed_title")
}

// TestItemService_Integration_AccessControl tests user isolation
func TestItemService_Integration_AccessControl(t *testing.T) {
	svc, _, _, userID, feedID := itemIntegrationTestSetup(t)
	ctx := context.Background()

	// Get items
	result, err := svc.ListItems(ctx, userID, ListItemOptions{FeedID: feedID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 4)

	itemID := result.Items[0].ID

	// Try to access with different user (should fail)
	_, err = svc.GetItem(ctx, "different-user-id", itemID)
	assert.Error(t, err, "Should not allow access to items from feeds user is not subscribed to")
	assert.ErrorIs(t, err, ErrItemNotFound)
}

// TestItemService_Integration_MarkAllRead tests marking all items as read
func TestItemService_Integration_MarkAllRead(t *testing.T) {
	svc, _, _, userID, feedID := itemIntegrationTestSetup(t)
	ctx := context.Background()

	// Mark all as read
	count, err := svc.MarkAllRead(ctx, userID, feedID)
	require.NoError(t, err)
	assert.Equal(t, 4, count, "Should mark all 4 items as read")

	// Verify all items are read
	result, err := svc.ListItems(ctx, userID, ListItemOptions{FeedID: feedID, Read: ptrBool(false), Limit: 10})
	require.NoError(t, err)
	assert.Len(t, result.Items, 0, "No unread items should remain")
}

func ptrBool(b bool) *bool {
	return &b
}
