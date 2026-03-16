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

func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.User{})
	require.NoError(t, err)

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("create user successfully", func(t *testing.T) {
		user := &model.User{
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			Nickname:     "TestUser",
			AuthProvider: "email",
		}
		err := user.GenerateID()
		require.NoError(t, err)

		err = repo.Create(ctx, user)
		assert.NoError(t, err)

		// Verify user was created
		var found model.User
		err = db.First(&found, "id = ?", user.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.PasswordHash, found.PasswordHash)
	})

	t.Run("create user with duplicate email fails", func(t *testing.T) {
		user1 := &model.User{
			Email:        "duplicate@example.com",
			PasswordHash: "hash1",
		}
		err := user1.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, user1)
		require.NoError(t, err)

		user2 := &model.User{
			Email:        "duplicate@example.com",
			PasswordHash: "hash2",
		}
		err = user2.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, user2)
		assert.Error(t, err) // Unique constraint violation
	})

	t.Run("create user with github id", func(t *testing.T) {
		user := &model.User{
			Email:        "github@example.com",
			AuthProvider: "github",
			GitHubID:     "github-12345",
		}
		err := user.GenerateID()
		require.NoError(t, err)

		err = repo.Create(ctx, user)
		assert.NoError(t, err)
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("get existing user", func(t *testing.T) {
		user := &model.User{
			Email:        "getbyid@example.com",
			PasswordHash: "hash",
			Nickname:     "GetByIDUser",
		}
		err := user.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.Nickname, found.Nickname)
	})

	t.Run("get non-existent user returns nil", func(t *testing.T) {
		found, err := repo.GetByID(ctx, "non-existent-id")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("get user by email", func(t *testing.T) {
		user := &model.User{
			Email:        "getbyemail@example.com",
			PasswordHash: "hash",
		}
		err := user.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.GetByEmail(ctx, "getbyemail@example.com")
		assert.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("get non-existent email returns nil", func(t *testing.T) {
		found, err := repo.GetByEmail(ctx, "nonexistent@example.com")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserRepository_GetByGitHubID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("get user by github id", func(t *testing.T) {
		user := &model.User{
			Email:        "githubuser@example.com",
			AuthProvider: "github",
			GitHubID:     "github-98765",
		}
		err := user.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.GetByGitHubID(ctx, "github-98765")
		assert.NoError(t, err)
		assert.Equal(t, user.GitHubID, found.GitHubID)
	})

	t.Run("get non-existent github id returns nil", func(t *testing.T) {
		found, err := repo.GetByGitHubID(ctx, "nonexistent-github-id")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestUserRepository_Update(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("update user successfully", func(t *testing.T) {
		user := &model.User{
			Email:        "update@example.com",
			PasswordHash: "old_hash",
			Nickname:     "OldNickname",
		}
		err := user.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, user)
		require.NoError(t, err)

		// Update fields
		user.Nickname = "NewNickname"
		user.AvatarURL = "https://example.com/avatar.png"
		time.Sleep(10 * time.Millisecond) // Ensure UpdatedAt changes
		err = repo.Update(ctx, user)
		assert.NoError(t, err)

		// Verify update
		found, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, "NewNickname", found.Nickname)
		assert.Equal(t, "https://example.com/avatar.png", found.AvatarURL)
	})

	t.Run("update non-existent user creates it (GORM Save upsert)", func(t *testing.T) {
		user := &model.User{
			Email:        "upsert@example.com",
			PasswordHash: "hash",
		}
		err := user.GenerateID()
		require.NoError(t, err)

		// GORM Save creates the record if it doesn't exist
		err = repo.Update(ctx, user)
		assert.NoError(t, err)

		// Verify the user was created
		found, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
	})
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("soft delete user", func(t *testing.T) {
		user := &model.User{
			Email:        "delete@example.com",
			PasswordHash: "hash",
		}
		err := user.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, user)
		require.NoError(t, err)

		err = repo.Delete(ctx, user.ID)
		assert.NoError(t, err)

		// Verify soft delete - normal query should not find it
		found, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Nil(t, found)

		// Verify soft delete - unscoped query should find it
		var deleted model.User
		err = db.Unscoped().First(&deleted, "id = ?", user.ID).Error
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
	})

	t.Run("delete non-existent user returns no error", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-id")
		assert.NoError(t, err) // GORM doesn't error on delete of non-existent record
	})
}
