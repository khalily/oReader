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
	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/config"
	"github.com/khalily/oreader/internal/infra/jwt"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/repository"
	"github.com/khalily/oreader/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupOAuthTestDB(t *testing.T) *gorm.DB {
	db := testutil.SetupTestDB(t)

	// Drop tables to ensure clean schema with latest model changes
	_ = db.Migrator().DropTable(&model.User{}, &model.RefreshToken{}, &model.OAuthState{}, &model.PendingOAuth{})

	err := db.AutoMigrate(&model.User{}, &model.RefreshToken{}, &model.OAuthState{}, &model.PendingOAuth{})
	require.NoError(t, err)
	return db
}

func setupOAuthTestConfig(t *testing.T) *config.Config {
	t.Setenv("DATABASE_URL", testutil.GetTestDatabaseURL())
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
	pendingOAuthRepo := repository.NewPendingOAuthRepository(db)

	baseURL := ""
	if githubServer != nil {
		baseURL = githubServer.URL
	}

	return NewOAuthHandler(cfg, jwtService, userRepo, tokenRepo, stateRepo, pendingOAuthRepo, baseURL)
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

// mockGitHubServerWithEmails creates a mock GitHub server with email endpoint support
func mockGitHubServerWithEmails(t *testing.T, userJSON string, tokenJSON string, emailsJSON string) *httptest.Server {
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

	// Mock user emails endpoint
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, emailsJSON)
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

	t.Run("callback with invalid state redirects with error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/github/callback?code=test_code&state=invalid_state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login?oauth_error=invalid_state", w.Header().Get("Location"))
	})

	t.Run("callback with expired state redirects with error", func(t *testing.T) {
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

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login?oauth_error=invalid_state", w.Header().Get("Location"))
	})

	t.Run("callback with missing state redirects with error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/github/callback?code=test_code", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login?oauth_error=invalid_state", w.Header().Get("Location"))
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

	t.Run("callback without code redirects with error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/github/callback?state=test_state_no_code", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login?oauth_error=access_denied", w.Header().Get("Location"))
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

	t.Run("callback with error parameter redirects with error", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?error=access_denied&error_description=user+denied+access&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login?oauth_error=access_denied", w.Header().Get("Location"))
	})
}

// TestGetBestEmail tests the getBestEmail helper function
func TestGetBestEmail(t *testing.T) {
	tests := []struct {
		name     string
		emails   []GitHubEmail
		expected string
	}{
		{
			name:     "empty emails",
			emails:   []GitHubEmail{},
			expected: "",
		},
		{
			name: "primary and verified is best",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: true},
				{Email: "b@example.com", Primary: true, Verified: true},
				{Email: "c@example.com", Primary: true, Verified: false},
			},
			expected: "b@example.com",
		},
		{
			name: "verified without primary",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: false},
				{Email: "b@example.com", Primary: false, Verified: true},
			},
			expected: "b@example.com",
		},
		{
			name: "primary without verified",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: true},
				{Email: "b@example.com", Primary: true, Verified: false},
			},
			expected: "a@example.com", // verified has higher priority than primary
		},
		{
			name: "first email as fallback",
			emails: []GitHubEmail{
				{Email: "a@example.com", Primary: false, Verified: false},
				{Email: "b@example.com", Primary: false, Verified: false},
			},
			expected: "a@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBestEmail(tt.emails)
			assert.Equal(t, tt.expected, result)
		})
	}
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

