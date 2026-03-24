package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"oreader/internal/config"
	"oreader/internal/handler"
	"oreader/internal/infra/database"
	"oreader/internal/infra/jwt"
	"oreader/internal/infra/logger"
	"oreader/internal/infra/ratelimit"
	"oreader/internal/infra/rss"
	"oreader/internal/middleware"
	"oreader/internal/model"
	"oreader/internal/repository"
	"oreader/internal/service"
	"oreader/internal/worker"
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

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	itemRepo := repository.NewItemRepository(db)
	userFeedRepo := repository.NewUserFeedRepository(db)
	userItemStateRepo := repository.NewUserItemStateRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	importJobRepo := repository.NewImportJobRepository(db)
	oauthStateRepo := repository.NewOAuthStateRepository(db)
	statsRepo := repository.NewStatsRepository(db)

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

	// Initialize RSS parser
	fetcher := rss.NewHTTPFetcher(30 * time.Second)
	parser := rss.NewParser(fetcher)

	// Initialize services
	feedService := service.NewFeedService(feedRepo, itemRepo, userFeedRepo, parser)
	itemService := service.NewItemService(itemRepo, userItemStateRepo, userFeedRepo)
	importService := service.NewImportService(feedService, importJobRepo, userFeedRepo, feedRepo)
	refreshWorkerService := service.NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)
	statsService := service.NewStatsService(statsRepo)

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

	// Initialize handlers
	authHandler := handler.NewHandler(cfg, jwtService, userRepo, tokenRepo)
	feedHandler := handler.NewFeedHandler(feedService)
	itemHandler := handler.NewItemHandler(itemService)
	importHandler := handler.NewImportHandler(feedService, importService, feedRepo)
	oauthHandler := handler.NewOAuthHandler(cfg, jwtService, userRepo, tokenRepo, oauthStateRepo, "")
	statsHandler := handler.NewStatsHandler(statsService)

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
			"status":  "ok",
			"version": "2.0.0",
			"time":    time.Now().Format(time.RFC3339),
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
			auth.GET("/github", oauthHandler.GitHubInitiate)
			auth.GET("/github/callback", oauthHandler.GitHubCallback)

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
			// Feed routes
			feeds := protected.Group("/feeds")
			{
				feeds.GET("", feedHandler.ListFeeds)
				feeds.POST("", feedHandler.CreateFeed)
				feeds.GET("/:id", feedHandler.GetFeed)
				feeds.DELETE("/:id", feedHandler.DeleteFeed)
				feeds.POST("/:id/refresh", feedHandler.RefreshFeed)
				feeds.POST("/:id/mark-all-read", itemHandler.MarkAllRead)
			}

			// Item routes
			items := protected.Group("/items")
			{
				items.GET("", itemHandler.ListItems)
				items.GET("/:id", itemHandler.GetItem)
				items.PUT("/:id/star", itemHandler.SetStar)
				items.PUT("/:id/read", itemHandler.SetRead)
			}

			// Stats routes
			protected.GET("/stats", statsHandler.GetStats)

			// OPML import
			opml := protected.Group("/opml")
			{
				opml.POST("/import", importHandler.ImportFeeds)
			}
		}

		// Optional auth routes (public but can use auth if present)
		optional := v1.Group("")
		optional.Use(middleware.OptionalAuthMiddleware(jwtService))
		{
			// OPML export (works with or without auth)
			optional.GET("/opml/export", importHandler.ExportFeeds)
			optional.GET("/opml/import/:job_id", importHandler.GetImportStatus)
		}
	}

	// Static file serving - use embedded files in production, local files in development
	var staticFS fs.FS
	if cfg.IsDevelopment() {
		// Development mode: try local web/dist first, fallback to API-only
		if _, err := os.Stat("web/dist"); err == nil {
			staticFS = os.DirFS("web/dist")
			log.Info().Str("path", "web/dist").Msg("Serving static files from local directory")
		} else {
			log.Info().Msg("API-only mode - run 'make frontend-dev' for frontend")
		}
	} else {
		// Production mode: must use embedded files
		staticFS = StaticFS()
		if staticFS == nil {
			log.Fatal().Msg("Production requires embedded frontend. Use 'make build-prod'")
		}
		log.Info().Msg("Serving static files from embedded filesystem")
	}

	// Create file server only if staticFS is available
	var fileServer http.Handler
	if staticFS != nil {
		fileServer = http.FileServer(http.FS(staticFS))
	}

	// Serve static assets directly (JS, CSS, images, etc.) - only if available
	if staticFS != nil {
		router.GET("/assets/*filepath", func(c *gin.Context) {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/assets")
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	// SPA fallback routing - serve index.html for all non-API routes
	// This allows React Router to handle client-side routing
	router.NoRoute(func(c *gin.Context) {
		// Skip if it's an API route that wasn't matched
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "The requested resource was not found",
				},
			})
			return
		}

		// Development mode without static files: redirect to Vite dev server
		if staticFS == nil && cfg.IsDevelopment() {
			frontendURL := cfg.Server.FrontendURL
			if frontendURL == "" {
				frontendURL = "http://localhost:5173" // Default fallback
			}
			viteURL := frontendURL + c.Request.URL.Path
			c.Redirect(http.StatusTemporaryRedirect, viteURL)
			return
		}

		// For all other routes with static files, serve index.html (SPA fallback)
		if staticFS != nil {
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// No frontend available
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "FRONTEND_NOT_AVAILABLE",
				"message": "Frontend not available. Run 'make frontend-dev' for development or 'make build-prod' for production",
			},
		})
	})

	// Start background refresh worker
	backgroundWorker := worker.NewRefreshWorker(cfg, refreshWorkerService)
	ctx := context.Background()
	if err := backgroundWorker.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start background worker")
		os.Exit(1)
	}
	log.Info().Msg("Background refresh worker started")

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Str("addr", addr).Msg("Server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Server error")
		}
	}()

	// Graceful shutdown - wait for SIGINT or SIGTERM
	backgroundWorker.WaitForShutdown()

	// Stop the background worker
	log.Info().Msg("Shutting down background refresh worker...")
	backgroundWorker.Stop()

	// Shutdown the HTTP server
	log.Info().Msg("Shutting down HTTP server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	log.Info().Msg("Server stopped")
}
