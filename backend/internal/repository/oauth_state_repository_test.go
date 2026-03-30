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

func setupOAuthStateTestDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	// Drop table to ensure clean schema with latest model changes
	_ = db.Migrator().DropTable(&model.OAuthState{})

	err := db.AutoMigrate(&model.OAuthState{})
	require.NoError(t, err)

	return db
}

func TestOAuthStateRepository_Create(t *testing.T) {
	db := setupOAuthStateTestDB(t)
	repo := NewOAuthStateRepository(db)
	ctx := context.Background()

	t.Run("create oauth state successfully", func(t *testing.T) {
		state := &model.OAuthState{
			State:     "test_state_123",
			Provider:  "github",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err := state.GenerateID()
		require.NoError(t, err)

		err = repo.Create(ctx, state)
		assert.NoError(t, err)

		// Verify state was created
		var found model.OAuthState
		err = db.First(&found, "id = ?", state.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "test_state_123", found.State)
		assert.Equal(t, "github", found.Provider)
	})

	t.Run("create state with duplicate value fails", func(t *testing.T) {
		state1 := &model.OAuthState{
			State:     "duplicate_state",
			Provider:  "github",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err := state1.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, state1)
		require.NoError(t, err)

		state2 := &model.OAuthState{
			State:     "duplicate_state",
			Provider:  "github",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err = state2.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, state2)
		assert.Error(t, err) // Unique constraint violation
	})
}

func TestOAuthStateRepository_GetByState(t *testing.T) {
	db := setupOAuthStateTestDB(t)
	repo := NewOAuthStateRepository(db)
	ctx := context.Background()

	t.Run("get state by value", func(t *testing.T) {
		state := &model.OAuthState{
			State:     "get_test_state",
			Provider:  "github",
			UserID:    "user123",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err := state.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, state)
		require.NoError(t, err)

		found, err := repo.GetByState(ctx, "get_test_state")
		assert.NoError(t, err)
		assert.Equal(t, "get_test_state", found.State)
		assert.Equal(t, "github", found.Provider)
		assert.Equal(t, "user123", found.UserID)
	})

	t.Run("get non-existent state returns nil", func(t *testing.T) {
		found, err := repo.GetByState(ctx, "nonexistent_state")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("get expired state still returns it", func(t *testing.T) {
		state := &model.OAuthState{
			State:     "expired_state",
			Provider:  "github",
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		}
		err := state.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, state)
		require.NoError(t, err)

		found, err := repo.GetByState(ctx, "expired_state")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.True(t, found.IsExpired())
	})
}

func TestOAuthStateRepository_Delete(t *testing.T) {
	db := setupOAuthStateTestDB(t)
	repo := NewOAuthStateRepository(db)
	ctx := context.Background()

	t.Run("delete state successfully", func(t *testing.T) {
		state := &model.OAuthState{
			State:     "delete_test_state",
			Provider:  "github",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err := state.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, state)
		require.NoError(t, err)

		err = repo.Delete(ctx, "delete_test_state")
		assert.NoError(t, err)

		// Verify state is deleted
		found, err := repo.GetByState(ctx, "delete_test_state")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("delete non-existent state returns no error", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent_delete_state")
		assert.NoError(t, err)
	})
}

func TestOAuthStateRepository_DeleteExpired(t *testing.T) {
	db := setupOAuthStateTestDB(t)
	repo := NewOAuthStateRepository(db)
	ctx := context.Background()

	t.Run("delete expired states only", func(t *testing.T) {
		// Create expired state
		expiredState := &model.OAuthState{
			State:     "expired_state_to_delete",
			Provider:  "github",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		err := expiredState.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, expiredState)
		require.NoError(t, err)

		// Create valid state
		validState := &model.OAuthState{
			State:     "valid_state_to_keep",
			Provider:  "github",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err = validState.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, validState)
		require.NoError(t, err)

		// Delete expired states
		err = repo.DeleteExpired(ctx)
		assert.NoError(t, err)

		// Verify expired state is deleted
		found, err := repo.GetByState(ctx, "expired_state_to_delete")
		assert.NoError(t, err)
		assert.Nil(t, found)

		// Verify valid state still exists
		found, err = repo.GetByState(ctx, "valid_state_to_keep")
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("delete expired when no expired states exists", func(t *testing.T) {
		validState := &model.OAuthState{
			State:     "another_valid_state",
			Provider:  "github",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err := validState.GenerateID()
		require.NoError(t, err)
		err = repo.Create(ctx, validState)
		require.NoError(t, err)

		// Should not error
		err = repo.DeleteExpired(ctx)
		assert.NoError(t, err)

		// Verify state still exists
		found, err := repo.GetByState(ctx, "another_valid_state")
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}
