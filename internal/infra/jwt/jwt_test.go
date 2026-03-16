package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secretKey := "test-secret-key-must-be-at-least-32-characters"
	service := NewService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "user-123"

	t.Run("generate access token", func(t *testing.T) {
		token, csrfToken, err := service.GenerateAccessToken(userID)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEmpty(t, csrfToken)
	})

	t.Run("validate access token", func(t *testing.T) {
		token, _, err := service.GenerateAccessToken(userID)
		require.NoError(t, err)

		claims, err := service.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("generate token with short TTL", func(t *testing.T) {
		shortService := NewService(secretKey, 100*time.Millisecond, 7*24*time.Hour)

		token, _, err := shortService.GenerateAccessToken(userID)
		require.NoError(t, err)

		time.Sleep(150 * time.Millisecond)

		_, err = shortService.ValidateAccessToken(token)
		assert.Error(t, err)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := service.ValidateAccessToken("invalid-token")
		assert.Error(t, err)
	})
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	secretKey := "test-secret-key-must-be-at-least-32-characters"
	service := NewService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "user-123"

	t.Run("generate refresh token", func(t *testing.T) {
		token, hash, err := service.GenerateRefreshToken(userID)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEmpty(t, hash)
	})

	t.Run("validate refresh token", func(t *testing.T) {
		token, _, err := service.GenerateRefreshToken(userID)
		require.NoError(t, err)

		claims, err := service.ValidateRefreshToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})
}
