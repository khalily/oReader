package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"oreader/internal/config"
	"oreader/internal/infra/cookie"
	"oreader/internal/infra/errors"
	"oreader/internal/infra/jwt"
	"oreader/internal/infra/password"
	"oreader/internal/model"
	"oreader/internal/service"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Handler handles authentication endpoints
type Handler struct {
	cfg         *config.Config
	jwtService  *jwt.Service
	userRepo    service.UserRepository
	tokenRepo   service.RefreshTokenRepository
}

// NewHandler creates a new auth handler
func NewHandler(cfg *config.Config, jwtService *jwt.Service, userRepo service.UserRepository, tokenRepo service.RefreshTokenRepository) *Handler {
	return &Handler{
		cfg:        cfg,
		jwtService: jwtService,
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname,omitempty"`
}

// Validate validates the register request
func (r *RegisterRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.Nickname = strings.TrimSpace(r.Nickname)

	if r.Email == "" {
		return errors.NewSimple(errors.ErrValidation, "Email is required")
	}
	if !emailRegex.MatchString(r.Email) {
		return errors.NewSimple(errors.ErrValidation, "Invalid email format")
	}
	if len(r.Password) < 8 {
		return errors.NewSimple(errors.ErrValidation, "Password must be at least 8 characters")
	}
	return nil
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendBadRequest(c, errors.Error{
			Code:    errors.ErrValidation,
			Message: "Invalid request body",
		})
		return
	}

	if err := req.Validate(); err != nil {
		errors.SendBadRequest(c, err.(errors.Error))
		return
	}

	// Check if user already exists
	existing, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to check existing user",
		})
		return
	}
	if existing != nil {
		errors.SendConflict(c, errors.Error{
			Code:    errors.ErrConflict,
			Message: "User already exists",
		})
		return
	}

	// Hash password
	passwordHash, err := password.Hash(req.Password)
	if err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to hash password",
		})
		return
	}

	// Create user
	user := &model.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		Nickname:     req.Nickname,
		AuthProvider: "email",
	}
	if err := user.GenerateID(); err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to generate user ID",
		})
		return
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		errors.SendInternal(c, errors.Error{
			Code:    errors.ErrInternal,
			Message: "Failed to create user",
		})
		return
	}

	// Generate tokens for the new user
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

	c.JSON(http.StatusCreated, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"nickname": user.Nickname,
		},
		"csrf_token": csrfToken,
	})
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates the login request
func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))

	if r.Email == "" {
		return errors.NewSimple(errors.ErrValidation, "Email is required")
	}
	if r.Password == "" {
		return errors.NewSimple(errors.ErrValidation, "Password is required")
	}
	return nil
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendBadRequest(c, errors.Error{
			Code:    errors.ErrValidation,
			Message: "Invalid request body",
		})
		return
	}

	if err := req.Validate(); err != nil {
		errors.SendBadRequest(c, err.(errors.Error))
		return
	}

	// Get user
	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil || user == nil {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "Invalid credentials",
		})
		return
	}

	// Verify password
	valid, err := password.Verify(req.Password, user.PasswordHash)
	if err != nil || !valid {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "Invalid credentials",
		})
		return
	}

	// Generate tokens
	h.setAuthCookies(c, user)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"nickname": user.Nickname,
		},
	})
}

// setAuthCookies generates tokens and sets auth cookies
func (h *Handler) setAuthCookies(c *gin.Context, user *model.User) {
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

	// Return user info and CSRF token in response body
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"nickname": user.Nickname,
		},
		"csrf_token": csrfToken,
	})
}

// Refresh handles token refresh
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken := cookie.GetRefreshToken(c.Request)
	if refreshToken == "" {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "Refresh token required",
		})
		return
	}

	// Validate refresh token
	claims, err := h.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "Invalid refresh token",
		})
		return
	}

	// Hash token for lookup
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Check if token exists and is valid
	storedToken, err := h.tokenRepo.GetByTokenHash(c.Request.Context(), tokenHash)
	if err != nil || storedToken == nil {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "Refresh token not found",
		})
		return
	}

	if !storedToken.IsValid() {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "Refresh token has been revoked or expired",
		})
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(c.Request.Context(), claims.UserID)
	if err != nil || user == nil {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "User not found",
		})
		return
	}

	// Revoke old token
	if err := h.tokenRepo.Revoke(c.Request.Context(), tokenHash); err != nil {
		// Log but continue
	}

	// Generate new tokens
	h.setAuthCookies(c, user)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"nickname": user.Nickname,
		},
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	refreshToken := cookie.GetRefreshToken(c.Request)
	if refreshToken != "" {
		// Hash token for lookup
		hash := sha256.Sum256([]byte(refreshToken))
		tokenHash := hex.EncodeToString(hash[:])

		// Revoke the token
		_ = h.tokenRepo.Revoke(c.Request.Context(), tokenHash)
	}

	// Clear cookies
	cookieCfg := cookie.DefaultConfig(h.cfg.IsProduction())
	cookie.ClearTokens(c.Writer, cookieCfg)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// Me returns the current authenticated user
func (h *Handler) Me(c *gin.Context) {
	accessToken := cookie.GetAccessToken(c.Request)
	if accessToken == "" {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "Access token required",
		})
		return
	}

	claims, err := h.jwtService.ValidateAccessToken(accessToken)
	if err != nil {
		if err == jwt.ErrTokenExpired {
			errors.SendUnauthorized(c, errors.Error{
				Code:    errors.ErrTokenExpired,
				Message: "Access token has expired",
			})
		} else {
			errors.SendUnauthorized(c, errors.Error{
				Code:    errors.ErrTokenInvalid,
				Message: "Invalid access token",
			})
		}
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), claims.UserID)
	if err != nil || user == nil {
		errors.SendUnauthorized(c, errors.Error{
			Code:    errors.ErrUnauthorized,
			Message: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"nickname":      user.Nickname,
			"avatar_url":    user.AvatarURL,
			"auth_provider": user.AuthProvider,
		},
	})
}
