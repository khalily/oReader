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

func setupPendingOAuthTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Drop table to ensure clean schema with latest model changes
	_ = db.Migrator().DropTable(&model.PendingOAuth{})

	err = db.AutoMigrate(&model.PendingOAuth{})
	require.NoError(t, err)

	return db
}

func TestPendingOAuthRepository_Create(t *testing.T) {
	db := setupPendingOAuthTestDB(t)
	repo := NewPendingOAuthRepository(db)
	ctx := context.Background()

	t.Run("create pending oauth successfully", func(t *testing.T) {
		pending := &model.PendingOAuth{
			Token:       "test-token-64-chars-padding-padding-padding-padding-padding-padding",
			GitHubID:    "12345",
			GitHubLogin: "testuser",
			Nickname:    "Test User",
			AvatarURL:   "https://example.com/avatar.png",
			ExpiresAt:   time.Now().Add(5 * time.Minute),
		}
		err := pending.GenerateID()
		require.NoError(t, err)

		err = repo.Create(ctx, pending)
		assert.NoError(t, err)

		// Verify pending oauth was created
		var found model.PendingOAuth
		err = db.First(&found, "id = ?", pending.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "test-token-64-chars-padding-padding-padding-padding-padding-padding", found.Token)
		assert.Equal(t, "12345", found.GitHubID)
		assert.Equal(t, "testuser", found.GitHubLogin)
	})

	t.Run("create with duplicate token fails", func(t *testing.T) {
		pending1 := &model.PendingOAuth{
			Token:       "duplicate-token-padding-padding-padding-padding-padding-padding-12",
			GitHubID:    "11111",
			GitHubLogin: "user1",
			ExpiresAt:   time.Now().Add(5 * time.Minute),
		}
		err := pending1.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, pending1)
		require.NoError(t, err)

		pending2 := &model.PendingOAuth{
			Token:       "duplicate-token-padding-padding-padding-padding-padding-padding-12",
			GitHubID:    "22222",
			GitHubLogin: "user2",
			ExpiresAt:   time.Now().Add(5 * time.Minute),
		}
		err = pending2.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, pending2)
		assert.Error(t, err) // Unique constraint violation
	})
}

func TestPendingOAuthRepository_GetByToken(t *testing.T) {
	db := setupPendingOAuthTestDB(t)
	repo := NewPendingOAuthRepository(db)
	ctx := context.Background()

	t.Run("get pending oauth by token", func(t *testing.T) {
		pending := &model.PendingOAuth{
			Token:       "get-test-token-padding-padding-padding-padding-padding-padding-34",
			GitHubID:    "12345",
			GitHubLogin: "testuser",
			Nickname:    "Test User",
			AvatarURL:   "https://example.com/avatar.png",
			ExpiresAt:   time.Now().Add(5 * time.Minute),
		}
		err := pending.GenerateID()
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, pending))

		found, err := repo.GetByToken(ctx, pending.Token)
		assert.NoError(t, err)
		assert.Equal(t, pending.GitHubID, found.GitHubID)
		assert.Equal(t, pending.GitHubLogin, found.GitHubLogin)
		assert.Equal(t, pending.Nickname, found.Nickname)
		assert.Equal(t, pending.AvatarURL, found.AvatarURL)
	})

	t.Run("get non-existent token returns nil", func(t *testing.T) {
		found, err := repo.GetByToken(ctx, "non-existent-token-padding-padding-padding-padding-padding")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("get expired pending oauth still returns it", func(t *testing.T) {
		pending := &model.PendingOAuth{
			Token:       "expired-token-padding-padding-padding-padding-padding-padding-56",
			GitHubID:    "99999",
			GitHubLogin: "expireduser",
			ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired
		}
		err := pending.GenerateID()
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, pending))

		found, err := repo.GetByToken(ctx, pending.Token)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.True(t, found.IsExpired())
	})
}

func TestPendingOAuthRepository_Delete(t *testing.T) {
	db := setupPendingOAuthTestDB(t)
	repo := NewPendingOAuthRepository(db)
	ctx := context.Background()

	t.Run("delete pending oauth successfully", func(t *testing.T) {
		pending := &model.PendingOAuth{
			Token:       "delete-test-token-padding-padding-padding-padding-padding-padding",
			GitHubID:    "12345",
			GitHubLogin: "testuser",
			ExpiresAt:   time.Now().Add(5 * time.Minute),
		}
		err := pending.GenerateID()
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, pending))

		err = repo.Delete(ctx, pending.Token)
		assert.NoError(t, err)

		// Verify deleted
		found, err := repo.GetByToken(ctx, pending.Token)
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("delete non-existent token returns no error", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-token-padding-padding-padding-padding-padding")
		assert.NoError(t, err)
	})
}

func TestPendingOAuth_IsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		pending := &model.PendingOAuth{
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		assert.False(t, pending.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		pending := &model.PendingOAuth{
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		assert.True(t, pending.IsExpired())
	})
}
