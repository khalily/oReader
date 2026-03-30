package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/config"
	"github.com/khalily/oreader/internal/infra/jwt"
	"github.com/khalily/oreader/internal/infra/password"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/repository"
	"github.com/khalily/oreader/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAuthTestDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)
	err := db.AutoMigrate(&model.User{}, &model.RefreshToken{})
	require.NoError(t, err)
	return db
}

func setupAuthTestConfig(t *testing.T) *config.Config {
	t.Setenv("DATABASE_URL", testutil.GetTestDatabaseURL())
	t.Setenv("JWT_SECRET_KEY", "test-secret-key-must-be-at-least-32-characters")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "168h")

	cfg, err := config.Load()
	require.NoError(t, err)
	return cfg
}

func createAuthHandler(t *testing.T, db *gorm.DB, cfg *config.Config) *Handler {
	accessTTL, _ := cfg.GetAccessTTL()
	refreshTTL, _ := cfg.GetRefreshTTL()
	jwtService := jwt.NewService(cfg.Auth.SecretKey, accessTTL, refreshTTL)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)

	return NewHandler(cfg, jwtService, userRepo, tokenRepo)
}

func createTestRouter(h *Handler) *gin.Engine {
	router := gin.New()
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
		auth.GET("/me", h.Me)
	}
	return router
}

