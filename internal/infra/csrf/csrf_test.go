package csrf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAndValidateToken(t *testing.T) {
	t.Run("generate token", func(t *testing.T) {
		token := Generate()
		assert.NotEmpty(t, token)
		assert.Len(t, token, 64) // 32 bytes hex encoded
	})

	t.Run("validate matching tokens", func(t *testing.T) {
		token := Generate()
		valid := Validate(token, token)
		assert.True(t, valid)
	})

	t.Run("validate non-matching tokens", func(t *testing.T) {
		token1 := Generate()
		token2 := Generate()
		valid := Validate(token1, token2)
		assert.False(t, valid)
	})

	t.Run("validate empty tokens", func(t *testing.T) {
		valid := Validate("", "")
		assert.False(t, valid)
	})
}
