package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"oreader/internal/config"
	"oreader/internal/infra/jwt"
	"oreader/internal/model"
	"oreader/internal/repository"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupOAuthTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Drop tables to ensure clean schema with latest model changes
	_ = db.Migrator().DropTable(&model.User{}, &model.RefreshToken{}, &model.OAuthState{})

	err = db.AutoMigrate(&model.User{}, &model.RefreshToken{}, &model.OAuthState{})
	require.NoError(t, err)
	return db
}

func setupOAuthTestConfig(t *testing.T) *config.Config {
	t.Setenv("DATABASE_URL", ":memory:")
	t.Setenv("JWT_SECRET_KEY", "test-secret-key-must-be-at-least-32-characters")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "168h")
	t.Setenv("GITHUB_CLIENT_ID", "test-github-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-github-client-secret")

	cfg, err := config.Load()
	require.NoError(t, err)
	return cfg
}

func createOAuthHandler(t *testing.T, db *gorm.DB, cfg *config.Config, githubServer *httptest.Server) *OAuthHandler {
	accessTTL, _ := cfg.GetAccessTTL()
	refreshTTL, _ := cfg.GetRefreshTTL()
	jwtService := jwt.NewService(cfg.Auth.SecretKey, accessTTL, refreshTTL)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	stateRepo := repository.NewOAuthStateRepository(db)

	baseURL := ""
	if githubServer != nil {
		baseURL = githubServer.URL
	}

	return NewOAuthHandler(cfg, jwtService, userRepo, tokenRepo, stateRepo, baseURL)
}

func createOAuthTestRouter(h *OAuthHandler) *gin.Engine {
	router := gin.New()
	oauth := router.Group("/api/v1/auth")
	{
		oauth.GET("/github", h.GitHubInitiate)
		oauth.GET("/github/callback", h.GitHubCallback)
		oauth.GET("/google", h.GoogleInitiate)
		oauth.GET("/google/callback", h.GoogleCallback)
		oauth.GET("/apple", h.AppleInitiate)
		oauth.GET("/apple/callback", h.AppleCallback)
	}
	return router
}

// mockGitHubServer creates a mock GitHub server for testing
func mockGitHubServer(t *testing.T, userJSON string, tokenJSON string) *httptest.Server {
	mux := http.NewServeMux()

	// Mock access token endpoint
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, tokenJSON)
	})

	// Mock user endpoint
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, userJSON)
	})

	return httptest.NewServer(mux)
}

func TestOAuthHandler_GitHubInitiate(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)
	h := createOAuthHandler(t, db, cfg, nil)
	router := createOAuthTestRouter(h)

	t.Run("initiate github oauth redirects to github", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/github", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)

		// Check redirect location
		location := w.Header().Get("Location")
		assert.Contains(t, location, "github.com/login/oauth/authorize")
		assert.Contains(t, location, "client_id=test-github-client-id")
		assert.Contains(t, location, "redirect_uri=")
		assert.Contains(t, location, "state=")

		// Extract and verify state was stored
		state := extractStateFromURL(t, location)
		assert.NotEmpty(t, state)

		// Verify state was stored in database
		stateRepo := repository.NewOAuthStateRepository(db)
		storedState, err := stateRepo.GetByState(context.Background(), state)
		assert.NoError(t, err)
		assert.NotNil(t, storedState)
		assert.Equal(t, "github", storedState.Provider)
		assert.False(t, storedState.IsExpired())
	})
}

func TestOAuthHandler_GitHubCallback_Success(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)

	userJSON := `{
		"id": 123456,
		"login": "testuser",
		"name": "Test User",
		"email": "testuser@example.com",
		"avatar_url": "https://example.com/avatar.png"
	}`

	tokenJSON := `{
		"access_token": "gh_test_access_token",
		"token_type": "bearer",
		"scope": "read:user,user:email"
	}`

	githubServer := mockGitHubServer(t, userJSON, tokenJSON)
	defer githubServer.Close()

	h := createOAuthHandler(t, db, cfg, githubServer)
	router := createOAuthTestRouter(h)

	// First, create and store a valid state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_valid_state_123",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err := state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("successful github oauth callback creates user and sets cookies", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?code=test_code&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should redirect after successful OAuth
		assert.Equal(t, http.StatusFound, w.Code)

		// Verify user was created in database
		userRepo := repository.NewUserRepository(db)
		user, err := userRepo.GetByEmail(context.Background(), "testuser@example.com")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "123456", user.GitHubID)
		assert.Equal(t, "Test User", user.Nickname) // GitHub name is preferred over login
		assert.Equal(t, "github", user.AuthProvider)
		assert.Equal(t, "https://example.com/avatar.png", user.AvatarURL)

		// Check auth cookies are set
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
}

