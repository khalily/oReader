package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnection(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expectError bool
	}{
		{
			name:        "sqlite in-memory database",
			dsn:         ":memory:",
			expectError: false,
		},
		{
			name:        "sqlite file database",
			dsn:         "test.db",
			expectError: false,
		},
		{
			name:        "empty dsn",
			dsn:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewConnection(tt.dsn)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, db)

			// Clean up
			sqlDB, err := db.DB()
			if err == nil {
				sqlDB.Close()
			}

			if tt.dsn == "test.db" {
				os.Remove("test.db")
			}
		})
	}
}

func TestConnectionPool(t *testing.T) {
	db, err := NewConnection(":memory:")
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Test pool settings
	stats := sqlDB.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0)
}
