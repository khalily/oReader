package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/testutil"
)

func setupRefreshTokenTestDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	err := db.AutoMigrate(&model.User{}, &model.RefreshToken{})
	require.NoError(t, err)

	return db
}

func createTestUserForRefreshToken(t *testing.T, db *gorm.DB) *model.User {
	user := &model.User{
		Email:        "refresh@example.com",
		PasswordHash: "hash",
	}
	err := user.GenerateID()
	require.NoError(t, err)
	err = db.Create(user).Error
	require.NoError(t, err)
	return user
}

func TestRefreshTokenRepository_Create(t *testing.T) {
	db := setupRefreshTokenTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()
	user := createTestUserForRefreshToken(t, db)

	t.Run("create refresh token successfully", func(t *testing.T) {
		token := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: "hash1",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := token.GenerateID()
		require.NoError(t, err)

		err = repo.Create(ctx, token)
		assert.NoError(t, err)

		// Verify token was created
		var found model.RefreshToken
		err = db.First(&found, "id = ?", token.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, user.ID, found.UserID)
		assert.Equal(t, "hash1", found.TokenHash)
	})

	t.Run("create token with duplicate hash fails", func(t *testing.T) {
		token1 := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: "duplicate_hash",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := token1.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, token1)
		require.NoError(t, err)

		token2 := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: "duplicate_hash",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err = token2.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, token2)
		assert.Error(t, err) // Unique constraint violation
	})
}

func TestRefreshTokenRepository_GetByTokenHash(t *testing.T) {
	db := setupRefreshTokenTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()
	user := createTestUserForRefreshToken(t, db)

	t.Run("get token by hash", func(t *testing.T) {
		token := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: "hash_get_test",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := token.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, token)
		require.NoError(t, err)

		found, err := repo.GetByTokenHash(ctx, "hash_get_test")
		assert.NoError(t, err)
		assert.Equal(t, token.TokenHash, found.TokenHash)
		assert.Equal(t, user.ID, found.UserID)
	})

	t.Run("get non-existent token returns nil", func(t *testing.T) {
		found, err := repo.GetByTokenHash(ctx, "nonexistent_hash")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	db := setupRefreshTokenTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()
	user := createTestUserForRefreshToken(t, db)

	t.Run("revoke token successfully", func(t *testing.T) {
		token := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: "hash_revoke",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			Revoked:   false,
		}
		err := token.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, token)
		require.NoError(t, err)

		err = repo.Revoke(ctx, "hash_revoke")
		assert.NoError(t, err)

		// Verify token is revoked
		found, err := repo.GetByTokenHash(ctx, "hash_revoke")
		assert.NoError(t, err)
		assert.True(t, found.Revoked)
	})

	t.Run("revoke non-existent token returns no error", func(t *testing.T) {
		err := repo.Revoke(ctx, "nonexistent_revoke_hash")
		assert.NoError(t, err)
	})
}

func TestRefreshTokenRepository_RevokeAllByUser(t *testing.T) {
	db := setupRefreshTokenTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()
	user := createTestUserForRefreshToken(t, db)

	// Create another user for multi-user testing
	user2 := &model.User{
		Email:        "refresh2@example.com",
		PasswordHash: "hash2",
	}
	err := user2.GenerateID()
	require.NoError(t, err)
	err = db.Create(user2).Error
	require.NoError(t, err)

	t.Run("revoke all tokens for user", func(t *testing.T) {
		// Create multiple tokens for user1
		for i := 0; i < 3; i++ {
			token := &model.RefreshToken{
				UserID:    user.ID,
				TokenHash: "user1_hash_" + string(rune('a'+i)),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			err := token.GenerateID()
			require.NoError(t, err)
			err = repo.Create(ctx, token)
			require.NoError(t, err)
		}

		// Create a token for user2
		token2 := &model.RefreshToken{
			UserID:    user2.ID,
			TokenHash: "user2_hash_a",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := token2.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, token2)
		require.NoError(t, err)

		// Revoke all tokens for user1
		err = repo.RevokeAllByUser(ctx, user.ID)
		assert.NoError(t, err)

		// Verify all user1 tokens are revoked
		var user1Tokens []model.RefreshToken
		err = db.Where("user_id = ?", user.ID).Find(&user1Tokens).Error
		assert.NoError(t, err)
		for _, tok := range user1Tokens {
			assert.True(t, tok.Revoked)
		}

		// Verify user2 token is NOT revoked
		var user2Tokens []model.RefreshToken
		err = db.Where("user_id = ?", user2.ID).Find(&user2Tokens).Error
		assert.NoError(t, err)
		assert.Len(t, user2Tokens, 1)
		assert.False(t, user2Tokens[0].Revoked)
	})
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	db := setupRefreshTokenTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()
	user := createTestUserForRefreshToken(t, db)

	t.Run("delete expired tokens only", func(t *testing.T) {
		// Create expired token
		expiredToken := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: "expired_hash",
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired yesterday
		}
		err := expiredToken.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, expiredToken)
		require.NoError(t, err)

		// Create valid token
		validToken := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: "valid_hash",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err = validToken.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, validToken)
		require.NoError(t, err)

		// Delete expired tokens
		err = repo.DeleteExpired(ctx)
		assert.NoError(t, err)

		// Verify expired token is deleted
		found, err := repo.GetByTokenHash(ctx, "expired_hash")
		assert.NoError(t, err)
		assert.Nil(t, found)

		// Verify valid token still exists
		found, err = repo.GetByTokenHash(ctx, "valid_hash")
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}
