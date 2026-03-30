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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/khalily/oreader/internal/config"
	"github.com/khalily/oreader/internal/infra/cookie"
	"github.com/khalily/oreader/internal/infra/errors"
	"github.com/khalily/oreader/internal/infra/jwt"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// OAuthHandler handles OAuth authentication endpoints
type OAuthHandler struct {
	cfg              *config.Config
	jwtService       *jwt.Service
	userRepo         service.UserRepository
	tokenRepo        service.RefreshTokenRepository
	stateRepo        service.OAuthStateRepository
	pendingOAuthRepo service.PendingOAuthRepository // For pending OAuth sessions
	githubBaseURL    string                         // For OAuth web flow (https://github.com)
	githubAPIURL     string                         // For REST API calls (https://api.github.com)
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(
	cfg *config.Config,
	jwtService *jwt.Service,
	userRepo service.UserRepository,
	tokenRepo service.RefreshTokenRepository,
	stateRepo service.OAuthStateRepository,
	pendingOAuthRepo service.PendingOAuthRepository,
	githubBaseURL string,
) *OAuthHandler {
	// Use default GitHub URL if not provided (for testing)
	if githubBaseURL == "" {
		githubBaseURL = "https://github.com"
	}

	// GitHub API uses a different base URL than OAuth
	// In production: OAuth uses github.com, API uses api.github.com
	// In testing: both use the same mock server URL
	githubAPIURL := "https://api.github.com"
	if githubBaseURL != "https://github.com" {
		// Testing mode: use same URL for both
		githubAPIURL = githubBaseURL
	}

	return &OAuthHandler{
		cfg:              cfg,
		jwtService:       jwtService,
		userRepo:         userRepo,
		tokenRepo:        tokenRepo,
		stateRepo:        stateRepo,
		pendingOAuthRepo: pendingOAuthRepo,
		githubBaseURL:    githubBaseURL,
		githubAPIURL:     githubAPIURL,
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

// GitHubEmail represents an email item from /user/emails
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// getBestEmail selects the best email from a list
// Priority: primary && verified > verified > primary > first
func getBestEmail(emails []GitHubEmail) string {
	var verified, primary string
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email // Best choice
		}
		if e.Verified && verified == "" {
			verified = e.Email
		}
		if e.Primary && primary == "" {
			primary = e.Email
		}
	}
	if verified != "" {
		return verified
	}
	if primary != "" {
		return primary
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
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
// Supports format=json query parameter to return JSON instead of 302 redirect
func (h *OAuthHandler) GitHubCallback(c *gin.Context) {
	// Check if client wants JSON response (for AJAX-based OAuth flow)
	wantJSON := c.Query("format") == "json"

	// Helper functions for response
	respondError := func(errorCode string) {
		if wantJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorCode})
		} else {
			c.Redirect(http.StatusFound, "/login?oauth_error="+errorCode)
		}
	}

	respondSuccess := func(user *model.User, csrfToken string) {
		if wantJSON {
			c.JSON(http.StatusOK, gin.H{
				"user": gin.H{
					"id":            user.ID,
					"email":         user.Email,
					"nickname":      user.Nickname,
					"avatar_url":    user.AvatarURL,
					"auth_provider": user.AuthProvider,
					"created_at":    user.CreatedAt,
					"updated_at":    user.UpdatedAt,
				},
				"csrf_token": csrfToken,
			})
		} else {
			c.Redirect(http.StatusFound, "/")
		}
	}

	respondPending := func(token string) {
		if wantJSON {
			c.JSON(http.StatusOK, gin.H{
				"redirect":      "pending",
				"pending_token": token,
			})
		} else {
			callbackHost := h.cfg.OAuth.GitHubCallbackHost
			if callbackHost == "" {
				callbackHost = h.cfg.Server.FrontendURL
			}
			c.Redirect(http.StatusFound, callbackHost+"/oauth/pending?token="+token)
		}
	}

	// Check for error response from GitHub
	if errorCode := c.Query("error"); errorCode != "" {
		respondError("access_denied")
		return
	}

	// Validate state parameter
	state := c.Query("state")
	if state == "" {
		respondError("invalid_state")
		return
	}

	// Retrieve and validate state from database
	oauthState, err := h.stateRepo.GetByState(c.Request.Context(), state)
	if err != nil || oauthState == nil {
		respondError("invalid_state")
		return
	}

	if oauthState.IsExpired() {
		// Clean up expired state
		_ = h.stateRepo.Delete(c.Request.Context(), state)
		respondError("invalid_state")
		return
	}

	if oauthState.Provider != "github" {
		respondError("invalid_state")
		return
	}

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		respondError("access_denied")
		return
	}

	// Clean up used state
	_ = h.stateRepo.Delete(c.Request.Context(), state)

	// Get redirect URI for token exchange (must match the one used in authorization)
	redirectURI := h.getRedirectURI(c, "github")
	log.Debug().Str("redirect_uri", redirectURI).Msg("GitHub OAuth callback - exchanging code for token")

	// Exchange code for access token
	accessToken, err := h.exchangeGitHubCode(c.Request.Context(), code, redirectURI)
	if err != nil {
		log.Error().Err(err).Msg("GitHub token exchange failed")
		respondError("github_error")
		return
	}

	// Get GitHub user profile
	log.Debug().Msg("Fetching GitHub user profile")
	githubUser, err := h.getGitHubUser(c.Request.Context(), accessToken)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch GitHub user profile")
		respondError("github_error")
		return
	}
	log.Debug().Str("github_id", fmt.Sprintf("%d", githubUser.ID)).Str("email", githubUser.Email).Msg("GitHub user profile fetched")

	// 1. Get email - if not in profile, fetch from /user/emails API
	email := githubUser.Email
	if email == "" {
		emails, err := h.fetchGitHubEmails(c.Request.Context(), accessToken)
		if err == nil && len(emails) > 0 {
			email = getBestEmail(emails)
		}
	}
	log.Debug().Str("email", email).Msg("Resolved email for GitHub user")

	// 2. Try to find existing user by GitHub ID
	githubIDStr := fmt.Sprintf("%d", githubUser.ID)
	user, err := h.userRepo.GetByGitHubID(c.Request.Context(), githubIDStr)
	if err == nil && user != nil {
		// Update GitHubLogin if changed
		if user.GitHubLogin != githubUser.Login {
			user.GitHubLogin = githubUser.Login
			_ = h.userRepo.Update(c.Request.Context(), user)
		}
		csrfToken, success := h.setAuthCookiesAndGetCSRF(c, user)
		if !success {
			respondError("github_error")
			return
		}
		respondSuccess(user, csrfToken)
		return
	}

	// 3. Not found by GitHub ID, check email for account linking or new user
	if email != "" {
		user, err := h.userRepo.GetByEmail(c.Request.Context(), email)
		if err == nil && user != nil {
			// Auto-link: connect GitHub to existing account
			user.GitHubID = githubIDStr
			user.GitHubLogin = githubUser.Login
			if user.AvatarURL == "" {
				user.AvatarURL = githubUser.AvatarURL
			}
			_ = h.userRepo.Update(c.Request.Context(), user)
			csrfToken, success := h.setAuthCookiesAndGetCSRF(c, user)
			if !success {
				respondError("github_error")
				return
			}
			respondSuccess(user, csrfToken)
			return
		}

		// Create new user with email
		newUser := &model.User{
			Email:        email,
			Nickname:     getNickname(githubUser),
			AvatarURL:    githubUser.AvatarURL,
			AuthProvider: "github",
			GitHubID:     githubIDStr,
			GitHubLogin:  githubUser.Login,
		}
		if err := newUser.GenerateID(); err != nil {
			log.Error().Err(err).Msg("Failed to generate user ID")
			respondError("github_error")
			return
		}
		if err := h.userRepo.Create(c.Request.Context(), newUser); err != nil {
			log.Error().Err(err).Msg("Failed to create user")
			respondError("github_error")
			return
		}
		csrfToken, success := h.setAuthCookiesAndGetCSRF(c, newUser)
		if !success {
			respondError("github_error")
			return
		}
		respondSuccess(newUser, csrfToken)
		return
	}

	// 4. No email available - create PendingOAuth and redirect to pending page
	token, err := generateState()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate pending token")
		respondError("github_error")
		return
	}

	pending := &model.PendingOAuth{
		Token:       token,
		GitHubID:    githubIDStr,
		GitHubLogin: githubUser.Login,
		Nickname:    getNickname(githubUser),
		AvatarURL:   githubUser.AvatarURL,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	if err := pending.GenerateID(); err != nil {
		log.Error().Err(err).Msg("Failed to generate pending OAuth ID")
		respondError("github_error")
		return
	}

	if err := h.pendingOAuthRepo.Create(c.Request.Context(), pending); err != nil {
		log.Error().Err(err).Msg("Failed to create pending OAuth")
		respondError("github_error")
		return
	}

	respondPending(token)
}

