package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerify(t *testing.T) {
	password := "mySecretPassword123"

	t.Run("hash password", func(t *testing.T) {
		hash, err := Hash(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 60) // bcrypt hash length
	})

	t.Run("verify correct password", func(t *testing.T) {
		hash, err := Hash(password)
		require.NoError(t, err)

		valid, err := Verify(password, hash)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("verify incorrect password", func(t *testing.T) {
		hash, err := Hash(password)
		require.NoError(t, err)

		valid, err := Verify("wrongPassword", hash)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("verify with invalid hash", func(t *testing.T) {
		valid, err := Verify(password, "invalid-hash")
		assert.Error(t, err)
		assert.False(t, valid)
	})
}
