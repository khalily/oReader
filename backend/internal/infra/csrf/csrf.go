package csrf

import (
	"crypto/rand"
	"encoding/hex"
)

// Generate creates a new CSRF token
func Generate() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// Validate checks if the provided token matches the expected token
func Validate(provided, expected string) bool {
	if provided == "" || expected == "" {
		return false
	}
	return provided == expected
}
