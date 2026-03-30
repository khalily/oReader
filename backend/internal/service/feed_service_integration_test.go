//go:build integration
// +build integration

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/infra/rss"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/repository"
	"github.com/khalily/oreader/internal/testutil"
)

// testParser is a mock RSS parser for testing
type testParser struct {
	feed *rss.ParsedFeed
	err  error
}

func (p *testParser) Parse(ctx context.Context, url string) (*rss.ParsedFeed, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.feed, nil
}

// testFeedServiceSetup creates a test environment with in-memory database
func testFeedServiceSetup(t *testing.T) (*feedService, *repository.FeedRepository, *repository.ItemRepository, *repository.UserFeedRepository, *testParser) {
	db := testutil.SetupTestDB(t)
	err := db.AutoMigrate(&model.Feed{}, &model.Item{}, &model.UserFeed{})
	require.NoError(t, err)

	feedRepo := repository.NewFeedRepository(db)
	itemRepo := repository.NewItemRepository(db)
	userFeedRepo := repository.NewUserFeedRepository(db)

	parser := &testParser{}
	svc := NewFeedService(feedRepo, itemRepo, userFeedRepo, parser).(*feedService)
	return svc, feedRepo, itemRepo, userFeedRepo, parser
}

// TestFeedService_Subscribe tests the Subscribe method with actual business logic
func TestFeedService_Subscribe(t *testing.T) {
	ctx := context.Background()

	t.Run("subscribe to new feed successfully", func(t *testing.T) {
		svc, _, _, _, parser := testFeedServiceSetup(t)

		parser.feed = &rss.ParsedFeed{
			Title:       "Test Feed",
			Link:        "https://example.com",
			Description: "A test feed",
			Items: []*rss.ParsedItem{
				{GUID: "item1", Title: "Item 1", Link: "https://example.com/item1"},
				{GUID: "item2", Title: "Item 2", Link: "https://example.com/item2"},
			},
		}

		result, err := svc.Subscribe(ctx, "user1", "https://example.com/feed.xml")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Feed)
		assert.Equal(t, "Test Feed", result.Feed.Title)
		assert.Equal(t, 2, result.NewItemCount)
	})

	t.Run("subscribe to existing feed already subscribed", func(t *testing.T) {
		svc, _, _, _, parser := testFeedServiceSetup(t)

		// First subscription
		parser.feed = &rss.ParsedFeed{
			Title:       "Existing Feed",
			Link:        "https://existing.com",
			Description: "A test feed",
			Items:       []*rss.ParsedItem{},
		}
		_, err := svc.Subscribe(ctx, "user1", "https://example.com/feed.xml")
		require.NoError(t, err)

		// Second subscription should fail
		parser.feed = &rss.ParsedFeed{
			Title:       "Existing Feed",
			Link:        "https://existing.com",
			Description: "A test feed",
			Items:       []*rss.ParsedItem{},
		}
		_, err = svc.Subscribe(ctx, "user1", "https://example.com/feed.xml")
		assert.Error(t, err)
		assert.Equal(t, ErrFeedAlreadySubscribed, err)
	})

	t.Run("subscribe with parser error", func(t *testing.T) {
		svc, _, _, _, parser := testFeedServiceSetup(t)

		parser.err = errors.New("failed to parse feed")

		_, err := svc.Subscribe(ctx, "user1", "https://invalid-url.com/feed.xml")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch")
	})
}

