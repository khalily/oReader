package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"oreader/internal/model"
)

func setupStatsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.Feed{},
		&model.UserFeed{},
		&model.Item{},
		&model.UserItemState{},
	)
	require.NoError(t, err)

	return db
}

func TestStatsRepository_GetUserStats(t *testing.T) {
	db := setupStatsTestDB(t)
	repo := NewStatsRepository(db)
	ctx := context.Background()

	// Create test user
	user := &model.User{
		Email:        "stats@example.com",
		PasswordHash: "hash",
	}
	err := user.GenerateID()
	require.NoError(t, err)
	err = db.Create(user).Error
	require.NoError(t, err)

	// Create test feed
	feed := &model.Feed{
		FeedURL: "https://example.com/feed.xml",
		Title:   "Test Feed",
	}
	err = feed.GenerateID()
	require.NoError(t, err)
	err = db.Create(feed).Error
	require.NoError(t, err)

	// Subscribe user to feed
	userFeed := &model.UserFeed{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	err = userFeed.GenerateID()
	require.NoError(t, err)
	err = db.Create(userFeed).Error
	require.NoError(t, err)

	t.Run("returns zero stats for user with no items", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), stats.Total)
		assert.Equal(t, int64(0), stats.Unread)
		assert.Equal(t, int64(0), stats.Starred)
		assert.Equal(t, int64(0), stats.Today)
	})

	// Create test items
	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	items := []*model.Item{
		{
			FeedID:  feed.ID,
			GUID:    "item-1",
			Title:   "Today Item 1",
			Link:    "https://example.com/1",
			PubDate: &today,
		},
		{
			FeedID:  feed.ID,
			GUID:    "item-2",
			Title:   "Today Item 2",
			Link:    "https://example.com/2",
			PubDate: &now, // Today
		},
		{
			FeedID:  feed.ID,
			GUID:    "item-3",
			Title:   "Yesterday Item",
			Link:    "https://example.com/3",
			PubDate: &yesterday,
		},
		{
			FeedID:  feed.ID,
			GUID:    "item-4",
			Title:   "Old Item",
			Link:    "https://example.com/4",
			PubDate: nil, // No pub_date
		},
	}

	for _, item := range items {
		err := item.GenerateID()
		require.NoError(t, err)
		err = db.Create(item).Error
		require.NoError(t, err)
	}

	t.Run("returns correct total count", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), stats.Total)
	})

	t.Run("returns correct today count", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), stats.Today)
	})

	// Mark one item as read
	state := &model.UserItemState{
		UserID:    user.ID,
		ItemID:    items[0].ID,
		IsRead:    true,
		IsStarred: false,
	}
	err = state.GenerateID()
	require.NoError(t, err)
	err = db.Create(state).Error
	require.NoError(t, err)

	t.Run("returns correct unread count", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), stats.Unread) // 4 total - 1 read
	})

	// Mark one item as starred
	starState := &model.UserItemState{
		UserID:    user.ID,
		ItemID:    items[1].ID,
		IsRead:    false,
		IsStarred: true,
	}
	err = starState.GenerateID()
	require.NoError(t, err)
	err = db.Create(starState).Error
	require.NoError(t, err)

	t.Run("returns correct starred count", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), stats.Starred)
	})

	t.Run("returns all stats together", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), stats.Total)
		assert.Equal(t, int64(2), stats.Today)
		assert.Equal(t, int64(3), stats.Unread)
		assert.Equal(t, int64(1), stats.Starred)
	})
}

func TestStatsRepository_GetUserStats_NoFeeds(t *testing.T) {
	db := setupStatsTestDB(t)
	repo := NewStatsRepository(db)
	ctx := context.Background()

	// Create user with no feeds
	user := &model.User{
		Email:        "nofeeds@example.com",
		PasswordHash: "hash",
	}
	err := user.GenerateID()
	require.NoError(t, err)
	err = db.Create(user).Error
	require.NoError(t, err)

	stats, err := repo.GetUserStats(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.Total)
	assert.Equal(t, int64(0), stats.Unread)
	assert.Equal(t, int64(0), stats.Starred)
	assert.Equal(t, int64(0), stats.Today)
}

func TestStatsRepository_GetUserStats_OnlyCountsUserFeeds(t *testing.T) {
	db := setupStatsTestDB(t)
	repo := NewStatsRepository(db)
	ctx := context.Background()

	// Create user 1
	user1 := &model.User{
		Email:        "user1@example.com",
		PasswordHash: "hash",
	}
	err := user1.GenerateID()
	require.NoError(t, err)
	err = db.Create(user1).Error
	require.NoError(t, err)

	// Create user 2
	user2 := &model.User{
		Email:        "user2@example.com",
		PasswordHash: "hash",
	}
	err = user2.GenerateID()
	require.NoError(t, err)
	err = db.Create(user2).Error
	require.NoError(t, err)

	// Create feeds for each user
	feed1 := &model.Feed{
		FeedURL: "https://feed1.com/rss.xml",
		Title:   "Feed 1",
	}
	err = feed1.GenerateID()
	require.NoError(t, err)
	err = db.Create(feed1).Error
	require.NoError(t, err)

	feed2 := &model.Feed{
		FeedURL: "https://feed2.com/rss.xml",
		Title:   "Feed 2",
	}
	err = feed2.GenerateID()
	require.NoError(t, err)
	err = db.Create(feed2).Error
	require.NoError(t, err)

	// Subscribe users to their feeds
	userFeed1 := &model.UserFeed{
		UserID: user1.ID,
		FeedID: feed1.ID,
	}
	err = userFeed1.GenerateID()
	require.NoError(t, err)
	err = db.Create(userFeed1).Error
	require.NoError(t, err)

	userFeed2 := &model.UserFeed{
		UserID: user2.ID,
		FeedID: feed2.ID,
	}
	err = userFeed2.GenerateID()
	require.NoError(t, err)
	err = db.Create(userFeed2).Error
	require.NoError(t, err)

	// Create items in feed1
	now := time.Now().UTC()
	item1 := &model.Item{
		FeedID:  feed1.ID,
		GUID:    "feed1-item",
		Title:   "Feed 1 Item",
		Link:    "https://feed1.com/1",
		PubDate: &now,
	}
	err = item1.GenerateID()
	require.NoError(t, err)
	err = db.Create(item1).Error
	require.NoError(t, err)

	// Create items in feed2
	item2 := &model.Item{
		FeedID:  feed2.ID,
		GUID:    "feed2-item",
		Title:   "Feed 2 Item",
		Link:    "https://feed2.com/1",
		PubDate: &now,
	}
	err = item2.GenerateID()
	require.NoError(t, err)
	err = db.Create(item2).Error
	require.NoError(t, err)

	// User 1 should only see items from feed1
	stats1, err := repo.GetUserStats(ctx, user1.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), stats1.Total)
	assert.Equal(t, int64(1), stats1.Today)

	// User 2 should only see items from feed2
	stats2, err := repo.GetUserStats(ctx, user2.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), stats2.Total)
	assert.Equal(t, int64(1), stats2.Today)
}