func TestOAuthHandler_GitHubCallback_InvalidState(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)
	h := createOAuthHandler(t, db, cfg, nil)
	router := createOAuthTestRouter(h)

	t.Run("callback with invalid state returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/github/callback?code=test_code&state=invalid_state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		errObj := resp["error"].(map[string]interface{})
		assert.Contains(t, errObj["message"], "Invalid OAuth state")
	})

	t.Run("callback with expired state returns 400", func(t *testing.T) {
		stateRepo := repository.NewOAuthStateRepository(db)
		state := &model.OAuthState{
			State:     "expired_state",
			Provider:  "github",
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		}
		err := state.GenerateID()
		require.NoError(t, err)
		err = stateRepo.Create(context.Background(), state)
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/api/v1/auth/github/callback?code=test_code&state=expired_state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("callback with missing state returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/github/callback?code=test_code", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOAuthHandler_GitHubCallback_MissingCode(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)
	h := createOAuthHandler(t, db, cfg, nil)
	router := createOAuthTestRouter(h)

	// Create a valid state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_state_no_code",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err := state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("callback without code returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/github/callback?state=test_state_no_code", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		errObj := resp["error"].(map[string]interface{})
		assert.Contains(t, errObj["message"], "Authorization code is required")
	})
}

func TestOAuthHandler_ReservedProviders(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)
	h := createOAuthHandler(t, db, cfg, nil)
	router := createOAuthTestRouter(h)

	t.Run("google oauth initiate returns 501", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/google", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Google OAuth")
	})

	t.Run("google oauth callback returns 501", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/google/callback?code=test&state=test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})

	t.Run("apple oauth initiate returns 501", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/apple", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})

	t.Run("apple oauth callback returns 501", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/apple/callback?code=test&state=test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})
}

func TestOAuthHandler_ExistingUserLinking(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)

	userJSON := `{
		"id": 789012,
		"login": "existinguser",
		"name": "Existing User",
		"email": "existing@example.com",
		"avatar_url": "https://example.com/newavatar.png"
	}`

	tokenJSON := `{
		"access_token": "gh_test_access_token",
		"token_type": "bearer"
	}`

	githubServer := mockGitHubServer(t, userJSON, tokenJSON)
	defer githubServer.Close()

	h := createOAuthHandler(t, db, cfg, githubServer)
	router := createOAuthTestRouter(h)

	// Create existing user with same email
	userRepo := repository.NewUserRepository(db)
	existingUser := &model.User{
		Email:        "existing@example.com",
		PasswordHash: "some-hash",
		Nickname:     "OldNickname",
		AuthProvider: "email",
	}
	err := existingUser.GenerateID()
	require.NoError(t, err)
	err = userRepo.Create(context.Background(), existingUser)
	require.NoError(t, err)

	// Create OAuth state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_linking_state",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err = state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("oauth with existing email links account", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?code=test_code&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)

		// Verify existing user was updated (not duplicated)
		users, err := userRepo.GetByEmail(context.Background(), "existing@example.com")
		assert.NoError(t, err)
		assert.NotNil(t, users)
		assert.Equal(t, existingUser.ID, users.ID) // Same ID
		assert.Equal(t, "789012", users.GitHubID)   // GitHub ID linked

		// Check cookies were set
		cookies := w.Result().Cookies()
		var hasAccessToken bool
		for _, c := range cookies {
			if c.Name == "access_token" {
				hasAccessToken = true
				break
			}
		}
		assert.True(t, hasAccessToken)
	})
}

func TestOAuthHandler_ExistingGitHubUser(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)

	userJSON := `{
		"id": 345678,
		"login": "githubuser",
		"name": "GitHub User",
		"email": "github@example.com",
		"avatar_url": "https://example.com/avatar.png"
	}`

	tokenJSON := `{
		"access_token": "gh_test_access_token",
		"token_type": "bearer"
	}`

	githubServer := mockGitHubServer(t, userJSON, tokenJSON)
	defer githubServer.Close()

	h := createOAuthHandler(t, db, cfg, githubServer)
	router := createOAuthTestRouter(h)

	// Create existing user with same GitHub ID
	userRepo := repository.NewUserRepository(db)
	existingUser := &model.User{
		Email:        "old@example.com",
		PasswordHash: "some-hash",
		Nickname:     "OldNickname",
		AuthProvider: "github",
		GitHubID:     "345678",
	}
	err := existingUser.GenerateID()
	require.NoError(t, err)
	err = userRepo.Create(context.Background(), existingUser)
	require.NoError(t, err)

	// Create OAuth state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_existing_github_state",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err = state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("oauth with existing github id signs in existing user", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?code=test_code&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)

		// Verify no duplicate user was created
		user, err := userRepo.GetByGitHubID(context.Background(), "345678")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, existingUser.ID, user.ID)

		// Check cookies were set
		cookies := w.Result().Cookies()
		var hasAccessToken bool
		for _, c := range cookies {
			if c.Name == "access_token" {
				hasAccessToken = true
				break
			}
		}
		assert.True(t, hasAccessToken)
	})
}

func TestOAuthHandler_GitHubError(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)
	h := createOAuthHandler(t, db, cfg, nil)
	router := createOAuthTestRouter(h)

	// Create OAuth state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_error_state",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err := state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("callback with error parameter returns error", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?error=access_denied&error_description=user+denied+access&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		errObj := resp["error"].(map[string]interface{})
		assert.Contains(t, errObj["message"], "access_denied")
	})
}

// Helper function to extract state parameter from URL
func extractStateFromURL(t *testing.T, redirectURL string) string {
	u, err := url.Parse(redirectURL)
	require.NoError(t, err)

	stateValues := u.Query()["state"]
	if len(stateValues) == 0 {
		return ""
	}
	return stateValues[0]
}