// TestFeedService_GetUserFeeds tests the GetUserFeeds method with actual business logic
func TestFeedService_GetUserFeeds(t *testing.T) {
	ctx := context.Background()

	t.Run("get feeds for user with subscriptions", func(t *testing.T) {
		svc, _, _, _, parser := testFeedServiceSetup(t)

		// Subscribe to two feeds
		parser.feed = &rss.ParsedFeed{
			Title:       "Feed 1",
			Link:        "https://feed1.com",
			Description: "Test feed 1",
			Items:       []*rss.ParsedItem{},
		}
		_, err := svc.Subscribe(ctx, "user1", "https://feed1.com/feed.xml")
		require.NoError(t, err)

		parser.feed = &rss.ParsedFeed{
			Title:       "Feed 2",
			Link:        "https://feed2.com",
			Description: "Test feed 2",
			Items:       []*rss.ParsedItem{},
		}
		_, err = svc.Subscribe(ctx, "user1", "https://feed2.com/feed.xml")
		require.NoError(t, err)

		// Get user feeds
		feeds, total, err := svc.GetUserFeeds(ctx, "user1", ListOptions{Limit: 10, Offset: 0})

		require.NoError(t, err)
		assert.Equal(t, 2, int(total))
		assert.Len(t, feeds, 2)
	})

	t.Run("get feeds for user with no subscriptions", func(t *testing.T) {
		svc, _, _, _, _ := testFeedServiceSetup(t)

		feeds, total, err := svc.GetUserFeeds(ctx, "user-without-feeds", ListOptions{Limit: 10, Offset: 0})

		require.NoError(t, err)
		assert.Empty(t, feeds)
		assert.Equal(t, 0, int(total))
	})
}

// TestFeedService_RefreshFeed tests the RefreshFeed method with actual business logic
func TestFeedService_RefreshFeed(t *testing.T) {
	ctx := context.Background()

	t.Run("refresh feed with new items", func(t *testing.T) {
		svc, _, _, _, parser := testFeedServiceSetup(t)

		// Subscribe to a feed first
		parser.feed = &rss.ParsedFeed{
			Title:       "Test Feed",
			Link:        "https://example.com",
			Description: "Test feed",
			Items: []*rss.ParsedItem{
				{GUID: "item1", Title: "Item 1", Link: "https://example.com/item1"},
			},
		}
		result, err := svc.Subscribe(ctx, "user1", "https://example.com/feed.xml")
		require.NoError(t, err)
		feedID := result.Feed.ID

		// Now refresh with new items
		parser.feed = &rss.ParsedFeed{
			Title:       "Test Feed",
			Link:        "https://example.com",
			Description: "Updated description",
			Items: []*rss.ParsedItem{
				{GUID: "item1", Title: "Item 1", Link: "https://example.com/item1"},
				{GUID: "item2", Title: "Item 2 (New)", Link: "https://example.com/item2"},
				{GUID: "item3", Title: "Item 3 (New)", Link: "https://example.com/item3"},
			},
		}

		refreshResult, err := svc.RefreshFeed(ctx, "user1", feedID)

		require.NoError(t, err)
		assert.NotNil(t, refreshResult)
		assert.Equal(t, 2, refreshResult.NewItemCount) // 2 new items
	})

	t.Run("refresh feed not subscribed", func(t *testing.T) {
		svc, _, _, _, _ := testFeedServiceSetup(t)

		_, err := svc.RefreshFeed(ctx, "user1", "non-existent-feed-id")

		assert.Error(t, err)
		assert.Equal(t, ErrFeedNotFound, err)
	})
}

// TestFeedService_DeleteFeed tests the DeleteFeed method with actual business logic
func TestFeedService_DeleteFeed(t *testing.T) {
	ctx := context.Background()

	t.Run("delete feed successfully", func(t *testing.T) {
		svc, _, _, _, parser := testFeedServiceSetup(t)

		parser.feed = &rss.ParsedFeed{
			Title:       "Test Feed",
			Link:        "https://example.com",
			Description: "Test feed",
			Items:       []*rss.ParsedItem{},
		}
		result, err := svc.Subscribe(ctx, "user1", "https://example.com/feed.xml")
		require.NoError(t, err)
		feedID := result.Feed.ID

		err = svc.DeleteFeed(ctx, "user1", feedID)

		require.NoError(t, err)
	})

	t.Run("delete feed not subscribed", func(t *testing.T) {
		svc, _, _, _, _ := testFeedServiceSetup(t)

		err := svc.DeleteFeed(ctx, "user1", "non-existent-feed-id")

		assert.Error(t, err)
		assert.Equal(t, ErrFeedNotFound, err)
	})
}
