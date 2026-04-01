package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	Refresh   RefreshConfig
	RateLimit RateLimitConfig
	Logging   LoggingConfig
	OAuth     OAuthConfig
	Paper     PaperConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Env         string
	Port        int
	FrontendURL string // Frontend URL for dev mode redirects (e.g., "http://localhost:5173")
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	SecretKey   string
	AccessTTL   string
	RefreshTTL  string
}

// RefreshConfig holds RSS refresh configuration
type RefreshConfig struct {
	Interval    string
	MaxBodySize int64 // Maximum feed response body size in bytes
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled  bool
	RedisURL string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string
}

// OAuthConfig holds OAuth provider configuration
type OAuthConfig struct {
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackHost string // OAuth callback host (e.g., "10.37.126.68:8080") - used when behind proxy
}

// PaperConfig holds paper import configuration
type PaperConfig struct {
	GRPCAddr      string
	UploadDir     string
	MaxUploadSize int64
	GRPCTimeout   string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Use environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	v.SetDefault("ENV", "development")
	v.SetDefault("PORT", 8080)
	v.SetDefault("FRONTEND_URL", "http://localhost:5173")
	v.SetDefault("JWT_ACCESS_TTL", "15m")
	v.SetDefault("JWT_REFRESH_TTL", "168h")
	v.SetDefault("REFRESH_INTERVAL", "15m")
	v.SetDefault("FEED_MAX_BODY_SIZE", 5242880) // 5MB
	v.SetDefault("RATE_LIMIT_ENABLED", true)
	v.SetDefault("LOG_LEVEL", "info")

	// Paper defaults
	v.SetDefault("PAPER_GRPC_ADDR", "localhost:50051")
	v.SetDefault("PAPER_UPLOAD_DIR", "uploads/papers")
	v.SetDefault("PAPER_MAX_UPLOAD_SIZE", 52428800) // 50MB
	v.SetDefault("PAPER_GRPC_TIMEOUT", "30m")

	// Bind environment variables
	_ = v.BindEnv("DATABASE_URL")
	_ = v.BindEnv("JWT_SECRET_KEY")
	_ = v.BindEnv("ENV")
	_ = v.BindEnv("PORT")
	_ = v.BindEnv("FRONTEND_URL")
	_ = v.BindEnv("JWT_ACCESS_TTL")
	_ = v.BindEnv("JWT_REFRESH_TTL")
	_ = v.BindEnv("REFRESH_INTERVAL")
	_ = v.BindEnv("FEED_MAX_BODY_SIZE")
	_ = v.BindEnv("RATE_LIMIT_ENABLED")
	_ = v.BindEnv("RATE_LIMIT_REDIS_URL")
	_ = v.BindEnv("LOG_LEVEL")
	_ = v.BindEnv("GITHUB_CLIENT_ID")
	_ = v.BindEnv("GITHUB_CLIENT_SECRET")
	_ = v.BindEnv("GITHUB_CALLBACK_HOST")
	_ = v.BindEnv("PAPER_GRPC_ADDR")
	_ = v.BindEnv("PAPER_UPLOAD_DIR")
	_ = v.BindEnv("PAPER_MAX_UPLOAD_SIZE")
	_ = v.BindEnv("PAPER_GRPC_TIMEOUT")

	cfg := &Config{
		Server: ServerConfig{
			Env:         v.GetString("ENV"),
			Port:        v.GetInt("PORT"),
			FrontendURL: v.GetString("FRONTEND_URL"),
		},
		Database: DatabaseConfig{
			URL: v.GetString("DATABASE_URL"),
		},
		Auth: AuthConfig{
			SecretKey:  v.GetString("JWT_SECRET_KEY"),
			AccessTTL:  v.GetString("JWT_ACCESS_TTL"),
			RefreshTTL: v.GetString("JWT_REFRESH_TTL"),
		},
		Refresh: RefreshConfig{
			Interval:    v.GetString("REFRESH_INTERVAL"),
			MaxBodySize: v.GetInt64("FEED_MAX_BODY_SIZE"),
		},
		RateLimit: RateLimitConfig{
			Enabled:  v.GetBool("RATE_LIMIT_ENABLED"),
			RedisURL: v.GetString("RATE_LIMIT_REDIS_URL"),
		},
		Logging: LoggingConfig{
			Level: v.GetString("LOG_LEVEL"),
		},
		OAuth: OAuthConfig{
			GitHubClientID:     v.GetString("GITHUB_CLIENT_ID"),
			GitHubClientSecret: v.GetString("GITHUB_CLIENT_SECRET"),
			GitHubCallbackHost: v.GetString("GITHUB_CALLBACK_HOST"),
		},
		Paper: PaperConfig{
			GRPCAddr:      v.GetString("PAPER_GRPC_ADDR"),
			UploadDir:     v.GetString("PAPER_UPLOAD_DIR"),
			MaxUploadSize: v.GetInt64("PAPER_MAX_UPLOAD_SIZE"),
			GRPCTimeout:   v.GetString("PAPER_GRPC_TIMEOUT"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.Auth.SecretKey == "" {
		return fmt.Errorf("JWT_SECRET_KEY is required")
	}

	if len(c.Auth.SecretKey) < 32 {
		return fmt.Errorf("JWT_SECRET_KEY must be at least 32 characters")
	}

	return nil
}

// GetAccessTTL returns the access token TTL as a duration
func (c *Config) GetAccessTTL() (time.Duration, error) {
	return time.ParseDuration(c.Auth.AccessTTL)
}

// GetRefreshTTL returns the refresh token TTL as a duration
func (c *Config) GetRefreshTTL() (time.Duration, error) {
	return time.ParseDuration(c.Auth.RefreshTTL)
}

// GetRefreshInterval returns the refresh interval as a duration
func (c *Config) GetRefreshInterval() (time.Duration, error) {
	return time.ParseDuration(c.Refresh.Interval)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// GetGRPCTimeout returns the gRPC timeout as a duration
func (c *Config) GetGRPCTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Paper.GRPCTimeout)
}