// exchangeGitHubCode exchanges the authorization code for an access token
func (h *OAuthHandler) exchangeGitHubCode(ctx context.Context, code string, redirectURI string) (string, error) {
	tokenURL := fmt.Sprintf("%s/login/oauth/access_token", h.githubBaseURL)

	data := url.Values{}
	data.Set("client_id", h.cfg.OAuth.GitHubClientID)
	data.Set("client_secret", h.cfg.OAuth.GitHubClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI) // Required: must match the redirect_uri used in authorization

	log.Debug().Str("url", tokenURL).Str("client_id", h.cfg.OAuth.GitHubClientID).Msg("Exchanging GitHub code for token")

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("GitHub token exchange failed")
		return "", fmt.Errorf("GitHub token exchange failed: %s", string(body))
	}

	var tokenResp GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	log.Debug().Msg("GitHub token exchange successful")
	return tokenResp.AccessToken, nil
}

// getGitHubUser fetches the user profile from GitHub
func (h *OAuthHandler) getGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	// Use GitHub API URL (api.github.com), not OAuth URL (github.com)
	userURL := fmt.Sprintf("%s/user", h.githubAPIURL)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("GitHub user fetch failed")
		return nil, fmt.Errorf("GitHub user fetch failed: %d - %s", resp.StatusCode, string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// fetchGitHubEmails fetches the user's email list from GitHub /user/emails API
func (h *OAuthHandler) fetchGitHubEmails(ctx context.Context, accessToken string) ([]GitHubEmail, error) {
	url := h.githubAPIURL + "/user/emails"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("GitHub emails fetch failed")
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}

// findOrCreateUser finds an existing user or creates a new one from GitHub profile
//
//nolint:unused // reserved for future OAuth provider expansion
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
//
//nolint:unused // reserved for future OAuth provider expansion
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

// setAuthCookiesAndGetCSRF generates tokens, sets auth cookies, and returns the CSRF token
// This is used for JSON-based OAuth flow where we need to return the CSRF token in the response
// Returns (csrfToken, success) - caller must check success before proceeding
func (h *OAuthHandler) setAuthCookiesAndGetCSRF(c *gin.Context, user *model.User) (csrfToken string, success bool) {
	accessToken, csrfToken, err := h.jwtService.GenerateAccessToken(user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		return "", false
	}

	refreshToken, tokenHash, err := h.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		return "", false
	}

	refreshTTL, err := h.cfg.GetRefreshTTL()
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse refresh TTL")
		return "", false
	}

	refreshModel := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(refreshTTL),
	}
	if err := refreshModel.GenerateID(); err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token ID")
		return "", false
	}

	if err := h.tokenRepo.Create(c.Request.Context(), refreshModel); err != nil {
		log.Error().Err(err).Msg("Failed to store refresh token")
		return "", false
	}

	cookieCfg := cookie.DefaultConfig(h.cfg.IsProduction())
	cookie.SetAccessToken(c.Writer, accessToken, csrfToken, cookieCfg)
	cookie.SetRefreshToken(c.Writer, refreshToken, cookieCfg)

	return csrfToken, true
}