func TestAuthHandler_Register(t *testing.T) {
	db := setupAuthTestDB(t)
	cfg := setupAuthTestConfig(t)
	h := createAuthHandler(t, db, cfg)
	router := createTestRouter(h)

	t.Run("register successfully", func(t *testing.T) {
		body := `{"email": "newuser@example.com", "password": "password123"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		user := resp["user"].(map[string]interface{})
		assert.Equal(t, "newuser@example.com", user["email"])
	})

	t.Run("register with invalid email fails", func(t *testing.T) {
		body := `{"email": "invalid-email", "password": "password123"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("register duplicate email fails", func(t *testing.T) {
		body := `{"email": "newuser@example.com", "password": "password123"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("register with short password fails", func(t *testing.T) {
		body := `{"email": "shortpw@example.com", "password": "short"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	db := setupAuthTestDB(t)
	cfg := setupAuthTestConfig(t)
	h := createAuthHandler(t, db, cfg)
	router := createTestRouter(h)

	// Create test user
	passwordHash, err := password.Hash("password123")
	require.NoError(t, err)
	user := &model.User{
		Email:        "login@example.com",
		PasswordHash: passwordHash,
		Nickname:     "TestUser",
	}
	err = user.GenerateID()
	require.NoError(t, err)
	err = db.Create(user).Error
	require.NoError(t, err)

	t.Run("login successfully", func(t *testing.T) {
		body := `{"email": "login@example.com", "password": "password123"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		user := resp["user"].(map[string]interface{})
		assert.Equal(t, "login@example.com", user["email"])

		// Check cookies are set
		cookies := w.Result().Cookies()
		var hasAccessToken, hasRefreshToken, hasCSRFToken bool
		for _, c := range cookies {
			if c.Name == "access_token" {
				hasAccessToken = true
			}
			if c.Name == "refresh_token" {
				hasRefreshToken = true
			}
			if c.Name == "csrf_token" {
				hasCSRFToken = true
			}
		}
		assert.True(t, hasAccessToken, "access_token cookie should be set")
		assert.True(t, hasRefreshToken, "refresh_token cookie should be set")
		assert.True(t, hasCSRFToken, "csrf_token cookie should be set")
	})

	t.Run("login with wrong password fails", func(t *testing.T) {
		body := `{"email": "login@example.com", "password": "wrongpassword"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("login with non-existent user fails", func(t *testing.T) {
		body := `{"email": "nonexistent@example.com", "password": "password123"}`
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	db := setupAuthTestDB(t)
	cfg := setupAuthTestConfig(t)
	accessTTL, _ := cfg.GetAccessTTL()
	refreshTTL, _ := cfg.GetRefreshTTL()
	jwtService := jwt.NewService(cfg.Auth.SecretKey, accessTTL, refreshTTL)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	h := NewHandler(cfg, jwtService, userRepo, tokenRepo)
	router := createTestRouter(h)

	// Create test user
	passwordHash, err := password.Hash("password123")
	require.NoError(t, err)
	user := &model.User{
		Email:        "refresh@example.com",
		PasswordHash: passwordHash,
	}
	err = user.GenerateID()
	require.NoError(t, err)
	err = db.Create(user).Error
	require.NoError(t, err)

	t.Run("refresh successfully", func(t *testing.T) {
		// Generate refresh token
		refreshToken, tokenHash, err := jwtService.GenerateRefreshToken(user.ID)
		require.NoError(t, err)

		// Store refresh token in DB
		refreshModel := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err = refreshModel.GenerateID()
		require.NoError(t, err)
		err = tokenRepo.Create(context.Background(), refreshModel)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		userData := resp["user"].(map[string]interface{})
		assert.Equal(t, user.ID, userData["id"])

		// Check new access token is set
		cookies := w.Result().Cookies()
		var hasAccessToken bool
		for _, c := range cookies {
			if c.Name == "access_token" {
				hasAccessToken = true
			}
		}
		assert.True(t, hasAccessToken, "new access_token cookie should be set")
	})

	t.Run("refresh without token fails", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("refresh with invalid token fails", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: "invalid-token",
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("refresh with revoked token fails", func(t *testing.T) {
		// Generate refresh token
		refreshToken, tokenHash, err := jwtService.GenerateRefreshToken(user.ID)
		require.NoError(t, err)

		// Store and revoke refresh token
		refreshModel := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			Revoked:   true,
		}
		err = refreshModel.GenerateID()
		require.NoError(t, err)
		err = tokenRepo.Create(context.Background(), refreshModel)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	db := setupAuthTestDB(t)
	cfg := setupAuthTestConfig(t)
	accessTTL, _ := cfg.GetAccessTTL()
	refreshTTL, _ := cfg.GetRefreshTTL()
	jwtService := jwt.NewService(cfg.Auth.SecretKey, accessTTL, refreshTTL)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	h := NewHandler(cfg, jwtService, userRepo, tokenRepo)
	router := createTestRouter(h)

	// Create test user
	passwordHash, err := password.Hash("password123")
	require.NoError(t, err)
	user := &model.User{
		Email:        "logout@example.com",
		PasswordHash: passwordHash,
	}
	err = user.GenerateID()
	require.NoError(t, err)
	err = db.Create(user).Error
	require.NoError(t, err)

	t.Run("logout successfully", func(t *testing.T) {
		// Generate tokens
		refreshToken, tokenHash, err := jwtService.GenerateRefreshToken(user.ID)
		require.NoError(t, err)

		// Store refresh token
		refreshModel := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err = refreshModel.GenerateID()
		require.NoError(t, err)
		err = tokenRepo.Create(context.Background(), refreshModel)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check cookies are cleared
		cookies := w.Result().Cookies()
		for _, c := range cookies {
			if c.Name == "access_token" || c.Name == "refresh_token" || c.Name == "csrf_token" {
				assert.Equal(t, -1, c.MaxAge, "cookie %s should be cleared", c.Name)
			}
		}
	})

	t.Run("logout without token still succeeds", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Logout should succeed even without token (idempotent)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthHandler_Me(t *testing.T) {
	db := setupAuthTestDB(t)
	cfg := setupAuthTestConfig(t)
	accessTTL, _ := cfg.GetAccessTTL()
	refreshTTL, _ := cfg.GetRefreshTTL()
	jwtService := jwt.NewService(cfg.Auth.SecretKey, accessTTL, refreshTTL)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	h := NewHandler(cfg, jwtService, userRepo, tokenRepo)
	router := createTestRouter(h)

	// Create test user
	passwordHash, err := password.Hash("password123")
	require.NoError(t, err)
	user := &model.User{
		Email:        "me@example.com",
		PasswordHash: passwordHash,
		Nickname:     "MeUser",
	}
	err = user.GenerateID()
	require.NoError(t, err)
	err = db.Create(user).Error
	require.NoError(t, err)

	t.Run("get current user successfully", func(t *testing.T) {
		accessToken, _, err := jwtService.GenerateAccessToken(user.ID)
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: accessToken,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		userData := resp["user"].(map[string]interface{})
		assert.Equal(t, user.ID, userData["id"])
		assert.Equal(t, "me@example.com", userData["email"])
		assert.Equal(t, "MeUser", userData["nickname"])
	})

	t.Run("get me without token fails", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("get me with invalid token fails", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "invalid-token",
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Validation(t *testing.T) {
	db := setupAuthTestDB(t)
	cfg := setupAuthTestConfig(t)
	h := createAuthHandler(t, db, cfg)
	router := createTestRouter(h)

	t.Run("register with empty body fails", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("login with empty body fails", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
