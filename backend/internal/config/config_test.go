package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "default configuration",
			envVars: map[string]string{
				"DATABASE_URL":    "test.db",
				"JWT_SECRET_KEY": "test-secret-key-must-be-at-least-32-chars",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test.db", cfg.Database.URL)
				assert.Equal(t, "development", cfg.Server.Env)
				assert.Equal(t, 8080, cfg.Server.Port)
				assert.Equal(t, "15m", cfg.Auth.AccessTTL)
				assert.Equal(t, "168h", cfg.Auth.RefreshTTL)
				assert.Equal(t, "15m", cfg.Refresh.Interval)
				assert.True(t, cfg.RateLimit.Enabled)
				assert.Equal(t, "info", cfg.Logging.Level)
			},
		},
		{
			name: "custom configuration",
			envVars: map[string]string{
				"DATABASE_URL":      "mysql://user:pass@localhost/db",
				"JWT_SECRET_KEY":    "custom-secret-key-must-be-at-least-32-chars",
				"ENV":               "production",
				"PORT":              "3000",
				"JWT_ACCESS_TTL":    "30m",
				"JWT_REFRESH_TTL":   "72h",
				"REFRESH_INTERVAL":  "30m",
				"RATE_LIMIT_ENABLED": "false",
				"LOG_LEVEL":         "debug",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "mysql://user:pass@localhost/db", cfg.Database.URL)
				assert.Equal(t, "production", cfg.Server.Env)
				assert.Equal(t, 3000, cfg.Server.Port)
				assert.Equal(t, "30m", cfg.Auth.AccessTTL)
				assert.Equal(t, "72h", cfg.Auth.RefreshTTL)
				assert.Equal(t, "30m", cfg.Refresh.Interval)
				assert.False(t, cfg.RateLimit.Enabled)
				assert.Equal(t, "debug", cfg.Logging.Level)
			},
		},
		{
			name: "missing required DATABASE_URL",
			envVars: map[string]string{
				"JWT_SECRET_KEY": "test-secret-key-must-be-at-least-32-chars",
			},
			expectError: true,
		},
		{
			name: "missing required JWT_SECRET_KEY",
			envVars: map[string]string{
				"DATABASE_URL": "test.db",
			},
			expectError: true,
		},
		{
			name: "JWT secret too short",
			envVars: map[string]string{
				"DATABASE_URL":    "test.db",
				"JWT_SECRET_KEY": "too-short",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnvVars()

			// Set test environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			cfg, err := Load()

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Database: DatabaseConfig{URL: "test.db"},
				Auth:     AuthConfig{SecretKey: "test-secret-key-must-be-at-least-32-chars"},
			},
			expectError: false,
		},
		{
			name: "missing database URL",
			config: &Config{
				Database: DatabaseConfig{URL: ""},
				Auth:     AuthConfig{SecretKey: "test-secret-key-must-be-at-least-32-chars"},
			},
			expectError: true,
		},
		{
			name: "JWT secret too short",
			config: &Config{
				Database: DatabaseConfig{URL: "test.db"},
				Auth:     AuthConfig{SecretKey: "short"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPaperConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "test.db")
	t.Setenv("JWT_SECRET_KEY", "super-secret-key-at-least-32-characters!!")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "localhost:50051", cfg.Paper.GRPCAddr)
	assert.Equal(t, "uploads/papers", cfg.Paper.UploadDir)
	assert.Equal(t, int64(50*1024*1024), cfg.Paper.MaxUploadSize)
	assert.Equal(t, "5m", cfg.Paper.GRPCTimeout)
}

func TestPaperConfigFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "test.db")
	t.Setenv("JWT_SECRET_KEY", "super-secret-key-at-least-32-characters!!")
	t.Setenv("PAPER_GRPC_ADDR", "paper-service:50051")
	t.Setenv("PAPER_UPLOAD_DIR", "/data/papers")
	t.Setenv("PAPER_MAX_UPLOAD_SIZE", "104857600")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "paper-service:50051", cfg.Paper.GRPCAddr)
	assert.Equal(t, "/data/papers", cfg.Paper.UploadDir)
	assert.Equal(t, int64(104857600), cfg.Paper.MaxUploadSize)
}

func clearEnvVars() {
	envVars := []string{
		"DATABASE_URL",
		"JWT_SECRET_KEY",
		"ENV",
		"PORT",
		"JWT_ACCESS_TTL",
		"JWT_REFRESH_TTL",
		"REFRESH_INTERVAL",
		"RATE_LIMIT_ENABLED",
		"RATE_LIMIT_REDIS_URL",
		"LOG_LEVEL",
		"GITHUB_CLIENT_ID",
		"GITHUB_CLIENT_SECRET",
	}
	for _, v := range envVars {
		_ = os.Unsetenv(v)
	}
}
