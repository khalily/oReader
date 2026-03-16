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
	"oreader/internal/infra/database"
	"oreader/internal/infra/logger"
	"oreader/internal/middleware"
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

	// API routes will be added here
	// TODO: Add route groups for auth, feeds, items, etc.

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
