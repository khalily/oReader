package jwt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenInvalid   = errors.New("token is invalid")
	ErrTokenMalformed  = errors.New("token is malformed")
)

// Service handles JWT token operations
type Service struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewService creates a new JWT service
func NewService(secretKey string, accessTTL, refreshTTL time.Duration) *Service {
	if len(secretKey) < 32 {
		panic("secret key requires at least 32 characters")
	}
	return &Service{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// AccessTokenClaims contains claims for access tokens
type AccessTokenClaims struct {
	UserID    string `json:"user_id"`
	CSRFToken string `json:"csrf_token"`
	jwt.RegisteredClaims
}

// RefreshTokenClaims contains claims for refresh tokens
type RefreshTokenClaims struct {
	UserID  string `json:"user_id"`
	TokenID string `json:"token_id"`
	jwt.RegisteredClaims
}

// GenerateAccessToken generates a new access token for a user
func (s *Service) GenerateAccessToken(userID string) (string, string, error) {
	// Generate CSRF token
	csrfToken := generateRandomToken(32)

	now := time.Now()
	claims := AccessTokenClaims{
		UserID:    userID,
		CSRFToken: csrfToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:   "oreader",
			Subject:  userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, csrfToken, nil
}

// ValidateAccessToken validates an access token and returns the claims
func (s *Service) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok {
		return nil, ErrTokenMalformed
	}

	return claims, nil
}

// GenerateRefreshToken generates a new refresh token for a user
func (s *Service) GenerateRefreshToken(userID string) (string, string, error) {
	// Generate random token ID
	tokenID := generateRandomToken(32)

	now := time.Now()
	claims := RefreshTokenClaims{
		UserID: userID,
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:   "oreader",
			Subject:  userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate hash for storage
	hash := sha256.Sum256([]byte(tokenString))

	return tokenString, hex.EncodeToString(hash[:]), nil
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (s *Service) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok {
		return nil, ErrTokenMalformed
	}

	return claims, nil
}

// generateRandomToken generates a random hex string
func generateRandomToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