// TestOAuthHandler_GitHubCallbackHost tests that the configured callback host
// is used instead of the request Host header when GITHUB_CALLBACK_HOST is set
func TestOAuthHandler_GitHubCallbackHost(t *testing.T) {
	db := setupOAuthTestDB(t)

	// Set callback host to test that it's used in redirect_uri
	t.Setenv("GITHUB_CALLBACK_HOST", "10.37.126.68:8080")
	cfg := setupOAuthTestConfig(t)

	// Verify config loaded correctly
	assert.Equal(t, "10.37.126.68:8080", cfg.OAuth.GitHubCallbackHost)

	h := createOAuthHandler(t, db, cfg, nil)
	router := createOAuthTestRouter(h)

	t.Run("uses configured callback host in redirect_uri", func(t *testing.T) {
		// Send request with different Host header (simulating proxy)
		req, _ := http.NewRequest("GET", "/api/v1/auth/github", nil)
		req.Host = "localhost:8080" // This should be ignored
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)

		location := w.Header().Get("Location")
		// Should use configured host, not request host
		assert.Contains(t, location, "redirect_uri=http%3A%2F%2F10.37.126.68%3A8080%2Fapi%2Fv1%2Fauth%2Fgithub%2Fcallback")
		assert.NotContains(t, location, "localhost")
	})

	t.Run("falls back to request host when callback host not configured", func(t *testing.T) {
		// Create new config without callback host
		t.Setenv("GITHUB_CALLBACK_HOST", "")
		cfgNoHost, err := config.Load()
		require.NoError(t, err)
		assert.Empty(t, cfgNoHost.OAuth.GitHubCallbackHost)

		hNoHost := createOAuthHandler(t, db, cfgNoHost, nil)
		routerNoHost := createOAuthTestRouter(hNoHost)

		req, _ := http.NewRequest("GET", "/api/v1/auth/github", nil)
		req.Host = "localhost:3000"
		w := httptest.NewRecorder()
		routerNoHost.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)

		location := w.Header().Get("Location")
		// Should use request host when not configured
		assert.Contains(t, location, "redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fapi%2Fv1%2Fauth%2Fgithub%2Fcallback")
	})
}

// TestGitHubCallback_NoEmail_CreatesPendingOAuth tests the pending OAuth flow
// when a GitHub user has no public email and no emails accessible via API
func TestGitHubCallback_NoEmail_CreatesPendingOAuth(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)

	// GitHub user with no public email
	userJSON := `{
		"id": 99999,
		"login": "noemailuser",
		"name": "No Email User",
		"email": "",
		"avatar_url": "https://example.com/noemail-avatar.png"
	}`

	tokenJSON := `{
		"access_token": "gh_test_noemail_token",
		"token_type": "bearer",
		"scope": "read:user,user:email"
	}`

	// Empty emails list - no emails accessible
	emailsJSON := `[]`

	githubServer := mockGitHubServerWithEmails(t, userJSON, tokenJSON, emailsJSON)
	defer githubServer.Close()

	h := createOAuthHandler(t, db, cfg, githubServer)
	router := createOAuthTestRouter(h)

	// Create OAuth state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_noemail_state",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err := state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("oauth without email creates pending oauth and redirects", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?code=test_code&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should redirect to pending page
		assert.Equal(t, http.StatusFound, w.Code)

		location := w.Header().Get("Location")
		assert.Contains(t, location, "/oauth/pending?token=")

		// Extract token from location
		token := extractTokenFromLocation(t, location)
		assert.NotEmpty(t, token)

		// Verify PendingOAuth was created
		pendingOAuthRepo := repository.NewPendingOAuthRepository(db)
		pending, err := pendingOAuthRepo.GetByToken(context.Background(), token)
		require.NoError(t, err)
		assert.NotNil(t, pending)
		assert.Equal(t, "99999", pending.GitHubID)
		assert.Equal(t, "noemailuser", pending.GitHubLogin)
		assert.Equal(t, "No Email User", pending.Nickname)
		assert.Equal(t, "https://example.com/noemail-avatar.png", pending.AvatarURL)
		assert.False(t, pending.IsExpired())

		// Verify no user was created
		userRepo := repository.NewUserRepository(db)
		user, err := userRepo.GetByGitHubID(context.Background(), "99999")
		require.NoError(t, err) // Should not error, just return nil
		assert.Nil(t, user)     // Should not find user
	})
}

