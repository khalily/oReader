package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserModel(t *testing.T) {
	t.Run("valid user", func(t *testing.T) {
		user := User{
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			Nickname:     "Test User",
		}

		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "hashed_password", user.PasswordHash)
		assert.Equal(t, "Test User", user.Nickname)
	})

	t.Run("UUID generation", func(t *testing.T) {
		user := User{
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		}

		require.NoError(t, user.GenerateID())
		assert.NotEmpty(t, user.ID)
		assert.Len(t, user.ID, 36) // UUID format
	})
}

func TestFeedModel(t *testing.T) {
	t.Run("valid feed", func(t *testing.T) {
		feed := Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       "Test Feed",
			Description: "A test feed",
		}

		assert.Equal(t, "https://example.com/feed.xml", feed.FeedURL)
		assert.Equal(t, "Test Feed", feed.Title)
	})

	t.Run("UUID generation", func(t *testing.T) {
		feed := Feed{
			FeedURL: "https://example.com/feed.xml",
			Title:   "Test Feed",
		}

		require.NoError(t, feed.GenerateID())
		assert.NotEmpty(t, feed.ID)
		assert.Len(t, feed.ID, 36)
	})
}

func TestItemModel(t *testing.T) {
	t.Run("valid item", func(t *testing.T) {
		now := time.Now()
		item := Item{
			FeedID:      "feed-uuid",
			GUID:        "item-guid-123",
			Title:       "Test Article",
			Link:        "https://example.com/article",
			Description: "Article description",
			Content:     "Article content",
			PubDate:     &now,
			Creator:     "Author",
		}

		assert.Equal(t, "feed-uuid", item.FeedID)
		assert.Equal(t, "item-guid-123", item.GUID)
		assert.Equal(t, "Test Article", item.Title)
	})

	t.Run("UUID generation", func(t *testing.T) {
		item := Item{
			FeedID: "feed-uuid",
			GUID:   "item-guid-456",
			Title:  "Test Article",
		}

		require.NoError(t, item.GenerateID())
		assert.NotEmpty(t, item.ID)
		assert.Len(t, item.ID, 36)
	})
}

func TestUserFeedModel(t *testing.T) {
	t.Run("valid user feed", func(t *testing.T) {
		userFeed := UserFeed{
			UserID:   "user-uuid",
			FeedID:   "feed-uuid",
			Position: 1,
		}

		assert.Equal(t, "user-uuid", userFeed.UserID)
		assert.Equal(t, "feed-uuid", userFeed.FeedID)
		assert.Equal(t, 1, userFeed.Position)
	})

	t.Run("UUID generation", func(t *testing.T) {
		userFeed := UserFeed{
			UserID: "user-uuid",
			FeedID: "feed-uuid",
		}

		require.NoError(t, userFeed.GenerateID())
		assert.NotEmpty(t, userFeed.ID)
		assert.Len(t, userFeed.ID, 36)
	})
}

func TestUserItemStateModel(t *testing.T) {
	t.Run("valid user item state", func(t *testing.T) {
		state := UserItemState{
			UserID:    "user-uuid",
			ItemID:    "item-uuid",
			IsStarred: true,
			IsRead:    true,
		}

		assert.Equal(t, "user-uuid", state.UserID)
		assert.Equal(t, "item-uuid", state.ItemID)
		assert.True(t, state.IsStarred)
		assert.True(t, state.IsRead)
	})

	t.Run("UUID generation", func(t *testing.T) {
		state := UserItemState{
			UserID: "user-uuid",
			ItemID: "item-uuid",
		}

		require.NoError(t, state.GenerateID())
		assert.NotEmpty(t, state.ID)
		assert.Len(t, state.ID, 36)
	})
}

func TestRefreshTokenModel(t *testing.T) {
	t.Run("valid refresh token", func(t *testing.T) {
		token := RefreshToken{
			UserID:    "user-uuid",
			TokenHash: "hashed-token",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		assert.Equal(t, "user-uuid", token.UserID)
		assert.Equal(t, "hashed-token", token.TokenHash)
		assert.False(t, token.Revoked)
	})

	t.Run("UUID generation", func(t *testing.T) {
		token := RefreshToken{
			UserID:    "user-uuid",
			TokenHash: "hashed-token",
		}

		require.NoError(t, token.GenerateID())
		assert.NotEmpty(t, token.ID)
		assert.Len(t, token.ID, 36)
	})

	t.Run("is expired", func(t *testing.T) {
		token := RefreshToken{
			UserID:    "user-uuid",
			TokenHash: "hashed-token",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}

		assert.True(t, token.IsExpired())

		token.ExpiresAt = time.Now().Add(1 * time.Hour)
		assert.False(t, token.IsExpired())
	})

	t.Run("is valid", func(t *testing.T) {
		token := RefreshToken{
			UserID:    "user-uuid",
			TokenHash: "hashed-token",
			ExpiresAt: time.Now().Add(1 * time.Hour),
			Revoked:   false,
		}

		assert.True(t, token.IsValid())

		token.Revoked = true
		assert.False(t, token.IsValid())

		token.Revoked = false
		token.ExpiresAt = time.Now().Add(-1 * time.Hour)
		assert.False(t, token.IsValid())
	})
}

func TestImportJobModel(t *testing.T) {
	t.Run("valid import job", func(t *testing.T) {
		job := ImportJob{
			UserID:      "user-uuid",
			Status:      ImportJobStatusPending,
			TotalFeeds:  10,
			Processed:   0,
			Failed:      0,
		}

		assert.Equal(t, "user-uuid", job.UserID)
		assert.Equal(t, ImportJobStatusPending, job.Status)
		assert.Equal(t, 10, job.TotalFeeds)
	})

	t.Run("UUID generation", func(t *testing.T) {
		job := ImportJob{
			UserID: "user-uuid",
		}

		require.NoError(t, job.GenerateID())
		assert.NotEmpty(t, job.ID)
		assert.Len(t, job.ID, 36)
	})
}
