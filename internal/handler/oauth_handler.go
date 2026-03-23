package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"oreader/internal/config"
	"oreader/internal/infra/cookie"
	"oreader/internal/infra/errors"
	"oreader/internal/infra/jwt"
	"oreader/internal/model"
	"oreader/internal/service"
)

// OAuthHandler handles OAuth authentication endpoints
type OAuthHandler struct {
	cfg         *config.Config
	jwtService  *jwt.Service
	userRepo    service.UserRepository
	tokenRepo   service.RefreshTokenRepository
	stateRepo   service.OAuthStateRepository
	githubBaseURL string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(
	cfg *config.Config,
	jwtService *jwt.Service,
	userRepo service.UserRepository,
	tokenRepo service.RefreshTokenRepository,
	stateRepo service.OAuthStateRepository,
	githubBaseURL string,
) *OAuthHandler {
	// Use default GitHub URL if not provided (for testing)
	if githubBaseURL == "" {
		githubBaseURL = "https://github.com"
	}

	return &OAuthHandler{
		cfg:           cfg,
		jwtService:    jwtService,
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		stateRepo:     stateRepo,
		githubBaseURL: githubBaseURL,
	}
}

// generateState generates a secure random state parameter
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GitHubUser represents the GitHub user profile response
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubTokenResponse represents GitHub's token response
type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// GitHubInitiate initiates the GitHub OAuth flow
func (h *OAuthHandler) GitHubInitiate(c *gin.Context) {
	// Generate secure state parameter
	state, err := generateState()
	if err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to generate OAuth state",
		})
		return
	}

	// Store state in database
	oauthState := &model.OAuthState{
		State:     state,
		Provider:  "github",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := oauthState.GenerateID(); err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to generate state ID",
		})
		return
	}

	if err := h.stateRepo.Create(c.Request.Context(), oauthState); err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to store OAuth state",
		})
		return
	}

	// Build GitHub authorization URL
	redirectURI := h.getRedirectURI(c, "github")
	authURL := fmt.Sprintf("%s/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user%%20user:email&state=%s",
		h.githubBaseURL,
		h.cfg.OAuth.GitHubClientID,
		url.QueryEscape(redirectURI),
		state,
	)

	c.Redirect(http.StatusFound, authURL)
}

// GitHubCallback handles the GitHub OAuth callback
func (h *OAuthHandler) GitHubCallback(c *gin.Context) {
	// Check for error response from GitHub
	if errorCode := c.Query("error"); errorCode != "" {
		c.Redirect(http.StatusFound, "/login?oauth_error=access_denied")
		return
	}

	// Validate state parameter
	state := c.Query("state")
	if state == "" {
		c.Redirect(http.StatusFound, "/login?oauth_error=invalid_state")
		return
	}

	// Retrieve and validate state from database
	oauthState, err := h.stateRepo.GetByState(c.Request.Context(), state)
	if err != nil || oauthState == nil {
		c.Redirect(http.StatusFound, "/login?oauth_error=invalid_state")
		return
	}

	if oauthState.IsExpired() {
		// Clean up expired state
		_ = h.stateRepo.Delete(c.Request.Context(), state)
		c.Redirect(http.StatusFound, "/login?oauth_error=invalid_state")
		return
	}

	if oauthState.Provider != "github" {
		c.Redirect(http.StatusFound, "/login?oauth_error=invalid_state")
		return
	}

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusFound, "/login?oauth_error=access_denied")
		return
	}

	// Clean up used state
	_ = h.stateRepo.Delete(c.Request.Context(), state)

	// Exchange code for access token
	accessToken, err := h.exchangeGitHubCode(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?oauth_error=github_error")
		return
	}

	// Get GitHub user profile
	githubUser, err := h.getGitHubUser(c.Request.Context(), accessToken)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?oauth_error=github_error")
		return
	}

	// Find or create user
	user, err := h.findOrCreateUser(c.Request.Context(), githubUser)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?oauth_error=github_error")
		return
	}

	// Set auth cookies
	h.setAuthCookies(c, user)

	// Redirect to frontend
	c.Redirect(http.StatusFound, "/")
}

// exchangeGitHubCode exchanges the authorization code for an access token
func (h *OAuthHandler) exchangeGitHubCode(ctx context.Context, code string) (string, error) {
	tokenURL := fmt.Sprintf("%s/login/oauth/access_token", h.githubBaseURL)

	data := url.Values{}
	data.Set("client_id", h.cfg.OAuth.GitHubClientID)
	data.Set("client_secret", h.cfg.OAuth.GitHubClientSecret)
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	if err != nil {
		return "", err
	}
	req.URL.RawQuery = data.Encode()
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub token exchange failed: %s", string(body))
	}

	var tokenResp GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// getGitHubUser fetches the user profile from GitHub