// getRedirectURI constructs the redirect URI for OAuth callbacks
func (h *OAuthHandler) getRedirectURI(c *gin.Context, provider string) string {
	scheme := "http"
	if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	// Prefer configured callback host (useful when behind proxy with changeOrigin)
	host := h.cfg.OAuth.GitHubCallbackHost
	if host == "" {
		host = c.Request.Host // Fallback to request Host header
	}

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

// GetPendingOAuth handles GET /api/v1/auth/oauth/pending
// Returns pending OAuth information for email completion flow
func (h *OAuthHandler) GetPendingOAuth(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "token is required",
		})
		return
	}

	pending, err := h.pendingOAuthRepo.GetByToken(c.Request.Context(), token)
	if err != nil || pending == nil || pending.IsExpired() {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "token not found or expired",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"github_login": pending.GitHubLogin,
		"nickname":     pending.Nickname,
		"avatar_url":   pending.AvatarURL,
	})
}

// CompleteOAuthRequest represents the request body for CompleteOAuth
type CompleteOAuthRequest struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// CompleteOAuth handles POST /api/v1/auth/oauth/complete
// Completes user registration with email for OAuth users without public email
func (h *OAuthHandler) CompleteOAuth(c *gin.Context) {
	var req CompleteOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Validate token
	if req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "token is required",
		})
		return
	}

	// Validate email format
	if req.Email == "" || !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "valid email is required",
		})
		return
	}

	// Get pending OAuth data
	pending, err := h.pendingOAuthRepo.GetByToken(c.Request.Context(), req.Token)
	if err != nil || pending == nil || pending.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "token not found or expired",
		})
		return
	}

	// Check if email is already used
	existingUser, _ := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "EMAIL_ALREADY_USED",
				"message": "该邮箱已被注册，请使用其他邮箱",
			},
		})
		return
	}

	// Create new user
	user := &model.User{
		Email:        req.Email,
		Nickname:     pending.Nickname,
		AvatarURL:    pending.AvatarURL,
		AuthProvider: "github",
		GitHubID:     pending.GitHubID,
		GitHubLogin:  pending.GitHubLogin,
	}
	if err := user.GenerateID(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	// Delete pending record
	_ = h.pendingOAuthRepo.Delete(c.Request.Context(), req.Token)

	// Set auth cookies (same flow as existing OAuth login)
	h.setAuthCookies(c, user)

	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"email":         user.Email,
		"nickname":      user.Nickname,
		"avatar_url":    user.AvatarURL,
		"auth_provider": user.AuthProvider,
	})
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	if len(email) > 255 {
		return false
	}
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// getNickname returns the best nickname from GitHub user profile
func getNickname(user *GitHubUser) string {
	if user.Name != "" {
		return user.Name
	}
	return user.Login
}
