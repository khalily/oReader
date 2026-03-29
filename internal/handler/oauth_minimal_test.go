package handler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"oreader/internal/model"
	"oreader/internal/testutil"
)

func TestOAuthUserCreation(t *testing.T) {
	db := testutil.SetupTestDB(t)

	_ = db.Migrator().DropTable(&model.User{})
	err := db.AutoMigrate(&model.User{})
	require.NoError(t, err)

	user := &model.User{
		Email:        "oauth@example.com",
		Nickname:     "OAuth User",
		AuthProvider: "github",
		GitHubID:     "123456",
		PasswordHash: "", // Empty password for OAuth
	}
	err = user.GenerateID()
	require.NoError(t, err)

	// This should succeed with the nullable password_hash
	err = db.Create(user).Error
	require.NoError(t, err, "User creation should succeed with empty password hash")

	// Verify user was created
	var found model.User
	err = db.First(&found, "id = ?", user.ID).Error
	require.NoError(t, err)
	require.Equal(t, "oauth@example.com", found.Email)
	require.Equal(t, "123456", found.GitHubID)
}