func (h *OAuthHandler) getGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	userURL := fmt.Sprintf("%s/user", h.githubBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", userURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub user fetch failed: %d - %s", resp.StatusCode, string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// findOrCreateUser finds an existing user or creates a new one from GitHub profile
func (h *OAuthHandler) findOrCreateUser(ctx context.Context, githubUser *GitHubUser) (*model.User, error) {
	githubID := fmt.Sprintf("%d", githubUser.ID)

	// First, try to find user by GitHub ID
	user, err := h.userRepo.GetByGitHubID(ctx, githubID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		// Update user profile from GitHub
		h.updateUserFromGitHub(user, githubUser)
		if err := h.userRepo.Update(ctx, user); err != nil {
			return nil, err
		}
		return user, nil
	}

	// Try to find user by email (for account linking)
	if githubUser.Email != "" {
		user, err = h.userRepo.GetByEmail(ctx, githubUser.Email)
		if err != nil {
			return nil, err
		}
		if user != nil {
			// Link GitHub account to existing user
			user.GitHubID = githubID
			user.AuthProvider = "github" // Update auth provider
			h.updateUserFromGitHub(user, githubUser)
			if err := h.userRepo.Update(ctx, user); err != nil {
				return nil, err
			}
			return user, nil
		}
	}

	// Create new user
	nickname := githubUser.Name
	if nickname == "" {
		nickname = githubUser.Login
	}

	user = &model.User{
		Email:        githubUser.Email,
		Nickname:     nickname,
		AvatarURL:    githubUser.AvatarURL,
		AuthProvider: "github",
		GitHubID:     githubID,
		PasswordHash: "", // No password for OAuth users
	}
	if err := user.GenerateID(); err != nil {
		return nil, err
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// updateUserFromGitHub updates user fields from GitHub profile
func (h *OAuthHandler) updateUserFromGitHub(user *model.User, githubUser *GitHubUser) {
	if githubUser.Name != "" {
		user.Nickname = githubUser.Name
	} else if githubUser.Login != "" {
		user.Nickname = githubUser.Login
	}
	if githubUser.AvatarURL != "" {
		user.AvatarURL = githubUser.AvatarURL
	}
	if githubUser.Email != "" {
		user.Email = githubUser.Email
	}
	user.AuthProvider = "github"
}

// setAuthCookies generates tokens and sets auth cookies
func (h *OAuthHandler) setAuthCookies(c *gin.Context, user *model.User) {
	accessToken, csrfToken, err := h.jwtService.GenerateAccessToken(user.ID)
	if err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to generate access token",
		})
		return
	}

	refreshToken, tokenHash, err := h.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to generate refresh token",
		})
		return
	}

	refreshTTL, err := h.cfg.GetRefreshTTL()
	if err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to parse refresh token TTL",
		})
		return
	}

	refreshModel := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(refreshTTL),
	}
	if err := refreshModel.GenerateID(); err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to generate refresh token ID",
		})
		return
	}

	if err := h.tokenRepo.Create(c.Request.Context(), refreshModel); err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to store refresh token",
		})
		return
	}

	cookieCfg := cookie.DefaultConfig(h.cfg.IsProduction())
	cookie.SetAccessToken(c.Writer, accessToken, csrfToken, cookieCfg)
	cookie.SetRefreshToken(c.Writer, refreshToken, cookieCfg)
}

// getRedirectURI constructs the redirect URI for OAuth callbacks
func (h *OAuthHandler) getRedirectURI(c *gin.Context, provider string) string {
	scheme := "http"
	if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	host := c.Request.Host
	return fmt.Sprintf("%s://%s/api/v1/auth/%s/callback", scheme, host, provider)
}

// GoogleInitiate handles Google OAuth initiation (reserved, returns 501)
func (h *OAuthHandler) GoogleInitiate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Google OAuth is not yet implemented",
	})
}

// GoogleCallback handles Google OAuth callback (reserved, returns 501)
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Google OAuth is not yet implemented",
	})
}

// AppleInitiate handles Apple OAuth initiation (reserved, returns 501)
func (h *OAuthHandler) AppleInitiate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Apple OAuth is not yet implemented",
	})
}

// AppleCallback handles Apple OAuth callback (reserved, returns 501)
func (h *OAuthHandler) AppleCallback(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Apple OAuth is not yet implemented",
	})
}
