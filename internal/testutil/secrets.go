// Package testutil provides shared utilities for testing.
package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"os"
)

// GetTestJWTSecret returns a JWT secret for testing.
// It first checks the TEST_JWT_SECRET environment variable,
// then falls back to a generated random key for local development.
//
// Usage in tests:
//
//	secret := testutil.GetTestJWTSecret(t)
//	jwtService := jwt.NewService(secret, 15*time.Minute, 7*24*time.Hour)
//
// To use a custom secret, set the environment variable:
//
//	TEST_JWT_SECRET=your-secret-key go test ./...
func GetTestJWTSecret() string {
	if secret := os.Getenv("TEST_JWT_SECRET"); secret != "" {
		return secret
	}

	// Generate a random 32-byte key for this test session
	// This ensures different test runs use different keys
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a deterministic key if random generation fails
		return "test-secret-key-fallback-must-be-32chars"
	}
	return hex.EncodeToString(bytes)
}

// GetTestOAuthClientSecret returns an OAuth client secret for testing.
// It first checks the TEST_OAUTH_CLIENT_SECRET environment variable,
// then falls back to a placeholder value.
func GetTestOAuthClientSecret() string {
	if secret := os.Getenv("TEST_OAUTH_CLIENT_SECRET"); secret != "" {
		return secret
	}
	return "test-oauth-client-secret-placeholder"
}

// GetTestDatabaseURL returns a database URL for testing.
// It first checks the TEST_DATABASE_URL environment variable,
// then falls back to a local MySQL test database.
func GetTestDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	return "oreader:oreader@tcp(localhost:3306)/oreader_test?charset=utf8mb4&parseTime=True&loc=Local"
}
