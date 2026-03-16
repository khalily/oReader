package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"oreader/internal/config"
	"oreader/internal/handler"
	"oreader/internal/infra/database"
	"oreader/internal/infra/jwt"
	"oreader/internal/infra/logger"
	"oreader/internal/infra/ratelimit"
	"oreader/internal/middleware"
	"oreader/internal/model"
	"oreader/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.Logging.Level, cfg.IsDevelopment())
	log.Info().Str("env", cfg.Server.Env).Msg("Starting oReader server")

	// Connect to database
	db, err := database.NewConnection(cfg.Database.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get database connection")
		os.Exit(1)
	}
	defer sqlDB.Close()

	log.Info().Str("database", cfg.Database.URL).Msg("Connected to database")

	// Auto-migrate database schemas
	if err := db.AutoMigrate(
		&model.User{},
		&model.Feed{},
		&model.UserFeed{},
		&model.Item{},
		&model.UserItemState{},
		&model.RefreshToken{},
		&model.ImportJob{},
		&model.OAuthState{},
	); err != nil {
		log.Fatal().Err(err).Msg("Failed to auto-migrate database")
		os.Exit(1)
	}
	log.Info().Msg("Database migrations completed")

	// Initialize services
	accessTTL, err := cfg.GetAccessTTL()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse access token TTL")
		os.Exit(1)
	}
	refreshTTL, err := cfg.GetRefreshTTL()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse refresh token TTL")
		os.Exit(1)
	}
	jwtService := jwt.NewService(cfg.Auth.SecretKey, accessTTL, refreshTTL)

	// Initialize rate limiter
	rateLimiter := ratelimit.NewMemoryLimiter()

	// Configure rate limits per endpoint type
	rateLimits := middleware.RouteLimits{
		Routes: map[string]middleware.RouteLimit{
			// Auth endpoints - stricter limits to prevent brute force
			"/api/v1/auth/register": {Requests: 5, Window: time.Minute},
			"/api/v1/auth/login":    {Requests: 10, Window: time.Minute},
			// Refresh endpoint - moderate limit
			"/api/v1/auth/refresh": {Requests: 20, Window: time.Minute},
		},
		Default: middleware.RouteLimit{Requests: 60, Window: time.Minute},
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)

	// Initialize handlers
	authHandler := handler.NewHandler(cfg, jwtService, userRepo, tokenRepo)

	// Setup Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Apply global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.RequestLoggerMiddleware())
	router.Use(middleware.RateLimitMiddleware(rateLimiter, rateLimits))

	// CORS configuration
	allowOrigins := []string{"*"}
	if cfg.IsDevelopment() {
		allowOrigins = []string{"http://localhost:5173", "http://localhost:3000", "*"}
	}
	router.Use(middleware.CORSMiddleware(allowOrigins))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)

			// Protected auth routes
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(jwtService))
			{
				authProtected.GET("/me", authHandler.Me)
			}
		}

		// Protected API routes (require authentication + CSRF)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(jwtService))
		protected.Use(middleware.CSRFMiddleware())
		{
			// Feed routes will be added here
			// feeds := protected.Group("/feeds")
			// {
			//     feeds.GET("", feedHandler.List)
			//     feeds.POST("", feedHandler.Create)
			//     feeds.GET("/:id", feedHandler.Get)
			//     feeds.DELETE("/:id", feedHandler.Delete)
			// }

			// Item routes will be added here
			// items := protected.Group("/items")
			// {
			//     items.GET("", itemHandler.List)
			//     items.GET("/:id", itemHandler.Get)
			//     items.POST("/:id/star", itemHandler.Star)
			//     items.POST("/:id/read", itemHandler.MarkRead)
			// }
		}

		// Optional auth routes (public but can use auth if present)
		optional := v1.Group("")
		optional.Use(middleware.OptionalAuthMiddleware(jwtService))
		{
			// Public routes that benefit from auth context will be added here
		}
	}

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Server shutdown error")
		}

		log.Info().Msg("Server stopped")
	}()

	log.Info().Str("addr", addr).Msg("Server listening")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server error")
	}
}
