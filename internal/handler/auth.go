package handler

import (
	"net/http"
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

// Handler handles authentication endpoints
type Handler struct {
	cfg         *config.Config
	jwtService  *jwt.Service
	userRepo     service.UserRepository
	tokenRepo   service.RefreshTokenRepository
}

// NewHandler creates a new auth handler
func NewHandler(cfg *config.Config, jwtService *jwt.Service, userRepo service.UserRepository, tokenRepo service.RefreshTokenRepository) *Handler {
	return &Handler{
		cfg:         cfg,
		jwtService:   jwtService,
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
	 }
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname,omitempty"`
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

    c.JSON(http.StatusCreated, gin.H{
        "user": gin.H{
            "id":       user.ID,
            "email":  user.Email,
            "nickname": user.Nickname,
        },
    })
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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
    accessToken, csrfToken, err := h.jwtService.GenerateAccessToken(user.ID)
    if err != nil {
        errors.SendInternal(c, errors.Error{
            Code:    errors.ErrInternal,
            Message: "Failed to generate access token",
        })
        return
    }

    // Generate refresh token
    refreshToken, tokenHash, err := h.jwtService.GenerateRefreshToken(user.ID)
    if err != nil {
        errors.SendInternal(c, errors.Error{
            Code:    errors.ErrInternal,
            Message: "Failed to generate refresh token",
        })
        return
    }

    // Get refresh token TTL
    refreshTTL, err := h.cfg.GetRefreshTTL()
    if err != nil {
        errors.SendInternal(c, errors.Error{
            Code:    errors.ErrInternal,
            Message: "Failed to parse refresh token TTL",
        })
        return
    }

    // Store refresh token
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

    // Set cookies
    cookieCfg := cookie.DefaultConfig(h.cfg.IsProduction())
    cookie.SetAccessToken(c.Writer, accessToken, csrfToken, cookieCfg)
    cookie.SetRefreshToken(c.Writer, refreshToken, cookieCfg)

    c.JSON(http.StatusOK, gin.H{
        "user": gin.H{
            "id":       user.ID,
            "email":  user.Email,
            "nickname": user.Nickname,
        },
    })
}