// TestGitHubCallback_FetchesEmailFromAPI tests fetching email from /user/emails API
func TestGitHubCallback_FetchesEmailFromAPI(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)

	// GitHub user with no public email
	userJSON := `{
		"id": 88888,
		"login": "privateemail",
		"name": "Private Email User",
		"email": "",
		"avatar_url": "https://example.com/private-avatar.png"
	}`

	tokenJSON := `{
		"access_token": "gh_test_private_token",
		"token_type": "bearer",
		"scope": "read:user,user:email"
	}`

	// Emails from API - one verified and primary
	emailsJSON := `[
		{
			"email": "verified@example.com",
			"primary": true,
			"verified": true
		},
		{
			"email": "unverified@example.com",
			"primary": false,
			"verified": false
		}
	]`

	githubServer := mockGitHubServerWithEmails(t, userJSON, tokenJSON, emailsJSON)
	defer githubServer.Close()

	h := createOAuthHandler(t, db, cfg, githubServer)
	router := createOAuthTestRouter(h)

	// Create OAuth state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_fetch_email_state",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err := state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("oauth fetches email from API and creates user", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?code=test_code&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should redirect after successful OAuth
		assert.Equal(t, http.StatusFound, w.Code)

		location := w.Header().Get("Location")
		assert.Contains(t, location, "/")

		// Verify user was created with correct email
		userRepo := repository.NewUserRepository(db)
		user, err := userRepo.GetByGitHubID(context.Background(), "88888")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "verified@example.com", user.Email) // Email from API
		assert.Equal(t, "88888", user.GitHubID)
		assert.Equal(t, "Private Email User", user.Nickname)
		assert.Equal(t, "github", user.AuthProvider)
		assert.Equal(t, "https://example.com/private-avatar.png", user.AvatarURL)

		// No PendingOAuth should be created when email is successfully fetched
		// User was created successfully, which means no pending OAuth was needed
	})
}

// TestGitHubCallback_FetchesNonPrimaryEmailFromAPI tests fetching non-primary but verified email
func TestGitHubCallback_FetchesNonPrimaryEmailFromAPI(t *testing.T) {
	db := setupOAuthTestDB(t)
	cfg := setupOAuthTestConfig(t)

	// GitHub user with no public email
	userJSON := `{
		"id": 77777,
		"login": "nonprimary",
		"name": "Non Primary User",
		"email": "",
		"avatar_url": "https://example.com/nonprimary-avatar.png"
	}`

	tokenJSON := `{
		"access_token": "gh_test_nonprimary_token",
		"token_type": "bearer",
		"scope": "read:user,user:email"
	}`

	// Emails from API - verified but not primary
	emailsJSON := `[
		{
			"email": "nonprimary@example.com",
			"primary": false,
			"verified": true
		}
	]`

	githubServer := mockGitHubServerWithEmails(t, userJSON, tokenJSON, emailsJSON)
	defer githubServer.Close()

	h := createOAuthHandler(t, db, cfg, githubServer)
	router := createOAuthTestRouter(h)

	// Create OAuth state
	stateRepo := repository.NewOAuthStateRepository(db)
	state := &model.OAuthState{
		State:     "test_nonprimary_state",
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	err := state.GenerateID()
	require.NoError(t, err)
	err = stateRepo.Create(context.Background(), state)
	require.NoError(t, err)

	t.Run("oauth fetches verified non-primary email", func(t *testing.T) {
		callbackURL := fmt.Sprintf("/api/v1/auth/github/callback?code=test_code&state=%s", state.State)
		req, _ := http.NewRequest("GET", callbackURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should redirect after successful OAuth
		assert.Equal(t, http.StatusFound, w.Code)

		// Verify user was created with verified email
		userRepo := repository.NewUserRepository(db)
		user, err := userRepo.GetByGitHubID(context.Background(), "77777")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "nonprimary@example.com", user.Email)
	})
}

// extractTokenFromLocation extracts the token parameter from a location URL
func extractTokenFromLocation(t *testing.T, location string) string {
	u, err := url.Parse(location)
	require.NoError(t, err)

	token := u.Query().Get("token")
	return token
}
